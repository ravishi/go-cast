package main

import (
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/oleksandr/bonjour"
	"github.com/ravishi/go-cast/cast"
	"github.com/ravishi/go-cast/cast/ctrl"
	"github.com/ravishi/go-vlc"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	err := actualMain(os.Args)
	if err == nil || err == Canceled {
		os.Exit(0)
	} else {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func actualMain(args []string) error {
	if len(args) < 3 {
		return errors.New("Not enough args. Please, try: example MOVIE_FILE --any-extra-vlc-args")
	}

	resolver, err := bonjour.NewResolver(nil)
	if err != nil {
		return err
	}
	defer func() { resolver.Exit <- true }()

	services := make(chan *bonjour.ServiceEntry)

	err = resolver.Browse("_googlecast._tcp", ".local", services)
	if err != nil {
		return err
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, os.Kill)
	defer signal.Stop(sigCh)

	for {
		fmt.Println("Searching devices...")

		select {
		case service, ok := <-services:
			if ok {
				fmt.Println("Found device:", service.Instance)
				err := consumeService(sigCh, service, args)
				if err != nil {
					return err
				} else {
					log.Println("Device disconnected:", service.Instance)
				}
			} else {
				return errors.New("Discovery closed unexpectedly")
			}
		case <-sigCh:
			return nil
		}
	}
}

var Canceled = errors.New("Canceled")

func consumeService(cancel <-chan os.Signal, service *bonjour.ServiceEntry, args []string) error {
	addr := fmt.Sprintf("%s:%d", service.AddrIPv4, service.Port)

	conn, err := tls.Dial("tcp", addr, &tls.Config{
		InsecureSkipVerify: true,
	})
	if err != nil {
		return fmt.Errorf("Failed to connect: %s", err)
	}
	defer conn.Close()

	device := cast.NewDevice(conn)
	defer device.Close()

	go device.Run()

	// Send a connect in 2s
	connection := ctrl.NewConnectionController(device, "sender-0", "receiver-0")
	defer connection.Close()

	err = connection.Connect()
	if err != nil {
		return fmt.Errorf("Failed to connect: %s", err)
	}

	heartbeat := ctrl.NewHeartbeatController(device, "sender-0", "receiver-0")
	defer heartbeat.Close()

	// send a PING every 5s, bail after 10.
	heartbeatError := make(chan error)
	go func() {
		err := heartbeat.Beat(time.Second*5, 2)
		if err != nil {
			heartbeatError <- err
		}
	}()

	receiver := ctrl.NewReceiverController(device, "sender-0", "receiver-0")
	defer receiver.Close()

	go func() {
		err := playSomething(device, receiver, args, cancel)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}()

	select {
	case err := <-heartbeatError:
		return err
	case <-cancel:
		return Canceled
	}

	return nil
}

func playSomething(device *cast.Device, receiver *ctrl.ReceiverController, args []string, cancel <-chan os.Signal) error {
	appId := "CC1AD845"
	sourceId := "client-123"

	status, err := receiver.Launch(appId)
	if err != nil {
		return fmt.Errorf("Failed to launch: %s", err)
	}

	session := new(ctrl.ApplicationSession)
	for _, s := range status.Applications {
		if s.AppID == appId {
			session = &s
			break
		}
	}

	if session == nil {
		return errors.New("Apparently we couldnt launch")
	}

	mediaConnection := ctrl.NewConnectionController(device, sourceId, session.TransportId)
	defer mediaConnection.Close()

	err = mediaConnection.Connect()
	if err != nil {
		return fmt.Errorf("Failed to connect to the media receiver: %s", err)
	}

	vlc, err := vlc.NewInstance(args[2:]...)
	if err != nil {
		return fmt.Errorf("Failed to create VLC instance: %s", err)
	}

	url := args[1]
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		absUrl, err := filepath.Abs(url)
		if err != nil {
			url = absUrl
		}
	}

	vlcMedia, err := vlc.AddBroadcast(
		"foo", url,
		"#transcode{acodec=mp3,vcodec=h264}:http{dst=:8090/stream,mux=avformat{mux=matroska},access=http{mime=video/x-matroska}}",
		nil,
	)
	if err != nil {
		return fmt.Errorf("Failed to create VLC Broadcast: %s", err)
	}

	err = vlc.Play(vlcMedia)
	if err != nil {
		return fmt.Errorf("Failed to play VLC Broadcast: %s", err)
	}

	mediaInfo := ctrl.MediaInfo{
		ContentID:   "http://192.168.25.142:8090/stream",
		ContentType: "video/x-matroska",
		StreamType:  ctrl.StreamTypeLive,
		Metadata: map[string]interface{}{
			"type":         0,
			"metadataType": 0,
			"title":        "Big Buck Bunny",
		},
	}

	media := ctrl.NewMediaController(device, sourceId, session.TransportId)
	sessionStatus, err := media.Load(mediaInfo, ctrl.LoadOptions{
		AutoPlay: true,
	})

	if err != nil {
		return fmt.Errorf("Error while loading media: %s", err)
	}

	waitTime := time.Second * 2

	select {
	case <-cancel:
	case <-time.After(waitTime):
	}

	sessionId := sessionStatus[0].MediaSessionID

	for {
		sessionStatus, err = media.Play(sessionId)
		if err != nil {
			return fmt.Errorf("Failed to play: %s", err)
		}

		if sessionStatus[0].PlayerState != "PLAYING" {
			log.Println("Player still", sessionStatus[0].PlayerState, "... Trying again in", waitTime)
			if sessionStatus[0].MediaSessionID > 0 {
				sessionId = sessionStatus[0].MediaSessionID
			}
		} else {
			break
		}

		select {
		case <-cancel:
		case <-time.After(waitTime):
		}
	}

SEEK:
	select {
	case <-time.After(time.Second * 15):
		err := vlc.Seek(vlcMedia, 0.3)
		if err != nil {
			return fmt.Errorf("Failed to seek: %s", err)
		}

		// Fake seek to flush buffers (?)
		sessionStatus, err = media.Seek(sessionId, float64(time.Now().Unix()+1)/1000.0)
		if err != nil {
			return fmt.Errorf("Failed to seek: %s", err)
		}
		if sessionStatus[0].MediaSessionID > 0 {
			sessionId = sessionStatus[0].MediaSessionID
		}
		goto SEEK
	case <-cancel:
	}

	return nil
}

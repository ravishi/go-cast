package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/oleksandr/bonjour"
	"github.com/ravishi/go-cast/pkg/cast"
	"github.com/ravishi/go-cast/pkg/cast/ctrl"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	urlFlag         = kingpin.Flag("url", "Stream URL.").Short('u').Required().URL()
	titleFlag       = kingpin.Flag("title", "The title of the stream.").Short('t').String()
	contentTypeFlag = kingpin.Flag("content-type", "The content-type of the stream.").Short('c').String()
)

func main() {
	kingpin.UsageTemplate(kingpin.CompactUsageTemplate).Author("Dirley Rodrigues")
	kingpin.CommandLine.Help = "A simple command line player for your Chromecast."
	kingpin.Parse()
	err := actualMain()
	if err == nil || err == context.Canceled {
		os.Exit(0)
	} else {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func actualMain() error {
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

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		select {
		case <-sigCh:
			cancel()
		}
	}()

	for {
		fmt.Println("Searching devices...")

		select {
		case service, ok := <-services:
			if ok {
				fmt.Println("Found device:", service.Instance)
				err := consumeService(service, ctx)
				if err != nil {
					return err
				} else {
					log.Println("Device disconnected:", service.Instance)
				}
			} else {
				return errors.New("Discovery closed unexpectedly")
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func consumeService(service *bonjour.ServiceEntry, ctx context.Context) error {
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

	// Beat until it fails
	heartbeatError := make(chan error)
	go func() {
		// send a PING every 5s, bail after 10.
		err := heartbeat.Beat(time.Second*5, 2)
		if err != nil {
			heartbeatError <- err
		}
	}()

	receiver := ctrl.NewReceiverController(device, "sender-0", "receiver-0")
	defer receiver.Close()

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

	mediaInfo := ctrl.MediaInfo{
		ContentID:   (*urlFlag).String(),
		ContentType: *contentTypeFlag,
		StreamType:  ctrl.StreamTypeBuffered,
		Metadata: map[string]interface{}{
			"type":         0,
			"metadataType": 0,
			"title":        *titleFlag,
		},
	}

	media := ctrl.NewMediaController(device, sourceId, session.TransportId)
	_, err = media.Load(mediaInfo, ctrl.LoadOptions{
		AutoPlay: true,
	})
	if err != nil {
		return fmt.Errorf("Error while loading media: %s", err)
	}

	select {
	case <-ctx.Done():
		return nil
	}
}

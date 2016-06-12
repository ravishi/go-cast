package main

import (
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/oleksandr/bonjour"
	"github.com/ravishi/go-cast/cast"
	"github.com/ravishi/go-cast/cast/ctrl"
	"log"
	"os"
	"os/signal"
	"time"
)

func main() {
	err := actualMain()
	if err == nil || err == Canceled {
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

	for {
		fmt.Println("Searching devices...")

		select {
		case service, ok := <-services:
			if ok {
				fmt.Println("Found device:", service.Instance)
				err := consumeService(sigCh, service)
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

func consumeService(cancel <-chan os.Signal, service *bonjour.ServiceEntry) error {
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
		appId := "CC1AD845"
		sourceId := "client-123"

		status, err := receiver.Launch(appId)
		if err != nil {
			log.Println("Failed to launch:", err)
			return
		}

		session := new(ctrl.ApplicationSession)
		for _, s := range status.Applications {
			if s.AppID == appId {
				session = &s
				break
			}
		}

		if session == nil {
			log.Println("Apparently we couldnt launch")
			return
		}

		mediaConnection := ctrl.NewConnectionController(device, sourceId, session.TransportId)
		defer mediaConnection.Close()

		err = mediaConnection.Connect()
		if err != nil {
			log.Println("Failed to connect to the media receiver:", err)
			return
		}

		mediaInfo := ctrl.MediaInfo{
			ContentID:   "http://commondatastorage.googleapis.com/gtv-videos-bucket/big_buck_bunny_1080p.mp4",
			ContentType: "video/mp4",
			StreamType:  ctrl.StreamTypeBuffered,
			Metadata: map[string]interface{}{
				"type":         0,
				"metadataType": 0,
				"title":        "Big Buck Bunny",
			},
		}

		media := ctrl.NewMediaController(device, sourceId, session.TransportId)
		_, err = media.Load(mediaInfo, ctrl.LoadOptions{
			AutoPlay: true,
		})

		if err != nil {
			log.Println("Error while loading media", err)
			return
		}

		select {
		case <-cancel:
			return
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

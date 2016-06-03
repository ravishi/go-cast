package main

import (
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/oleksandr/bonjour"
	"github.com/ravishi/go-cast/cast"
	"github.com/ravishi/go-cast/cast/ctrl"
	"golang.org/x/net/context"
	"log"
	"os"
	"os/signal"
	"time"
)

func main() {
	err := actualMain()
	if err == nil {
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
	defer close(services)

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
				log.Println("Device disconnected:", err)
				if err == context.Canceled {
					err = nil
				}
				return err
			} else {
				return errors.New("Discovery closed unexpectedly")
			}
		case <-sigCh:
			return nil
		}
	}
}

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

	// Send a connect in 2s
	connection := ctrl.NewConnectionController(device, "sender-0", "receiver-0")
	defer connection.Close()

	select {
	case <-cancel:
		return context.Canceled
	case <-time.After(time.Second * 2):
		err := connection.Connect()
		if err != nil {
			return fmt.Errorf("Failed to connect: %s", err)
		}
	}

	// send a PING every 5s
	heartbeat := ctrl.NewHeartbeatController(device, "sender-0", "receiver-0")
	heartbeatCtx, cancelHeartbeat := context.WithCancel(context.Background())
	defer cancelHeartbeat()
	defer heartbeat.Close()
	go func() {
		for {
			select {
			case <-heartbeatCtx.Done():
				return
			case <-time.After(time.Second * 5):
				err := heartbeat.Ping()
				if err != nil {
					log.Println("Failed to send PING:", err)
				}
			}
		}
	}()

	deviceCtx, stopDevice := context.WithCancel(context.Background())
	defer stopDevice()

	go device.Run(deviceCtx)

	select {
	case <-cancel:
		return context.Canceled
	}

	return nil
}

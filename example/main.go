package main

import (
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/oleksandr/bonjour"
	"github.com/ravishi/go-castv2/cast"
	"github.com/ravishi/go-castv2/cast/ctrl"
	"golang.org/x/net/context"
	"log"
	"os"
	"os/signal"
	"time"
)

func main() {
	resolver, err := bonjour.NewResolver(nil)
	if err != nil {
		log.Println("Failed to initialize resolver:", err.Error())
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	results := make(chan *bonjour.ServiceEntry)
	timeoutCh := make(chan bool)
	timeoutSeconds := 20
	go func(ctx context.Context, entriesCh <-chan *bonjour.ServiceEntry, exitCh chan<- bool, timeoutCh chan<- bool) {
		for {
			select {
			case <-ctx.Done():
				exitCh <- true
			case <-time.After(time.Second * time.Duration(timeoutSeconds)):
				timeoutCh <- true
			case entry, ok := <-entriesCh:
				if ok {
					log.Println("Device found:", entry.Instance)

					err := consumeService(entry, ctx)
					log.Println(entry.Instance, "disconnected:", err)
					continue
				}
			}
			return
		}
	}(ctx, results, resolver.Exit, timeoutCh)

	err = resolver.Browse("_googlecast._tcp", ".local", results)
	if err != nil {
		log.Println("Failed to browse:", err.Error())
		os.Exit(1)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, os.Kill)

	// Block until the context is canceled or a signal is received.
	select {
	case s := <-sigCh:
		fmt.Println("Got signal:", s)
	case <-timeoutCh:
		fmt.Println("No device found after", timeoutSeconds, "seconds")
		os.Exit(1)
	}

	os.Exit(0)
}

func consumeService(entry *bonjour.ServiceEntry, ctx context.Context) error {
	addr := fmt.Sprintf("%s:%d", entry.AddrIPv4, entry.Port)

	conn, err := tls.Dial("tcp", addr, &tls.Config{
		InsecureSkipVerify: true,
	})
	if err != nil {
		return fmt.Errorf("Failed to connect: %s", err)
	}

	// Cancel all pending goroutines before closing the connection. Hopefully.
	defer conn.Close()

	chanmgr := cast.NewChanneler(ctx, conn)
	defer chanmgr.Close()

	// Consume the connection in another goroutine
	go chanmgr.Consume(ctx)

	// Send a connect in 2s
	connection := ctrl.NewConnectionController(chanmgr, "sender-0", "receiver-0")
	select {
	case <-time.After(time.Second * 2):
		err := connection.Connect()
		if err != nil {
			return fmt.Errorf("Failed to connect: %s", err)
		}
	}

	defer connection.Close()

	// ping-pong every 5s, timeout after 3 missed messages.
	heartbeat := ctrl.NewHeartbeatController(chanmgr, "sender-0", "receiver-0")
	interval := time.Duration(time.Second * 5)
	tf := 2
PING:
	for {
		ctx, _ := context.WithTimeout(ctx, interval*time.Duration(tf))

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
		}

		for i := 0; i < tf; i++ {
			requestId, err := heartbeat.Ping()
			if err != nil {
				log.Println("Failed to PING:", err)
				continue
			}

			err = heartbeat.WaitPong(ctx, requestId)
			if err == nil {
				continue PING
			} else if ctrl.IsParseError(err) || ctrl.IsUnexpectedResponseError(err) {
				// We can ignore these a few times
				continue
			} else {
				// Anything else we should just bail.
				return err
			}
		}

		return errors.New("timeout")
	}

	return nil
}

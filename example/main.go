package main

import (
	"crypto/tls"
	"fmt"
	"github.com/davecgh/go-spew/spew"
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
	entriesCh := make(chan *bonjour.ServiceEntry, 4)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		for entry := range entriesCh {
			log.Print("New device found:")
			spew.Dump(entry)

			err := consume(entry, ctx)
			log.Println("Device disconnected:", err)
		}
	}()

	resolver, err := bonjour.NewResolver(nil)
	if err != nil {
		log.Println("Failed to initialize resolver:", err.Error())
		os.Exit(1)
	}

	go func() {
		log.Println("Searching for cast devices...")
		err := resolver.Browse("_googlecast._tcp", ".local", entriesCh)
		if err != nil {
			log.Println("Failed to browse:", err.Error())
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)

	// Block until a signal is received.
	s := <-c
	fmt.Println("Got signal:", s)
	cancel()
}

// FIXME It works, but something is holding the context open when the remote connection is closed. Maybe it's a deadlock? Who knows.
func consume(entry *bonjour.ServiceEntry, ctx context.Context) error {
	addr := fmt.Sprintf("%s:%d", entry.AddrIPv4, entry.Port)

	conn, err := tls.Dial("tcp", addr, &tls.Config{
		InsecureSkipVerify: true,
	})
	if err != nil {
		return fmt.Errorf("Failed to connect: %s", err)
	}

	// Cancel all pending goroutines before closing the connection. Hopefully.
	defer conn.Close()

	// Consume the connection forever.
	chanmgr := cast.NewChanneler(ctx, conn)
	go func() {
		defer chanmgr.Close()
		chanmgr.Consume(ctx)
	}()

	// Send a connect in 2s
	connection := ctrl.NewConnectionController(chanmgr, "sender-0", "receiver-0")
	select {
	case <-ctx.Done():
		err := connection.Close()
		if err != nil {
			log.Println("Failed to close connection:", err)
		}
	case <-time.After(time.Second * 2):
		err := connection.Connect()
		if err != nil {
			log.Println("Failed to connect:", err)
		}
	}

	// ping-pong each 5s, timeout after 3 missed messages.
	heartbeat := ctrl.NewHeartbeatController(chanmgr, "sender-0", "receiver-0")
	go func() {
		interval := time.Duration(time.Second * 5)
		for {
			select {
			case <-ctx.Done():
				log.Println("Stopping beat:", ctx.Err())
				return
			case <-time.After(interval):
			}

			ctx, _ := context.WithTimeout(ctx, interval*time.Duration(3))
			err := pingThenWaitPong(heartbeat, ctx)
			if err == context.DeadlineExceeded || err == context.Canceled {
				return
			} else if err != nil {
				log.Println("Beat error:", err)
			}
		}
	}()

	select {
	case <-ctx.Done():
	}
	return ctx.Err()
}

func pingThenWaitPong(c *ctrl.HeartbeatController, ctx context.Context) error {
	requestId, err := c.Ping()
	if err != nil {
		return err
	}
	return c.WaitPong(ctx, requestId)
}

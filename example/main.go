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
			log.Print("Found a Chromecast:")
			spew.Dump(entry)
			connect(entry, ctx)
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

func connect(entry *bonjour.ServiceEntry, ctx context.Context) {
	addr := fmt.Sprintf("%s:%d", entry.AddrIPv4, entry.Port)

	conn, err := tls.Dial("tcp", addr, &tls.Config{
		InsecureSkipVerify: true,
	})
	if err != nil {
		log.Println("Failed to connect:", err)
		return
	}

	ctx, cancel := context.WithCancel(ctx)

	chanmgr := cast.NewChanneler(ctx, conn)

	go func() {
		defer cancel()
		defer chanmgr.Close()
		chanmgr.Consume(ctx)
	}()

	connection := ctrl.NewConnectionController(chanmgr, "sender-0", "receiver-0")

	time.Sleep(time.Second * 3)

	connection.Connect()

	select {
	case <-ctx.Done():
	}
}

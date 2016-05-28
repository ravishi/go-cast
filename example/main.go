package main

import (
	"crypto/tls"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/oleksandr/bonjour"
	"github.com/ravishi/go-castv2/cast"
	"github.com/ravishi/go-castv2/cast/ctrl"
	"log"
	"os"
	"os/signal"
	"time"
)

func main() {
	entriesCh := make(chan *bonjour.ServiceEntry, 4)

	go func() {
		for entry := range entriesCh {
			log.Print("Found a Chromecast:")
			spew.Dump(entry)

			addr := fmt.Sprintf("%s:%d", entry.AddrIPv4, entry.Port)

			conn, err := tls.Dial("tcp", addr, &tls.Config{
				InsecureSkipVerify: true,
			})
			if err != nil {
				log.Println("Failed to connect:", err)
				return
			}

			chanmgr := cast.NewChanneler(conn)

			connection := ctrl.NewConnectionController(chanmgr, "sender-0", "receiver-0")

			go chanmgr.Run()

			time.Sleep(time.Second * 1)

			connection.Connect()

			select {}
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
}

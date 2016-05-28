package cast

import (
	"github.com/davecgh/go-spew/spew"
	"io"
	"log"
	"sync"
)

type channeler struct {
	conn    io.ReadWriteCloser
	input   chan *CastMessage
	reg     chan *channel
	unreg   chan *channel
	outputs map[*channel]bool
	done    chan interface{}
}

type Channeler interface {
	Stop() error
	Run()
}

func (b *channeler) broadcast(m *CastMessage) {
	log.Print("Broadcasting received message: ")
	spew.Dump(m)
	for c := range b.outputs {
		if *m.Namespace != c.namespace ||
			*m.DestinationId != c.sourceId && *m.DestinationId != "*" ||
			*m.SourceId != c.destinationId {
			continue
		}
		c.ch <- m
	}
}

func (b *channeler) send(m *CastMessage) {
	if b != nil {
		b.input <- m
	}
}

func (b *channeler) unregister(c *channel) {
	b.unreg <- c
}

// New creates a new Channeler with the given input channel buffer length.
func NewChanneler(conn io.ReadWriteCloser) Channeler {
	return &channeler{
		conn:    conn,
		input:   make(chan *CastMessage),
		reg:     make(chan *channel),
		unreg:   make(chan *channel),
		outputs: make(map[*channel]bool),
		done:    make(chan interface{}),
	}
}

func (b *channeler) NewChannel(namespace, sourceId, destinationId string) Channel {
	ch := newChannel(b, namespace, sourceId, destinationId)
	b.reg <- ch
	return ch
}

func (b *channeler) Run() {
	// Use a wait group so we can spawn goroutines but still block before returning.
	// This way the caller can still choose how to call us.
	wg := sync.WaitGroup{}
	defer wg.Wait()

	// The first goroutine will just broadcast messages arriving at `<-b.input`
	// until either that or `<-b.done` are closed.
	wg.Add(1)
	go func(wg sync.WaitGroup, b *channeler) {
		defer wg.Done()
		defer log.Println("Stopping with the broadcast...")
		for {
			select {
			case c := <-b.unreg:
				delete(b.outputs, c)
				close(c.ch)
			case m := <-b.input:
				b.broadcast(m)
			case c, ok := <-b.reg:
				if ok {
					b.outputs[c] = true
				} else {
					return
				}
			case <-b.done:
				return
			}
		}
	}(wg, b)

	// The second goroutine will read the connection until it is closed.
	wg.Add(1)
	go func(wg sync.WaitGroup, r io.Reader, messageCh chan<- *CastMessage, done <-chan interface{}) {
		defer wg.Done()
		for {
			select {
			case <-done:
				log.Println("Returning as request")
				return
			default:
				message, err := Read(r)
				if err == nil {
					messageCh <- message
				} else if err == io.EOF {
					log.Println("Underlying connection was closed")
					return
				} else if err != io.ErrNoProgress {
					log.Println("Error while reading message:", err)
				}
			}
		}
	}(wg, b.conn, b.input, b.done)
}

func (b *channeler) Stop() error {
	close(b.done)
	close(b.reg)
	return nil
}

package cast

import (
	"github.com/davecgh/go-spew/spew"
	"io"
	"log"
	"sync"
)

type channeler struct {
	wg      *sync.WaitGroup
	conn    io.ReadWriteCloser
	input   chan *CastMessage
	reg     chan *Channel
	unreg   chan *Channel
	outputs map[*Channel]bool
	done    chan interface{}
}

type Channeler interface {
	Run()
	Stop() error
	NewChannel(namespace, sourceId, destinationId string) *Channel
}

func (b *channeler) broadcast(m *CastMessage) {
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

func (b *channeler) unregister(c *Channel) {
	b.unreg <- c
}
func (b *channeler) broadcastForever() {
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
}

// New creates a new Channeler with the given input channel buffer length.
func NewChanneler(conn io.ReadWriteCloser) Channeler {
	w := &channeler{
		wg:      &sync.WaitGroup{},
		conn:    conn,
		input:   make(chan *CastMessage),
		reg:     make(chan *Channel),
		unreg:   make(chan *Channel),
		outputs: make(map[*Channel]bool),
		done:    make(chan interface{}),
	}

	// This goroutine will just broadcast messages arriving at `<-b.input`
	// until either it or `<-b.done` are closed.
	go func() {
		w.wg.Add(1)
		defer w.wg.Done()
		w.broadcastForever()
	}()

	return w
}

func (b *channeler) NewChannel(namespace, sourceId, destinationId string) *Channel {
	ch := newChannel(b, namespace, sourceId, destinationId)
	b.reg <- ch
	return ch
}

func (b *channeler) Run() {
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func(wg *sync.WaitGroup, r io.Reader, messageCh chan<- *CastMessage, done <-chan interface{}) {
		defer wg.Done()
		for {
			select {
			case <-done:
				log.Println("Returning as request")
				return
			default:
				message, err := Read(r)
				if err == nil {
					log.Println("Incoming message:")
					spew.Dump(message)
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

	wg.Wait()
}

func (b *channeler) Stop() error {
	close(b.done)
	close(b.reg)
	return nil
}

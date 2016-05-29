package cast

import (
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"golang.org/x/net/context"
	"io"
	"log"
	"sync"
)

type Channeler struct {
	conn    io.ReadWriter
	input   chan *CastMessage
	reg     chan *Channel
	unreg   chan *Channel
	outputs map[*Channel]bool
	wg      *sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
}

func (b *Channeler) broadcast(m *CastMessage) {
	log.Println("-> Broadcasting message:")
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

func (b *Channeler) send(m *CastMessage) {
	b.input <- m
}

func (b *Channeler) unregister(c *Channel) {
	b.unreg <- c
}

// This function will just broadcast messages arriving at `<-b.input`
// until either it or `<-b.done` are closed.
func (b *Channeler) broadcastForever() {
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
		case <-b.ctx.Done():
			return
		}
	}
}

func NewChanneler(ctx context.Context, conn io.ReadWriter) *Channeler {
	ctx, cancel := context.WithCancel(ctx)

	w := &Channeler{
		ctx:     ctx,
		cancel:  cancel,
		conn:    conn,
		wg:      &sync.WaitGroup{},
		reg:     make(chan *Channel),
		unreg:   make(chan *Channel),
		input:   make(chan *CastMessage),
		outputs: make(map[*Channel]bool),
	}

	go func() {
		w.wg.Add(1)
		defer w.wg.Done()
		w.broadcastForever()
	}()

	return w
}

func (b *Channeler) NewChannel(namespace, sourceId, destinationId string) *Channel {
	ch := newChannel(b, namespace, sourceId, destinationId)
	b.reg <- ch
	return ch
}

func (b *Channeler) Consume(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			message, err := Read(b.conn)
			if err == nil {
				b.input <- message
			} else if err != io.ErrNoProgress && err != io.EOF {
				return fmt.Errorf("Failed to read message: %s", err)
			} else {
				return nil
			}
		}
	}
}

func (b *Channeler) Close() error {
	b.ctx.Done()
	close(b.reg)
	b.cancel()
	return nil
}

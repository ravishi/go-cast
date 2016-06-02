package cast

import (
	"github.com/hashicorp/go.net/context"
	"io"
	"log"
)

type (
	Device struct {
		bc        *messageBroadcaster
		ctx       context.Context
		conn      io.ReadWriter
		cancel    context.CancelFunc
		handlerId int32
	}

	Handler func(context.Context) error
)

func NewDevice(connection io.ReadWriter) *Device {
	ctx, cancel := context.WithCancel(context.Background())
	return &Device{
		bc:        newMessageBroadcaster(),
		ctx:       ctx,
		conn:      connection,
		cancel:    cancel,
		handlerId: 0,
	}
}

func (d *Device) NewChannel(namespace, sourceId, destinationId string, size int) *Channel {
	return newChannel(d, namespace, sourceId, destinationId, size)
}

func (d *Device) Send(message *CastMessage) error {
	log.Println("->", message)
	return Write(d.conn, message)
}

func (d *Device) Run(ctx context.Context) error {
	for {
		select {
		case <-d.ctx.Done():
			return nil
		case <-ctx.Done():
			return nil
		default:
		}

		message, err := Read(d.conn)
		if err == io.ErrNoProgress {
			continue
		} else if message != nil {
			log.Println("<-", message)
			// if err == io.EOF, message can be not null
			// and that's why we have this weird branching here.
			d.bc.Pub() <- message
		}

		if err != nil {
			return err
		}
	}
}

func (d *Device) Close() {
	d.cancel()
	d.bc.Close()
}

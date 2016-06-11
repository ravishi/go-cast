package ctrl

import (
	"encoding/json"
	"errors"
	"github.com/ravishi/go-cast/cast"
	"golang.org/x/net/context"
)

const (
	ConnectionNamespace = "urn:x-cast:com.google.cast.tp.connection"
)

var (
	CloseCommand   = PayloadHeaders{Type: "CLOSE"}
	ConnectCommand = PayloadHeaders{Type: "CONNECT"}
	Closed         = errors.New("Closed")
)

type ConnectionController struct {
	ch     *cast.Channel
	ctx    context.Context
	close  context.CancelFunc
	closed chan struct{}
	err    error
}

func NewConnectionController(device *cast.Device, sourceId, destinationId string) *ConnectionController {
	ctx, close := context.WithCancel(context.Background())
	c := &ConnectionController{
		ch:     device.NewChannel(ConnectionNamespace, sourceId, destinationId),
		ctx:    ctx,
		close:  close,
		closed: make(chan struct{}),
		err:    nil,
	}

	go c.waitClose()

	return c
}

func (c *ConnectionController) Connect() error {
	return send(c.ch, ConnectCommand)
}

func (c *ConnectionController) Close() {
	if c.err == nil {
		send(c.ch, CloseCommand)
		close(c.closed)
	}
	c.ch.Close()
}

func (c *ConnectionController) Closed() <-chan struct{} {
	return c.closed
}

func (c *ConnectionController) waitClose() {
	for {
		select {
		case <-c.ctx.Done():
			return
		case message, ok := <-c.ch.Read():
			if !ok {
				return
			}

			headers := &PayloadHeaders{}
			err := json.Unmarshal([]byte(*message.PayloadUtf8), headers)
			if err != nil {
				c.err = err
				close(c.closed)
				return
			}

			if headers.Type == "CLOSE" {
				c.err = Closed
				close(c.closed)
				return
			}
		}
	}
}

package ctrl

import (
	"github.com/ravishi/go-cast/cast"
)

var (
	CloseCommand   = PayloadHeaders{Type: "CLOSE"}
	ConnectCommand = PayloadHeaders{Type: "CONNECT"}

	ConnectionNamespace = "urn:x-cast:com.google.cast.tp.connection"
)

type ConnectionController struct {
	ch *cast.Channel
}

func NewConnectionController(chanmgr *cast.Device, sourceId, destinationId string) *ConnectionController {
	return &ConnectionController{
		ch: chanmgr.NewChannel(ConnectionNamespace, sourceId, destinationId, 1),
	}
}

func (c *ConnectionController) Connect() error {
	return send(c.ch, ConnectCommand)
}

func (c *ConnectionController) Close() {
	send(c.ch, CloseCommand)
	c.ch.Close()
}

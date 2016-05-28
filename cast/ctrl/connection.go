package ctrl

import (
	"github.com/ravishi/go-castv2/cast"
)

var (
	CloseCommand   = PayloadHeaders{Type: "CLOSE"}
	ConnectCommand = PayloadHeaders{Type: "CONNECT"}

	ConnectionNamespace = "urn:x-cast:com.google.cast.tp.connection"
)

type ConnectionController struct {
	ch *cast.Channel
}

func NewConnectionController(chanmgr *cast.Channeler, sourceId, destinationId string) *ConnectionController {
	return &ConnectionController{
		ch: chanmgr.NewChannel(ConnectionNamespace, sourceId, destinationId),
	}
}

func (c *ConnectionController) Connect() error {
	return send(c.ch, ConnectCommand)
}

func (c *ConnectionController) Close() error {
	return send(c.ch, CloseCommand)
}

package ctrl

import (
	"encoding/json"
	"github.com/ravishi/go-castv2/cast"
)

var (
	CloseCommand   = PayloadHeaders{Type: "CLOSE"}
	ConnectCommand = PayloadHeaders{Type: "CONNECT"}

	ConnectionNamespace = "urn:x-cast:com.google.cast.tp.connection"
)

type connectionController struct {
	ch *cast.Channel
}

func NewConnectionController(chmgr cast.Channeler, sourceId, destinationId string) *connectionController {
	return &connectionController{
		ch: chmgr.NewChannel(ConnectionNamespace, sourceId, destinationId),
	}
}

func (c *connectionController) Connect() error {
	return send(c.ch, ConnectCommand)
}

func (c *connectionController) Close() error {
	return send(c.ch, CloseCommand)
}

func send(ch *cast.Channel, payload interface{}) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return ch.Send(string(jsonData))
}

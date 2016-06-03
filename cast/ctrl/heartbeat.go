package ctrl

import (
	"github.com/ravishi/go-cast/cast"
)

const (
	HeartbeatNamespace = "urn:x-cast:com.google.cast.tp.heartbeat"
)

var (
	PingCommand = PayloadHeaders{Type: "PING"}
	PongCommand = PayloadHeaders{Type: "PONG"}
)

type HeartbeatController struct {
	ch *cast.Channel
}

func NewHeartbeatController(chanmgr *cast.Device, sourceId, destinationId string) *HeartbeatController {
	return &HeartbeatController{
		ch: chanmgr.NewChannel(HeartbeatNamespace, sourceId, destinationId, 5),
	}
}

func (c *HeartbeatController) Ping() error {
	return send(c.ch, &PayloadHeaders{
		Type: PingCommand.Type,
	})
}

func (c *HeartbeatController) Close() {
	c.ch.Close()
}

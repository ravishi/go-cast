package ctrl

import (
	"github.com/ravishi/go-cast/cast"
	"golang.org/x/net/context"
	"time"
)

const (
	HeartbeatNamespace = "urn:x-cast:com.google.cast.tp.heartbeat"
)

var (
	PingCommand = PayloadHeaders{Type: "PING"}
)

type HeartbeatController struct {
	ch    *cast.Channel
	ctx   context.Context
	close context.CancelFunc
}

func NewHeartbeatController(chanmgr *cast.Device, sourceId, destinationId string) *HeartbeatController {
	ctx, cancel := context.WithCancel(chanmgr.Context())
	return &HeartbeatController{
		ch:    chanmgr.NewChannel(HeartbeatNamespace, sourceId, destinationId, 5),
		ctx:   ctx,
		close: cancel,
	}
}

func (c *HeartbeatController) Ping() error {
	return send(c.ch, &PayloadHeaders{
		Type: PingCommand.Type,
	})
}

func (c *HeartbeatController) Close() {
	c.close()
	c.ch.Close()
}

func (c *HeartbeatController) Beat(interval time.Duration, timeoutFactor int) error {
NEXT:
	for {
		ctx, _ := context.WithTimeout(c.ctx, interval*time.Duration(timeoutFactor))

		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case message, ok := <-c.ch.Read():
				if !ok {
					// XXX Channel got closed. What should we do?
					return nil
				}
				payload, err := getHeaders(message)
				if err == nil && payload.Type == "PONG" {
					continue NEXT
				}
			case <-time.After(interval):
			}

			err := c.Ping()
			if err != nil {
				return err
			}
		}
	}
}

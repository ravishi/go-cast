package ctrl

import (
	"encoding/json"
	"github.com/ravishi/go-cast/cast"
	"golang.org/x/net/context"
	"log"
	"time"
)

const (
	HeartbeatNamespace = "urn:x-cast:com.google.cast.tp.heartbeat"
)

var (
	pingCommand = PayloadHeaders{Type: "PING"}
	pongCommand = PayloadHeaders{Type: "PONG"}
)

type HeartbeatController struct {
	ch    *cast.Channel
	ctx   context.Context
	close context.CancelFunc
}

func NewHeartbeatController(device *cast.Device, sourceId, destinationId string) *HeartbeatController {
	ctx, cancel := context.WithCancel(device.Context())
	return &HeartbeatController{
		ch:    device.NewChannel(HeartbeatNamespace, sourceId, destinationId),
		ctx:   ctx,
		close: cancel,
	}
}

func (c *HeartbeatController) Ping() error {
	return send(c.ch, &pingCommand)
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
			case message, ok := <-c.ch.Read():
				if !ok {
					return nil
				}

				payload := &PayloadHeaders{}
				err := json.Unmarshal([]byte(*message.PayloadUtf8), payload)
				if err != nil {
					log.Println("Error while unmarshaling beat response:", err)
				} else if payload.Type == pongCommand.Type {
					continue NEXT
				}
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(interval):
			}

			err := c.Ping()
			if err != nil {
				return err
			}
		}
	}
}

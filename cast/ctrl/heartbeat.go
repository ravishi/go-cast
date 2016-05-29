package ctrl

import (
	"fmt"
	"sync/atomic"

	"github.com/ravishi/go-castv2/cast"
	"golang.org/x/net/context"
	"strings"
)

const (
	requestIdKey       = "requestId"
	HeartbeatNamespace = "urn:x-cast:com.google.cast.tp.heartbeat"
)

var (
	heartbeatRequestId = int32(0)

	PingCommand = PayloadHeaders{Type: "PING"}
	PongCommand = PayloadHeaders{Type: "PONG"}
)

type HeartbeatController struct {
	ch *cast.Channel
}

func NewHeartbeatController(chanmgr *cast.Channeler, sourceId, destinationId string) *HeartbeatController {
	return &HeartbeatController{
		ch: chanmgr.NewChannel(HeartbeatNamespace, sourceId, destinationId),
	}
}

func (c *HeartbeatController) Ping() (int, error) {
	i := int(atomic.AddInt32(&heartbeatRequestId, 1))
	err := c.ping(i)
	if err != nil {
		return 0, err
	}
	return i, nil
}

func (c *HeartbeatController) ping(requestId int) error {
	return send(c.ch, &PayloadHeaders{
		RequestId: &requestId,
		Type:      PingCommand.Type,
	})
}

func (c *HeartbeatController) Pong() error {
	return send(c.ch, PongCommand)
}

// Ping the server and wait for a PONG.
//
// Returns nil once PONG arrives or:
// - an error if we fail the parse any incoming message.
//   Pro tip, use `IsPareError` to check.
// - an error if we end up getting an unexpected message
//   Like not a PONG, or a message that was "addressed at
//   someone else", who knows what can happen in the guts
//   of the implementation. Anyway, use `IsUnexpectedResponseError`
//   to check.
func (c *HeartbeatController) WaitPong(ctx context.Context, requestId int) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case message := <-c.ch.Channel():
		h, err := getHeaders(message)
		if err != nil {
			return fmt.Errorf("%s: %s", parseErrorPrefix, err)
		}
		if h.Type == PongCommand.Type {
			if h.RequestId != nil && *h.RequestId != requestId {
				return fmt.Errorf(
					"%s: %s; while we were expecting %s",
					unexpectedResponseErrorPrefix,
					h.RequestId, requestId)
			}
		}
	}
	return nil
}

// \_(ツ)_/¯ What could I do?

const (
	parseErrorPrefix              = "Failed to parse message"
	unexpectedResponseErrorPrefix = "Unknown PONG received"
)

func IsParseError(err error) bool {
	return strings.HasPrefix(err.Error(), parseErrorPrefix)
}

func IsUnexpectedResponseError(err error) bool {
	return strings.HasPrefix(err.Error(), unexpectedResponseErrorPrefix)
}

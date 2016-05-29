package cast

import (
	"github.com/davecgh/go-spew/spew"
	"log"
)

type Channel struct {
	mgr           *Channeler
	ch            chan *CastMessage
	namespace     string
	sourceId      string
	destinationId string
}

func newChannel(mgr *Channeler, namespace, sourceId, destinationId string) *Channel {
	return &Channel{
		mgr:           mgr,
		sourceId:      sourceId,
		destinationId: destinationId,
		namespace:     namespace,
		ch:            make(chan *CastMessage),
	}
}

func (c *Channel) Close() error {
	c.mgr.unregister(c)
	return nil
}

func (c *Channel) Channel() <-chan *CastMessage {
	return c.ch
}

func (c *Channel) Namespace() string {
	return c.namespace
}

func (c *Channel) SourceId() string {
	return c.sourceId
}

func (c *Channel) DestinationId() string {
	return c.destinationId
}

func (c *Channel) Send(payload string) error {
	message := &CastMessage{
		ProtocolVersion: CastMessage_CASTV2_1_0.Enum(),
		SourceId:        &c.sourceId,
		DestinationId:   &c.destinationId,
		Namespace:       &c.namespace,
		PayloadType:     CastMessage_STRING.Enum(),
		PayloadUtf8:     &payload,
	}
	log.Print("<- Sending message:")
	spew.Dump(message)
	return Write(c.mgr.conn, message)
}

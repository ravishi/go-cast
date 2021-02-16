package cast

type Channel struct {
	device        *Device
	ch            chan *CastMessage
	namespace     string
	sourceId      string
	destinationId string
}

func newChannel(device *Device, namespace, sourceId, destinationId string) *Channel {
	c := &Channel{
		ch:            make(chan *CastMessage),
		device:        device,
		namespace:     namespace,
		sourceId:      sourceId,
		destinationId: destinationId,
	}
	c.device.bc.FilteredSubscribe(c.ch, c.expects)
	return c
}

func (c *Channel) expects(m *CastMessage) bool {
	return *m.Namespace == c.namespace &&
		*m.SourceId == c.destinationId &&
		*m.DestinationId == "*" || *m.DestinationId == c.sourceId
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

func (c *Channel) Read() <-chan *CastMessage {
	return c.ch
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
	return c.device.Send(message)
}

func (c *Channel) Close() {
	c.device.bc.Unsub(c.ch)
	close(c.ch)
}

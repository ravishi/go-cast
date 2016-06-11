package cast

type Channel struct {
	in            chan *CastMessage
	out           chan *CastMessage
	device        *Device
	namespace     string
	sourceId      string
	destinationId string
}

func (c *Channel) filterForever() {
	for message := range c.in {
		if c.expects(message) {
			c.out <- message
		}
	}
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
	return c.out
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
	// The broadcaster is the only one writing on c.in.
	c.device.bc.UnSub() <- c.in
	// And only messages coming from c.in are written on c.out.
	// Therefore, we should close them in this order.
	close(c.in)
	close(c.out)
}

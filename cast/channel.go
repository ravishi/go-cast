package cast

type Channel struct {
	in            chan *CastMessage
	out           chan *CastMessage
	ring          *messageRingBuffer
	device        *Device
	namespace     string
	sourceId      string
	destinationId string
}

func newChannel(device *Device, namespace, sourceId, destinationId string, size int) *Channel {
	in := make(chan *CastMessage)
	out := make(chan *CastMessage, size)
	c := &Channel{
		in:            in,
		out:           out,
		device:        device,
		namespace:     namespace,
		sourceId:      sourceId,
		destinationId: destinationId,
	}
	device.bc.Sub() <- c.in
	c.ring = newMessageRingBuffer(in, out, c.filter)
	return c
}

func (c *Channel) filter(m *CastMessage) bool {
	return *m.Namespace == c.namespace &&
		*m.SourceId == c.destinationId &&
		*m.DestinationId == "*" ||
		*m.DestinationId == c.sourceId
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
	return c.ring.out
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
	c.device.bc.UnSub() <- c.in
	close(c.in)
	close(c.out)
}

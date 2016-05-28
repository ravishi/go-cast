package cast

type channel struct {
	mgr           *channeler
	ch            chan *CastMessage
	namespace     string
	sourceId      string
	destinationId string
}

type Channel interface {
	Close() error
}

func newChannel(mgr *channeler, namespace, sourceId, destinationId string) *channel {
	return &channel{
		mgr:           mgr,
		sourceId:      sourceId,
		destinationId: destinationId,
		namespace:     namespace,
		ch:            make(chan *CastMessage),
	}
}

func (c *channel) Close() error {
	c.mgr.unregister(c)
	return nil
}

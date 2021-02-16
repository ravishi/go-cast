package cast

type broadcaster struct {
	pub       chan *CastMessage
	subs      map[chan<- *CastMessage]filterFunc
	subReqs   chan *subRequest
	unsubReqs chan chan<- *CastMessage
}

type subRequest struct {
	ch     chan<- *CastMessage
	filter filterFunc
}

type filterFunc func(*CastMessage) bool

func newMessageBroadcaster() *broadcaster {
	b := &broadcaster{
		pub:       make(chan *CastMessage),
		subs:      make(map[chan<- *CastMessage]filterFunc),
		subReqs:   make(chan *subRequest),
		unsubReqs: make(chan chan<- *CastMessage),
	}
	go b.broadcastForever()
	return b
}

func (b *broadcaster) broadcast(m *CastMessage) {
	for ch := range b.subs {
		if filter := b.subs[ch]; filter == nil || filter(m) {
			ch <- m
		}
	}
}

func (b *broadcaster) broadcastForever() {
	defer close(b.pub)
	defer close(b.unsubReqs)
	for {
		select {
		case m := <-b.pub:
			b.broadcast(m)
		case req, ok := <-b.subReqs:
			if ok {
				b.subs[req.ch] = req.filter
			} else {
				return
			}
		case ch := <-b.unsubReqs:
			delete(b.subs, ch)
		}
	}
}

func (b *broadcaster) Publish(m *CastMessage) {
	b.pub <- m
}

func noFilter(*CastMessage) bool {
	return true
}

func (b *broadcaster) Subscribe(ch chan<- *CastMessage) {
	b.FilteredSubscribe(ch, noFilter)
}

func (b *broadcaster) FilteredSubscribe(ch chan<- *CastMessage, filter filterFunc) {
	b.subReqs <- &subRequest{ch: ch, filter: filter}
}

func (b *broadcaster) Unsub(ch chan<- *CastMessage) {
	b.unsubReqs <- ch
}

func (b *broadcaster) Close() {
	close(b.subReqs)
}

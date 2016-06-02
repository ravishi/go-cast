package cast

type messageBroadcaster struct {
	pub     chan *CastMessage
	subbers map[chan<- *CastMessage]bool

	sub   chan chan<- *CastMessage
	unsub chan chan<- *CastMessage
}

func newMessageBroadcaster() *messageBroadcaster {
	b := &messageBroadcaster{
		pub:     make(chan *CastMessage),
		sub:     make(chan chan<- *CastMessage),
		unsub:   make(chan chan<- *CastMessage),
		subbers: make(map[chan<- *CastMessage]bool),
	}
	go b.broadcastForever()
	return b
}

func (b *messageBroadcaster) broadcast(m *CastMessage) {
	for ch := range b.subbers {
		ch <- m
	}
}

func (b *messageBroadcaster) broadcastForever() {
	defer close(b.pub)
	defer close(b.unsub)
	for {
		select {
		case m := <-b.pub:
			b.broadcast(m)
		case ch, ok := <-b.sub:
			if ok {
				b.subbers[ch] = true
			} else {
				return
			}
		case ch := <-b.unsub:
			delete(b.subbers, ch)
		}
	}
}

func (b *messageBroadcaster) Pub() chan<- *CastMessage {
	return b.pub
}

func (b *messageBroadcaster) Sub() chan<- chan<- *CastMessage {
	return b.sub
}

func (b *messageBroadcaster) UnSub() chan<- chan<- *CastMessage {
	return b.unsub
}

func (b *messageBroadcaster) Close() {
	close(b.sub)
}

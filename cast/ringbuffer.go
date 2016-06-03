package cast

type messageFilter func(*CastMessage) bool

type messageRingBuffer struct {
	in     <-chan *CastMessage
	out    chan *CastMessage
	filter messageFilter
}

func newMessageRingBuffer(in <-chan *CastMessage, out chan *CastMessage, filter messageFilter) *messageRingBuffer {
	r := &messageRingBuffer{in, out, filter}
	go r.run()
	return r
}

func (r *messageRingBuffer) run() {
	for v := range r.in {
		if !r.filter(v) {
			continue
		}
		select {
		case r.out <- v:
		default:
			<-r.out
			r.out <- v
		}
	}
}

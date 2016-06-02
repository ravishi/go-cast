package cast

type messageRingBuffer struct {
	in  <-chan *CastMessage
	out chan *CastMessage
}

func ringBuffer(in <-chan *CastMessage, out chan *CastMessage) *messageRingBuffer {
	r := &messageRingBuffer{in, out}
	go r.run()
	return r
}

func (r *messageRingBuffer) run() {
	for v := range r.in {
		select {
		case r.out <- v:
		default:
			<-r.out
			r.out <- v
		}
	}
}

package ctrl

import (
	"encoding/json"
	"fmt"
	"sync/atomic"

	"github.com/ravishi/go-cast/pkg/cast"
	"golang.org/x/net/context"
)

type Request interface {
	setRequestId(int32)
}

type ResponseHeaders struct {
	header  *RequestHeader
	message *json.RawMessage
}

func (r *ResponseHeaders) Type() string {
	return r.header.Type
}

func (r *ResponseHeaders) RequestId() int {
	return int(r.header.RequestId)
}

func (r *ResponseHeaders) Unmarshal(v interface{}) error {
	return json.Unmarshal(*r.message, v)
}

type requestManager struct {
	ch              *cast.Channel
	ctx             context.Context
	close           context.CancelFunc
	requestId       int32
	requestHandlers map[int32]chan<- *ResponseHeaders
}

type RequestHeader struct {
	PayloadHeaders
	RequestId int32 `json:"requestId"`
}

type ResponseHeader RequestHeader

func (h *RequestHeader) setRequestId(requestId int32) {
	h.RequestId = requestId
}

func newRequestManager(ch *cast.Channel) *requestManager {
	ctx, close := context.WithCancel(context.Background())
	m := &requestManager{
		ch:              ch,
		ctx:             ctx,
		close:           close,
		requestId:       0,
		requestHandlers: make(map[int32]chan<- *ResponseHeaders),
	}

	go m.handleForever()

	return m
}

func (m *requestManager) handleForever() {
	for {
		select {
		case message, ok := <-m.ch.Read():
			if !ok {
				return
			} else {
				// TODO Notify errors.
				rawMessage := &json.RawMessage{}
				err := json.Unmarshal([]byte(*message.PayloadUtf8), rawMessage)
				if err != nil {
					continue
				}

				header := &RequestHeader{}
				err = json.Unmarshal(*rawMessage, header)
				if err != nil {
					continue
				}

				ch := m.requestHandlers[header.RequestId]
				if ch == nil {
					continue
				}

				ch <- &ResponseHeaders{header: header, message: rawMessage}
			}
		case <-m.ctx.Done():
			return
		}
	}
}

func (m *requestManager) unregister(ch chan<- *ResponseHeaders) {
	for i, c := range m.requestHandlers {
		if ch == c {
			delete(m.requestHandlers, i)
			return
		}
	}
}

func (m *requestManager) Send(payload Request, response chan<- *ResponseHeaders) error {
	requestId := atomic.AddInt32(&m.requestId, 1)

	payload.setRequestId(requestId)

	m.requestHandlers[requestId] = response

	err := send(m.ch, payload)
	if err != nil {
		return err
	}

	return nil
}

type responseError struct {
	ResponseHeaders
	reason string `json:"reason"`
}

func (e *responseError) Error() string {
	return e.reason
}

func (m *requestManager) Request(payload Request /*, timeout time.Duration */) (*ResponseHeaders, error) {
	responseCh := make(chan *ResponseHeaders)
	defer close(responseCh)

	err := m.Send(payload, responseCh)
	defer m.unregister(responseCh)
	if err != nil {
		return nil, err
	}

	select {
	case <-m.ctx.Done():
		return nil, m.ctx.Err()
	case response, ok := <-responseCh:
		if !ok {
			return nil, fmt.Errorf("Response channel unexpectedly closed")
		}

		if response.Type() == "INVALID_REQUEST" {
			responseErr := &responseError{}
			if err := response.Unmarshal(responseErr); err != nil {
				return nil, err
			}
			return nil, responseErr
		}

		return response, nil
	}

}

func (m *requestManager) Close() {
	m.close()
}

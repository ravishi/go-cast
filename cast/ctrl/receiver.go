package ctrl

import (
	"github.com/ravishi/go-cast/cast"
	"golang.org/x/net/context"
)

var (
	ReceiverNamespace = "urn:x-cast:com.google.cast.receiver"

	GetStatusCommand = PayloadHeaders{Type: "GET_STATUS"}
)

type ReceiverController struct {
	ch    *cast.Channel
	ctx   context.Context
	close context.CancelFunc
	rm    *requestManager
}

type ReceiverStatus interface{}

func NewReceiverController(device *cast.Device, sourceId, destinationId string) *ReceiverController {
	ctx, close := context.WithCancel(context.Background())
	ch := device.NewChannel(ReceiverNamespace, sourceId, destinationId, 2)
	return &ReceiverController{
		ch:    ch,
		ctx:   ctx,
		close: close,
		rm:    newRequestManager(ch),
	}
}

type statusResponse struct {
	MessageHeader
	Status *ReceiverStatus `json:"status,omitempty"`
}

func (r *ReceiverController) GetStatus() (*ReceiverStatus, error) {
	response, err := r.rm.Request(&MessageHeader{
		PayloadHeaders: GetStatusCommand,
	})
	if err != nil {
		return nil, err
	}

	statusResponse := &statusResponse{}
	err = response.Unmarshal(statusResponse)
	if err != nil {
		return nil, err
	}

	return statusResponse.Status, nil
}

func (r *ReceiverController) Close() {
	r.rm.Close()
	r.close()
}

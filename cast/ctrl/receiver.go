package ctrl

import (
	"fmt"
	"github.com/ravishi/go-cast/cast"
	"golang.org/x/net/context"
)

const (
	ReceiverNamespace = "urn:x-cast:com.google.cast.receiver"
)

type ReceiverController struct {
	ch    *cast.Channel
	ctx   context.Context
	close context.CancelFunc
	rm    *requestManager
}

type ReceiverStatus struct {
	Applications []*ApplicationSession `json:"applications"`
	Volume       *Volume               `json:"volume,omitempty"`
}

type ApplicationSession struct {
	AppID       *string      `json:"appId,omitempty"`
	DisplayName *string      `json:"displayName,omitempty"`
	Namespaces  []*Namespace `json:"namespaces"`
	SessionID   *string      `json:"sessionId,omitempty"`
	StatusText  *string      `json:"statusText,omitempty"`
	TransportId *string      `json:"transportId,omitempty"`
}

type Namespace struct {
	Name string `json:"name"`
}

type Volume struct {
	Level *float64 `json:"level,omitempty"`
	Muted *bool    `json:"muted,omitempty"`
}

func NewReceiverController(device *cast.Device, sourceId, destinationId string) *ReceiverController {
	ctx, close := context.WithCancel(context.Background())
	ch := device.NewChannel(ReceiverNamespace, sourceId, destinationId)
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
	return r.requestStatus(&MessageHeader{
		PayloadHeaders: PayloadHeaders{"GET_STATUS"},
	})
}

func (r *ReceiverController) SetVolume(level float64) (*ReceiverStatus, error) {
	request := &struct {
		MessageHeader
		Volume *Volume `json:"volume"`
	}{
		MessageHeader: MessageHeader{
			PayloadHeaders: PayloadHeaders{Type: "SET_VOLUME"},
		},
		Volume: &Volume{Level: &level},
	}

	return r.requestStatus(request)
}

func (r *ReceiverController) SetMuted(muted bool) (*ReceiverStatus, error) {
	request := &struct {
		MessageHeader
		Volume *Volume `json:"volume"`
	}{
		MessageHeader: MessageHeader{
			PayloadHeaders: PayloadHeaders{Type: "SET_VOLUME"},
		},
		Volume: &Volume{Muted: &muted},
	}

	return r.requestStatus(request)
}

func (r *ReceiverController) Launch(appId string) (*ReceiverStatus, error) {
	request := &struct {
		MessageHeader
		AppID string `json:"appId"`
	}{
		MessageHeader: MessageHeader{
			PayloadHeaders: PayloadHeaders{Type: "LAUNCH"},
		},
		AppID: appId,
	}

	response, err := r.rm.Request(request)
	if err != nil {
		return nil, err
	}

	responseHeader := &MessageHeader{}
	err = response.Unmarshal(responseHeader)
	if err != nil {
		return nil, err
	}

	if responseHeader.Type == "LAUNCH_ERROR" {
		errorReponse := &struct {
			MessageHeader
			Reason interface{}
		}{}
		err = response.Unmarshal(errorReponse)
		err = fmt.Errorf("Launch error: %s", err)
		return nil, err
	}

	statusResponse := &statusResponse{}
	err = response.Unmarshal(statusResponse)
	if err != nil {
		return nil, err
	}

	return statusResponse.Status, nil
}

func (r *ReceiverController) Stop(sessionId string) (*ReceiverStatus, error) {
	request := &struct {
		MessageHeader
		SessionID string `json:"sesssionId"`
	}{
		MessageHeader: MessageHeader{
			PayloadHeaders: PayloadHeaders{Type: "STOP"},
		},
		SessionID: sessionId,
	}

	return r.requestStatus(request)
}

func (r *ReceiverController) Close() {
	r.rm.Close()
	r.close()
}

func (r *ReceiverController) requestStatus(request Request) (*ReceiverStatus, error) {
	response, err := r.rm.Request(request)
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

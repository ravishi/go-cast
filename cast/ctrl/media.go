package ctrl

import (
	"errors"
	"github.com/ravishi/go-cast/cast"
	"golang.org/x/net/context"
	"time"
)

const (
	MediaNamespace = "urn:x-cast:com.google.cast.media"
)

type MediaController struct {
	ch    *cast.Channel
	ctx   context.Context
	close context.CancelFunc
	rm    *requestManager
}

type mediaRequest struct {
	RequestHeader
	MediaSessionID int `json:"mediaSessionId"`
}

func NewMediaController(device *cast.Device, sourceId, destinationId string) *MediaController {
	ctx, close := context.WithCancel(context.Background())
	ch := device.NewChannel(MediaNamespace, sourceId, destinationId)
	return &MediaController{
		ch:    ch,
		ctx:   ctx,
		close: close,
		rm:    newRequestManager(ch),
	}
}

type mediaStatusResponse struct {
	ResponseHeader
	Status []MediaStatus `json:"status,omitempty"`
}

type MediaStatus struct {
	MediaSessionID         int                    `json:"mediaSessionId"`
	PlaybackRate           float64                `json:"playbackRate"`
	PlayerState            string                 `json:"playerState"`
	CurrentTime            float64                `json:"currentTime"`
	SupportedMediaCommands int                    `json:"supportedMediaCommands"`
	CustomData             map[string]interface{} `json:"customData"`
	IdleReason             string                 `json:"idleReason"`
}

func (r *MediaController) GetStatus() ([]MediaStatus, error) {
	return r.requestStatus(&RequestHeader{
		PayloadHeaders: PayloadHeaders{"GET_STATUS"},
	})
}

type StreamType string
type MediaType int
type TrackType int
type TrackSubtype int

const (
	UnknownDuration time.Duration = -1

	//StreamTypeInvalid  StreamType = -1
	//StreamTypeNone     StreamType = 0
	StreamTypeBuffered StreamType = "BUFFERED"
	StreamTypeLive     StreamType = "LIVE"

	MediaTypeGeneric    MediaType = 0
	MediaTypeMovie      MediaType = 1
	MediaTypeTVShow     MediaType = 1
	MediaTypeMusicTrack MediaType = 3
	MediaTypePhoto      MediaType = 4
	MediaTypeUser       MediaType = 4

	TrackTypeUnknown TrackType = 0
	TrackTypeText    TrackType = 1
	TrackTypeAudio   TrackType = 2
	TrackTypeVideo   TrackType = 3

	TrackSubtypeUnknown      TrackSubtype = -1
	TrackSubtypeNone         TrackSubtype = 0
	TrackSubtypeCaptions     TrackSubtype = 2
	TrackSubtypeChapters     TrackSubtype = 4
	TrackSubtypeDescriptions TrackSubtype = 3
	TrackSubtypeMetadata     TrackSubtype = 5
)

var (
	LoadFailed    = errors.New("Load failed")
	LoadCancelled = errors.New("Load cancelled")
)

type MediaInfo struct {
	ContentID      string                 `json:"contentId"`
	ContentType    string                 `json:"contentType"`
	CustomData     interface{}            `json:"customData,omitempty"`
	MediaTracks    []MediaTrack           `json:"mediaTracks,omitempty"`
	Metadata       map[string]interface{} `json:"mediaMetadata,omitempty"`
	StreamDuration time.Duration          `json:"streamDuration,omitempty"`
	StreamType     StreamType             `json:"streamType"`
	// TODO TextTrackStyle
}

type MediaTrack struct {
	ID          int64        `json:"id"`
	ContentID   string       `json:"contentId"`
	ContentType string       `json:"contentType"`
	CustomData  interface{}  `json:"customData"`
	Language    string       `json:"language"`
	Name        string       `json:"name"`
	Type        TrackType    `json:"type"`
	Subtype     TrackSubtype `json:"subtype"`
}

type LoadOptions struct {
	AutoPlay       bool          `json:"autoplay,omitempty"`
	PlayPosition   time.Duration `json:"currentTime,omitempty"`
	ActiveTrackIDs []int64       `json:"activeTrackIds,omitempty"`
	CustomData     interface{}   `json:"customData,omitempty"`
	// TODO RepeatMode?
}

func (r *MediaController) Load(media MediaInfo, options LoadOptions) ([]MediaStatus, error) {
	request := &struct {
		LoadOptions
		RequestHeader
		Media MediaInfo `json:"media"`
	}{
		RequestHeader: RequestHeader{
			PayloadHeaders: PayloadHeaders{Type: "LOAD"},
		},
		LoadOptions: options,
		Media:       media,
	}

	response, err := r.rm.Request(request)
	if err != nil {
		return nil, err
	}

	responseHeader := &ResponseHeader{}
	err = response.Unmarshal(responseHeader)
	if err != nil {
		return nil, err
	}

	if responseHeader.Type == "LOAD_FAILED" {
		return nil, LoadFailed
	} else if responseHeader.Type == "LOAD_CANCELLED" {
		return nil, LoadCancelled
	}

	mediaStatusResponse := &mediaStatusResponse{}
	err = response.Unmarshal(mediaStatusResponse)
	if err != nil {
		return nil, err
	}

	return mediaStatusResponse.Status, nil
}

type sessionRequest struct {
	RequestHeader
	MediaSessionID int `json:"mediaSessionId"`
}

func (r *MediaController) Play(sessionId int) ([]MediaStatus, error) {
	return r.sessionRequest(sessionId, "PLAY")
}

func (r *MediaController) Seek(sessionId int, position float64) ([]MediaStatus, error) {
	request := &struct {
		sessionRequest
		CurrentTime float64 `json:"currentTime"`
	}{
		sessionRequest: sessionRequest{
			RequestHeader: RequestHeader{
				PayloadHeaders: PayloadHeaders{Type: "SEEK"},
			},
			MediaSessionID: sessionId,
		},
		CurrentTime: position,
	}
	return r.requestStatus(request)
}

func (r *MediaController) sessionRequest(sessionId int, typ string) ([]MediaStatus, error) {
	request := &sessionRequest{
		RequestHeader: RequestHeader{
			PayloadHeaders: PayloadHeaders{Type: typ},
		},
		MediaSessionID: sessionId,
	}
	return r.requestStatus(request)
}

func (r *MediaController) requestStatus(request Request) ([]MediaStatus, error) {
	response, err := r.rm.Request(request)
	if err != nil {
		return nil, err
	}

	mediaStatusResponse := &mediaStatusResponse{}
	err = response.Unmarshal(mediaStatusResponse)
	if err != nil {
		return nil, err
	}

	return mediaStatusResponse.Status, nil
}

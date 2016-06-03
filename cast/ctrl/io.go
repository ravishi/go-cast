package ctrl

import (
	"encoding/json"
	"github.com/ravishi/go-cast/cast"
)

func send(ch *cast.Channel, payload interface{}) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return ch.Send(string(jsonData))
}

func getHeaders(message *cast.CastMessage) (*PayloadHeaders, error) {
	header := &PayloadHeaders{}
	err := json.Unmarshal([]byte(*message.PayloadUtf8), header)
	if err != nil {
		return nil, err
	}
	return header, nil
}

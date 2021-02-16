package ctrl

import (
	"encoding/json"

	"github.com/ravishi/go-cast/pkg/cast"
)

func send(ch *cast.Channel, payload interface{}) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return ch.Send(string(jsonData))
}

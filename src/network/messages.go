package network

import (
	"elevator/types"
	"encoding/json"
)

func JsonToMsg[T types.Content](encoded []byte) (*types.Msg[T], error) {
	var msg types.Msg[T]

	err := json.Unmarshal(encoded, &msg)

	if err != nil {
		return nil, err
	}

	return &msg, nil
}

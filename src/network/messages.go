package network

import (
	"elevator/types"
	"encoding/json"
)

func JsonToMsg[T types.Content](encoded []byte) types.Msg[T] {
	var msg types.Msg[T]

	err := json.Unmarshal(encoded, &msg)

	if err != nil {
		panic(err)
	}

	return msg
}

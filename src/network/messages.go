package network

import (
	"elevator/types"
	"encoding/json"
)

const SIZE_OF_HEADER = 23

func GetMsgHeader(encodedMsg []byte) (*types.Header, error) {
	var header types.Header

	encodedHeader := encodedMsg[:SIZE_OF_HEADER]

	err := json.Unmarshal(encodedHeader, &header)

	if err != nil {
		return nil, err
	}

	return &header, nil
}

func GetMsgContent[T types.Content](encodedMsg []byte) (*T, error) {
	var content T

	encodedHeader := encodedMsg[SIZE_OF_HEADER:]

	err := json.Unmarshal(encodedHeader, &content)

	if err != nil {
		return nil, err
	}

	return &content, nil
}

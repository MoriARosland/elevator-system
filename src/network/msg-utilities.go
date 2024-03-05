package network

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
)

/*
 * Function using Marshal to send messges as JSON structs
 */
func MarshalToBytes(msg interface{}) ([]byte, error) {
	data, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}
	return data, nil
}

/*
 * Function using Unmarshal to decode messages as JSON structs
 */
func UnmarshalFromBytes(data []byte, msg interface{}) error {
	err := json.Unmarshal(data, msg)
	if err != nil {
		return err
	}
	return nil
}

/*
 * Function to seriealize any type of value using go generic functions,
 * copied from: https://medium.com/lyonas/go-1-21-generic-functions-comprehensive-guide-6528b37feb5c
 */
func Serialize[T any](msg T) ([]byte, error) {
	buffer := bytes.Buffer{}

	encoder := gob.NewEncoder(&buffer)
	err := encoder.Encode(msg)
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

/*
 * Function to deseriealize any type of value using go generic functions,
 * copied from: https://medium.com/lyonas/go-1-21-generic-functions-comprehensive-guide-6528b37feb5c
 */
func Deserialize[T any](b []byte) (T, error) {
	buffer := bytes.Buffer{}
	buffer.Write(b)

	decoder := gob.NewDecoder(&buffer)
	var data T
	err := decoder.Decode(&data)
	if err != nil {
		return data, err
	}

	return data, nil
}

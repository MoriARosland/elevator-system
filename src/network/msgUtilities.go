package network

import (
	"bytes"
	"elevator/types"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"reflect"
)

/*
 * Inspired by example code from the GitHub repository: https://github.com/TTK4145/Network-go
 * The input message should be a custom message struct defined in types.go, such as bid, assig, reassign, served or sync.
 */
func MsgToJson(content any, authorID int) []byte {

	jsonContent, err := json.Marshal(content)
	if err != nil {
		panic(err)
	}

	msg := types.JsonStrMsg{
		AuthorID: authorID,
		Type:     reflect.TypeOf(content).String(),
		Content:  jsonContent,
	}

	jsonMsg, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}

	return jsonMsg
}

func JsonToMsg(jsonMsg []byte) types.Msg {
	var tempMsg types.JsonStrMsg

	err := json.Unmarshal(jsonMsg, &tempMsg)
	if err != nil {
		panic(err)
	}

	var msg types.Msg

	switch tempMsg.Type {
	case "types.Bid":
		var bid types.Bid
		_ = json.Unmarshal(tempMsg.Content, &bid)
		msg = types.Msg{
			AuthorID: tempMsg.AuthorID,
			Type:     tempMsg.Type,
			Content:  bid,
		}
	case "types.Assign":
		var assign types.Assign
		_ = json.Unmarshal(tempMsg.Content, &assign)
		msg = types.Msg{
			AuthorID: tempMsg.AuthorID,
			Type:     tempMsg.Type,
			Content:  assign,
		}
	case "types.Reassign":
		var reassign types.Reassign
		_ = json.Unmarshal(tempMsg.Content, &reassign)
		msg = types.Msg{
			AuthorID: tempMsg.AuthorID,
			Type:     tempMsg.Type,
			Content:  reassign,
		}
	case "types.Served":
		var served types.Served
		_ = json.Unmarshal(tempMsg.Content, &served)
		msg = types.Msg{
			AuthorID: tempMsg.AuthorID,
			Type:     tempMsg.Type,
			Content:  served,
		}
	case "types.Sync":
		var sync types.Sync
		_ = json.Unmarshal(tempMsg.Content, &sync)
		msg = types.Msg{
			AuthorID: tempMsg.AuthorID,
			Type:     tempMsg.Type,
			Content:  sync,
		}
	default:
		/*
		 * Do nothing
		 */
	}

	fmt.Println("msg: ", msg)

	return msg
}

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

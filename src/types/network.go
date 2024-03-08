package types

import (
	"bytes"
	"encoding/json"
)

type MsgTypes int

const (
	BID MsgTypes = iota
	ASSIGN
	REASSIGN
	SERVED
	SYNC
)

type Bid struct {
	Order        Order
	TimeToServed []int
}

type Assign struct {
	Order    Order
	Assignee int
}

type Reassign struct {
	Order       Order
	NewAssignee int
	OldAssignee int
}

type Served struct {
	Order Order
}

type Sync struct {
	Orders [][][]bool
}

type Header struct {
	Type     MsgTypes
	AuthorID int // must contain a single digit number [0, 9] in order to properly decode messages
}

type Content interface {
	Bid | Assign | Reassign | Served | Sync
}

type Msg[T Content] struct {
	Header  Header
	Content T
}

func (msg Msg[T]) ToJson() []byte {
	encodedContent, err := json.Marshal(msg.Content)

	if err != nil {
		panic(err)
	}

	encodedHeader, err := json.Marshal(msg.Header)

	if err != nil {
		panic(err)
	}

	separator := []byte("")

	return bytes.Join([][]byte{encodedHeader, encodedContent}, separator)
}

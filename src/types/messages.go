package types

import (
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

type Content interface {
	Bid | Assign | Reassign | Served | Sync
}

type Msg[T Content] struct {
	Content T
}

type MsgHeader struct {
	Type     MsgTypes
	AuthorID int // must contain a single digit number [0, 9] in order to properly decode messages
}

func (msg Msg[T]) ToJson() ([]byte, error) {
	encodedMsg, err := json.Marshal(msg)

	if err != nil {
		return nil, err
	}

	return encodedMsg, nil
}

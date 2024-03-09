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

func FormatAssignMsg(order types.Order, assignee int, author int) []byte {
	msg := types.Msg[types.Assign]{
		Header: types.Header{
			Type:     types.ASSIGN,
			AuthorID: author,
		},
		Content: types.Assign{
			Order:    order,
			Assignee: assignee,
		},
	}

	return msg.ToJson()
}

func FormatServedMsg(order types.Order, author int) []byte {
	msg := types.Msg[types.Served]{
		Header: types.Header{
			Type:     types.SERVED,
			AuthorID: author,
		},
		Content: types.Served{
			Order: order,
		},
	}

	return msg.ToJson()
}

func FormatBidMsg(timeToServed []int, order types.Order, NumNodes int, author int) []byte {
	if len(timeToServed) == 0 {

		timeToServed = make([]int, NumNodes)

		for NodeID := range timeToServed {
			timeToServed[NodeID] = -1
		}
	}

	msg := types.Msg[types.Bid]{
		Header: types.Header{
			Type:     types.BID,
			AuthorID: author,
		},
		Content: types.Bid{
			Order:        order,
			TimeToServed: timeToServed,
		},
	}

	return msg.ToJson()
}

func FormatSyncMsg(target types.NextNode, orders [][][]bool, author int) []byte {
	msg := types.Msg[types.Sync]{
		Header: types.Header{
			Type:     types.SYNC,
			AuthorID: author,
		},
		Content: types.Sync{
			Orders: orders,
			Target: target,
		},
	}

	return msg.ToJson()
}

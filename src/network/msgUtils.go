package network

import (
	"elevator/types"
)

func FormatBidMsg(
	timeToServed []int,
	order types.Order,
	oldAssignee int,
	NumNodes int,
	recipient int,
	author int,
	loopCounter ...int,
) types.Msg[types.Bid] {
	tempLoopCounter := 0
	if len(loopCounter) > 0 {
		tempLoopCounter = loopCounter[0]
	}

	if len(timeToServed) == 0 {
		timeToServed = make([]int, NumNodes)

		for NodeID := range timeToServed {
			timeToServed[NodeID] = -1
		}
	}

	msg := types.Msg[types.Bid]{
		Header: types.Header{
			AuthorID:  author,
			Recipient: recipient,
			LoopCounter: tempLoopCounter,
		},
		Content: types.Bid{
			Order:        order,
			TimeToServed: timeToServed,
			OldAssignee:  oldAssignee,
		},
	}

	return msg
}

func FormatAssignMsg(
	order types.Order,
	newAssignee int,
	oldAssignee int,
	recipient int,
	author int,
	loopCounter ...int,
) types.Msg[types.Assign] {
	tempLoopCounter := 0
	if len(loopCounter) > 0 {
		tempLoopCounter = loopCounter[0]
	}

	msg := types.Msg[types.Assign]{
		Header: types.Header{
			AuthorID:  author,
			Recipient: recipient,
			LoopCounter: tempLoopCounter,
		},
		Content: types.Assign{
			Order:       order,
			NewAssignee: newAssignee,
			OldAssignee: oldAssignee,
		},
	}

	return msg
}

func FormatServedMsg(
	order types.Order,
	recipient int,
	author int,
	loopCounter ...int,
) types.Msg[types.Served] {
	tempLoopCounter := 0
	if len(loopCounter) > 0 {
		tempLoopCounter = loopCounter[0]
	}

	msg := types.Msg[types.Served]{
		Header: types.Header{
			AuthorID:  author,
			Recipient: recipient,
			LoopCounter: tempLoopCounter,
		},
		Content: types.Served{
			Order: order,
		},
	}

	return msg
}

func FormatSyncMsg(
	orders [][][]bool,
	syncTarget int,
	recipient int,
	author int,
	loopCounter ...int,
) types.Msg[types.Sync] {
	tempLoopCounter := 0
	if len(loopCounter) > 0 {
		tempLoopCounter = loopCounter[0]
	}

	msg := types.Msg[types.Sync]{
		Header: types.Header{
			AuthorID:  author,
			Recipient: recipient,
			LoopCounter: tempLoopCounter,
		},
		Content: types.Sync{
			Orders:   orders,
			TargetID: syncTarget,
		},
	}

	return msg
}

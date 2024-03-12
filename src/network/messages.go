package network

import (
	"elevator/types"
)

func FormatAssignMsg(
	order types.Order,
	newAssignee int,
	oldAssignee int,
	recipient int,
	author int,
) types.Msg[types.Assign] {

	msg := types.Msg[types.Assign]{
		Header: types.Header{
			AuthorID:  author,
			Recipient: recipient,
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
) types.Msg[types.Served] {

	msg := types.Msg[types.Served]{
		Header: types.Header{
			AuthorID:  author,
			Recipient: recipient,
		},
		Content: types.Served{
			Order: order,
		},
	}

	return msg
}

func FormatBidMsg(
	timeToServed []int,
	order types.Order,
	oldAssignee int,
	NumNodes int,
	recipient int,
	author int,
) types.Msg[types.Bid] {

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
		},
		Content: types.Bid{
			Order:        order,
			TimeToServed: timeToServed,
			OldAssignee:  oldAssignee,
		},
	}

	return msg
}

func FormatSyncMsg(
	orders [][][]bool,
	targetID int,
	recipient int,
	author int,
) types.Msg[types.Sync] {

	msg := types.Msg[types.Sync]{
		Header: types.Header{
			AuthorID:  author,
			Recipient: recipient,
		},
		Content: types.Sync{
			Orders:   orders,
			TargetID: targetID,
		},
	}

	return msg
}

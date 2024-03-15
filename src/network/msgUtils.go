package network

import (
	"crypto/rand"
	"elevator/types"
	"fmt"
)

/*
 * From stackoverflow:
 * https://shorturl.at/bfxAI
 */
func pseudo_uuid() (uuid string) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		fmt.Println("Error: ", err)
		return
	}

	uuid = fmt.Sprintf("%X-%X-%X-%X-%X", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])

	return
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
			UUID:      pseudo_uuid(),
			LoopCounter: 0,
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
) types.Msg[types.Assign] {
	msg := types.Msg[types.Assign]{
		Header: types.Header{
			AuthorID:  author,
			Recipient: recipient,
			UUID:      pseudo_uuid(),
			LoopCounter: 0,
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
			UUID:      pseudo_uuid(),
			LoopCounter: 0,
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
) types.Msg[types.Sync] {
	msg := types.Msg[types.Sync]{
		Header: types.Header{
			AuthorID:  author,
			Recipient: recipient,
			UUID:      pseudo_uuid(),
			LoopCounter: 0,
		},
		Content: types.Sync{
			Orders:   orders,
			TargetID: syncTarget,
		},
	}

	return msg
}

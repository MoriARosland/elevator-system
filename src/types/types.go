package types

import (
	"Driver-go/elevio"
)

type Order = elevio.ButtonEvent

type DirnBehaviourPair struct {
	Dirn      elevio.MotorDirection
	Behaviour ElevBehaviour
}

type ElevBehaviour int

const (
	EB_Idle ElevBehaviour = iota
	EB_DoorOpen
	EB_Moving
)

type FsmOutput struct {
	ElevDirn       elevio.MotorDirection
	MotorDirn      elevio.MotorDirection
	SetMotor       bool
	Door           bool
	StartDoorTimer bool
	ClearOrders    [3]bool
}

type NextNode struct {
	ID   int
	Addr string
}

type ElevConfig struct {
	NodeID           int
	NumNodes         int
	NumFloors        int
	NumButtons       int
	DoorOpenDuration int
	BroadcastPort    int
}

type ElevState struct {
	Floor           int
	Dirn            elevio.MotorDirection
	DoorObstr       bool
	Orders          [][][]bool
	NextNode        NextNode
	WaitingForReply bool
}

type Msg struct {
	AuthorID int
	Type     string // The type of the value in Content, stored as a string
	Content  any    // The content of a message should be one of the structs below (bid, assign, reassign, served or sync)
}

type JsonStrMsg struct {
	AuthorID int
	Type     string
	Content  []byte
}

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

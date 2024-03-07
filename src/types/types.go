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
	Requests        [][][]bool
	NextNode        NextNode
	WaitingForReply bool
}

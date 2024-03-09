package types

import "Driver-go/elevio"

type Order = elevio.ButtonEvent

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
	ProcessingOrder bool
	Disconnected    bool
}

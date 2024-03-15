package types

import "Driver-go/elevio"

type Order = elevio.ButtonEvent

type ElevConfig struct {
	NodeID           int
	NumNodes         int
	NumFloors        int
	NumButtons       int
	DoorOpenDuration int
}

type ElevState struct {
	Floor              int
	Dirn               elevio.MotorDirection
	StuckBetweenFloors bool
	DoorObstr          bool
	Orders             [][][]bool
	NextNodeID         int
}

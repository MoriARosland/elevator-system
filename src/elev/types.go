package elev

import "Driver-go/elevio"

type NextNode struct {
	ID   int
	Addr string
}

type ElevConfig struct {
	NodeID           int
	NumNodes         int
	NumFloors        int
	DoorOpenDuration int
	BroadcastPort    int
}

type ElevState struct {
	Floor    int
	Dirn     elevio.MotorDirection
	Requests [][]bool
	NextNode NextNode
}

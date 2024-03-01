package fsm

type ElevBehaviour int

const (
	EB_Idle ElevBehaviour = iota
	EB_DoorOpen
	EB_Moving
)

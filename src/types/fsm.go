package types

import (
	"Driver-go/elevio"
)

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

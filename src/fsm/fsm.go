package fsm

import (
	"Driver-go/elevio"
	"elevator/requests"
	"elevator/types"
)

var state types.ElevBehaviour = types.EB_Idle

func OnInitBetweenFloors() {
	state = types.EB_Moving
}

func OnOrderAssigned(
	newOrder elevio.ButtonEvent,
	elevState *types.ElevState,
	elevConfig *types.ElevConfig,
) types.FsmOutput {

	output := types.FsmOutput{
		ElevDirn:       elevState.Dirn,
		Door:           state == types.EB_DoorOpen,
		StartDoorTimer: false,
	}

	switch state {
	case types.EB_DoorOpen:
		if requests.ShouldClearImmediately(elevState, newOrder) {
			output.StartDoorTimer = true
			output.ClearOrders[newOrder.Button] = true
		}

	case types.EB_Idle:
		pair := requests.ChooseDirection(elevState, elevConfig)

		output.ElevDirn = pair.Dirn
		state = pair.Behaviour

		switch state {
		case types.EB_DoorOpen:
			output.ClearOrders = requests.ClearAtCurrentFloor(elevState, elevConfig)
			output.Door = true
			output.StartDoorTimer = true

		case types.EB_Moving:
			output.MotorDirn = pair.Dirn
			output.SetMotor = true
		}
	}

	return output
}

func OnFloorArrival(
	elevState *types.ElevState,
	elevConfig *types.ElevConfig,
) types.FsmOutput {

	output := types.FsmOutput{
		ElevDirn:       elevState.Dirn,
		Door:           state == types.EB_DoorOpen,
		StartDoorTimer: false,
	}

	shouldStop := requests.ShouldStop(elevState, elevConfig)

	if state == types.EB_Moving && shouldStop {
		output.MotorDirn = elevio.MD_Stop
		output.SetMotor = true

		output.Door = true
		output.StartDoorTimer = true

		output.ClearOrders = requests.ClearAtCurrentFloor(elevState, elevConfig)

		state = types.EB_DoorOpen
	}

	return output
}

func OnDoorTimeout(
	elevState *types.ElevState,
	elevConfig *types.ElevConfig,
) types.FsmOutput {

	output := types.FsmOutput{
		ElevDirn:       elevState.Dirn,
		Door:           state == types.EB_DoorOpen,
		StartDoorTimer: false,
	}

	if state != types.EB_DoorOpen {
		return output
	}

	pair := requests.ChooseDirection(elevState, elevConfig)

	output.ElevDirn = pair.Dirn
	state = pair.Behaviour

	if state == types.EB_DoorOpen {
		output.StartDoorTimer = true
		output.ClearOrders = requests.ClearAtCurrentFloor(elevState, elevConfig)
	} else {
		output.Door = false
		output.MotorDirn = pair.Dirn
		output.SetMotor = true
	}

	return output
}

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
		Dirn:           elevState.Dirn,
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

		output.Dirn = pair.Dirn
		state = pair.Behaviour

		switch state {
		case types.EB_DoorOpen:
			output.ClearOrders = requests.ClearAtCurrentFloor(elevState, elevConfig)
			output.Door = true
			output.StartDoorTimer = true

		case types.EB_Moving:
			output.Dirn = pair.Dirn
		}
	}

	return output
}

func OnFloorArrival(
	elevState *types.ElevState,
	elevConfig *types.ElevConfig,
) types.FsmOutput {

	output := types.FsmOutput{
		Dirn:           elevState.Dirn,
		Door:           state == types.EB_DoorOpen,
		StartDoorTimer: false,
	}

	if state == types.EB_Moving && requests.ShouldStop(elevState, elevConfig) {
		output.Dirn = elevio.MD_Stop

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
		Dirn:           elevState.Dirn,
		Door:           state == types.EB_DoorOpen,
		StartDoorTimer: false,
	}

	if state != types.EB_DoorOpen {
		return output
	}

	pair := requests.ChooseDirection(elevState, elevConfig)

	output.Dirn = pair.Dirn
	state = pair.Behaviour

	if state == types.EB_DoorOpen {
		output.StartDoorTimer = true
		output.ClearOrders = requests.ClearAtCurrentFloor(elevState, elevConfig)
	} else {
		elevio.SetDoorOpenLamp(false)
		elevio.SetMotorDirection(elevState.Dirn)

		output.Door = false
		output.Dirn = pair.Dirn
	}

	return output
}

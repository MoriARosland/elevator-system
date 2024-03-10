package fsm

import (
	"Driver-go/elevio"
	"elevator/orders"
	"elevator/types"
)

var state types.ElevBehaviour = types.EB_Idle

func OnInitBetweenFloors() {
	state = types.EB_Moving
}

func OnOrderAssigned(
	newOrder types.Order,
	elevState *types.ElevState,
	elevConfig *types.ElevConfig,
) types.FsmOutput {

	output := types.FsmOutput{
		ElevDirn: elevState.Dirn,
		Door:     state == types.EB_DoorOpen,
	}

	switch state {
	case types.EB_DoorOpen:
		if orders.ShouldClearImmediately(elevState, newOrder) {
			output.StartDoorTimer = true
			output.ClearOrders[newOrder.Button] = true
		}

	case types.EB_Idle:
		pair := orders.ChooseDirection(elevState, elevConfig)

		output.ElevDirn = pair.Dirn
		state = pair.Behaviour

		switch state {
		case types.EB_DoorOpen:
			output.ClearOrders = orders.ClearAtCurrentFloor(elevState, elevConfig)
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
		ElevDirn: elevState.Dirn,
		Door:     state == types.EB_DoorOpen,
	}

	shouldStop := orders.ShouldStop(elevState, elevConfig)

	if state == types.EB_Moving && shouldStop {
		output.MotorDirn = elevio.MD_Stop
		output.SetMotor = true

		output.Door = true
		output.StartDoorTimer = true

		output.ClearOrders = orders.ClearAtCurrentFloor(elevState, elevConfig)

		state = types.EB_DoorOpen
	}

	return output
}

func OnDoorTimeout(
	elevState *types.ElevState,
	elevConfig *types.ElevConfig,
) types.FsmOutput {

	output := types.FsmOutput{
		ElevDirn: elevState.Dirn,
		Door:     state == types.EB_DoorOpen,
	}

	if state != types.EB_DoorOpen {
		return output
	}

	pair := orders.ChooseDirection(elevState, elevConfig)

	output.ElevDirn = pair.Dirn
	state = pair.Behaviour

	if state == types.EB_DoorOpen {
		output.StartDoorTimer = true
		output.ClearOrders = orders.ClearAtCurrentFloor(elevState, elevConfig)
	} else {
		output.Door = false
		output.MotorDirn = pair.Dirn
		output.SetMotor = true
	}

	return output
}

func OnSync(
	elevState *types.ElevState,
	elevConfig *types.ElevConfig,
) types.FsmOutput {

	output := types.FsmOutput{
		ElevDirn: elevState.Dirn,
		Door:     state == types.EB_DoorOpen,
	}

	pair := orders.ChooseDirection(elevState, elevConfig)

	output.ElevDirn = pair.Dirn
	state = pair.Behaviour

	if state == types.EB_DoorOpen {
		output.StartDoorTimer = true
		output.ClearOrders = orders.ClearAtCurrentFloor(elevState, elevConfig)
	} else {
		output.Door = false
		output.MotorDirn = pair.Dirn
		output.SetMotor = true
	}

	return output
}

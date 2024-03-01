package fsm

import (
	"Driver-go/elevio"
	"elevator/elev"
	"elevator/requests"
	"elevator/timer"
	"elevator/types"
)

var state types.ElevBehaviour = types.EB_Idle

func OnInitBetweenFloors() {
	state = types.EB_Moving
}

func OnRequestButtonPress(
	buttonPress elevio.ButtonEvent,
	elevState *types.ElevState,
	elevConfig *types.ElevConfig,
) {
	switch state {
	case types.EB_DoorOpen:
		if requests.ShouldClearImmediately(elevState, buttonPress) {
			timer.Start(elevConfig.DoorOpenDuration)
		} else {
			elevState.Requests[buttonPress.Floor][buttonPress.Button] = true
		}

	case types.EB_Moving:
		elevState.Requests[buttonPress.Floor][buttonPress.Button] = true

	case types.EB_Idle:
		elevState.Requests[buttonPress.Floor][buttonPress.Button] = true
		pair := requests.ChooseDirection(elevState, elevConfig)

		elevState.Dirn = pair.Dirn
		state = pair.Behaviour

		switch pair.Behaviour {
		case types.EB_DoorOpen:
			elevio.SetDoorOpenLamp(true)
			timer.Start(elevConfig.DoorOpenDuration)
			requests.ClearAtcurrentFloor(elevState, elevConfig)

		case types.EB_Moving:
			elevio.SetMotorDirection(pair.Dirn)
		}
	}

	elev.SetAllLights(elevState, elevConfig)
}

func OnFloorArrival(
	floor int,
	elevState *types.ElevState,
	elevConfig *types.ElevConfig,
) {
	elevState.Floor = floor
	elevio.SetFloorIndicator(elevState.Floor)

	if state == types.EB_Moving && requests.ShouldStop(elevState, elevConfig) {
		elevio.SetMotorDirection(elevio.MD_Stop)
		elevio.SetDoorOpenLamp(true)

		requests.ClearAtcurrentFloor(elevState, elevConfig)

		timer.Start(elevConfig.DoorOpenDuration)
		elev.SetAllLights(elevState, elevConfig)
		state = types.EB_DoorOpen
	}
}

func OnDoorTimeout(elevState *types.ElevState, elevConfig *types.ElevConfig) {
	if state != types.EB_DoorOpen {
		return
	}

	pair := requests.ChooseDirection(elevState, elevConfig)

	elevState.Dirn = pair.Dirn
	state = pair.Behaviour

	if state == types.EB_DoorOpen {
		timer.Start(elevConfig.DoorOpenDuration)
		requests.ClearAtcurrentFloor(elevState, elevConfig)
		elev.SetAllLights(elevState, elevConfig)
	} else {
		elevio.SetDoorOpenLamp(false)
		elevio.SetMotorDirection(elevState.Dirn)
	}
}

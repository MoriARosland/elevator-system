package requests

import (
	"Driver-go/elevio"
	"elevator/types"
)

func requestsAbove(elevState *types.ElevState, elevConfig *types.ElevConfig) bool {
	for floor := elevState.Floor + 1; floor < elevConfig.NumFloors; floor++ {
		for btn := 0; btn < elevConfig.NumButtons; btn++ {
			if elevState.Requests[elevConfig.NodeID][floor][btn] {
				return true
			}
		}
	}

	return false
}

func requestsBelow(elevState *types.ElevState, elevConfig *types.ElevConfig) bool {
	for floor := 0; floor < elevState.Floor; floor++ {
		for btn := 0; btn < elevConfig.NumButtons; btn++ {
			if elevState.Requests[elevConfig.NodeID][floor][btn] {
				return true
			}
		}
	}

	return false
}

func requestsHere(elevState *types.ElevState, elevConfig *types.ElevConfig) bool {
	for btn := 0; btn < elevConfig.NumButtons; btn++ {
		if elevState.Requests[elevConfig.NodeID][elevState.Floor][btn] {
			return true
		}
	}

	return false
}

func ChooseDirection(elevState *types.ElevState, elevConfig *types.ElevConfig) types.DirnBehaviourPair {
	switch elevState.Dirn {
	case elevio.MD_Up:
		if requestsAbove(elevState, elevConfig) {
			return types.DirnBehaviourPair{Dirn: elevio.MD_Up, Behaviour: types.EB_Moving}
		} else if requestsHere(elevState, elevConfig) {
			return types.DirnBehaviourPair{Dirn: elevio.MD_Down, Behaviour: types.EB_DoorOpen}
		} else if requestsBelow(elevState, elevConfig) {
			return types.DirnBehaviourPair{Dirn: elevio.MD_Down, Behaviour: types.EB_Moving}
		} else {
			return types.DirnBehaviourPair{Dirn: elevio.MD_Stop, Behaviour: types.EB_Idle}
		}

	case elevio.MD_Down:
		if requestsBelow(elevState, elevConfig) {
			return types.DirnBehaviourPair{Dirn: elevio.MD_Down, Behaviour: types.EB_Moving}
		} else if requestsHere(elevState, elevConfig) {
			return types.DirnBehaviourPair{Dirn: elevio.MD_Up, Behaviour: types.EB_DoorOpen}
		} else if requestsAbove(elevState, elevConfig) {
			return types.DirnBehaviourPair{Dirn: elevio.MD_Up, Behaviour: types.EB_Moving}
		} else {
			return types.DirnBehaviourPair{Dirn: elevio.MD_Stop, Behaviour: types.EB_Idle}
		}

	case elevio.MD_Stop:
		if requestsHere(elevState, elevConfig) {
			return types.DirnBehaviourPair{Dirn: elevio.MD_Stop, Behaviour: types.EB_DoorOpen}
		} else if requestsAbove(elevState, elevConfig) {
			return types.DirnBehaviourPair{Dirn: elevio.MD_Up, Behaviour: types.EB_Moving}
		} else if requestsBelow(elevState, elevConfig) {
			return types.DirnBehaviourPair{Dirn: elevio.MD_Down, Behaviour: types.EB_Moving}
		} else {
			return types.DirnBehaviourPair{Dirn: elevio.MD_Stop, Behaviour: types.EB_Idle}
		}

	default:
		return types.DirnBehaviourPair{Dirn: elevio.MD_Stop, Behaviour: types.EB_Idle}
	}
}

func ShouldStop(elevState *types.ElevState, elevConfig *types.ElevConfig) bool {
	switch elevState.Dirn {
	case elevio.MD_Down:
		return (elevState.Requests[elevConfig.NodeID][elevState.Floor][elevio.BT_HallDown] ||
			elevState.Requests[elevConfig.NodeID][elevState.Floor][elevio.BT_Cab] ||
			!requestsBelow(elevState, elevConfig))

	case elevio.MD_Up:
		return (elevState.Requests[elevConfig.NodeID][elevState.Floor][elevio.BT_HallUp] ||
			elevState.Requests[elevConfig.NodeID][elevState.Floor][elevio.BT_Cab] ||
			!requestsAbove(elevState, elevConfig))

	default:
		return true
	}
}

func ShouldClearImmediately(elevState *types.ElevState, buttonPress elevio.ButtonEvent) bool {
	/*
	 * TODO: check project requirements to make sure clearing is handled correctly
	 */
	return elevState.Floor == buttonPress.Floor
}

func ClearAtcurrentFloor(elevState *types.ElevState, elevConfig *types.ElevConfig) {
	/*
	 * ...same here
	 */
	for btn := 0; btn < elevConfig.NumButtons; btn++ {
		elevState.Requests[elevConfig.NodeID][elevState.Floor][btn] = false
	}
}

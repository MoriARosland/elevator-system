package orders

import (
	"Driver-go/elevio"
	"elevator/types"
)

func ordersAbove(elevState *types.ElevState, elevConfig *types.ElevConfig) bool {
	for floor := elevState.Floor + 1; floor < elevConfig.NumFloors; floor++ {
		for btn := 0; btn < elevConfig.NumButtons; btn++ {
			if elevState.Orders[elevConfig.NodeID][floor][btn] {
				return true
			}
		}
	}

	return false
}

func ordersBelow(elevState *types.ElevState, elevConfig *types.ElevConfig) bool {
	for floor := 0; floor < elevState.Floor; floor++ {
		for btn := 0; btn < elevConfig.NumButtons; btn++ {
			if elevState.Orders[elevConfig.NodeID][floor][btn] {
				return true
			}
		}
	}

	return false
}

func ordersHere(elevState *types.ElevState, elevConfig *types.ElevConfig) bool {
	for btn := 0; btn < elevConfig.NumButtons; btn++ {
		if elevState.Orders[elevConfig.NodeID][elevState.Floor][btn] {
			return true
		}
	}

	return false
}

func ChooseDirection(elevState *types.ElevState, elevConfig *types.ElevConfig) types.DirnBehaviourPair {
	switch elevState.Dirn {
	case elevio.MD_Up:
		if ordersAbove(elevState, elevConfig) {
			return types.DirnBehaviourPair{Dirn: elevio.MD_Up, Behaviour: types.EB_Moving}
		} else if ordersHere(elevState, elevConfig) {
			return types.DirnBehaviourPair{Dirn: elevio.MD_Down, Behaviour: types.EB_DoorOpen}
		} else if ordersBelow(elevState, elevConfig) {
			return types.DirnBehaviourPair{Dirn: elevio.MD_Down, Behaviour: types.EB_Moving}
		} else {
			return types.DirnBehaviourPair{Dirn: elevio.MD_Stop, Behaviour: types.EB_Idle}
		}

	case elevio.MD_Down:
		if ordersBelow(elevState, elevConfig) {
			return types.DirnBehaviourPair{Dirn: elevio.MD_Down, Behaviour: types.EB_Moving}
		} else if ordersHere(elevState, elevConfig) {
			return types.DirnBehaviourPair{Dirn: elevio.MD_Up, Behaviour: types.EB_DoorOpen}
		} else if ordersAbove(elevState, elevConfig) {
			return types.DirnBehaviourPair{Dirn: elevio.MD_Up, Behaviour: types.EB_Moving}
		} else {
			return types.DirnBehaviourPair{Dirn: elevio.MD_Stop, Behaviour: types.EB_Idle}
		}

	case elevio.MD_Stop:
		if ordersHere(elevState, elevConfig) {
			return types.DirnBehaviourPair{Dirn: elevio.MD_Stop, Behaviour: types.EB_DoorOpen}
		} else if ordersAbove(elevState, elevConfig) {
			return types.DirnBehaviourPair{Dirn: elevio.MD_Up, Behaviour: types.EB_Moving}
		} else if ordersBelow(elevState, elevConfig) {
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
		return (elevState.Orders[elevConfig.NodeID][elevState.Floor][elevio.BT_HallDown] ||
			elevState.Orders[elevConfig.NodeID][elevState.Floor][elevio.BT_Cab] ||
			!ordersBelow(elevState, elevConfig))

	case elevio.MD_Up:
		return (elevState.Orders[elevConfig.NodeID][elevState.Floor][elevio.BT_HallUp] ||
			elevState.Orders[elevConfig.NodeID][elevState.Floor][elevio.BT_Cab] ||
			!ordersAbove(elevState, elevConfig))

	default:
		return true
	}
}

func ShouldClearImmediately(elevState *types.ElevState, order types.Order) bool {
	return elevState.Floor == order.Floor && ((elevState.Dirn == elevio.MD_Up && order.Button == elevio.BT_HallUp) ||
		(elevState.Dirn == elevio.MD_Down && order.Button == elevio.BT_HallDown) ||
		elevState.Dirn == elevio.MD_Stop ||
		order.Button == elevio.BT_Cab)
}

func ClearAtCurrentFloor(
	elevState *types.ElevState,
	elevConfig *types.ElevConfig,
) [3]bool {

	var clearOrders [3]bool
	floor := elevState.Floor
	id := elevConfig.NodeID

	clearOrders[elevio.BT_Cab] = true

	switch elevState.Dirn {
	case elevio.MD_Up:
		if !ordersAbove(elevState, elevConfig) &&
			!elevState.Orders[id][floor][elevio.BT_HallUp] {
			clearOrders[elevio.BT_HallDown] = true
		}
		clearOrders[elevio.BT_HallUp] = true

	case elevio.MD_Down:
		if !ordersBelow(elevState, elevConfig) &&
			!elevState.Orders[id][floor][elevio.BT_HallDown] {
			clearOrders[elevio.BT_HallUp] = true
		}
		clearOrders[elevio.BT_HallDown] = true

	default:
		clearOrders[elevio.BT_HallUp] = true
		clearOrders[elevio.BT_HallDown] = true
	}

	return clearOrders
}

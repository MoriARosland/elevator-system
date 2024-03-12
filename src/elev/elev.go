package elev

import (
	"Driver-go/elevio"
	"elevator/fsm"
	"elevator/network"
	"elevator/types"
)

/*
 * Takes in output from fsm, performs side effects and return new elev state
 */
func SetState(
	oldState *types.ElevState,
	elevConfig *types.ElevConfig,
	stateChanges types.FsmOutput,
	servedTx chan types.Msg[types.Served],
	doorTimer chan<- types.TimerActions,
	floorTimer chan<- types.TimerActions,
) *types.ElevState {

	if stateChanges.SetMotor {
		elevio.SetMotorDirection(stateChanges.MotorDirn)

		if stateChanges.MotorDirn != elevio.MD_Stop {
			floorTimer <- types.START
		}
	}

	elevio.SetDoorOpenLamp(stateChanges.Door)

	if stateChanges.StartDoorTimer {
		doorTimer <- types.START
	}

	newState := types.ElevState{
		Floor:        oldState.Floor,
		Dirn:         stateChanges.ElevDirn,
		DoorObstr:    oldState.DoorObstr,
		Orders:       oldState.Orders,
		Disconnected: oldState.Disconnected,
	}

	/*
	 * Clear served orders
	 */
	for order, clearOrder := range stateChanges.ClearOrders {
		if !clearOrder {
			continue
		}

		if newState.Disconnected {
			newState = *SetOrderStatus(
				&newState,
				elevConfig,
				elevConfig.NodeID,
				types.Order{
					Button: elevio.ButtonType(order),
					Floor:  newState.Floor,
				},
				false,
			)
		} else {
			servedTx <- network.FormatServedMsg(
				types.Order{
					Button: elevio.ButtonType(order),
					Floor:  newState.Floor,
				},
				newState.NextNodeID,
				elevConfig.NodeID,
			)
		}
	}

	return &newState
}

func SetHallLights(orders [][][]bool, elevConfig *types.ElevConfig) {
	// We are here skipping the cab buttons by subtracting 1 from elevConfig.NumButtons.
	// See type ButtonType in lib/driver-go-master/elevio/elevator_io.go for reference.

	combinedOrders := make([][]bool, elevConfig.NumFloors)

	for floor := range combinedOrders {
		combinedOrders[floor] = make([]bool, elevConfig.NumButtons-1)
	}

	for elevator := range orders {
		for floor := range orders[elevator] {
			for orderType := 0; orderType < elevConfig.NumButtons-1; orderType++ {
				combinedOrders[floor][orderType] = orders[elevator][floor][orderType] || combinedOrders[floor][orderType]
			}
		}
	}

	for floor := range combinedOrders {
		for orderType := 0; orderType < elevConfig.NumButtons-1; orderType++ {
			elevio.SetButtonLamp(elevio.ButtonType(orderType), floor, combinedOrders[floor][orderType])
		}
	}
}

func SetCabLights(orders [][]bool, elevConfig *types.ElevConfig) {
	for floor := range orders {
		elevio.SetButtonLamp(elevio.BT_Cab, floor, orders[floor][elevio.BT_Cab])
	}
}

func HandleDoorObstr(
	elevState *types.ElevState,
	isObstructed bool,
	obstrTimer chan types.TimerActions,
	doorTimer chan types.TimerActions,
) *types.ElevState {

	if elevState.DoorObstr == isObstructed {
		return elevState
	}

	if isObstructed {
		obstrTimer <- types.START
	} else {
		obstrTimer <- types.STOP
	}

	doorTimer <- types.START
	elevState.DoorObstr = isObstructed

	return elevState
}

func SetOrderStatus(
	elevState *types.ElevState,
	elevConfig *types.ElevConfig,
	assignee int,
	order types.Order,
	newStatus bool,
) *types.ElevState {

	elevState.Orders[assignee][order.Floor][order.Button] = newStatus

	SetCabLights(elevState.Orders[elevConfig.NodeID], elevConfig)
	SetHallLights(elevState.Orders, elevConfig)

	return elevState
}

/*
 * Merges incoming order list with the current order list
 * Hall orders are overwritten while cab orders are ored
 */
func OnSync(elevState *types.ElevState,
	elevConfig *types.ElevConfig,
	newOrders [][][]bool,
) *types.ElevState {

	for elevator := range newOrders {
		for floor := range newOrders[elevator] {
			for orderType := range newOrders[elevator][floor] {
				if orderType == elevio.BT_Cab && elevator == elevConfig.NodeID {
					// Merge cab orders
					newOrderStatus := newOrders[elevator][floor][orderType] || elevState.Orders[elevator][floor][orderType]
					elevState.Orders[elevator][floor][orderType] = newOrderStatus
				} else {
					// Overwrite hall orders
					elevState.Orders[elevator][floor][orderType] = newOrders[elevator][floor][orderType]
				}
			}
		}
	}

	SetCabLights(elevState.Orders[elevConfig.NodeID], elevConfig)
	SetHallLights(elevState.Orders, elevConfig)

	return elevState
}

func ReassignOrders(
	elevState *types.ElevState,
	elevConfig *types.ElevConfig,
	nodeID int,
	bidTx chan types.Msg[types.Bid],
) {

	for floor := range elevState.Orders[nodeID] {
		for orderType, orderStatus := range elevState.Orders[nodeID][floor] {
			if !orderStatus || orderType == elevio.BT_Cab {
				continue
			}

			order := types.Order{
				Button: elevio.ButtonType(orderType),
				Floor:  floor,
			}

			bidTx <- network.FormatBidMsg(
				nil,
				order,
				nodeID,
				elevConfig.NumNodes,
				elevState.NextNodeID,
				elevConfig.NodeID,
			)
		}
	}
}

func SelfAssignOrder(
	elevState *types.ElevState,
	elevConfig *types.ElevConfig,
	order types.Order,
	servedTx chan types.Msg[types.Served],
	doorTimer chan<- types.TimerActions,
	floorTimer chan<- types.TimerActions,
) *types.ElevState {
	elevState = SetOrderStatus(
		elevState,
		elevConfig,
		elevConfig.NodeID,
		order,
		true,
	)

	fsmOutput := fsm.OnOrderAssigned(order, elevState, elevConfig)

	elevState = SetState(
		elevState,
		elevConfig,
		fsmOutput,
		servedTx,
		doorTimer,
		floorTimer,
	)

	return elevState
}

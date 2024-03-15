package elev

import (
	"Driver-go/elevio"
	"elevator/fsm"
	"elevator/network"
	"elevator/types"
	"slices"
	"strconv"
)

/*
 * Takes in output from fsm, performs side effects and return new elev state
 */
func SetState(
	oldState *types.ElevState,
	elevConfig *types.ElevConfig,
	stateChanges types.FsmOutput,
	servedTx chan types.Msg[types.Served],
	syncTx chan types.Msg[types.Sync],
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
		Floor:      oldState.Floor,
		Dirn:       stateChanges.ElevDirn,
		DoorObstr:  oldState.DoorObstr,
		Orders:     oldState.Orders,
		NextNodeID: oldState.NextNodeID,
	}

	/*
	 * Clear served orders
	 */
	for order, clearOrder := range stateChanges.ClearOrders {
		if !clearOrder {
			continue
		}

		order := types.Order{
			Button: elevio.ButtonType(order),
			Floor:  newState.Floor,
		}

		isAlone := newState.NextNodeID == elevConfig.NodeID
		disconnected := newState.NextNodeID == -1

		if isAlone || disconnected {
			newState = *SetOrderStatus(
				&newState,
				elevConfig,
				elevConfig.NodeID,
				order,
				false,
			)
		} else {
			servedTx <- network.FormatServedMsg(
				order,
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
	bidTxSecure chan types.Msg[types.Bid],
) {

	isAlone := elevState.NextNodeID == elevConfig.NodeID
	disconnected := elevState.NextNodeID == -1

	if isAlone || disconnected {
		return
	}

	for floor := range elevState.Orders[nodeID] {
		for orderType, orderStatus := range elevState.Orders[nodeID][floor] {
			if !orderStatus || orderType == elevio.BT_Cab {
				continue
			}

			order := types.Order{
				Button: elevio.ButtonType(orderType),
				Floor:  floor,
			}

			bidTxSecure <- network.FormatBidMsg(
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
	syncTx chan types.Msg[types.Sync],
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
		syncTx,
		doorTimer,
		floorTimer,
	)

	return elevState
}

func strArrToInt(strArr []string) []int {
	intArr := make([]int, len(strArr))

	for i, el := range strArr {
		num, _ := strconv.Atoi(el)
		intArr[i] = num
	}

	return intArr
}

func indexOf(arr []int, val int) int {
	for i := range arr {
		if arr[i] == val {
			return i
		}
	}

	return -1
}

func SetNextNodeID(
	elevState *types.ElevState,
	elevConfig *types.ElevConfig,
	peersStr []string,
) *types.ElevState {
	peers := strArrToInt(peersStr)
	slices.Sort(peers)

	indexOfNodeID := indexOf(peers, elevConfig.NodeID)

	if len(peers) == 0 {
		elevState.NextNodeID = -1
		return elevState
	}

	if 0 > indexOfNodeID {
		return elevState
	}

	if indexOfNodeID >= len(peers)-1 {
		elevState.NextNodeID = peers[0]
	} else {
		elevState.NextNodeID = peers[indexOfNodeID+1]
	}

	return elevState
}

func ShouldSendSync(
	nodeID int,
	oldNextNode int,
	newNextNode int,
	newPeerStr string,
) bool {
	if len(newPeerStr) == 0 {
		return false
	}

	newPeerInt, _ := strconv.Atoi(newPeerStr)
	return newNextNode == newPeerInt && oldNextNode != -1 && newNextNode != nodeID
}

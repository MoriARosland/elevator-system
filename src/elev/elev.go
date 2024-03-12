package elev

import (
	"Driver-go/elevio"
	"elevator/fsm"
	"elevator/network"
	"elevator/types"
	"fmt"
)

func InitConfig(
	nodeID int,
	numNodes int,
	numFloors int,
	numButtons int,
	doorOpenDuration int,
	basePort int,
) *types.ElevConfig {

	if nodeID+1 > numNodes {
		panic("Node id greater than number of nodes")
	}

	elevator := types.ElevConfig{
		NodeID:           nodeID,
		NumNodes:         numNodes,
		NumFloors:        numFloors,
		NumButtons:       numButtons,
		DoorOpenDuration: doorOpenDuration,
		BroadcastPort:    basePort + nodeID,
	}

	return &elevator
}

func InitState(elevConfig *types.ElevConfig) *types.ElevState {
	orders := make([][][]bool, elevConfig.NumNodes)

	for elevator := range orders {
		orders[elevator] = make([][]bool, elevConfig.NumFloors)
		for floor := range orders[elevator] {
			orders[elevator][floor] = make([]bool, elevConfig.NumButtons)
		}
	}

	elevState := types.ElevState{
		Floor:  -1,
		Dirn:   elevio.MD_Stop,
		Orders: orders,
	}

	return &elevState
}

/*
 * Takes in output from fsm, performs side effects and return new elev state
 */
func SetState(
	oldState *types.ElevState,
	elevConfig *types.ElevConfig,
	stateChanges types.FsmOutput,
	sendSecureMsg chan<- []byte,
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
		NextNode:     oldState.NextNode,
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
			sendSecureMsg <- network.FormatServedMsg(
				types.Order{
					Button: elevio.ButtonType(order),
					Floor:  newState.Floor,
				},
				elevConfig.NodeID,
			)
		}
	}

	return &newState
}

/*
 * Initiate elevator driver and elevator polling
 */
func InitDriver(
	elevState *types.ElevState,
	elevConfig *types.ElevConfig,
	port int,
) (chan elevio.ButtonEvent, chan int, chan bool) {

	elevio.Init(fmt.Sprintf("localhost:%d", port), elevConfig.NumFloors)

	drvButtons := make(chan elevio.ButtonEvent)
	drvFloors := make(chan int)
	drvObstr := make(chan bool)

	go elevio.PollButtons(drvButtons)
	go elevio.PollFloorSensor(drvFloors)
	go elevio.PollObstructionSwitch(drvObstr)

	/*
	 * Reset elevator to known state
	 */
	elevio.SetDoorOpenLamp(false)
	SetCabLights(elevState.Orders[elevConfig.NodeID], elevConfig)
	SetHallLights(elevState.Orders, elevConfig)

	return drvButtons, drvFloors, drvObstr
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
	sendSecureMsg chan<- []byte,
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

			sendSecureMsg <- network.FormatBidMsg(
				nil,
				order,
				nodeID,
				elevConfig.NumNodes,
				elevConfig.NodeID,
			)
		}
	}
}

func SelfAssignOrder(
	elevState *types.ElevState,
	elevConfig *types.ElevConfig,
	order types.Order,
	sendSecureMsg chan<- []byte,
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
		sendSecureMsg,
		doorTimer,
		floorTimer,
	)

	return elevState
}

func HandleNewOrder(
	elevState *types.ElevState,
	elevConfig *types.ElevConfig,
	order types.Order,
	sendSecureMsg chan<- []byte,
	doorTimer chan<- types.TimerActions,
	floorTimer chan<- types.TimerActions,
) *types.ElevState {

	isCabOrder := order.Button == elevio.BT_Cab

	if elevState.Disconnected && isCabOrder {
		fmt.Println("disconnected cab call")
		/*
		 * When disconnected we only handle new cab orders
		 */
		elevState = SelfAssignOrder(
			elevState,
			elevConfig,
			order,
			sendSecureMsg,
			doorTimer,
			floorTimer,
		)
	} else if isCabOrder {
		fmt.Println("connected cab call")
		/*
		 * Cab orders are selfassigned (over the network)
		 */
		sendSecureMsg <- network.FormatAssignMsg(
			order,
			elevConfig.NodeID,
			int(types.UNASSIGNED),
			elevConfig.NodeID,
		)
	} else {
		fmt.Println("connected hall call")
		/*
		 * Hall orders are assigned after a bidding round
		 */
		sendSecureMsg <- network.FormatBidMsg(
			nil,
			order,
			int(types.UNASSIGNED),
			elevConfig.NumNodes,
			elevConfig.NodeID,
		)
	}

	return elevState
}

func HandleFloorArrival(
	elevState *types.ElevState,
	elevConfig *types.ElevConfig,
	newFloor int,
	sendSecureMsg chan<- []byte,
	doorTimer chan<- types.TimerActions,
	floorTimer chan<- types.TimerActions,
) *types.ElevState {

	oldFloor := elevState.Floor

	elevState.Floor = newFloor
	elevio.SetFloorIndicator(newFloor)

	floorTimer <- types.STOP
	elevState.StuckBetweenFloors = false

	fsmOutput := fsm.OnFloorArrival(elevState, elevConfig)

	elevState = SetState(
		elevState,
		elevConfig,
		fsmOutput,
		sendSecureMsg,
		doorTimer,
		floorTimer,
	)

	if !fsmOutput.SetMotor && oldFloor != -1 {
		floorTimer <- types.START
	}

	return elevState
}

func HandleDoorTimeout(
	elevState *types.ElevState,
	elevConfig *types.ElevConfig,
	sendSecureMsg chan<- []byte,
	doorTimer chan<- types.TimerActions,
	floorTimer chan<- types.TimerActions,
) *types.ElevState {

	if elevState.DoorObstr {
		doorTimer <- types.START
		return elevState
	}
	doorTimer <- types.STOP

	fsmOutput := fsm.OnDoorTimeout(elevState, elevConfig)

	elevState = SetState(
		elevState,
		elevConfig,
		fsmOutput,
		sendSecureMsg,
		doorTimer,
		floorTimer,
	)

	return elevState
}

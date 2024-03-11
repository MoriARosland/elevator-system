package elev

import (
	"Driver-go/elevio"
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
func UpdateState(
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
			newState = *OnOrderChanged(
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
			for btn := 0; btn < elevConfig.NumButtons-1; btn++ {
				combinedOrders[floor][btn] = orders[elevator][floor][btn] || combinedOrders[floor][btn]
			}
		}
	}

	for floor := range combinedOrders {
		for btn := 0; btn < elevConfig.NumButtons-1; btn++ {
			elevio.SetButtonLamp(elevio.ButtonType(btn), floor, combinedOrders[floor][btn])
		}
	}
}

func SetCabLights(orders [][]bool, elevConfig *types.ElevConfig) {
	for floor := range orders {
		elevio.SetButtonLamp(elevio.BT_Cab, floor, orders[floor][elevio.BT_Cab])
	}
}

func OnOrderChanged(
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
			for btn := range newOrders[elevator][floor] {
				if btn == elevio.BT_Cab && elevator == elevConfig.NodeID {
					// Merge cab orders
					elevState.Orders[elevator][floor][btn] = newOrders[elevator][floor][btn] || elevState.Orders[elevator][floor][btn]
				} else {
					// Overwrite hall orders
					elevState.Orders[elevator][floor][btn] = newOrders[elevator][floor][btn]
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
		for btn, order := range elevState.Orders[nodeID][floor] {
			if order && btn != elevio.BT_Cab {
				sendSecureMsg <- network.FormatBidMsg(
					nil,
					types.Order{
						Button: elevio.ButtonType(btn),
						Floor:  floor,
					},
					nodeID,
					elevConfig.NumNodes,
					elevConfig.NodeID,
				)
			}
		}
	}
}

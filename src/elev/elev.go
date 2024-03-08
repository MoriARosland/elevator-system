package elev

import (
	"Driver-go/elevio"
	"elevator/network"
	"elevator/timer"
	"elevator/types"
	"errors"
	"fmt"
)

func InitConfig(
	nodeID int,
	numNodes int,
	numFloors int,
	numButtons int,
	doorOpenDuration int,
	basePort int,
) (*types.ElevConfig, error) {

	if nodeID+1 > numNodes {
		return nil, errors.New("node id greater than number of nodes")
	}

	elevator := types.ElevConfig{
		NodeID:           nodeID,
		NumNodes:         numNodes,
		NumFloors:        numFloors,
		NumButtons:       numButtons,
		DoorOpenDuration: doorOpenDuration,
		BroadcastPort:    basePort + nodeID,
	}

	return &elevator, nil
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
 * 
 */
func UpdateState(
	oldState *types.ElevState,
	elevConfig *types.ElevConfig,
	stateChanges types.FsmOutput,
	sendSecureMsg chan<- []byte,
) *types.ElevState {

	if stateChanges.SetMotor {
		elevio.SetMotorDirection(stateChanges.MotorDirn)
	}
	elevio.SetDoorOpenLamp(stateChanges.Door)

	if stateChanges.StartDoorTimer {
		timer.Start(elevConfig.DoorOpenDuration)
	}

	newState := types.ElevState{
		Floor:     oldState.Floor,
		Dirn:      stateChanges.ElevDirn,
		DoorObstr: oldState.DoorObstr,
		Orders:    oldState.Orders,
		NextNode:  oldState.NextNode,
	}

	/*
	 * Clear served orders
	 */
	for order, clearOrder := range stateChanges.ClearOrders {
		if clearOrder {
			sendSecureMsg <- network.FormatServedMsg(
				types.Order{
					Button: elevio.ButtonType(order),
					Floor:  newState.Floor,
				},
				elevConfig.NodeID,
			)
		}
	}

	cabcalls := newState.Orders[elevConfig.NodeID]
	SetCabLights(cabcalls, elevConfig)
	SetHallLights(newState.Orders, elevConfig)

	return &newState
}

/*
 * Initiate elevator driver and elevator polling
 */
func InitDriver(
	port int,
	numFloors int,
) (chan elevio.ButtonEvent, chan int, chan bool) {

	elevio.Init(fmt.Sprintf("localhost:%d", port), numFloors)

	drvButtons := make(chan elevio.ButtonEvent)
	drvFloors := make(chan int)
	drvObstr := make(chan bool)

	go elevio.PollButtons(drvButtons)
	go elevio.PollFloorSensor(drvFloors)
	go elevio.PollObstructionSwitch(drvObstr)

	return drvButtons, drvFloors, drvObstr
}

func FindNextNodeID(elevConfig *types.ElevConfig) int {
	var nextNodeID int

	if elevConfig.NodeID+1 >= elevConfig.NumNodes {
		nextNodeID = 0
	} else {
		nextNodeID = elevConfig.NodeID + 1
	}

	return nextNodeID
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
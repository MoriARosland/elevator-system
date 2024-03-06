package elev

import (
	"Driver-go/elevio"
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
	requests := make([][][]bool, elevConfig.NumNodes)

	for elevator := range requests {
		requests[elevator] = make([][]bool, elevConfig.NumFloors)
		for floor := range requests[elevator] {
			requests[elevator][floor] = make([]bool, elevConfig.NumButtons)
		}
	}

	elevState := types.ElevState{
		Floor:    -1,
		Dirn:     elevio.MD_Stop,
		Requests: requests,
	}

	return &elevState
}

func UpdateState(
	oldState *types.ElevState,
	stateChanges types.FsmOutput,
	elevConfig *types.ElevConfig,
) *types.ElevState {

	if stateChanges.SetMotor {
		elevio.SetMotorDirection(stateChanges.MotorDirn)
	}
	elevio.SetDoorOpenLamp(stateChanges.Door)

	if stateChanges.StartDoorTimer {
		timer.Start(elevConfig.DoorOpenDuration)
	}

	newState := types.ElevState{
		Floor:           oldState.Floor,
		Dirn:            stateChanges.ElevDirn,
		DoorObstr:       oldState.DoorObstr,
		Requests:        oldState.Requests,
		NextNode:        oldState.NextNode,
		WaitingForReply: oldState.WaitingForReply,
	}

	for order, clearOrder := range stateChanges.ClearOrders {
		if clearOrder {
			// TODO: Handle order clearing correctly (sendSecure through network)
			newState.Requests[elevConfig.NodeID][newState.Floor][order] = false
		}
	}

	cabcalls := newState.Requests[elevConfig.NodeID]
	SetCabLights(cabcalls, elevConfig)
	SetHallLights(newState.Requests, elevConfig)

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

func SetHallLights(requests [][][]bool, elevConfig *types.ElevConfig) {
	for elevator := range requests {
		for floor := range requests[elevator] {
			// Skip the cab buttons by subtracting 1 from elevConfig.NumButtons.
			// See type ButtonType in lib/driver-go-master/elevio/elevator_io.go for reference.
			for btn := 0; btn < elevConfig.NumButtons-1; btn++ {
				elevio.SetButtonLamp(elevio.ButtonType(btn), floor, requests[elevator][floor][btn])
			}
		}
	}
}

func SetCabLights(cabcalls [][]bool, elevConfig *types.ElevConfig) {
	for floor := range cabcalls {
		elevio.SetButtonLamp(elevio.BT_Cab, floor, cabcalls[floor][elevio.BT_Cab])
	}
}

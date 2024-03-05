package elev

import (
	"Driver-go/elevio"
	"elevator/types"
	"errors"
	"fmt"
)

const NUM_BUTTONS = 3

func InitConfig(
	nodeID int,
	numNodes int,
	numFloors int,
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
		NumButtons:       NUM_BUTTONS,
		DoorOpenDuration: doorOpenDuration,
		BroadcastPort:    basePort + nodeID,
	}

	return &elevator, nil
}

func InitState(numElevators int, numFloors int) *types.ElevState {
	requests := make([][][]bool, numElevators)

	for elevator := range requests {
		requests[elevator] = make([][]bool, numFloors)
		for floor := range requests[elevator] {
			requests[elevator][floor] = make([]bool, NUM_BUTTONS)
		}
	}

	fmt.Println(requests)

	elevState := types.ElevState{
		Floor:    -1,
		Dirn:     elevio.MD_Stop,
		Requests: requests,
	}

	return &elevState
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

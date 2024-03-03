package elev

import (
	"Driver-go/elevio"
	"elevator/types"
	"errors"
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

func InitState(numFloors int) *types.ElevState {
	requests := make([][]bool, numFloors)

	for floor := range requests {
		requests[floor] = make([]bool, NUM_BUTTONS)
	}

	elevState := types.ElevState{
		Floor:    -1,
		Dirn:     elevio.MD_Stop,
		Requests: requests,
	}

	return &elevState
}

func SetAllLights(elevState *types.ElevState, elevConfig *types.ElevConfig) {
	for floor := 0; floor < elevConfig.NumFloors; floor++ {
		for btn := 0; btn < elevConfig.NumButtons; btn++ {
			elevio.SetButtonLamp(elevio.ButtonType(btn), floor, elevState.Requests[floor][btn])
		}
	}
}

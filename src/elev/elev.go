package elev

import (
	"Driver-go/elevio"
	"errors"
)

const NUM_BUTTONS = 3

func InitConfig(
	nodeID int,
	numNodes int,
	numFloors int,
	doorOpenDuration int,
	basePort int,
) (*ElevConfig, error) {

	if nodeID+1 > numNodes {
		return nil, errors.New("node id greater than number of nodes")
	}

	elevator := ElevConfig{
		NodeID:        nodeID,
		NumNodes:      numNodes,
		NumFloors:     numFloors,
		BroadcastPort: basePort + nodeID,
	}

	return &elevator, nil
}

func InitState(numFloors int) *ElevState {
	requests := make([][]bool, numFloors)

	for floor := range requests {
		requests[floor] = make([]bool, NUM_BUTTONS)
	}

	elevState := ElevState{
		Floor:    -1,
		Dirn:     elevio.MD_Stop,
		Requests: requests,
	}

	return &elevState
}

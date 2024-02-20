package elevator

import "errors"

func InitElevator(
	nodeID int,
	numNodes int,
	basePort int,
) (*Elevator, error) {

	if nodeID+1 > numNodes {
		return nil, errors.New("node id greater than number of nodes")
	}

	elevator := Elevator{
		NodeID:        nodeID,
		NumNodes:      numNodes,
		BroadCastPort: basePort + nodeID,
	}

	return &elevator, nil
}

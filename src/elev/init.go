package elev

import (
	"Driver-go/elevio"
	"elevator/types"
	"fmt"
)

func InitConfig(
	nodeID int,
	numNodes int,
	numFloors int,
	numButtons int,
	doorOpenDuration int,
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
		NextNodeID: -1,
	}

	return &elevState
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

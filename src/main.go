package main

import (
	"Driver-go/elevio"
	"elevator/elev"
	"elevator/fsm"
	"elevator/network"
	"flag"
	"fmt"
	"os"
)

const NUM_FLOORS = 4
const DOOR_OPEN_DURATION = 3000

func main() {
	/*
	 * Read command line arguments
	 */
	nodeID := flag.Int("id", -1, "Node id")
	numNodes := flag.Int("num", -1, "Number of nodes")
	baseBroadcastPort := flag.Int("bport", -1, "Base Broadcasting port")
	elevServerPort := flag.Int("sport", -1, "Elevator server port")

	flag.Parse()

	if *nodeID < 0 || *numNodes < 0 || *baseBroadcastPort < 0 || *elevServerPort < 0 {
		fmt.Println("Missing flags, use flag -h to see usage")
		os.Exit(1)
	}

	/*
	 * Clear terminal window
	 */
	fmt.Print("\033[2J")

	/*
	 * Initiate elevator config
	 */
	elevConfig, err := elev.InitConfig(
		*nodeID,
		*numNodes,
		NUM_FLOORS,
		DOOR_OPEN_DURATION,
		*baseBroadcastPort,
	)

	if err != nil {
		panic(err)
	}

	/*
	 * Initiate elevator state
	 */
	elevState := elev.InitState(elevConfig.NumFloors)

	fmt.Println(elevState.Requests)

	/*
	 * Initiate elevator driver and elevator polling
	 */
	elevio.Init(fmt.Sprintf("localhost:%d", *elevServerPort), NUM_FLOORS)

	drvButtons := make(chan elevio.ButtonEvent)
	drvFloors := make(chan int)
	drvObstr := make(chan bool)

	go elevio.PollButtons(drvButtons)
	go elevio.PollFloorSensor(drvFloors)
	go elevio.PollObstructionSwitch(drvObstr)

	currentFloor := elevio.GetFloor()

	if 0 > currentFloor {
		elevio.SetMotorDirection(elevio.MD_Down)
		elevState.Dirn = elevio.MD_Down

		fsm.OnInitBetweenFloors()
	}

	/*
	 * Start "I'm alive" broadcasting
	 */
	go network.Broadcast(elevConfig.BroadcastPort)

	/*
	 * Monitor next nodes and update NextNode in elevConfig
	 */
	var nextNodeID int

	if elevConfig.NodeID+1 >= elevConfig.NumNodes {
		nextNodeID = 0
	} else {
		nextNodeID = elevConfig.NodeID + 1
	}

	updateNextNode := make(chan elev.NextNode)

	go network.MonitorNextNode(
		elevConfig.NodeID,
		elevConfig.NumNodes,
		*baseBroadcastPort,
		nextNodeID,
		make(chan bool),
		updateNextNode,
	)

	for {
		select {
		case newNextNode := <-updateNextNode:
			elevState.NextNode = newNextNode

		case buttonPress := <-drvButtons:
			fsm.OnRequestButtonPress(buttonPress, elevState)

		case newCurrentFloor := <-drvFloors:
			fsm.OnFloorArrival(newCurrentFloor, elevState)

		case isObstructed := <-drvObstr:
			/*
			 * isObstructed ? reset door timer
			 */
			fmt.Println("Obstruction: ", isObstructed)

		/*
		 * TODO: create door timer
		 */

		default:
			/*
			 * For now, do nothing
			 */
			continue
		}
	}
}

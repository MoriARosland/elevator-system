package main

import (
	"Driver-go/elevio"
	"elevator/elev"
	"elevator/fsm"
	"elevator/network"
	"elevator/timer"
	"elevator/types"
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

	updateNextNode := make(chan types.NextNode)

	go network.MonitorNextNode(
		elevConfig.NodeID,
		elevConfig.NumNodes,
		*baseBroadcastPort,
		nextNodeID,
		make(chan bool),
		updateNextNode,
	)

	localIP, err := network.LocalIP()

	if err != nil {
		fmt.Println(err)
		localIP = "DISCONNECTED"
	}

	incomingMessageChannel := make(chan []byte)

	go network.ListenForMessages(localIP, elevConfig.BroadcastPort, incomingMessageChannel)

	for {
		select {
		case newNextNode := <-updateNextNode:
			elevState.NextNode = newNextNode

			/*
			 * Temporary display id and next node
			 */
			fmt.Print("\033[J\033[2;0H\r  ")
			fmt.Printf("ID: %d | NextID: %d | NextAddr: %s ", elevConfig.NodeID, elevState.NextNode.ID, elevState.NextNode.Addr)

		case buttonPress := <-drvButtons:
			fsm.OnRequestButtonPress(buttonPress, elevState, elevConfig)

		case newCurrentFloor := <-drvFloors:
			fsm.OnFloorArrival(newCurrentFloor, elevState, elevConfig)

		case isObstructed := <-drvObstr:
			timer.Start(elevConfig.DoorOpenDuration)
			elevState.DoorObstr = isObstructed

		case message := <-incomingMessageChannel:
			fmt.Println(string(message))

		default:
			if timer.TimedOut() {
				if elevState.DoorObstr {
					timer.Start(elevConfig.DoorOpenDuration)
					continue
				}

				timer.Stop()
				fsm.OnDoorTimeout(elevState, elevConfig)
			}
			continue
		}
	}
}

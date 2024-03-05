package main

import (
	"Driver-go/elevio"
	"elevator/elev"
	"elevator/fsm"
	"elevator/network"
	"elevator/timer"
	"elevator/types"
	"encoding/binary"
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

	/*
	 * Continuously listen for messages from previous node
	 */
	localIP, err := network.LocalIP()

	if err != nil {
		fmt.Println(err)
		localIP = "DISCONNECTED"
	}

	incomingMessageChannel := make(chan []byte)
	go network.ListenForMessages(localIP, elevConfig.BroadcastPort, incomingMessageChannel)

	/*
	 * Channels used by secure send
	 */
	updateNextNodeAddr := make(chan string)
	replyReceived := make(chan bool)

	/*
	 * Main for/select
	 */
	for {
		select {
		/*
		 * Handle new next node
		 */
		case newNextNode := <-updateNextNode:
			if elevState.NextNode == newNextNode {
				continue
			}

			elevState.NextNode = newNextNode

			if elevState.WaitingForReply {
				updateNextNodeAddr <- elevState.NextNode.Addr
			}

		/*
		 * Handle button presses
		 */
		case buttonPress := <-drvButtons:
			fsm.OnRequestButtonPress(buttonPress, elevState, elevConfig)

		/*
		 * Handle floor arrivals
		 */
		case newCurrentFloor := <-drvFloors:
			fsm.OnFloorArrival(newCurrentFloor, elevState, elevConfig)

		/*
		 * Handle door obstructions
		 */
		case isObstructed := <-drvObstr:
			if elevState.DoorObstr == isObstructed {
				continue
			}

			timer.Start(elevConfig.DoorOpenDuration)
			elevState.DoorObstr = isObstructed

			/*
			 * For testing: send a secure message
			 */
			if elevState.WaitingForReply {
				continue
			}

			elevState.WaitingForReply = true

			buffer := make([]byte, 4)
			binary.BigEndian.PutUint32(buffer, uint32(elevConfig.BroadcastPort))

			go network.SecureSend(
				elevState.NextNode.Addr,
				buffer,
				replyReceived,
				updateNextNodeAddr,
			)

		/*
		 * Handle incomming UDP messages
		 */
		case message := <-incomingMessageChannel:
			/*
			 * ...and receive (or forward) the secure msg
			 */

			receivedMsg := binary.BigEndian.Uint32(message)

			if receivedMsg == uint32(elevConfig.BroadcastPort) && elevState.WaitingForReply {
				replyReceived <- true
				elevState.WaitingForReply = false
				fmt.Println("Recieved reply!")
			} else {
				network.Send(elevState.NextNode.Addr, message)
				fmt.Println("Received: ", receivedMsg)
			}

		/*
		 * Handle door time outs
		 */
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

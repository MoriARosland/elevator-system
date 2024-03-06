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

const NUM_BUTTONS = 3
const NUM_FLOORS = 4
const DOOR_OPEN_DURATION = 3000

/*
 * Parse command line arguments
 */
func parseCommandlineFlags() (int, int, int, int) {
	nodeID := flag.Int("id", -1, "Node id")
	numNodes := flag.Int("num", -1, "Number of nodes")
	baseBroadcastPort := flag.Int("bport", -1, "Base Broadcasting port")
	elevServerPort := flag.Int("sport", -1, "Elevator server port")

	flag.Parse()

	if *nodeID < 0 || *numNodes < 0 || *baseBroadcastPort < 0 || *elevServerPort < 0 {
		fmt.Println("Missing flags, use flag -h to see usage")
		os.Exit(1)
	}

	return *nodeID, *numNodes, *baseBroadcastPort, *elevServerPort
}

func main() {
	nodeID, numNodes, baseBroadcastPort, elevServerPort := parseCommandlineFlags()

	/*
	 * Clear terminal window
	 */
	fmt.Print("\033[2J")

	/*
	 * Initiate elevator config
	 */
	elevConfig, err := elev.InitConfig(
		nodeID,
		numNodes,
		NUM_FLOORS,
		NUM_BUTTONS,
		DOOR_OPEN_DURATION,
		baseBroadcastPort,
	)

	if err != nil {
		panic(err)
	}

	/*
	 * Initiate elevator state
	 */
	elevState := elev.InitState(elevConfig)

	/*
	 * Initiate elevator driver
	 */
	drvButtons, drvFloors, drvObstr := elev.InitDriver(elevServerPort, elevConfig.NumFloors)

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
	updateNextNode := make(chan types.NextNode)

	go network.MonitorNextNode(
		elevConfig.NodeID,
		elevConfig.NumNodes,
		baseBroadcastPort,
		elev.FindNextNodeID(elevConfig),
		make(chan bool),
		updateNextNode,
	)

	/*
	 * Continuously listen for messages from previous node
	 */
	localIP, err := network.LocalIP()

	if err != nil {
		panic(err)
	}

	incomingMessageChannel := make(chan []byte)
	go network.ListenForMessages(
		localIP,
		elevConfig.BroadcastPort,
		incomingMessageChannel,
	)

	/*
	 * Channels used by secure send
	 */
	updateNextNodeAddr := make(chan string)

	/*
	 * Main for/select
	 */
	for {
		select {
		/*
		 * Handle new next node
		 */
		case newNextNode := <-updateNextNode:
			/*
			 * TODO: handle reassignment of the dead nodes hall orders
			 */
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
		case buttonEvent := <-drvButtons:

			/*
			 * TODO: assign order properly
			 */

			elevState.Orders[elevConfig.NodeID][buttonEvent.Floor][buttonEvent.Button] = true

			output := fsm.OnOrderAssigned(buttonEvent, elevState, elevConfig)

			elevState = elev.UpdateState(
				elevState,
				output,
				elevConfig,
			)

		/*
		 * Handle floor arrivals
		 */
		case newCurrentFloor := <-drvFloors:
			elevState.Floor = newCurrentFloor
			elevio.SetFloorIndicator(newCurrentFloor)

			output := fsm.OnFloorArrival(elevState, elevConfig)

			elevState = elev.UpdateState(
				elevState,
				output,
				elevConfig,
			)

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
			* Test MsgToJson() og JsonToMsg()
			 */

			bid := types.Bid{
				Order:        types.Order{Floor: 1, Button: 2},
				TimeToServed: []int{1, 2, 3},
			}

			assign := types.Assign{
				Order:    types.Order{Floor: 2, Button: 0},
				Assignee: 1,
			}

			encoded_msg := network.MsgToJson(bid, 1)
			decoded_msg := network.JsonToMsg(encoded_msg)

			fmt.Print("\n", decoded_msg, "\n\n")
			fmt.Print(decoded_msg.Content, "\n")

			encoded_msg = network.MsgToJson(assign, 1)
			decoded_msg = network.JsonToMsg(encoded_msg)

			encoded_msg = network.MsgToJson(decoded_msg.Content, 2)
			decoded_msg = network.JsonToMsg(encoded_msg)

			encoded_msg = network.MsgToJson(decoded_msg.Content, 3)
			decoded_msg = network.JsonToMsg(encoded_msg)

			fmt.Print("\n", decoded_msg, "\n\n")
			fmt.Print(decoded_msg.Content, "\n")

		/*
		 * Handle incomming UDP messages
		 */
		case <-incomingMessageChannel:
			continue

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

				output := fsm.OnDoorTimeout(elevState, elevConfig)

				elevState = elev.UpdateState(
					elevState,
					output,
					elevConfig,
				)
			}
		}
	}
}

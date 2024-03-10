package main

import (
	"Driver-go/elevio"
	"elevator/elev"
	"elevator/fsm"
	"elevator/network"
	"elevator/timer"
	"elevator/types"
	"time"
)

const NUM_BUTTONS = 3
const NUM_FLOORS = 6
const DOOR_OPEN_DURATION = 3000

func main() {
	nodeID, numNodes, baseBroadcastPort, elevServerPort := parseCommandlineFlags()

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
	 * Continuously listen for messages from previous node
	 */
	incomingMessageChannel := make(chan []byte)

	go network.ListenForMessages(
		network.LocalIP(),
		elevConfig.BroadcastPort,
		incomingMessageChannel,
	)

	/*
	 * Monitor next nodes and update NextNode in elevState
	 * Makes sure we always know which node to send messages to
	 */
	updateNextNode := make(chan types.NextNode)
	syncNextNode := make(chan int)

	go network.MonitorNextNode(
		elevConfig,
		elev.FindNextNodeID(elevConfig),

		updateNextNode,
		syncNextNode,

		make(chan bool),
		make(chan bool),
	)

	/*
	 * Setup secure message sending
	 */
	updateNextNodeAddr := make(chan string)
	replyReceived := make(chan types.Header)
	sendSecureMsg := make(chan []byte)

	go network.SecureSend(
		updateNextNodeAddr,
		replyReceived,
		sendSecureMsg,
	)

	/*
	 * Initiate elevator driver
	 */
	drvButtons, drvFloors, drvObstr := elev.InitDriver(elevState, elevConfig, elevServerPort)

	/*
	 * Wait until we know the status of the other nodes in the circle
	 */
	elevState.NextNode = <-updateNextNode
	updateNextNodeAddr <- elevState.NextNode.Addr

	printNextNode(elevState, elevConfig)

	/*
	 * In case we start between two floors
	 */
	if 0 > elevio.GetFloor() {
		elevio.SetMotorDirection(elevio.MD_Down)
		elevState.Dirn = elevio.MD_Down
		fsm.OnInitBetweenFloors()
	}

	/*
	 * Setup timers
	 */
	doorTimeout, doorTimer := timer.New(DOOR_OPEN_DURATION * time.Millisecond)

	/*
	 * Start "I'm alive" broadcasting, notifies the other nodes that we are ready
	 */
	go network.Broadcast(elevConfig.BroadcastPort)

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
			updateNextNodeAddr <- elevState.NextNode.Addr

			printNextNode(elevState, elevConfig)

		/*
		 * Sync new next node
		 */
		case targetNode := <-syncNextNode:
			sendSecureMsg <- network.FormatSyncMsg(
				targetNode,
				elevState.Orders,
				elevConfig.NodeID,
			)

		/*
		 * Handle button presses
		 */
		case newOrder := <-drvButtons:
			if elevState.ProcessingOrder {
				continue
			}

			elevState.ProcessingOrder = true

			/*
			 * Cab orders are directly selfassigned
			 */
			if newOrder.Button == elevio.BT_Cab {
				sendSecureMsg <- network.FormatAssignMsg(
					newOrder,
					elevConfig.NodeID,
					elevConfig.NodeID,
				)

				continue
			}

			/*
			 * Hall orders are assigned after a bidding round
			 */
			sendSecureMsg <- network.FormatBidMsg(
				nil,
				newOrder,
				elevConfig.NumNodes,
				elevConfig.NodeID,
			)

		/*
		 * Handle floor arrivals
		 */
		case newCurrentFloor := <-drvFloors:
			elevState.Floor = newCurrentFloor
			elevio.SetFloorIndicator(newCurrentFloor)

			fsmOutput := fsm.OnFloorArrival(elevState, elevConfig)

			elevState = elev.UpdateState(
				elevState,
				elevConfig,
				fsmOutput,
				sendSecureMsg,
				doorTimer,
			)

		/*
		 * Handle door obstructions
		 */
		case isObstructed := <-drvObstr:
			if elevState.DoorObstr == isObstructed {
				continue
			}

			doorTimer <- types.START
			elevState.DoorObstr = isObstructed

		/*
		 * Handle incomming UDP messages
		 */
		case encodedMsg := <-incomingMessageChannel:
			header, err := network.GetMsgHeader(encodedMsg)

			if err != nil {
				continue
			}

			isReply := header.AuthorID == elevConfig.NodeID

			if isReply {
				replyReceived <- *header
			}

			switch header.Type {
			case types.BID:
				/*
				 * Handle bid
				 */
				bidMsg, err := network.GetMsgContent[types.Bid](encodedMsg)

				if err != nil {
					continue
				}

				bidMsg.TimeToServed[elevConfig.NodeID] = fsm.TimeToOrderServed(
					elevState,
					elevConfig,
					bidMsg.Order,
				)

				if isReply {
					assignee := minTimeToServed(bidMsg.TimeToServed)

					sendSecureMsg <- network.FormatAssignMsg(
						bidMsg.Order,
						assignee,
						elevConfig.NodeID,
					)

					continue
				}

				encodedMsg = network.FormatBidMsg(
					bidMsg.TimeToServed,
					bidMsg.Order,
					elevConfig.NumNodes,
					header.AuthorID,
				)

			case types.ASSIGN:
				/*
				 * Handle assign
				 */
				assignMsg, err := network.GetMsgContent[types.Assign](encodedMsg)

				if err != nil {
					continue
				}

				elevState = elev.OnOrderChanged(
					elevState,
					elevConfig,
					assignMsg.Assignee,
					assignMsg.Order,
					true,
				)

				/*
				 * Make sure that the message is forwarded before updating
				 * state in case the order is to be cleared immediately
				 */
				if !isReply {
					network.Send(elevState.NextNode.Addr, encodedMsg)
				} else {
					elevState.ProcessingOrder = false
				}

				if assignMsg.Assignee != elevConfig.NodeID {
					continue
				}

				fsmOutput := fsm.OnOrderAssigned(
					assignMsg.Order,
					elevState,
					elevConfig,
				)

				elevState = elev.UpdateState(
					elevState,
					elevConfig,
					fsmOutput,
					sendSecureMsg,
					doorTimer,
				)

				continue

			case types.REASSIGN:
				/*
				 * Handle reassign
				 */

			case types.SERVED:
				/*
				 * Handle served
				 */
				servedMsg, err := network.GetMsgContent[types.Served](encodedMsg)

				if err != nil {
					continue
				}

				elevState = elev.OnOrderChanged(
					elevState,
					elevConfig,
					header.AuthorID,
					servedMsg.Order,
					false,
				)

			case types.SYNC:
				/*
				 * Handle sync
				 */
				syncMsg, err := network.GetMsgContent[types.Sync](encodedMsg)

				if err != nil {
					continue
				}

				if syncMsg.TargetID != elevConfig.NodeID {
					break
				}

				elevState = elev.OnSync(
					elevState,
					elevConfig,
					syncMsg.Orders,
				)

				if elevState.Dirn != elevio.MD_Stop {
					break
				}

				fsmOutput := fsm.OnSync(elevState, elevConfig)

				elevState = elev.UpdateState(
					elevState,
					elevConfig,
					fsmOutput,
					sendSecureMsg,
					doorTimer,
				)

			}

			/*
			 * Forward message
			 */
			if !isReply {
				network.Send(elevState.NextNode.Addr, encodedMsg)
			}

		/*
		 * Handle door timeouts
		 */
		case <-doorTimeout:
			if elevState.DoorObstr {
				doorTimer <- types.START
				continue
			}
			doorTimer <- types.STOP

			fsmOutput := fsm.OnDoorTimeout(elevState, elevConfig)

			elevState = elev.UpdateState(
				elevState,
				elevConfig,
				fsmOutput,
				sendSecureMsg,
				doorTimer,
			)

		default:
			/*
			 * Do nothing
			 */
			continue
		}
	}
}

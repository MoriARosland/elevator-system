package main

import (
	"Driver-go/elevio"
	"elevator/elev"
	"elevator/fsm"
	"elevator/network"
	"elevator/timer"
	"elevator/types"
	"fmt"
)

const NUM_BUTTONS = 3
const NUM_FLOORS = 6
const DOOR_OPEN_DURATION = 3000

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
	 * Monitor next nodes and update NextNode in elevState
	 * Makes sure we always know which node to send messages to
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
			updateNextNodeAddr <- elevState.NextNode.Addr

			fmt.Print("\033[J\033[2;0H\r  ")
			fmt.Printf("ID: %d | NextID: %d | NextAddr: %s ",
				elevConfig.NodeID,
				elevState.NextNode.ID,
				elevState.NextNode.Addr,
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
					assignee := MinTimeToServed(bidMsg.TimeToServed)

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
		default:
			if timer.TimedOut() {
				if elevState.DoorObstr {
					timer.Start(elevConfig.DoorOpenDuration)
					continue
				}
				timer.Stop()

				fsmOutput := fsm.OnDoorTimeout(elevState, elevConfig)

				elevState = elev.UpdateState(
					elevState,
					elevConfig,
					fsmOutput,
					sendSecureMsg,
				)
			}
		}
	}
}

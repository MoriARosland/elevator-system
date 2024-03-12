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
const DOOR_OBSTR_TIMEOUT = 6000
const FLOOR_ARRIVAL_TIMEOUT = 6000

func main() {
	nodeID, numNodes, baseBroadcastPort, elevServerPort := parseCommandlineFlags()

	elevConfig := elev.InitConfig(
		nodeID,
		numNodes,
		NUM_FLOORS,
		NUM_BUTTONS,
		DOOR_OPEN_DURATION,
		baseBroadcastPort,
	)

	elevState := elev.InitState(elevConfig)

	incomingMessage, disableListen := network.InitReceiver(elevConfig.BroadcastPort)

	updateNextNode, nextNodeRevived, nextNodeDied := network.InitWatchdog(elevConfig)

	updateSecureSendAddr, replyReceived, sendSecureMsg, disableSecureSend := network.InitSecureSend()

	drvButtons, drvFloors, drvObstr := elev.InitDriver(elevState, elevConfig, elevServerPort)

	doorTimeout, doorTimer := timer.New(DOOR_OPEN_DURATION * time.Millisecond)
	obstrTimeout, obstrTimer := timer.New(DOOR_OBSTR_TIMEOUT * time.Millisecond)
	floorTimeout, floorTimer := timer.New(FLOOR_ARRIVAL_TIMEOUT * time.Millisecond)

	if 0 > elevio.GetFloor() {
		elevio.SetMotorDirection(elevio.MD_Down)
		elevState.Dirn = elevio.MD_Down
		fsm.OnInitBetweenFloors()
		floorTimer <- types.START
	}

	/*
	 * Wait until we know the status of the other nodes in the circle...
	 */
	elevState.NextNode = <-updateNextNode
	updateSecureSendAddr <- elevState.NextNode.Addr
	printNextNode(elevState, elevConfig)

	/*
	 * ...before we notify the other nodes that we are ready
	 */
	networkStatus := make(chan bool)
	go network.Broadcast(elevConfig.BroadcastPort, networkStatus)

	for {
		select {
		case newNextNode := <-updateNextNode:
			if elevState.NextNode == newNextNode {
				continue
			}

			elevState.NextNode = newNextNode
			updateSecureSendAddr <- elevState.NextNode.Addr
			printNextNode(elevState, elevConfig)

		case nodeID := <-nextNodeRevived:
			sendSecureMsg <- network.FormatSyncMsg(
				elevState.Orders,
				nodeID,
				elevConfig.NodeID,
			)

		case nodeID := <-nextNodeDied:
			elev.ReassignOrders(
				elevState,
				elevConfig,
				nodeID,
				sendSecureMsg,
			)

		case disconnected := <-networkStatus:
			if !disconnected {
				updateNextNode, nextNodeRevived, nextNodeDied = network.InitWatchdog(elevConfig)
			}

			elevState.Disconnected = disconnected

			disableListen <- disconnected
			disableSecureSend <- disconnected

		case newOrder := <-drvButtons:
			elevState = elev.HandleNewOrder(
				elevState,
				elevConfig,
				newOrder,
				sendSecureMsg,
				doorTimer,
				floorTimer,
			)

		case newFloor := <-drvFloors:
			elevState = elev.HandleFloorArrival(
				elevState,
				elevConfig,
				newFloor,
				sendSecureMsg,
				doorTimer,
				floorTimer,
			)

		case isObstructed := <-drvObstr:
			elevState = elev.HandleDoorObstr(
				elevState,
				isObstructed,
				obstrTimer,
				doorTimer,
			)

		case encodedMsg := <-incomingMessage:
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
				bidMsg, err := network.GetMsgContent[types.Bid](encodedMsg)

				if err != nil {
					continue
				}

				if !elevState.DoorObstr && !elevState.StuckBetweenFloors {
					bidMsg.TimeToServed[elevConfig.NodeID] = fsm.TimeToOrderServed(
						elevState,
						elevConfig,
						bidMsg.Order,
					)
				}

				if isReply {
					assignee := minTimeToServed(bidMsg.TimeToServed)

					sendSecureMsg <- network.FormatAssignMsg(
						bidMsg.Order,
						assignee,
						bidMsg.OldAssignee,
						elevConfig.NodeID,
					)

					continue
				}

				encodedMsg = network.FormatBidMsg(
					bidMsg.TimeToServed,
					bidMsg.Order,
					bidMsg.OldAssignee,
					elevConfig.NumNodes,
					header.AuthorID,
				)

			case types.ASSIGN:
				assignMsg, err := network.GetMsgContent[types.Assign](encodedMsg)

				if err != nil {
					continue
				}

				if assignMsg.OldAssignee != int(types.UNASSIGNED) {
					elevState = elev.SetOrderStatus(
						elevState,
						elevConfig,
						assignMsg.OldAssignee,
						assignMsg.Order,
						false,
					)
				}

				elevState = elev.SetOrderStatus(
					elevState,
					elevConfig,
					assignMsg.NewAssignee,
					assignMsg.Order,
					true,
				)

				/*
				 * Make sure that the message is forwarded before updating
				 * state in case the order is to be cleared immediately
				 */
				if !isReply {
					network.Send(elevState.NextNode.Addr, encodedMsg)
				}

				if assignMsg.NewAssignee != elevConfig.NodeID {
					continue
				}

				fsmOutput := fsm.OnOrderAssigned(
					assignMsg.Order,
					elevState,
					elevConfig,
				)

				elevState = elev.SetState(
					elevState,
					elevConfig,
					fsmOutput,
					sendSecureMsg,
					doorTimer,
					floorTimer,
				)

				continue

			case types.SERVED:
				servedMsg, err := network.GetMsgContent[types.Served](encodedMsg)

				if err != nil {
					continue
				}

				elevState = elev.SetOrderStatus(
					elevState,
					elevConfig,
					header.AuthorID,
					servedMsg.Order,
					false,
				)

			case types.SYNC:
				syncMsg, err := network.GetMsgContent[types.Sync](encodedMsg)

				if err != nil {
					continue
				}

				elevState = elev.OnSync(
					elevState,
					elevConfig,
					syncMsg.Orders,
				)

				isTarget := syncMsg.TargetID == elevConfig.NodeID

				if isTarget && elevState.Dirn == elevio.MD_Stop {
					fsmOutput := fsm.OnSync(elevState, elevConfig)

					elevState = elev.SetState(
						elevState,
						elevConfig,
						fsmOutput,
						sendSecureMsg,
						doorTimer,
						floorTimer,
					)
				}

				encodedMsg = network.FormatSyncMsg(elevState.Orders, syncMsg.TargetID, header.AuthorID)
			}

			if !isReply {
				network.Send(elevState.NextNode.Addr, encodedMsg)
			}

		case <-doorTimeout:
			elevState = elev.HandleDoorTimeout(
				elevState,
				elevConfig,
				sendSecureMsg,
				doorTimer,
				floorTimer,
			)

		case <-obstrTimeout:
			obstrTimer <- types.STOP
			elev.ReassignOrders(
				elevState,
				elevConfig,
				elevConfig.NodeID,
				sendSecureMsg,
			)

		case <-floorTimeout:
			elevState.StuckBetweenFloors = true
			elev.ReassignOrders(
				elevState,
				elevConfig,
				elevConfig.NodeID,
				sendSecureMsg,
			)

		default:
			continue
		}
	}
}

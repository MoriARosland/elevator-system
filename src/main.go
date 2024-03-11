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

	incomingMessageChannel, disableListen := network.InitReceiver(elevConfig.BroadcastPort)

	updateNextNode, nextNodeRevived, nextNodeDied := network.InitWatchdog(elevConfig)

	updateNextNodeAddr, replyReceived, sendSecureMsg, disableSecureSend := network.InitSecureSend()

	drvButtons, drvFloors, drvObstr := elev.InitDriver(elevState, elevConfig, elevServerPort)

	/*
	 * Wait until we know the status of the other nodes in the circle
	 */
	elevState.NextNode = <-updateNextNode
	updateNextNodeAddr <- elevState.NextNode.Addr
	printNextNode(elevState, elevConfig)

	/*
	 * Setup timers
	 */
	doorTimeout, doorTimer := timer.New(DOOR_OPEN_DURATION * time.Millisecond)
	obstrTimeout, obstrTimer := timer.New(DOOR_OBSTR_TIMEOUT * time.Millisecond)
	floorTimeout, floorTimer := timer.New(FLOOR_ARRIVAL_TIMEOUT * time.Millisecond)

	/*
	 * In case we start between two floors
	 */
	if 0 > elevio.GetFloor() {
		elevio.SetMotorDirection(elevio.MD_Down)
		elevState.Dirn = elevio.MD_Down
		fsm.OnInitBetweenFloors()
		floorTimer <- types.START
	}

	/*
	 * Start "I'm alive" broadcasting, notifies the other nodes that we are ready
	 */
	networkStatus := make(chan bool)
	go network.Broadcast(elevConfig.BroadcastPort, networkStatus)

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

		/*
		 * Update elevator network status
		 */
		case disconnected := <-networkStatus:
			if !disconnected {
				updateNextNode, nextNodeRevived, nextNodeDied = network.InitWatchdog(elevConfig)

				/*
				 * Wait until we know the status of the other nodes in the circle
				 */
				elevState.NextNode = <-updateNextNode
				updateNextNodeAddr <- elevState.NextNode.Addr

				printNextNode(elevState, elevConfig)
			}

			elevState.Disconnected = disconnected

			disableListen <- disconnected
			disableSecureSend <- disconnected

		/*
		 * Handle button presses
		 */
		case newOrder := <-drvButtons:
			/*
			 * If elevator is disconnected from the network, it should only accept orders from its own cab
			 */
			if elevState.Disconnected {
				if newOrder.Button == elevio.BT_Cab {
					elevState = elev.OnOrderChanged(
						elevState,
						elevConfig,
						elevConfig.NodeID,
						newOrder,
						true,
					)

					fsmOutput := fsm.OnOrderAssigned(newOrder, elevState, elevConfig)

					elevState = elev.UpdateState(
						elevState,
						elevConfig,
						fsmOutput,
						sendSecureMsg,
						doorTimer,
						floorTimer,
					)
				}

				continue
			}

			/*
			 * Cab orders are directly selfassigned
			 */
			if newOrder.Button == elevio.BT_Cab {
				sendSecureMsg <- network.FormatAssignMsg(
					newOrder,
					elevConfig.NodeID,
					int(types.UNASSIGNED),
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
				int(types.UNASSIGNED),
				elevConfig.NumNodes,
				elevConfig.NodeID,
			)

		/*
		 * Handle floor arrivals
		 */
		case newCurrentFloor := <-drvFloors:
			oldFloor := elevState.Floor

			elevState.Floor = newCurrentFloor
			elevio.SetFloorIndicator(newCurrentFloor)

			floorTimer <- types.STOP
			elevState.StuckBetweenFloors = false

			fsmOutput := fsm.OnFloorArrival(elevState, elevConfig)

			elevState = elev.UpdateState(
				elevState,
				elevConfig,
				fsmOutput,
				sendSecureMsg,
				doorTimer,
				floorTimer,
			)

			if !fsmOutput.SetMotor && oldFloor != -1 {
				floorTimer <- types.START
			}

		/*
		 * Handle door obstructions
		 */
		case isObstructed := <-drvObstr:
			if elevState.DoorObstr == isObstructed {
				continue
			}

			if isObstructed {
				obstrTimer <- types.START
			} else {
				obstrTimer <- types.STOP
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

				if !elevState.DoorObstr || !elevState.StuckBetweenFloors {
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
					assignMsg.NewAssignee,
					assignMsg.Order,
					true,
				)

				/*
				 * In case of an order reassign
				 */
				if assignMsg.OldAssignee != int(types.UNASSIGNED) {
					elevState = elev.OnOrderChanged(
						elevState,
						elevConfig,
						assignMsg.OldAssignee,
						assignMsg.Order,
						false,
					)
				}

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

				elevState = elev.UpdateState(
					elevState,
					elevConfig,
					fsmOutput,
					sendSecureMsg,
					doorTimer,
					floorTimer,
				)

				continue

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

				elevState = elev.OnSync(
					elevState,
					elevConfig,
					syncMsg.Orders,
				)

				isTarget := syncMsg.TargetID == elevConfig.NodeID

				if isTarget && elevState.Dirn == elevio.MD_Stop {
					fsmOutput := fsm.OnSync(elevState, elevConfig)

					elevState = elev.UpdateState(
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
				floorTimer,
			)

		/*
		 * Reassign orders if door obstruction times out
		 */
		case <-obstrTimeout:
			obstrTimer <- types.STOP

			elev.ReassignOrders(
				elevState,
				elevConfig,
				elevConfig.NodeID,
				sendSecureMsg,
			)

		/*
		 * Reassign order if we are stuck between floors
		 */
		case <-floorTimeout:
			elevState.StuckBetweenFloors = true

			elev.ReassignOrders(
				elevState,
				elevConfig,
				elevConfig.NodeID,
				sendSecureMsg,
			)

		default:
			/*
			 * Do nothing
			 */
			continue
		}
	}
}

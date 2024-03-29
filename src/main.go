package main

import (
	"Driver-go/elevio"
	"Network-go/bcast"
	"Network-go/peers"
	"elevator/elev"
	"elevator/fsm"
	"elevator/network"
	"elevator/timer"
	"elevator/types"
	"slices"
	"strconv"
	"time"
)

const BCAST_PORT = 16491
const PEER_PORT = 17441

const NUM_BUTTONS = 3
const NUM_FLOORS = 4

const DOOR_OPEN_DURATION = 3000 // ms
const DOOR_OBSTR_TIMEOUT = 6000 // ms
const FLOOR_ARRIVAL_TIMEOUT = 6000 // ms

func main() {
	nodeID, numNodes, elevServerPort := parseCommandlineFlags()

	elevConfig := elev.InitConfig(
		nodeID,
		numNodes,
		NUM_FLOORS,
		NUM_BUTTONS,
		DOOR_OPEN_DURATION,
	)

	elevState := elev.InitState(elevConfig)

	drvButtons, drvFloors, drvObstr := elev.InitDriver(elevState, elevConfig, elevServerPort)

	doorTimeout, doorTimer := timer.New(DOOR_OPEN_DURATION * time.Millisecond)
	obstrTimeout, obstrTimer := timer.New(DOOR_OBSTR_TIMEOUT * time.Millisecond)
	floorTimeout, floorTimer := timer.New(FLOOR_ARRIVAL_TIMEOUT * time.Millisecond)

	/*
	 * Setup network communication channels
	 */

	bidTx := make(chan types.Msg[types.Bid])
	bidTxSecure := make(chan types.Msg[types.Bid])
	bidRx := make(chan types.Msg[types.Bid])

	bidSetRecipient := make(chan int)
	bidReplyReceived := make(chan string)

	go network.SecureTransmitter[types.Bid](
		bidSetRecipient,
		bidReplyReceived,
		bidTx,
		bidTxSecure,
	)

	assignTx := make(chan types.Msg[types.Assign])
	assignRx := make(chan types.Msg[types.Assign])
	assignTxSecure := make(chan types.Msg[types.Assign])

	assignSetRecipient := make(chan int)
	assignReplyReceived := make(chan string)

	go network.SecureTransmitter[types.Assign](
		assignSetRecipient,
		assignReplyReceived,
		assignTx,
		assignTxSecure,
	)

	servedTx := make(chan types.Msg[types.Served])
	servedRx := make(chan types.Msg[types.Served])
	servedTxSecure := make(chan types.Msg[types.Served])

	servedSetRecipient := make(chan int)
	servedReplyReceived := make(chan string)

	go network.SecureTransmitter[types.Served](
		servedSetRecipient,
		servedReplyReceived,
		servedTx,
		servedTxSecure,
	)

	syncTx := make(chan types.Msg[types.Sync])
	syncRx := make(chan types.Msg[types.Sync])
	syncTxSecure := make(chan types.Msg[types.Sync])

	syncSetRecipient := make(chan int)
	syncReplyReceived := make(chan string)

	go network.SecureTransmitter[types.Sync](
		syncSetRecipient,
		syncReplyReceived,
		syncTx,
		syncTxSecure,
	)

	go bcast.Transmitter(BCAST_PORT, bidTx, assignTx, servedTx, syncTx)
	go bcast.Receiver(BCAST_PORT, bidRx, assignRx, servedRx, syncRx)

	/*
	 * In case we start between two floors; choose a direction
	 */
	if 0 > elevio.GetFloor() {
		elevio.SetMotorDirection(elevio.MD_Down)
		elevState.Dirn = elevio.MD_Down
		fsm.OnInitBetweenFloors()
		floorTimer <- types.START
	}

	/*
	 * Wait until we know which floor we are on
	 */
	newFloor := <-drvFloors

	oldFloor := elevState.Floor

	elevState.Floor = newFloor
	elevio.SetFloorIndicator(newFloor)

	floorTimer <- types.STOP
	elevState.StuckBetweenFloors = false

	fsmOutput := fsm.OnFloorArrival(elevState, elevConfig)

	elevState = elev.SetState(
		elevState,
		elevConfig,
		fsmOutput,
		doorTimer,
		floorTimer,
	)

	elevState = elev.ClearOrdersAtFloor(
		elevState,
		elevConfig,
		fsmOutput.ClearOrders,
		servedTxSecure,
	)

	if !fsmOutput.SetMotor && oldFloor != -1 {
		floorTimer <- types.START
	}

	/*
	 * After setup is complete: start "I'm alive" broadcasting
	 */
	peerUpdate := make(chan peers.PeerUpdate)

	go peers.Transmitter(PEER_PORT, strconv.Itoa(elevConfig.NodeID), nil)
	go peers.Receiver(PEER_PORT, peerUpdate)

	for {
		select {
		case newPeerList := <-peerUpdate:
			oldNextNodeID := elevState.NextNodeID

			elevState = elev.SetNextNodeID(
				elevState,
				elevConfig,
				newPeerList.Peers,
			)

			printNextNode(elevState, elevConfig)

			if elevState.NextNodeID != oldNextNodeID {
				bidSetRecipient <- elevState.NextNodeID
				assignSetRecipient <- elevState.NextNodeID
				servedSetRecipient <- elevState.NextNodeID
				syncSetRecipient <- elevState.NextNodeID
			}

			shouldSendSync := elev.ShouldSendSync(
				elevConfig.NodeID,
				oldNextNodeID,
				elevState.NextNodeID,
				newPeerList.New,
			)

			oldNextDied := slices.Contains(newPeerList.Lost, strconv.Itoa(oldNextNodeID))
			disconnected := elevState.NextNodeID == -1

			if shouldSendSync {
				syncTxSecure <- network.FormatSyncMsg(
					elevState.Orders,
					elevState.NextNodeID,
					elevState.NextNodeID,
					elevConfig.NodeID,
				)
			} else if oldNextDied && !disconnected {
				elev.ReassignOrders(
					elevState,
					elevConfig,
					oldNextNodeID,
					bidTxSecure,
				)
			}

		case newOrder := <-drvButtons:
			isCabOrder := newOrder.Button == elevio.BT_Cab

			isAlone := elevState.NextNodeID == elevConfig.NodeID
			disconnected := elevState.NextNodeID == -1

			if (isAlone || disconnected) && isCabOrder {
				elevState = elev.SetOrderStatus(
					elevState,
					elevConfig,
					elevConfig.NodeID,
					newOrder,
					true,
				)

				fsmOutput := fsm.OnOrderAssigned(newOrder, elevState, elevConfig)

				elevState = elev.SetState(
					elevState,
					elevConfig,
					fsmOutput,
					doorTimer,
					floorTimer,
				)

				elevState = elev.ClearOrdersAtFloor(
					elevState,
					elevConfig,
					fsmOutput.ClearOrders,
					servedTxSecure,
				)
			} else if !isAlone && !disconnected && isCabOrder {
				assignTxSecure <- network.FormatAssignMsg(
					newOrder,
					elevConfig.NodeID,
					int(types.UNASSIGNED),
					elevState.NextNodeID,
					elevConfig.NodeID,
				)
			} else if !disconnected {
				bidTxSecure <- network.FormatBidMsg(
					nil,
					newOrder,
					int(types.UNASSIGNED),
					elevConfig.NumNodes,
					elevState.NextNodeID,
					elevConfig.NodeID,
				)
			}

		case newFloor := <-drvFloors:
			oldFloor := elevState.Floor

			elevState.Floor = newFloor
			elevio.SetFloorIndicator(newFloor)

			floorTimer <- types.STOP
			elevState.StuckBetweenFloors = false

			fsmOutput := fsm.OnFloorArrival(elevState, elevConfig)

			elevState = elev.SetState(
				elevState,
				elevConfig,
				fsmOutput,
				doorTimer,
				floorTimer,
			)

			elevState = elev.ClearOrdersAtFloor(
				elevState,
				elevConfig,
				fsmOutput.ClearOrders,
				servedTxSecure,
			)

			if !fsmOutput.SetMotor && oldFloor != -1 {
				floorTimer <- types.START
			}

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


		case <-doorTimeout:
			if elevState.DoorObstr {
				doorTimer <- types.START
				continue
			}
			doorTimer <- types.STOP

			fsmOutput := fsm.OnDoorTimeout(elevState, elevConfig)

			elevState = elev.SetState(
				elevState,
				elevConfig,
				fsmOutput,
				doorTimer,
				floorTimer,
			)

			elevState = elev.ClearOrdersAtFloor(
				elevState,
				elevConfig,
				fsmOutput.ClearOrders,
				servedTxSecure,
			)

		case <-obstrTimeout:
			obstrTimer <- types.STOP

			disconnected := elevState.NextNodeID == -1

			if disconnected {
				continue
			}

			elev.ReassignOrders(
				elevState,
				elevConfig,
				elevConfig.NodeID,
				bidTxSecure,
			)

		case <-floorTimeout:
			elevState.StuckBetweenFloors = true

			disconnected := elevState.NextNodeID == -1

			if disconnected {
				continue
			}

			elev.ReassignOrders(
				elevState,
				elevConfig,
				elevConfig.NodeID,
				bidTxSecure,
			)

		case bid := <-bidRx:
			if bid.Header.Recipient != elevConfig.NodeID {
				continue
			}

			isReply := bid.Header.AuthorID == elevConfig.NodeID

			if !elevState.DoorObstr && !elevState.StuckBetweenFloors {
				bid.Content.TimeToServed[elevConfig.NodeID] = fsm.TimeToOrderServed(
					elevState,
					elevConfig,
					bid.Content.Order,
				)
			}

			if !isReply && bid.Header.LoopCounter < elevConfig.NumNodes {
				bid.Header.Recipient = elevState.NextNodeID
				bid.Header.LoopCounter += 1
				bidTx <- bid
			} else {
				bidReplyReceived <- bid.Header.UUID

				assignee := minTimeToServed(bid.Content.TimeToServed)

				assignTxSecure <- network.FormatAssignMsg(
					bid.Content.Order,
					assignee,
					bid.Content.OldAssignee,
					elevState.NextNodeID,
					elevConfig.NodeID,
				)
			}

		case assign := <-assignRx:
			if assign.Header.Recipient != elevConfig.NodeID {
				continue
			}

			if assign.Content.OldAssignee != int(types.UNASSIGNED) {
				elevState = elev.SetOrderStatus(
					elevState,
					elevConfig,
					assign.Content.OldAssignee,
					assign.Content.Order,
					false,
				)
			}

			elevState = elev.SetOrderStatus(
				elevState,
				elevConfig,
				assign.Content.NewAssignee,
				assign.Content.Order,
				true,
			)

			isReply := assign.Header.AuthorID == elevConfig.NodeID

			if !isReply && assign.Header.LoopCounter < elevConfig.NumNodes {
				assign.Header.Recipient = elevState.NextNodeID
				assign.Header.LoopCounter += 1
				assignTx <- assign
			} else {
				assignReplyReceived <- assign.Header.UUID
			}

			if assign.Content.NewAssignee != elevConfig.NodeID {
				continue
			}

			fsmOutput := fsm.OnOrderAssigned(
				assign.Content.Order,
				elevState,
				elevConfig,
			)

			elevState = elev.SetState(
				elevState,
				elevConfig,
				fsmOutput,
				doorTimer,
				floorTimer,
			)

			elevState = elev.ClearOrdersAtFloor(
				elevState,
				elevConfig,
				fsmOutput.ClearOrders,
				servedTxSecure,
			)

		case served := <-servedRx:
			if served.Header.Recipient != elevConfig.NodeID {
				continue
			}

			elevState = elev.SetOrderStatus(
				elevState,
				elevConfig,
				served.Header.AuthorID,
				served.Content.Order,
				false,
			)

			isReply := served.Header.AuthorID == elevConfig.NodeID

			if !isReply && served.Header.LoopCounter < elevConfig.NumNodes {
				served.Header.Recipient = elevState.NextNodeID
				served.Header.LoopCounter += 1
				servedTx <- served
			} else {
				servedReplyReceived <- served.Header.UUID
			}

		case sync := <-syncRx:
			if sync.Header.Recipient != elevConfig.NodeID {
				continue
			}

			elevState = elev.MergeOrderLists(
				elevState,
				elevConfig,
				sync.Content.Orders,
			)

			isTarget := sync.Content.TargetID == elevConfig.NodeID

			if isTarget && elevState.Dirn == elevio.MD_Stop {
				fsmOutput := fsm.OnSync(elevState, elevConfig)

				elevState = elev.SetState(
					elevState,
					elevConfig,
					fsmOutput,
					doorTimer,
					floorTimer,
				)

				elevState = elev.ClearOrdersAtFloor(
					elevState,
					elevConfig,
					fsmOutput.ClearOrders,
					servedTxSecure,
				)
			}

			isReply := sync.Header.AuthorID == elevConfig.NodeID

			if !isReply && sync.Header.LoopCounter < elevConfig.NumNodes {
				sync.Header.Recipient = elevState.NextNodeID
				sync.Header.LoopCounter += 1
				sync.Content.Orders = elevState.Orders
				syncTx <- sync
			} else {
				syncReplyReceived <- sync.Header.UUID
			}

		default:
			continue
		}
	}
}

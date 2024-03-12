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
	"fmt"
	"slices"
	"strconv"
	"time"
)

const BCAST_PORT = 16421
const PEER_PORT = 17421

const NUM_BUTTONS = 3
const NUM_FLOORS = 4

const DOOR_OPEN_DURATION = 3000
const DOOR_OBSTR_TIMEOUT = 6000
const FLOOR_ARRIVAL_TIMEOUT = 6000

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

	if 0 > elevio.GetFloor() {
		elevio.SetMotorDirection(elevio.MD_Down)
		elevState.Dirn = elevio.MD_Down
		fsm.OnInitBetweenFloors()
		floorTimer <- types.START
	}

	peerUpdate := make(chan peers.PeerUpdate)
	peerTxEnable := make(chan bool)

	go peers.Transmitter(PEER_PORT, strconv.Itoa(elevConfig.NodeID), peerTxEnable)
	go peers.Receiver(PEER_PORT, peerUpdate)

	bidTx := make(chan types.Msg[types.Bid])
	bidRx := make(chan types.Msg[types.Bid])

	assignTx := make(chan types.Msg[types.Assign])
	assignRx := make(chan types.Msg[types.Assign])

	servedTx := make(chan types.Msg[types.Served])
	servedRx := make(chan types.Msg[types.Served])

	syncTx := make(chan types.Msg[types.Sync])
	syncRx := make(chan types.Msg[types.Sync])

	go bcast.Transmitter(BCAST_PORT, bidTx, assignTx, servedTx, syncTx)
	go bcast.Receiver(BCAST_PORT, bidRx, assignRx, servedRx, syncRx)

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

			shoudSendSync := elev.ShouldSendSync(
				elevConfig.NodeID,
				oldNextNodeID,
				elevState.NextNodeID,
				newPeerList.New,
			)

			oldNextDied := slices.Contains(newPeerList.Lost, strconv.Itoa(oldNextNodeID))

			if shoudSendSync {
				syncTx <- network.FormatSyncMsg(
					elevState.Orders,
					nodeID,
					elevState.NextNodeID,
					elevConfig.NodeID,
				)
			} else if oldNextDied {
				elev.ReassignOrders(
					elevState,
					elevConfig,
					nodeID,
					bidTx,
				)
			}

		case newOrder := <-drvButtons:
			elevState = elev.HandleNewOrder(
				elevState,
				elevConfig,
				newOrder,
				servedTx,
				bidTx,
				assignTx,
				doorTimer,
				floorTimer,
			)

		case newFloor := <-drvFloors:
			elevState = elev.HandleFloorArrival(
				elevState,
				elevConfig,
				newFloor,
				servedTx,
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

		case <-doorTimeout:
			elevState = elev.HandleDoorTimeout(
				elevState,
				elevConfig,
				servedTx,
				doorTimer,
				floorTimer,
			)

		case <-obstrTimeout:
			obstrTimer <- types.STOP
			elev.ReassignOrders(
				elevState,
				elevConfig,
				elevConfig.NodeID,
				bidTx,
			)

		case <-floorTimeout:
			elevState.StuckBetweenFloors = true
			elev.ReassignOrders(
				elevState,
				elevConfig,
				elevConfig.NodeID,
				bidTx,
			)

		case bid := <-bidRx:
			if bid.Header.Recipient != elevConfig.NodeID {
				continue
			}
			fmt.Println("Received bid")

		case assign := <-assignRx:
			if assign.Header.Recipient != elevConfig.NodeID {
				continue
			}
			fmt.Println("Received assign")

		case served := <-servedRx:
			if served.Header.Recipient != elevConfig.NodeID {
				continue
			}
			fmt.Println("Received served")

		case sync := <-syncRx:
			if sync.Header.Recipient != elevConfig.NodeID {
				continue
			}
			fmt.Println("Received sync")

		default:
			continue
		}
	}
}

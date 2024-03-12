package main

import (
	"Driver-go/elevio"
	"Network-go/network/bcast"
	"Network-go/network/peers"
	"elevator/elev"
	"elevator/fsm"
	"elevator/timer"
	"elevator/types"
	"fmt"
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

	// updateSecureSendAddr, replyReceived, sendSecureMsg, disableSecureSend := network.InitSecureSend()

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

	go peers.Transmitter(PEER_PORT, string(elevConfig.NodeID), peerTxEnable)
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
			// update next node id in elevstate
			fmt.Println("New peer list: ", newPeerList.Peers)

		case newOrder := <-drvButtons:
			fmt.Println("New order: ", newOrder)

		case newFloor := <-drvFloors:
			fmt.Println("New floor: ", newFloor)

		case isObstructed := <-drvObstr:
			fmt.Println("Obstr: ", isObstructed)

		case <-doorTimeout:
			fmt.Println("Door timed out")

		case <-obstrTimeout:
			obstrTimer <- types.STOP
			fmt.Println("Door not closing")

		case <-floorTimeout:
			elevState.StuckBetweenFloors = true
			fmt.Println("Stuck")

		case bid := <-bidRx:
			fmt.Println(bid)

		case assign := <-assignRx:
			fmt.Println(assign)

		case served := <-servedRx:
			fmt.Println(served)

		case sync := <-syncRx:
			fmt.Println(sync)

		default:
			continue
		}
	}
}

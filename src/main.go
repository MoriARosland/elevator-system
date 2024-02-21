package main

import (
	"elevator/elevator"
	"elevator/network"
	"flag"
	"fmt"
	"os"
)

func main() {
	/*
	 * Read command line arguments
	 */
	nodeID := flag.Int("id", -1, "Node id")
	numNodes := flag.Int("num", -1, "Number of nodes")
	basePort := flag.Int("port", -1, "Base broadcasting port")

	flag.Parse()

	if *nodeID < 0 || *numNodes < 0 || *basePort < 0 {
		fmt.Println("Missing flags, use flag -h to see usage")
		os.Exit(1)
	}

	/*
	 * Initiate elevator state
	 */
	elevator, err := elevator.InitElevator(*nodeID, *numNodes, *basePort)

	if err != nil {
		panic(err)
	}

	/*
	 * Clear terminal window
	 */
	fmt.Print("\033[2J")

	go network.Broadcast(elevator.BroadCastPort)

	/*
	 * Monitor next nodes and update NextNodeAddr
	 */
	var nextNodeID int

	if elevator.NodeID+1 >= elevator.NumNodes {
		nextNodeID = 0
	} else {
		nextNodeID = elevator.NodeID + 1
	}

	updateCurrentNextAddr := make(chan string)

	go network.MonitorNext(
		elevator.NodeID,
		elevator.NumNodes,
		elevator.BroadCastPort,
		nextNodeID,
		make(chan bool),
		updateCurrentNextAddr,
	)

	for {
		select {
		case newNextNodeAddr := <-updateCurrentNextAddr:
			elevator.NextNodeAddr = newNextNodeAddr

			/*
			 * Temporary display id and next node
			 */
			fmt.Print("\033[J\033[2;0H\r  ")
			fmt.Printf("ID: %d | Broadcasting: %d | Next: %s ", elevator.NodeID, elevator.BroadCastPort, elevator.NextNodeAddr)

		default:
			/*
			 * For now, do nothing
			 */
		}
	}
}

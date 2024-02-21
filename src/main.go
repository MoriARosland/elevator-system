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
	elevState, err := elevator.InitElevator(*nodeID, *numNodes, *basePort)

	if err != nil {
		panic(err)
	}

	/*
	 * Clear terminal window
	 */
	fmt.Print("\033[2J")

	go network.Broadcast(elevState.BroadCastPort)

	/*
	 * Monitor next nodes and update NextNodeAddr
	 */
	var nextNodeID int

	if elevState.NodeID+1 >= elevState.NumNodes {
		nextNodeID = 0
	} else {
		nextNodeID = elevState.NodeID + 1
	}

	updateNextNode := make(chan elevator.Next)

	go network.MonitorNext(
		elevState.NodeID,
		elevState.NumNodes,
		*basePort,
		nextNodeID,
		make(chan bool),
		updateNextNode,
	)

	for {
		select {
		case newNextNode := <-updateNextNode:
			elevState.Next = newNextNode

			/*
			 * Temporary display id and next node
			 */
			fmt.Print("\033[J\033[2;0H\r  ")
			fmt.Printf("ID: %d | NextID: %d | NextAddr: %s ", elevState.NodeID, elevState.Next.ID, elevState.Next.Addr)

		default:
			/*
			 * For now, do nothing
			 */
			continue
		}
	}
}

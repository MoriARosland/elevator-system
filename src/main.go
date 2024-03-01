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
	 * Clear terminal window
	 */
	fmt.Print("\033[2J")

	/*
	 * Initiate elevator state
	 */
	elevState, err := elevator.InitElevator(*nodeID, *numNodes, *basePort)

	if err != nil {
		panic(err)
	}

	go network.Broadcast(elevState.BroadCastPort)

	/*
	 * Monitor next nodes and update NextNode in elevState
	 */
	var nextNodeID int

	if elevState.NodeID+1 >= elevState.NumNodes {
		nextNodeID = 0
	} else {
		nextNodeID = elevState.NodeID + 1
	}

	updateNextNode := make(chan elevator.NextNode)

	go network.MonitorNextNode(
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
			elevState.NextNode = newNextNode

			/*
			 * Temporary display id and next node
			 */
			fmt.Print("\033[J\033[2;0H\r  ")
			fmt.Printf("ID: %d | NextID: %d | NextAddr: %s ", elevState.NodeID, elevState.NextNode.ID, elevState.NextNode.Addr)

		default:
			/*
			 * For now, do nothing
			 */
			continue
		}
	}
}

package main

import (
	"elevator/types"
	"flag"
	"fmt"
	"os"
	"slices"
)

/*
 * Parse command line arguments
 */
func parseCommandlineFlags() (int, int, int, int) {
	nodeID := flag.Int("id", -1, "Node id")
	numNodes := flag.Int("num", -1, "Number of nodes")
	baseBroadcastPort := flag.Int("bport", -1, "Base Broadcasting port")
	elevServerPort := flag.Int("sport", -1, "Elevator server port")

	flag.Parse()

	if *nodeID < 0 || *numNodes < 0 || *baseBroadcastPort < 0 || *elevServerPort < 0 {
		fmt.Println("Missing flags, use flag -h to see usage")
		os.Exit(1)
	}

	return *nodeID, *numNodes, *baseBroadcastPort, *elevServerPort
}

/*
 * Find the index of the lowest value that is not -1
 */
func minTimeToServed(timeToServed []int) int {
	result := slices.Max(timeToServed)

	for _, value := range timeToServed {
		if 0 > value {
			continue
		} else if value < result {
			result = value
		}
	}

	return slices.Index(timeToServed, result)
}

func printNextNode(elevState *types.ElevState, elevConfig *types.ElevConfig) {
	fmt.Print("\033[2J\033[2;0H\r  ")
	fmt.Printf("ID: %d | NextID: %d | NextAddr: %s ",
		elevConfig.NodeID,
		elevState.NextNode.ID,
		elevState.NextNode.Addr,
	)
}

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

	go network.Broadcast(*basePort + elevator.NodeID)
	go network.NextWatchDog(*nodeID, *numNodes, *basePort)

	select {}
}

package network

import (
	"elevator/types"
	"fmt"
	"net"
	"time"

	"github.com/libp2p/go-reuseport"
)

const LISTEN_TIMEOUT = 300
const BUF_SIZE = 2

/*
 * Recursively monitors the other nodes.
 * The closest (forward in the circle) node that is
 * alive is updated on the updateNextNode channel.
 */
func MonitorNextNode(
	elevConfig *types.ElevConfig,
	nextNodeID int,

	updateNextNode chan<- types.NextNode,
	nodeRevived chan<- int,
	nodeDied chan<- int,

	terminationComplete chan bool,
	selfDestruct <-chan bool,
) {

	hasSubroutine := false
	isAlive := true
	previouslyAlive := false

	destroySubroutine := make(chan bool)

	msgBuffer := make([]byte, BUF_SIZE)

	basePort := elevConfig.BroadcastPort - elevConfig.NodeID
	nextNodePort := basePort + nextNodeID

	packetConnection, err := reuseport.ListenPacket("udp4", fmt.Sprintf(":%d", nextNodePort))

	if err != nil {
		panic(err)
	}
	defer packetConnection.Close()

	for {
		select {
		case <-selfDestruct:
			if hasSubroutine {
				destroySubroutine <- true
			} else {
				terminationComplete <- true
			}

			return

		/*
		 * Listen for Broadcasting from the next node,
		 * create new subroutine if non is received.
		 */
		default:
			deadline := time.Now().Add(LISTEN_TIMEOUT * time.Millisecond)
			err := packetConnection.SetReadDeadline(deadline)

			if err != nil {
				panic(err)
			}

			_, addr, err := packetConnection.ReadFrom(msgBuffer)

			/*
			 * UDP read successful, the next node is alive
			 */
			if err == nil {
				/*
				 * Destroy all subroutines
				 * Blocks until all routines are dead
				 */
				if hasSubroutine {
					destroySubroutine <- true
					<-terminationComplete
					hasSubroutine = false
				}

				updateNextNode <- types.NextNode{
					ID:   nextNodeID,
					Addr: addr.String(),
				}

				if !isAlive {
					nodeRevived <- nextNodeID
				}

				isAlive = true
				previouslyAlive = true
				continue
			}

			isAlive = false

			/*
			 * Error is not a timeout error
			 */
			nErr, ok := err.(net.Error)
			if !ok || !nErr.Timeout() {
				panic(err)
			}

			/*
			 * UDP read timed out
			 */

			if hasSubroutine {
				continue
			}

			if previouslyAlive {
				nodeDied <- nextNodeID
				previouslyAlive = false
			}

			/*
			 * There are no other nodes alive
			 */
			if nextNodeID == calcPrevNodeID(elevConfig) {
				updateNextNode <- types.NextNode{
					ID:   -1,
					Addr: "",
				}
				continue
			}

			/*
			 * If we have not come full circle:
			 * spawn new subroutine to monitor the "next" nextNode
			 */
			go MonitorNextNode(
				elevConfig,
				calcNextNodeID(elevConfig, nextNodeID),

				updateNextNode,
				nodeRevived,
				nodeDied,

				terminationComplete,
				destroySubroutine,
			)

			hasSubroutine = true
		}
	}
}

func calcPrevNodeID(elevConfig *types.ElevConfig) int {
	var prevNodeID int

	if elevConfig.NodeID == 0 {
		prevNodeID = elevConfig.NumNodes - 1
	} else {
		prevNodeID = elevConfig.NodeID - 1
	}

	return prevNodeID
}

func calcNextNodeID(elevConfig *types.ElevConfig, nodeID int) int {
	var nextNodeID int

	if nodeID+1 >= elevConfig.NumNodes {
		nextNodeID = 0
	} else {
		nextNodeID = nodeID + 1
	}

	return nextNodeID
}

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
	nodeID int,
	numNodes int,
	basePort int,
	nextNodeID int,
	selfDestruct chan bool,
	updateNextNode chan types.NextNode,
) {
	var prevNodeID int
	hasSubroutine := false

	if nodeID == 0 {
		prevNodeID = numNodes - 1
	} else {
		prevNodeID = nodeID - 1
	}

	destroySubroutine := make(chan bool)
	buf := make([]byte, BUF_SIZE)

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

			_, addr, err := packetConnection.ReadFrom(buf)

			/*
			 * UDP read successful, the next node is alive
			 */
			if err == nil {
				if hasSubroutine {
					destroySubroutine <- true
					hasSubroutine = false
				}

				updateNextNode <- types.NextNode{ID: nextNodeID, Addr: addr.String()}
				break
			}

			/*
			 * UDP read timed out
			 */
			if nErr, ok := err.(net.Error); ok && nErr.Timeout() {
				if hasSubroutine {
					break
				}

				/*
				 * If we have not come full circle:
				 * spawn new subroutine to monitor the "next" nextNode
				 */
				if nextNodeID != prevNodeID {
					var nextNextNodeID int

					if nextNodeID+1 >= numNodes {
						nextNextNodeID = 0
					} else {
						nextNextNodeID = nextNodeID + 1
					}

					go MonitorNextNode(
						nodeID,
						numNodes,
						basePort,
						nextNextNodeID,
						destroySubroutine,
						updateNextNode,
					)
					hasSubroutine = true
				}

				updateNextNode <- types.NextNode{ID: -1, Addr: ""}
				break
			}

			panic(err)
		}
	}
}

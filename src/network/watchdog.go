package network

import (
	"encoding/binary"
	"fmt"
	"net"
	"time"

	"github.com/libp2p/go-reuseport"
)

const LISTEN_TIMEOUT = 1000
const BUF_SIZE = 4

/*
 * Recursively monitors the other nodes.
 * The closest (forward in the circle) node that is
 * alive is updated on the updateCurrentNext channel.
 */
func MonitorNext(
	nodeID int,
	numNodes int,
	basePort int,
	nextNodeID int,
	selfDestruct chan bool,
	updateCurrentNext chan string,
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
		 * Listen for broadcasting from the next node,
		 * create new subroutine if non is received.
		 */
		default:
			deadline := time.Now().Add(LISTEN_TIMEOUT * time.Millisecond)
			err := packetConnection.SetReadDeadline(deadline)

			if err != nil {
				panic(err)
			}

			_, addr, err := packetConnection.ReadFrom(buf)

			received := binary.BigEndian.Uint32(buf)

			/*
			 * UDP read successful, the next node is alive
			 */
			if err == nil {
				if hasSubroutine {
					destroySubroutine <- true
					hasSubroutine = false
				}

				updateCurrentNext <- fmt.Sprintf("%s - %d", addr.String(), received)
				break
			}

			/*
			 * UDP read timed out
			 */
			if nErr, ok := err.(net.Error); ok && nErr.Timeout() {
				if hasSubroutine {
					break
				}

				if nextNodeID != prevNodeID {
					var nextNextNodeID int

					if nextNodeID+1 >= numNodes {
						nextNextNodeID = 0
					} else {
						nextNextNodeID = nextNodeID + 1
					}

					go MonitorNext(nodeID, numNodes, basePort, nextNextNodeID, destroySubroutine, updateCurrentNext)
					hasSubroutine = true
				}

				updateCurrentNext <- ""
				break
			}

			panic(err)
		}
	}
}

package network

import (
	"fmt"
	"net"
	"runtime"
	"time"

	"github.com/libp2p/go-reuseport"
)

const LISTEN_TIMEOUT = 50
const BUFF_SIZE = 2

/*
 * Recursivly monitors the other nodes.
 * The closest (forward in the circle) node that is
 * alive is updated on the updateCurrentNext channel.
 */
func monitorNext(
	nodeID int,
	nextNodeID int,
	numNodes int,
	basePort int,
	selfDestruct chan bool,
	updateCurrentNext chan int,
) {
	var prevNodeID int
	hasSubroutine := false

	if nodeID == 0 {
		prevNodeID = numNodes - 1
	} else {
		prevNodeID = nodeID - 1
	}

	destroySubroutine := make(chan bool)

	pc, err := reuseport.ListenPacket("udp4", fmt.Sprintf(":%d", basePort+nextNodeID))

	if err != nil {
		panic(err)
	}
	defer pc.Close()

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
			pc.SetReadDeadline(deadline)

			buf := make([]byte, BUFF_SIZE)

			// TODO: second return value can be used to get IP of the sender
			_, _, err := pc.ReadFrom(buf)

			/*
			 * UDP read successful, the next node is alive
			 */
			if err == nil {
				if hasSubroutine {
					destroySubroutine <- true
					hasSubroutine = false
				}

				updateCurrentNext <- nextNodeID
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

					go monitorNext(nodeID, nextNextNodeID, numNodes, basePort, destroySubroutine, updateCurrentNext)
					hasSubroutine = true
				}

				updateCurrentNext <- -1
				break
			}

			panic(err)
		}
	}
}

/*
 * Monitor and print next nodes
 * TODO: move current next state to main function,
 * pass it out on a channel
 */
func NextWatchDog(nodeID int, numNodes int, basePort int) {
	var nextNode int

	if nodeID+1 >= numNodes {
		nextNode = 0
	} else {
		nextNode = nodeID + 1
	}

	currentNextNode := nextNode
	updateCurrentNext := make(chan int)

	go monitorNext(nodeID, nextNode, numNodes, basePort, make(chan bool), updateCurrentNext)

	for {
		fmt.Print("\033[J\033[2;0H\r  ")
		fmt.Printf("ID: %d | Next: %d | Routines: %d \n  ", nodeID, currentNextNode, runtime.NumGoroutine())

		newNextID := <-updateCurrentNext
		currentNextNode = newNextID
	}
}

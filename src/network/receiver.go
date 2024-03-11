package network

import (
	"fmt"
	"time"

	"github.com/libp2p/go-reuseport"
)

/*
 * Listen for incoming messages on specified IP and port.
 */
func ListenForMessages(
	ip string,
	port int,
	outgoingMessage chan<- []byte,
	disableListen chan bool,
) {

	const BUFFER_SIZE = 1024
	const LISTEN_TIMEOUT = 500

	packetConnection, err := reuseport.ListenPacket("udp4", fmt.Sprintf("%s:%d", ip, port))

	if err != nil {
		panic("Could not connect to network")
	}

	defer packetConnection.Close()

	buffer := make([]byte, BUFFER_SIZE)

	var disabled bool

	for {
		select {
		case disabled = <-disableListen:

		default:
			if disabled {
				continue
			}

			deadline := time.Now().Add(LISTEN_TIMEOUT * time.Millisecond)
			err := packetConnection.SetReadDeadline(deadline)

			if err != nil {
				continue
			}

			n, _, err := packetConnection.ReadFrom(buffer)

			if err != nil {
				continue
			}

			outgoingMessage <- buffer[:n]
		}
	}
}

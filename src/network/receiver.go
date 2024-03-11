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
	messageChannel chan<- []byte,
	disconnectedChannel chan bool,
) {

	const BUFFER_SIZE = 1024
	const LISTEN_TIMEOUT = 500

	packetConnection, err := reuseport.ListenPacket("udp4", fmt.Sprintf("%s:%d", ip, port))

	if err != nil {
		panic(err)
	}

	defer packetConnection.Close()

	buffer := make([]byte, BUFFER_SIZE)

	var disconnected bool

	for {
		select {
		case disconnected = <-disconnectedChannel:
			fmt.Println("listen disconnect: ", disconnected)

		default:
			if disconnected {
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

			messageChannel <- buffer[:n]
		}
	}
}

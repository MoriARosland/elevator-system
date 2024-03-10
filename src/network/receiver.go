package network

import (
	"fmt"

	"github.com/libp2p/go-reuseport"
)

const BUFFER_SIZE = 1024

/*
 * Listen for incoming messages on specified IP and port.
 */
func ListenForMessages(
	ip string,
	port int,
	messageChannel chan<- []byte,
	disconnectedChannel chan bool,
) {
	conn, err := reuseport.ListenPacket("udp4", fmt.Sprintf("%s:%d", ip, port))

	if err != nil {
		panic(err)
	}

	defer conn.Close()

	buffer := make([]byte, BUFFER_SIZE)

	var disconnected bool

	for {
		select {
		case disconnected = <-disconnectedChannel:
		default:
			if disconnected {
				continue
			}

			n, _, _ := conn.ReadFrom(buffer)
			messageChannel <- buffer[:n]
		}
	}
}

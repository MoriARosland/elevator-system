package network

import (
	"fmt"

	"github.com/libp2p/go-reuseport"
)

const BUFFER_SIZE = 1024

/*
 * Listen for incoming messages on specified IP and port.
 */
func ListenForMessages(ip string, port int, messageChannelchan chan<- []byte) {
	conn, err := reuseport.ListenPacket("udp4", fmt.Sprintf("%s:%d", ip, port))

	if err != nil {
		messageChannelchan <- []byte("CONNECTION ERROR")
		panic(err)
	}

	defer conn.Close()

	buffer := make([]byte, BUFFER_SIZE)

	for {
		_, _, err := conn.ReadFrom(buffer)

		if err != nil {
			panic(err)
		}

		messageChannelchan <- buffer
	}
}

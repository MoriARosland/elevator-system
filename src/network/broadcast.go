package network

import (
	"fmt"
	"net"
	"time"

	"github.com/libp2p/go-reuseport"
)

const BROADCAST_IP = "255.255.255.255"
const BROADCAST_INTERVAL = 100

/*
 * Broadcasts "I'm alive" on specified port.
 */
func Broadcast(port int) {
	packetConnection, err := reuseport.ListenPacket("udp4", fmt.Sprintf(":%d", port))

	if err != nil {
		panic(err)
	}
	defer packetConnection.Close()

	addr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%d", BROADCAST_IP, port))

	if err != nil {
		panic(err)
	}

	for {
		time.Sleep(BROADCAST_INTERVAL * time.Millisecond)

		_, err := packetConnection.WriteTo([]byte(""), addr)

		if err != nil {
			panic(err)
		}
	}
}

package network

import (
	"fmt"
	"net"
	"time"
)

const BROADCAST_ADDR = "255.255.255.255"
const BROADCAST_INTERVAL = 10

/*
 * Broadcasts "I'm alive" on specified port.
 */
func Broadcast(port int) {
	conn, err := net.Dial("udp4", fmt.Sprintf("%s:%d", BROADCAST_ADDR, port))

	if err != nil {
		panic(err)
	}
	defer conn.Close()

	for {
		time.Sleep(BROADCAST_INTERVAL * time.Millisecond)

		_, err := conn.Write([]byte(""))

		if err != nil {
			panic(err)
		}
	}
}

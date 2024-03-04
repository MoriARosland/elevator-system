package network

import (
	"net"

	"github.com/libp2p/go-reuseport"
)

/*
 *	Send-function using UDP
 */
func Send(addr string, msg []byte) {
	packetConnection, err := reuseport.ListenPacket("udp4", addr)
	if err != nil {
		panic(err)
	}
	defer packetConnection.Close()

	resolvedAddr, err := net.ResolveUDPAddr("udp4", addr)
	if err != nil {
		panic(err)
	}

	_, err = packetConnection.WriteTo(msg, resolvedAddr)
	if err != nil {
		panic(err)
	}
}

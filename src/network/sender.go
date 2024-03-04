package network

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/libp2p/go-reuseport"
)

const MSG_TIMEOUT = 2000

/*
 *	Send message using UDP protocol to the sepcified address
 */
func Send(addr string, msg []byte) {
	if addr == "" {
		return
	}

	receiverPort := strings.Split(addr, ":")[1]
	packetConnection, err := reuseport.ListenPacket("udp4", fmt.Sprintf(":%s", receiverPort))
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

/*
 * Send message to all nodes in the network
 * Resend if no reply is received within timeout
 */
func SecureSend(
	initialAddr string,
	msg []byte,
	replyReceived <-chan bool,
	updateAddr <-chan string,
) {
	addr := initialAddr

	Send(addr, msg)

	msgTimedOut := time.NewTicker(MSG_TIMEOUT * time.Millisecond)

	for {
		select {
		case newAddr := <-updateAddr:
			addr = newAddr

		case <-replyReceived:
			msgTimedOut.Stop()
			return

		case <-msgTimedOut.C:
			Send(addr, msg)

		default:
			/*
			 * Do nothing
			 */
			continue
		}
	}
}

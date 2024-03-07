package network

import (
	"bytes"
	"elevator/types"
	"encoding/binary"
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
func Send(addr string, msgType types.MsgTypes, msg []byte) {
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

	typeBuffer := make([]byte, 4)
	binary.BigEndian.PutUint32(typeBuffer, uint32(msgType))

	byteSlice := [][]byte{typeBuffer, msg}
	seperator := []byte(",")
	jointBytes := bytes.Join(byteSlice, seperator)

	_, err = packetConnection.WriteTo(jointBytes, resolvedAddr)
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
	msgType types.MsgTypes,
	msg []byte,
	replyReceived <-chan bool,
	updateAddr <-chan string,
) {
	addr := initialAddr

	Send(addr, msgType, msg)

	msgTimedOut := time.NewTicker(MSG_TIMEOUT * time.Millisecond)

	for {
		select {
		case newAddr := <-updateAddr:
			addr = newAddr

		case <-replyReceived:
			msgTimedOut.Stop()
			return

		case <-msgTimedOut.C:
			Send(addr, msgType, msg)

		default:
			/*
			 * Do nothing
			 */
			continue
		}
	}
}

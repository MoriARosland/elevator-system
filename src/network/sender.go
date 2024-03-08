package network

import (
	"elevator/types"
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
 * Send message to next node and wait for reply
 * Resend if no reply is received within timeout
 */
func SecureSend(
	updateAddr <-chan string,
	replyReceived <-chan types.Header,
	msgChan <-chan []byte,
) {

	var addr string
	var msgBuffer [][]byte

	msgTimeOut := time.NewTicker(MSG_TIMEOUT * time.Millisecond)
	msgTimeOut.Stop()

	for {
		select {
		case newAddr := <-updateAddr:
			addr = newAddr

		case replyHeader := <-replyReceived:
			if len(msgBuffer) == 0 {
				continue
			}

			msgHeader, _ := GetMsgHeader(msgBuffer[0])

			validReply := replyHeader == *msgHeader

			if !validReply {
				continue
			}

			msgBuffer = msgBuffer[1:]

			if len(msgBuffer) == 0 {
				msgTimeOut.Stop()
				continue
			}

			Send(addr, msgBuffer[0])
			msgTimeOut.Reset(MSG_TIMEOUT * time.Millisecond)

		case msg := <-msgChan:
			msgBuffer = append(msgBuffer, msg)

			if len(msgBuffer) == 1 {
				Send(addr, msgBuffer[0])
				msgTimeOut = time.NewTicker(MSG_TIMEOUT * time.Millisecond)
			}

		case <-msgTimeOut.C:
			if len(msgBuffer) > 0 {
				Send(addr, msgBuffer[0])
			}

		default:
			/*
			 * Do nothing
			 */
			continue
		}
	}
}

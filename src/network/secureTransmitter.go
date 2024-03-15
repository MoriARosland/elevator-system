package network

import (
	"elevator/types"
	"time"
)

const REPLY_TIMEOUT = 300

/*
 * Ensures messages are not lost in the event of network errors:
 * - Sends messages in the message buffer and waits for a reply
 * - Resends if no reply is received within a timeout
 */
func SecureTransmitter[T types.Content](
	setRecipient <-chan int,
	replyReceived <-chan string,
	msgTx chan<- types.Msg[T],
	msg <-chan types.Msg[T],
) {

	var msgBuffer []types.Msg[T]

	replyTimeout := time.NewTicker(REPLY_TIMEOUT * time.Millisecond)
	replyTimeout.Stop()

	for {
		select {
		case newRecipient := <-setRecipient:
			for i := range msgBuffer {
				msgBuffer[i].Header.Recipient = newRecipient
			}

		case replyId := <-replyReceived:
			if len(msgBuffer) == 0 {
				continue
			}

			validReply := replyId == msgBuffer[0].Header.UUID

			if !validReply {
				continue
			}

			msgBuffer = msgBuffer[1:]

			if len(msgBuffer) == 0 {
				replyTimeout.Stop()
				continue
			}

			msgTx <- msgBuffer[0]
			replyTimeout.Reset(REPLY_TIMEOUT * time.Millisecond)

		case newMsg := <-msg:
			msgBuffer = append(msgBuffer, newMsg)

			if len(msgBuffer) == 1 {
				msgTx <- msgBuffer[0]
				replyTimeout = time.NewTicker(REPLY_TIMEOUT * time.Millisecond)
			}

		case <-replyTimeout.C:
			if len(msgBuffer) > 0 {
				msgTx <- msgBuffer[0]
			}
		}
	}
}

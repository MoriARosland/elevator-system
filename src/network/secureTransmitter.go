package network

import (
	"elevator/types"
	"time"
)

const REPLY_TIMEOUT = 300

func SecureTransmitter[T types.Content](
	setRecipient <-chan int,
	replyReceived <-chan bool,
	transmit chan<- types.Msg[T],
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

		case <-replyReceived:
			if len(msgBuffer) == 0 {
				continue
			}

			msgBuffer = msgBuffer[1:]

			if len(msgBuffer) == 0 {
				replyTimeout.Stop()
				continue
			}

			transmit <- msgBuffer[0]
			replyTimeout.Reset(REPLY_TIMEOUT * time.Millisecond)

		case newMsg := <-msg:
			msgBuffer = append(msgBuffer, newMsg)

			if len(msgBuffer) == 1 {
				transmit <- msgBuffer[0]
				replyTimeout = time.NewTicker(REPLY_TIMEOUT * time.Millisecond)
			}

		case <-replyTimeout.C:
			if len(msgBuffer) > 0 {
				transmit <- msgBuffer[0]
			}
		}
	}
}

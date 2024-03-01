package network

import (
	"net"
	"strconv"
	"time"
)

/*
Listen for incoming messages on specified IP and port.
*/
func ListenForMessages(ip string, port int, messageChannelchan chan<- []byte) {
	conn, err := net.ListenPacket("udp4", ip+":"+strconv.Itoa(port))

	if err != nil {
		messageChannelchan <- []byte("CONNECTION ERROR")
		return
	}

	defer conn.Close()

	buffer := make([]byte, 1024)

	for {
		_, _, err := conn.ReadFrom(buffer)

		if err != nil {
			panic(err)
		}

		messageChannelchan <- buffer
	}
}

// ----- TEST FUNCTION ----- //

/*
Sends a message on specified IP and port every 5 seconds (debug function).
*/
func SendUDPMessages(ip string, port int) {
	conn, err := net.Dial("udp", ip+":"+strconv.Itoa(port))
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	message := []byte("Oh, so you wanna talk about the non-local quantum hologram, the phase-conjugate adaptive waves resonating in micro-tubules in the brain, which of course requires some closed-timelike curves and Lorentzian manifold, and... you'll catch up, I'll wait.")

	for {
		_, err = conn.Write(message)
		if err != nil {
			panic(err)
		}

		time.Sleep(5 * time.Second)
	}
}

package network

import (
	"net"
	"strings"
)

var localIP string

/*
 * Fetch the local IP address of the machine.
 * If the IP address has already been fetched, the function returns the cached value.
 * This code is copied from: https://github.com/TTK4145/Network-go/blob/master/network/localip/localip.go
 */
func LocalIP() (string, error) {
	if localIP == "" {
		conn, err := net.DialTCP("tcp4", nil, &net.TCPAddr{IP: []byte{8, 8, 8, 8}, Port: 53})
		if err != nil {
			return "", err
		}
		defer conn.Close()
		localIP = strings.Split(conn.LocalAddr().String(), ":")[0]
	}
	return localIP, nil
}

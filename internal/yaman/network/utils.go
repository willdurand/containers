package network

import (
	"net"
	"strconv"
	"strings"
)

// GetRandomPort returns a random (host) port if it finds one.
func GetRandomPort() (int, error) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	defer l.Close()

	parts := strings.Split(l.Addr().String(), ":")

	return strconv.Atoi(parts[len(parts)-1])
}

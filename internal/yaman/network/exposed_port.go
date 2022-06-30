package network

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var ErrInvalidExposedPort = errors.New("invalid exposed port")

type ExposedPort struct {
	Proto     string
	HostAddr  string
	HostPort  int
	GuestAddr string
	GuestPort int
}

// String returns a human-readable representation of an exposed port.
func (p ExposedPort) String() string {
	if p.HostPort == 0 {
		return fmt.Sprintf("%d/%s", p.GuestPort, p.Proto)
	}

	return fmt.Sprintf("%s:%d->%d/%s", p.HostAddr, p.HostPort, p.GuestPort, p.Proto)
}

// ParseExposedPorts parses the exposed ports listed in an image configuration
// and returns a list of exposed ports.
//
// If the configuration is invalid, an error will be returned.
func ParseExposedPorts(exposedPorts map[string]struct{}) ([]ExposedPort, error) {
	ports := make([]ExposedPort, 0)

	for host := range exposedPorts {
		parts := strings.Split(host, "/")
		if len(parts) != 2 {
			return ports, ErrInvalidExposedPort
		}

		guestPort, err := strconv.Atoi(parts[0])
		if err != nil {
			return ports, ErrInvalidExposedPort
		}

		ports = append(ports, ExposedPort{
			Proto:     parts[1],
			HostAddr:  "0.0.0.0",
			HostPort:  0,
			GuestPort: guestPort,
		})
	}

	return ports, nil
}

package ipc

import (
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"
)

func EnsureValidSockAddr(sockAddr string, mustExist bool) error {
	if sockAddr == "" {
		return fmt.Errorf("socket address '%s' is empty", sockAddr)
	}
	if len(sockAddr) > 108 {
		// LOL: https://github.com/moby/moby/pull/13408
		return fmt.Errorf("socket address '%s' is too long", sockAddr)
	}
	if _, err := os.Stat(sockAddr); mustExist && errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("socket address '%s' does not exist", sockAddr)
	}

	return nil
}

func AwaitMessage(conn net.Conn, expectedMessage string) error {
	buf := make([]byte, len(expectedMessage))

	n, err := conn.Read(buf)
	if err != nil {
		return fmt.Errorf("failed to read from socket: %w", err)
	}

	msg := string(buf[0:n])
	if msg != expectedMessage {
		return fmt.Errorf("received unexpected message: %s", msg)
	}

	return nil
}

func SendMessage(conn net.Conn, message string) error {
	if _, err := conn.Write([]byte(message)); err != nil {
		return fmt.Errorf("failed to send message '%s': %w", message, err)
	}

	return nil
}

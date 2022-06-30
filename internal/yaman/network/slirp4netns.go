package network

import (
	"encoding/json"
	"fmt"
	"net"
	"os/exec"
	"strconv"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Slirp4netnsCommand struct {
	Execute string                 `json:"execute"`
	Args    map[string]interface{} `json:"arguments,omitempty"`
}

type Slirp4netns struct {
	pid        int
	binaryPath string
	socketPath string
}

func NewSlirp4netns(pid int, socketPath string) (*Slirp4netns, error) {
	slirp4netns, err := exec.LookPath("slirp4netns")
	if err != nil {
		return nil, err
	}

	return &Slirp4netns{
		pid:        pid,
		binaryPath: slirp4netns,
		socketPath: socketPath,
	}, nil
}

func (s *Slirp4netns) Start() (int, error) {
	cmd := exec.Command(s.binaryPath, []string{
		"--configure",
		"--mtu=65520",
		"--disable-host-loopback",
		"--api-socket", s.socketPath,
		strconv.Itoa(s.pid),
		"en0",
	}...)

	logrus.WithField("command", cmd.String()).Debug("starting slirp4netns")

	if err := cmd.Start(); err != nil {
		return 0, err
	}
	defer cmd.Process.Release()

	return cmd.Process.Pid, nil
}

func (s *Slirp4netns) ExposePorts(ports []ExposedPort) error {
	for _, port := range ports {
		if port.HostPort == 0 {
			continue
		}

		if err := s.addHostFwd(port); err != nil {
			return err
		}
	}

	return nil
}

func (s *Slirp4netns) addHostFwd(port ExposedPort) error {
	cmd := Slirp4netnsCommand{
		Execute: "add_hostfwd",
		Args: map[string]interface{}{
			"proto":     port.Proto,
			"host_addr": port.HostAddr,
			"host_port": port.HostPort,
			// TODO: add support for "guest_addr"
			"guest_port": port.GuestPort,
		},
	}

	return s.executeCommand(cmd)
}

func (s *Slirp4netns) executeCommand(cmd Slirp4netnsCommand) error {
	data, err := json.Marshal(cmd)
	if err != nil {
		return err
	}

	conn, err := net.Dial("unix", s.socketPath)
	if err != nil {
		return err
	}
	defer func() {
		if err := conn.Close(); err != nil {
			logrus.WithError(err).Error("failed to close slirp4netns socket")
		}
	}()

	if _, err := conn.Write(data); err != nil {
		return err
	}

	if err := conn.(*net.UnixConn).CloseWrite(); err != nil {
		return errors.New("failed to close write slirp4netns socket")
	}

	buf := make([]byte, 2048)
	len, err := conn.Read(buf)
	if err != nil {
		return err
	}

	var response map[string]interface{}
	if err := json.Unmarshal(buf[0:len], &response); err != nil {
		return err
	}

	if err, ok := response["error"]; ok {
		return fmt.Errorf("%s failed: %s", cmd.Execute, err)
	}

	return nil
}

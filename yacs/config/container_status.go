package config

import (
	"encoding/json"
	"syscall"
)

// ContainerStatus represents the container process status and especially the
// (wait) status after the process has exited.
type ContainerStatus struct {
	PID        int
	WaitStatus *syscall.WaitStatus
}

// ExitStatus returns the exit status (code) of the container process when it
// has exited. When the process hasn't been started yet or is still running,
// `-1` is returned.
func (s *ContainerStatus) ExitStatus() int {
	if s.WaitStatus == nil {
		return -1
	}

	return s.WaitStatus.ExitStatus()
}

// MarshalJSON returns the JSON encoding of the container status when the
// container process has exited. When the process hasn't been started yet or is
// still running, an empty JSON object is returned.
func (s *ContainerStatus) MarshalJSON() ([]byte, error) {
	if s.WaitStatus == nil {
		return json.Marshal(map[string]interface{}{})
	}

	return json.Marshal(map[string]interface{}{
		"pid":        s.PID,
		"exited":     s.WaitStatus.Exited(),
		"exitStatus": s.WaitStatus.ExitStatus(),
	})
}

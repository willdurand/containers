package config

import (
	"encoding/json"
	"syscall"
)

type ContainerStatus struct {
	PID        int                 `json:"pid"`
	WaitStatus *syscall.WaitStatus `json:"-"`
}

func (s *ContainerStatus) ExitStatus() int {
	if s.WaitStatus == nil {
		return -1
	}

	return s.WaitStatus.ExitStatus()
}

func (s *ContainerStatus) MarshalJSON() ([]byte, error) {
	if s.WaitStatus == nil {
		return json.Marshal(map[string]interface{}{})
	}

	return json.Marshal(map[string]interface{}{
		"exited":     s.WaitStatus.Exited(),
		"exitStatus": s.WaitStatus.ExitStatus(),
	})
}

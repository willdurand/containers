package yacs

import (
	"encoding/json"
	"syscall"
	"testing"
)

func TestJSON(t *testing.T) {
	wstatus := syscall.WaitStatus(0)
	s1 := &ContainerStatus{
		PID:        123,
		WaitStatus: &wstatus,
	}

	if !s1.Exited() {
		t.Error("s1 is not exited")
	}

	data, err := json.Marshal(s1)
	if err != nil {
		t.Error(err)
	}

	s2 := new(ContainerStatus)
	if json.Unmarshal(data, &s2); err != nil {
		t.Error(err)
	}

	if s1.PID != s2.PID {
		t.Errorf("%d != %d", s1.PID, s2.PID)
	}

	if !s2.Exited() {
		t.Error("s1 is not exited")
	}
}

package network

import "testing"

func TestGetRandomPort(t *testing.T) {
	port, err := GetRandomPort()
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	if port < 1 {
		t.Errorf("expected strictly positive port number, got: %d", port)
	}
}

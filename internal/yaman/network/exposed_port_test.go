package network

import "testing"

func TestExposedPort(t *testing.T) {
	for _, tc := range []struct {
		port     ExposedPort
		expected string
	}{
		{
			port: ExposedPort{
				Proto:     "tcp",
				HostAddr:  "1.2.3.4",
				HostPort:  1234,
				GuestPort: 4567,
			},
			expected: "1.2.3.4:1234->4567/tcp",
		},
		{
			port: ExposedPort{
				Proto:     "tcp",
				HostAddr:  "0.0.0.0",
				HostPort:  0,
				GuestPort: 4567,
			},
			expected: "4567/tcp",
		},
	} {
		if tc.port.String() != tc.expected {
			t.Errorf("expected: %s, got: %s", tc.expected, tc.port.String())
		}
	}
}

func TestParseExposedPorts(t *testing.T) {
	exposedPorts := map[string]struct{}{
		"1234/tcp": {},
		"53/udp":   {},
	}

	ports, err := ParseExposedPorts(exposedPorts)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	if len(ports) != len(exposedPorts) {
		t.Errorf("expected %d ports, got: %d", len(exposedPorts), len(ports))
	}
}

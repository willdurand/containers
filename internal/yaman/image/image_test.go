package image

import "testing"

func TestIsNameValid(t *testing.T) {
	for _, tc := range []struct {
		name     string
		expected bool
	}{
		{"", false},
		{"alpine:latest", false},
		{"alpine", true},
		{"library/alpine", true},
		{"gcr.io/project/image", true},
	} {
		if valid := isNameValid(tc.name); valid != tc.expected {
			t.Errorf("name: %s - got: %t, want: %t", tc.name, valid, tc.expected)
		}
	}
}

package registry

import (
	"testing"
)

func TestParsePullPolicy(t *testing.T) {
	for _, tc := range []struct {
		value    string
		expected PullPolicy
		err      error
	}{
		{"always", PullAlways, nil},
		{"missing", PullMissing, nil},
		{"never", PullNever, nil},
		{"", "", ErrInvalidPullPolicy},
		{"invalid", "", ErrInvalidPullPolicy},
	} {
		policy, err := ParsePullPolicy(tc.value)
		if policy != tc.expected {
			t.Errorf("value: %s - got policy: %s, want: %s", tc.value, policy, tc.expected)
		}
		if err != tc.err {
			t.Errorf("value: %s - got error: %t, want: %t", tc.value, err, tc.err)
		}
	}
}

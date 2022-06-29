package user

import (
	"bytes"
	"testing"
)

func TestParseSubordinateIDs(t *testing.T) {
	var buffer bytes.Buffer
	buffer.WriteString("vagrant:100000:65536\n")
	buffer.WriteString("ubuntu:165536:65536\n")

	ids, err := parseSubordinateIDs(&buffer)
	if err != nil {
		t.Error("failed to parse subuid")
	}

	if len(ids) != 2 {
		t.Errorf("expected %d IDs, got: %d", 2, len(ids))
	}

	for i, expected := range []SubordinateID{
		{User: "vagrant", ID: 100000, Size: 65536},
		{User: "ubuntu", ID: 165536, Size: 65536},
	} {
		if ids[i].User != expected.User {
			t.Errorf("expected: %s, got: %s", expected.User, ids[i].User)
		}
		if ids[i].ID != expected.ID {
			t.Errorf("expected: %d, got: %d", expected.ID, ids[i].ID)
		}
		if ids[i].Size != expected.Size {
			t.Errorf("expected: %d, got: %d", expected.Size, ids[i].Size)
		}
	}
}

func TestParseSubordinateIDsEmptyFile(t *testing.T) {
	var buffer bytes.Buffer

	ids, err := parseSubordinateIDs(&buffer)
	if err != nil {
		t.Error("failed to parse subuid")
	}

	if len(ids) != 0 {
		t.Errorf("expected %d IDs, got: %d", 0, len(ids))
	}
}

func TestParseSubordinateIDsInvalidFile(t *testing.T) {
	for _, tc := range []struct {
		content string
		err     error
	}{
		{content: "vagrant:\n", err: ErrInvalidEntry},
		{content: "::\n", err: ErrInvalidEntry},
		{content: "vagrant::\n", err: ErrInvalidEntry},
		{content: "vagrant:1:\n", err: ErrInvalidEntry},
	} {
		var buffer bytes.Buffer
		buffer.WriteString(tc.content)

		_, err := parseSubordinateIDs(&buffer)
		if err != tc.err {
			t.Errorf("expected error: %v, got: %v", tc.err, err)
		}
	}
}

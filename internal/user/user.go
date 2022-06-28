package user

import (
	"bufio"
	"errors"
	"io"
	"os"
	"os/user"
	"strconv"
	"strings"
)

var (
	ErrInvalidEntry = errors.New("invalid entry in subordinate file")
	ErrNotFound     = errors.New("no subordinate IDs found for current user")
)

// SubordinateID represents a range of subordinate (user or group) IDs.
//
// See: https://man7.org/linux/man-pages/man5/subuid.5.html
// See: https://man7.org/linux/man-pages/man5/subgid.5.html
type SubordinateID struct {
	User string
	ID   int
	Size int
}

func ParseSubUid() ([]SubordinateID, error) {
	file, err := os.Open("/etc/subuid")
	if err != nil {
		return []SubordinateID{}, err
	}
	defer file.Close()

	return parseSubordinateIDs(file)
}

func ParseSubGid() ([]SubordinateID, error) {
	file, err := os.Open("/etc/subgid")
	if err != nil {
		return []SubordinateID{}, err
	}
	defer file.Close()

	return parseSubordinateIDs(file)
}

// GetSubUid returns the range of subordinate IDs for the current user.
func GetSubUid() (SubordinateID, error) {
	ids, err := ParseSubUid()
	if err != nil {
		return SubordinateID{}, err
	}

	user, err := user.Current()
	if err != nil {
		return SubordinateID{}, err
	}

	for _, id := range ids {
		if user.Username == id.User || user.Uid == id.User {
			return id, nil
		}
	}

	return SubordinateID{}, ErrNotFound
}

// GetSubGid returns the range of subordinate IDs for the current user.
func GetSubGid() (SubordinateID, error) {
	ids, err := ParseSubGid()
	if err != nil {
		return SubordinateID{}, err
	}

	user, err := user.Current()
	if err != nil {
		return SubordinateID{}, err
	}

	for _, id := range ids {
		if user.Username == id.User || user.Gid == id.User {
			return id, nil
		}
	}

	return SubordinateID{}, ErrNotFound
}

func parseSubordinateIDs(r io.Reader) ([]SubordinateID, error) {
	var ids []SubordinateID

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		parts := strings.Split(scanner.Text(), ":")
		if len(parts) != 3 {
			return ids, ErrInvalidEntry
		}

		if parts[0] == "" {
			return ids, ErrInvalidEntry
		}

		id, err := strconv.Atoi(parts[1])
		if err != nil {
			return ids, ErrInvalidEntry
		}

		size, err := strconv.Atoi(parts[2])
		if err != nil {
			return ids, ErrInvalidEntry
		}

		ids = append(ids, SubordinateID{
			User: parts[0],
			ID:   id,
			Size: size,
		})
	}

	return ids, nil
}

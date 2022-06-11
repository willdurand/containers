package yacr

import (
	"encoding/json"
	"io"

	"github.com/willdurand/containers/internal/yacr/container"
)

func WriteState(rootDir, containerId string, w io.Writer) error {
	container, err := container.LoadWithBundleConfig(rootDir, containerId)
	if err != nil {
		return err
	}

	if err := json.NewEncoder(w).Encode(container.State()); err != nil {
		return err
	}

	return nil
}

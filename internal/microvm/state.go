package microvm

import (
	"encoding/json"
	"io"

	"github.com/willdurand/containers/internal/microvm/container"
)

func State(rootDir, containerId string, w io.Writer) error {
	container, err := container.LoadWithBundleConfig(rootDir, containerId)
	if err != nil {
		return err
	}

	if err := json.NewEncoder(w).Encode(container.State); err != nil {
		return err
	}

	return nil
}

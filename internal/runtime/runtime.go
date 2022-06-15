package runtime

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"

	runtimespec "github.com/opencontainers/runtime-spec/specs-go"
)

func LoadBundleConfig(bundle string) (runtimespec.Spec, error) {
	var spec runtimespec.Spec

	data, err := ioutil.ReadFile(filepath.Join(bundle, "config.json"))
	if err != nil {
		return spec, fmt.Errorf("failed to read config.json: %w", err)
	}
	if err := json.Unmarshal(data, &spec); err != nil {
		return spec, fmt.Errorf("failed to parse config.json: %w", err)
	}

	return spec, nil
}

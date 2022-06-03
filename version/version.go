package version

import (
	"runtime"
	"strings"

	"github.com/opencontainers/runtime-spec/specs-go"
)

var (
	VersionString string = "0.1.0"
	// GitCommit is the hash of the git commit at build time. It is set by `make build`.
	GitCommit string = "n/a"
)

func Version() string {
	return strings.Join([]string{
		VersionString,
		"commit: " + GitCommit,
		"spec: " + specs.Version,
		"go: " + runtime.Version(),
	}, "\n")
}

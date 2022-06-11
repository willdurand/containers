// Package version provides a function to print the version of this project,
// and it is usually exposed in a `version` command for each CLI application.
package version

import (
	"runtime"
	"strings"

	runtimespec "github.com/opencontainers/runtime-spec/specs-go"
)

var (
	// VersionString represents the project's version.
	VersionString string = "0.1.0"
	// GitCommit is the hash of the git commit at build time. It is set by the Makefile.
	GitCommit string = "n/a"
)

// Version returns a formatted string with version information (like git commit,
// OCI specification an go versions).
func Version() string {
	return strings.Join([]string{
		VersionString,
		"commit: " + GitCommit,
		"spec: " + runtimespec.Version,
		"go: " + runtime.Version(),
	}, "\n")
}

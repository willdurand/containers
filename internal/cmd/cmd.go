package cmd

import (
	"fmt"
	"os/exec"
	"strings"
)

// Run runs a given command and tries to return more meaningful information when
// something goes wrong.
func Run(c *exec.Cmd) error {
	if _, err := c.Output(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("%s: %w", strings.TrimSuffix(string(exitError.Stderr), "\n"), err)
		}
		return err
	}

	return nil
}

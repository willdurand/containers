package main

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/willdurand/containers/internal/cli"
	"github.com/willdurand/containers/internal/runtime"
)

func init() {
	cmd := &cobra.Command{
		Use:   "spec",
		Short: "Create a new specification file for a bundle",
		Run:   cli.HandleErrors(spec),
		Args:  cobra.NoArgs,
	}
	cmd.Flags().StringP("bundle", "b", "", "path to the root of the bundle directory")
	cmd.Flags().Bool("rootless", false, "generate a configuration for a rootless container")
	rootCmd.AddCommand(cmd)
}

func spec(cmd *cobra.Command, args []string) error {
	bundle, _ := cmd.Flags().GetString("bundle")
	rootless, _ := cmd.Flags().GetBool("rootless")

	configFile, err := os.Create(filepath.Join(bundle, "config.json"))
	if err != nil {
		return err
	}
	defer configFile.Close()

	encoder := json.NewEncoder(configFile)
	encoder.SetIndent("", "  ")

	rootfs, err := filepath.Abs(filepath.Join(bundle, "rootfs"))
	if err != nil {
		return err
	}

	spec, err := runtime.BaseSpec(rootfs, rootless)
	if err != nil {
		return err
	}

	return encoder.Encode(spec)
}

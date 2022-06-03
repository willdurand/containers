package main

import "github.com/spf13/cobra"

func appendGlobalFlags(cmd *cobra.Command, args []string) []string {
	if logFile, _ := cmd.Flags().GetString("log"); logFile != "" {
		args = append(args, "--log", logFile)
	}
	if logFormat, _ := cmd.Flags().GetString("log-format"); logFormat != "" {
		args = append(args, "--log-format", logFormat)
	}
	if debug, _ := cmd.Flags().GetBool("debug"); debug {
		args = append(args, "--debug")
	}

	return args
}

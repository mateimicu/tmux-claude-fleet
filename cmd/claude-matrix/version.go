package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version is set at build time via ldflags: -X main.Version=<value>
var Version string

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version of claude-matrix",
		Run: func(cmd *cobra.Command, args []string) {
			v := Version
			if v == "" {
				v = "dev"
			}
			fmt.Fprintf(cmd.OutOrStdout(), "claude-matrix %s\n", v)
		},
	}
}

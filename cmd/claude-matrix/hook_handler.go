package main

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/mateimicu/tmux-claude-matrix/internal/hooks"
)

func hookHandlerCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "hook-handler",
		Short:  "Handle Claude Code hook events (internal use)",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return hooks.HandleHookEvent(os.Stdin)
		},
	}
}

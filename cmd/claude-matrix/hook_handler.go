package main

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/mateimicu/tmux-claude-matrix/internal/hooks"
	"github.com/mateimicu/tmux-claude-matrix/internal/tmux"
)

func hookHandlerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "hook-handler",
		Short:  "Handle Claude Code hook events (internal use)",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return hooks.HandleHookEvent(os.Stdin, tmux.New())
		},
	}
	// The --from flag is used as a marker in the registered hook command
	// (e.g. "claude-matrix hook-handler --from=tmux-claude-matrix") so that
	// our entries can be identified and managed in Claude's settings.json.
	cmd.Flags().String("from", "", "Hook source identifier (used as marker)")
	return cmd
}

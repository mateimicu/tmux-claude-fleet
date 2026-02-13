package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/mateimicu/tmux-claude-matrix/internal/hooks"
)

func setupHooksCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "setup-hooks",
		Short: "Configure Claude Code hooks for status tracking",
		RunE: func(cmd *cobra.Command, args []string) error {
			binaryPath, err := os.Executable()
			if err != nil {
				return err
			}
			binaryPath, err = filepath.EvalSymlinks(binaryPath)
			if err != nil {
				return err
			}
			if err := hooks.SetupHooks(binaryPath); err != nil {
				return err
			}
			fmt.Println("✓ Claude Code hooks configured for status tracking")
			fmt.Println("  Hooks will update tmux window names with status indicators:")
			fmt.Println("  🟢 Running  ❓ Needs Input  💬 Ready")
			return nil
		},
	}
}

func removeHooksCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove-hooks",
		Short: "Remove Claude Code status tracking hooks",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := hooks.RemoveHooks(); err != nil {
				return err
			}
			fmt.Println("✓ Claude Code status tracking hooks removed")
			return nil
		},
	}
}

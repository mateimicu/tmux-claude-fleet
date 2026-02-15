package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

func main() {
	// Setup context with cancellation
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	rootCmd := &cobra.Command{
		Use:   "claude-matrix",
		Short: "Tmux session manager with Claude integration",
		Long: `Claude Matrix is a tmux session manager that integrates with Claude AI.
It helps you quickly create development environments for your repositories.`,
	}

	rootCmd.AddCommand(
		createCmd(),
		listCmd(),
		listReposCmd(),
		renameCmd(),
		diagnoseCmd(),
		refreshCmd(),
		hookHandlerCmd(),
		setupHooksCmd(),
		removeHooksCmd(),
	)

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		cancel()
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	cancel()
}

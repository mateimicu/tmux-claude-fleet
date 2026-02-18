package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/mateimicu/tmux-claude-matrix/internal/config"
	"github.com/mateimicu/tmux-claude-matrix/internal/logging"
	"github.com/mateimicu/tmux-claude-matrix/pkg/types"
)

// Context key types for storing config and logger.
type contextKey int

const (
	configKey contextKey = iota
	loggerKey
)

// configFromContext retrieves the Config from the command context.
func configFromContext(ctx context.Context) *types.Config {
	cfg, _ := ctx.Value(configKey).(*types.Config)
	return cfg
}

// loggerFromContext retrieves the Logger from the command context.
func loggerFromContext(ctx context.Context) *logging.Logger {
	log, _ := ctx.Value(loggerKey).(*logging.Logger)
	return log
}

func main() {
	// Setup context with cancellation
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	rootCmd := &cobra.Command{
		Use:   "claude-matrix",
		Short: "Tmux session manager with Claude integration",
		Long: `Claude Matrix is a tmux session manager that integrates with Claude AI.
It helps you quickly create development environments for your repositories.`,
		// PersistentPreRunE loads config once and creates the logger.
		// NOTE: Cobra does not chain PersistentPreRunE — if a subcommand defines
		// its own, it shadows this. Currently no subcommands define one. If one
		// is added in the future, it must call this logic explicitly.
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			// CLI flag overrides config/env when explicitly set
			if cmd.Flags().Changed("debug") {
				debugFlag, _ := cmd.Flags().GetBool("debug")
				cfg.Debug = debugFlag
			}

			log := logging.New(cfg.Debug)

			ctx := context.WithValue(cmd.Context(), configKey, cfg)
			ctx = context.WithValue(ctx, loggerKey, log)
			cmd.SetContext(ctx)

			return nil
		},
	}

	rootCmd.PersistentFlags().BoolP("debug", "d", false, "Enable debug logging")

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
		versionCmd(),
	)

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		cancel()
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	cancel()
}

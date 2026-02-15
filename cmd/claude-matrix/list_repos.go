package main

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/spf13/cobra"

	"github.com/mateimicu/tmux-claude-matrix/internal/config"
	"github.com/mateimicu/tmux-claude-matrix/internal/fzf"
	"github.com/mateimicu/tmux-claude-matrix/internal/repos"
)

func listReposCmd() *cobra.Command {
	var forceRefresh bool

	cmd := &cobra.Command{
		Use:   "list-repos",
		Short: "List repositories in FZF-compatible format",
		Long:  `Output discovered repositories as formatted lines for FZF consumption. Used internally by the FZF reload binding.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runListRepos(cmd.Context(), forceRefresh)
		},
	}

	cmd.Flags().BoolVar(&forceRefresh, "force-refresh", false, "Bypass cache TTL and fetch fresh data (falls back to stale cache on failure)")

	return cmd
}

func runListRepos(ctx context.Context, forceRefresh bool) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	sources, err := buildSources(ctx, cfg, io.Discard)
	if err != nil {
		return err
	}

	if forceRefresh {
		for _, s := range sources {
			if gh, ok := s.(*repos.GitHubSource); ok {
				gh.SetForceRefresh(true)
			}
		}
	}

	discoverer := repos.NewDiscoverer(sources...)

	discoveryCtx, discoveryCancel := context.WithTimeout(ctx, 30*time.Second)
	defer discoveryCancel()

	repoList, err := discoverer.ListAll(discoveryCtx)
	if err != nil {
		return fmt.Errorf("failed to discover repositories: %w", err)
	}

	for _, repo := range repoList {
		fmt.Println(fzf.FormatRepoLine(repo)) //nolint:errcheck // stdout write failure is unrecoverable
	}

	return nil
}

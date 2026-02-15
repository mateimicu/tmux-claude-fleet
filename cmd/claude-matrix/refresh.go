package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/mateimicu/tmux-claude-matrix/internal/config"
	"github.com/mateimicu/tmux-claude-matrix/internal/repos"
)

func refreshCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "refresh",
		Short: "Refresh the repository cache",
		Long:  `Force refresh the repository cache by fetching fresh data from GitHub API.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRefresh(cmd.Context())
		},
	}
}

func runRefresh(ctx context.Context) error {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	fmt.Println("🔄 Refreshing repository cache...")

	// Clear existing cache
	cachePath := filepath.Join(cfg.CacheDir, "github-repos.json")
	if err := os.Remove(cachePath); err != nil && !os.IsNotExist(err) {
		fmt.Printf("⚠️  Failed to clear cache: %v\n", err)
	}

	// Build sources list
	sources, err := buildSources(ctx, cfg, os.Stdout)
	if err != nil {
		return err
	}

	// Fetch repos (this will update the cache)
	discoverer := repos.NewDiscoverer(sources...)

	discoveryCtx, discoveryCancel := context.WithTimeout(ctx, 15*time.Second)
	defer discoveryCancel()

	repoList, err := discoverer.ListAll(discoveryCtx)
	if err != nil {
		return fmt.Errorf("failed to fetch repositories: %w", err)
	}

	fmt.Printf("✓ Cache refreshed with %d repositories\n", len(repoList))
	fmt.Printf("📁 Cache location: %s\n", cachePath)
	fmt.Printf("⏰ Cache TTL: %s\n", cfg.CacheTTL)

	return nil
}

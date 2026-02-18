package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

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
	cfg := configFromContext(ctx)
	log := loggerFromContext(ctx)

	log.Debugf("🔄 Refreshing repository cache...\n")

	// Clear existing cache
	cachePath := filepath.Join(cfg.CacheDir, "github-repos.json")
	if err := os.Remove(cachePath); err != nil && !os.IsNotExist(err) {
		log.Warnf("⚠️  Failed to clear cache: %v\n", err)
	}

	// Build sources list
	sources, err := buildSources(ctx, cfg, log)
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

	// User-facing success confirmation — always visible
	fmt.Printf("✓ Cache refreshed with %d repositories\n", len(repoList))
	log.Debugf("📁 Cache location: %s\n", cachePath)
	log.Debugf("⏰ Cache TTL: %s\n", cfg.CacheTTL)

	return nil
}

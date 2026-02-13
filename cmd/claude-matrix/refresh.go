package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	var sources []repos.Source

	if cfg.LocalConfigEnabled && cfg.LocalReposFile != "" {
		sources = append(sources, repos.NewLocalSource(cfg.LocalReposFile))
		fmt.Println("✓ Local repos source ready")
	}

	if cfg.GitHubEnabled {
		token, source := repos.GetGitHubToken(ctx)
		if token == "" {
			fmt.Println("⚠️  GitHub authentication not found, skipping GitHub repositories")
			fmt.Println("   To enable GitHub integration:")
			fmt.Println("   Option 1: Use gh CLI (recommended)")
			fmt.Println("     - Install: brew install gh")
			fmt.Println("     - Login: gh auth login")
			fmt.Println("   Option 2: Set token manually")
			fmt.Println("     - export GITHUB_TOKEN=\"ghp_your_token_here\"")
			fmt.Println("     - Get token at: https://github.com/settings/tokens")
		} else {
			fmt.Printf("✓ GitHub integration enabled (using %s)\n", source)
			if len(cfg.GitHubOrgs) > 0 {
				fmt.Printf("  Filtering by organizations: %s\n", strings.Join(cfg.GitHubOrgs, ", "))
			}
			sources = append(sources, repos.NewGitHubSource(token, cfg.CacheDir, cfg.CacheTTL, cfg.GitHubOrgs))
		}
	}

	if len(sources) == 0 {
		return fmt.Errorf("no repository sources configured")
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

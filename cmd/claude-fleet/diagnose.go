package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/mateimicu/tmux-claude-fleet/internal/config"
	"github.com/mateimicu/tmux-claude-fleet/internal/repos"
	"github.com/spf13/cobra"
)

func diagnoseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "diagnose",
		Short: "Diagnose repository discovery issues",
		Long:  `Show configuration and test repository sources.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDiagnose(cmd.Context())
		},
	}
}

func runDiagnose(ctx context.Context) error {
	fmt.Println("🔍 Diagnosing tmux-claude-fleet configuration...")
	fmt.Println()

	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("❌ Failed to load config: %w", err)
	}

	fmt.Println("✓ Configuration loaded successfully")
	fmt.Println()

	// Show configuration
	fmt.Println("📋 Configuration:")
	fmt.Printf("  Clone directory: %s\n", cfg.CloneDir)
	fmt.Printf("  Sessions directory: %s\n", cfg.SessionsDir)
	fmt.Printf("  Cache directory: %s\n", cfg.CacheDir)
	fmt.Printf("  Cache TTL: %s\n", cfg.CacheTTL)
	fmt.Println()

	// Check local repos
	fmt.Println("📁 Local Repository Source:")
	fmt.Printf("  Enabled: %v\n", cfg.LocalConfigEnabled)
	if cfg.LocalConfigEnabled {
		fmt.Printf("  File: %s\n", cfg.LocalReposFile)

		// Check if file exists
		if _, err := os.Stat(cfg.LocalReposFile); err == nil {
			fmt.Println("  Status: ✓ File exists")

			// Try to load repos
			source := repos.NewLocalSource(cfg.LocalReposFile)
			localRepos, err := source.List(ctx)
			if err != nil {
				fmt.Printf("  Error: ❌ %v\n", err)
			} else {
				fmt.Printf("  Repositories found: %d\n", len(localRepos))
				for i, repo := range localRepos {
					if i < 5 { // Show first 5
						fmt.Printf("    - %s\n", repo.Name)
					}
				}
				if len(localRepos) > 5 {
					fmt.Printf("    ... and %d more\n", len(localRepos)-5)
				}
			}
		} else {
			fmt.Println("  Status: ❌ File not found")
			fmt.Printf("  Create it with: echo 'https://github.com/user/repo' > %s\n", cfg.LocalReposFile)
		}
	} else {
		fmt.Println("  Status: Disabled")
	}
	fmt.Println()

	// Check GitHub
	fmt.Println("🐙 GitHub Repository Source:")
	fmt.Printf("  Enabled: %v\n", cfg.GitHubEnabled)

	if cfg.GitHubEnabled {
		token, source := repos.GetGitHubToken()
		if token == "" {
			fmt.Println("  Status: ❌ No GitHub authentication found")
			fmt.Println()
			fmt.Println("  To enable GitHub integration:")
			fmt.Println("    Option 1: Use gh CLI (recommended)")
			fmt.Println("      - Install: brew install gh  (macOS)")
			fmt.Println("      - Login: gh auth login")
			fmt.Println("      - Verify: gh auth status")
			fmt.Println()
			fmt.Println("    Option 2: Use token manually")
			fmt.Println("      - Get token: https://github.com/settings/tokens")
			fmt.Println("      - Export: export GITHUB_TOKEN=\"ghp_your_token\"")
			fmt.Println("      - Or run: ./setup-github.sh")
		} else {
			fmt.Printf("  Authentication: ✓ Using %s\n", source)
			fmt.Printf("  Token: %s...\n", token[:10])

			// Try to fetch repos
			fmt.Println("  Testing GitHub API...")
			if len(cfg.GitHubOrgs) > 0 {
				fmt.Printf("  Organization filter: %s\n", strings.Join(cfg.GitHubOrgs, ", "))
			}
			source := repos.NewGitHubSource(token, cfg.CacheDir, cfg.CacheTTL, cfg.GitHubOrgs)
			githubRepos, err := source.List(ctx)
			if err != nil {
				fmt.Printf("  Error: ❌ %v\n", err)
				fmt.Println()
				fmt.Println("  Common issues:")
				fmt.Println("    - Token expired or invalid")
				fmt.Println("    - Token missing 'repo' scope")
				fmt.Println("    - Network connectivity issues")
			} else {
				fmt.Printf("  Status: ✓ API working\n")
				fmt.Printf("  Repositories found: %d\n", len(githubRepos))
				for i, repo := range githubRepos {
					if i < 5 { // Show first 5
						fmt.Printf("    - %s\n", repo.Name)
					}
				}
				if len(githubRepos) > 5 {
					fmt.Printf("    ... and %d more\n", len(githubRepos)-5)
				}
			}
		}
	} else {
		fmt.Println("  Status: Disabled")
	}
	fmt.Println()

	// Summary
	fmt.Println("📊 Summary:")

	// Count total sources
	var sources []repos.Source
	if cfg.LocalConfigEnabled && cfg.LocalReposFile != "" {
		sources = append(sources, repos.NewLocalSource(cfg.LocalReposFile))
	}
	token, _ := repos.GetGitHubToken()
	if cfg.GitHubEnabled && token != "" {
		sources = append(sources, repos.NewGitHubSource(token, cfg.CacheDir, cfg.CacheTTL, cfg.GitHubOrgs))
	}

	if len(sources) == 0 {
		fmt.Println("  ❌ No repository sources configured!")
		fmt.Println()
		fmt.Println("  To fix:")
		fmt.Println("    1. Add local repos: echo 'https://github.com/user/repo' > ~/.tmux-claude-fleet/repos.txt")
		fmt.Println("    2. Or set GITHUB_TOKEN: ./setup-github.sh")
	} else {
		discoverer := repos.NewDiscoverer(sources...)
		allRepos, err := discoverer.ListAll(ctx)
		if err != nil {
			fmt.Printf("  Error discovering repos: %v\n", err)
		} else {
			fmt.Printf("  ✓ Total repositories available: %d\n", len(allRepos))
		}
	}

	fmt.Println()
	fmt.Println("For more help, see: https://github.com/mateimicu/tmux-claude-fleet")

	return nil
}

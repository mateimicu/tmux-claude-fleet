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
	"github.com/mateimicu/tmux-claude-matrix/internal/fzf"
	"github.com/mateimicu/tmux-claude-matrix/internal/git"
	"github.com/mateimicu/tmux-claude-matrix/internal/repos"
	"github.com/mateimicu/tmux-claude-matrix/internal/session"
	"github.com/mateimicu/tmux-claude-matrix/internal/tmux"
	"github.com/mateimicu/tmux-claude-matrix/pkg/types"
)

func createCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "create",
		Short: "Create a new tmux session",
		Long:  `Create a new tmux session by selecting a repository from configured sources.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreate(cmd.Context())
		},
	}
}

func runCreate(ctx context.Context) error {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Build sources list
	var sources []repos.Source

	if cfg.LocalConfigEnabled && cfg.LocalReposFile != "" {
		sources = append(sources, repos.NewLocalSource(cfg.LocalReposFile))
	}

	if cfg.GitHubEnabled {
		token, source := repos.GetGitHubToken()
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

	// Discover repos
	discoverer := repos.NewDiscoverer(sources...)
	fmt.Println("🔍 Discovering repositories...")

	repoList, err := discoverer.ListAll(ctx)
	if err != nil {
		return fmt.Errorf("failed to discover repositories: %w", err)
	}

	if len(repoList) == 0 {
		return fmt.Errorf("no repositories found")
	}

	fmt.Printf("✓ Found %d repositories\n", len(repoList))

	// Let user select
	selected, err := fzf.SelectRepository(repoList)
	if err != nil {
		return fmt.Errorf("repository selection cancelled: %w", err)
	}

	// Generate session name
	sessionMgr := session.NewManager(cfg.SessionsDir)
	repoName := git.ExtractRepoName(selected.URL)
	sessionName, err := sessionMgr.GenerateUniqueName(repoName)
	if err != nil {
		return fmt.Errorf("failed to generate session name: %w", err)
	}

	// Clone repo
	clonePath := filepath.Join(cfg.CloneDir, sessionName)
	gitMgr := git.New()

	// Check if already cloned
	if _, err := os.Stat(clonePath); err == nil {
		fmt.Printf("📦 Repository already exists at %s\n", clonePath)
	} else {
		fmt.Printf("📦 Cloning %s...\n", selected.URL)
		if err := gitMgr.Clone(selected.URL, clonePath); err != nil {
			return fmt.Errorf("failed to clone repository: %w", err)
		}
		fmt.Println("✓ Clone complete")
	}

	// Create tmux session with Claude
	tmuxMgr := tmux.New()

	var claudeCmd string
	if cfg.ClaudeBin != "" {
		claudeCmd = cfg.ClaudeBin + " " + strings.Join(cfg.ClaudeArgs, " ")
	}

	fmt.Printf("🚀 Creating tmux session '%s'...\n", sessionName)
	if err := tmuxMgr.CreateSession(sessionName, clonePath, claudeCmd); err != nil {
		return fmt.Errorf("failed to create tmux session: %w", err)
	}

	// Save metadata
	sess := &types.Session{
		Name:      sessionName,
		RepoURL:   selected.URL,
		ClonePath: clonePath,
		CreatedAt: time.Now(),
	}
	if err := sessionMgr.Save(sess); err != nil {
		fmt.Printf("⚠️  Failed to save session metadata: %v\n", err)
	}

	fmt.Println("✓ Session created successfully!")

	// Switch to session
	if err := tmuxMgr.SwitchToSession(sessionName); err != nil {
		fmt.Printf("⚠️  Failed to switch to session: %v\n", err)
		fmt.Printf("You can attach manually with: tmux attach -t %s\n", sessionName)
	}

	return nil
}

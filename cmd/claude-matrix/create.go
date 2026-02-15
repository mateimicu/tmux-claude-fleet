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
	sources, err := buildSources(ctx, cfg, os.Stdout)
	if err != nil {
		return err
	}

	// Discover repos
	discoverer := repos.NewDiscoverer(sources...)
	fmt.Println("🔍 Discovering repositories...")

	discoveryCtx, discoveryCancel := context.WithTimeout(ctx, 15*time.Second)
	defer discoveryCancel()

	repoList, err := discoverer.ListAll(discoveryCtx)
	if err != nil {
		return fmt.Errorf("failed to discover repositories: %w", err)
	}

	if len(repoList) == 0 {
		return fmt.Errorf("no repositories found")
	}

	fmt.Printf("✓ Found %d repositories\n", len(repoList))

	// Check FZF version before launching selection UI
	if err := fzf.CheckFZFVersion(); err != nil {
		return err
	}

	// Get binary path for FZF reload
	binaryPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get binary path: %w", err)
	}

	// Let user select
	selected, err := fzf.SelectRepository(repoList, binaryPath)
	if err != nil {
		return fmt.Errorf("repository selection cancelled: %w", err)
	}

	sessionMgr := session.NewManager(cfg.SessionsDir)
	gitMgr := git.New()
	tmuxMgr := tmux.New()

	if selected.IsWorkspace {
		return createWorkspaceSession(cfg, selected, sessionMgr, gitMgr, tmuxMgr)
	}

	return createRepoSession(cfg, selected, sessionMgr, gitMgr, tmuxMgr)
}

func createRepoSession(cfg *types.Config, selected *types.Repository, sessionMgr *session.Manager, gitMgr *git.Manager, tmuxMgr *tmux.Manager) error {
	repoName := git.ExtractRepoName(selected.URL)
	sessionName, err := sessionMgr.GenerateUniqueName(repoName)
	if err != nil {
		return fmt.Errorf("failed to generate session name: %w", err)
	}

	clonePath := filepath.Join(cfg.CloneDir, sessionName)

	if _, err := os.Stat(clonePath); err == nil {
		fmt.Printf("📦 Repository already exists at %s\n", clonePath)
	} else {
		fmt.Printf("📦 Cloning %s (using cache for faster cloning)...\n", selected.URL)
		if err := gitMgr.CloneWithCache(selected.URL, clonePath, cfg.CacheDir); err != nil {
			return fmt.Errorf("failed to clone repository: %w", err)
		}
		fmt.Println("✓ Clone complete")
	}

	var claudeCmd string
	if cfg.ClaudeBin != "" {
		claudeCmd = cfg.ClaudeBin + " " + strings.Join(cfg.ClaudeArgs, " ")
	}

	fmt.Printf("🚀 Creating tmux session '%s'...\n", sessionName)
	if err := tmuxMgr.CreateSession(sessionName, clonePath, claudeCmd); err != nil {
		return fmt.Errorf("failed to create tmux session: %w", err)
	}

	// Generate title from display name (org/repo format)
	title := sessionMgr.GenerateTitle(selected.Name)

	sess := &types.Session{
		Name:      sessionName,
		RepoURL:   selected.URL,
		Title:     title,
		ClonePath: clonePath,
		CreatedAt: time.Now(),
	}
	if err := sessionMgr.Save(sess); err != nil {
		fmt.Printf("⚠️  Failed to save session metadata: %v\n", err)
	}

	// Set tmux session env var for status bar display
	if err := tmuxMgr.SetSessionEnv(sessionName, "@claude-matrix-title", title); err != nil {
		fmt.Printf("⚠️  Failed to set session title env: %v\n", err)
	}

	fmt.Printf("✓ Session created: %s\n", title)

	if err := tmuxMgr.SwitchToSession(sessionName); err != nil {
		fmt.Printf("⚠️  Failed to switch to session: %v\n", err)
		fmt.Printf("You can attach manually with: tmux attach -t %s\n", sessionName)
	}

	return nil
}

func createWorkspaceSession(cfg *types.Config, selected *types.Repository, sessionMgr *session.Manager, gitMgr *git.Manager, tmuxMgr *tmux.Manager) error {
	sessionName, err := sessionMgr.GenerateUniqueName(selected.Name)
	if err != nil {
		return fmt.Errorf("failed to generate session name: %w", err)
	}

	workspacePath := filepath.Join(cfg.CloneDir, sessionName)
	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		return fmt.Errorf("failed to create workspace directory: %w", err)
	}

	fmt.Printf("📦 Setting up workspace '%s' with %d repos...\n", selected.Name, len(selected.WorkspaceRepos))

	for _, repoURL := range selected.WorkspaceRepos {
		repoName := git.ExtractRepoName(repoURL)
		// Replace slashes with dashes for directory name
		dirName := strings.ReplaceAll(repoName, "/", "-")
		clonePath := filepath.Join(workspacePath, dirName)

		if _, err := os.Stat(clonePath); err == nil {
			fmt.Printf("  ✓ %s already exists\n", repoName)
		} else {
			fmt.Printf("  📦 Cloning %s...\n", repoName)
			if err := gitMgr.Clone(repoURL, clonePath); err != nil {
				return fmt.Errorf("failed to clone %s: %w", repoURL, err)
			}
			fmt.Printf("  ✓ %s cloned\n", repoName)
		}
	}

	var claudeCmd string
	if cfg.ClaudeBin != "" {
		claudeCmd = cfg.ClaudeBin + " " + strings.Join(cfg.ClaudeArgs, " ")
	}

	fmt.Printf("🚀 Creating tmux session '%s'...\n", sessionName)
	if err := tmuxMgr.CreateSession(sessionName, workspacePath, claudeCmd); err != nil {
		return fmt.Errorf("failed to create tmux session: %w", err)
	}

	// Generate title for workspace
	wsTitle := sessionMgr.GenerateTitle(selected.Name)

	sess := &types.Session{
		Name:      sessionName,
		RepoURL:   "workspace:" + selected.Name,
		Title:     wsTitle,
		RepoURLs:  selected.WorkspaceRepos,
		ClonePath: workspacePath,
		CreatedAt: time.Now(),
	}
	if err := sessionMgr.Save(sess); err != nil {
		fmt.Printf("⚠️  Failed to save session metadata: %v\n", err)
	}

	// Set tmux session env var for status bar display
	if err := tmuxMgr.SetSessionEnv(sessionName, "@claude-matrix-title", wsTitle); err != nil {
		fmt.Printf("⚠️  Failed to set session title env: %v\n", err)
	}

	fmt.Printf("✓ Workspace session created: %s\n", wsTitle)

	if err := tmuxMgr.SwitchToSession(sessionName); err != nil {
		fmt.Printf("⚠️  Failed to switch to session: %v\n", err)
		fmt.Printf("You can attach manually with: tmux attach -t %s\n", sessionName)
	}

	return nil
}

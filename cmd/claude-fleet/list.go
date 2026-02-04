package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/mateimicu/tmux-claude-fleet/internal/config"
	"github.com/mateimicu/tmux-claude-fleet/internal/fzf"
	"github.com/mateimicu/tmux-claude-fleet/internal/session"
	"github.com/mateimicu/tmux-claude-fleet/internal/tmux"
	"github.com/mateimicu/tmux-claude-fleet/pkg/types"
	"github.com/spf13/cobra"
)

func listCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List and switch to existing sessions",
		Long:  `List all managed tmux sessions and switch to one.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd.Context())
		},
	}
}

func runList(ctx context.Context) error {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Load sessions
	sessionMgr := session.NewManager(cfg.SessionsDir)
	sessions, err := sessionMgr.List()
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}

	if len(sessions) == 0 {
		fmt.Println("No sessions found. Create one with: claude-fleet create")
		return nil
	}

	// Get tmux status
	tmuxMgr := tmux.New()
	activeSessions, _ := tmuxMgr.ListSessions()
	activeMap := make(map[string]bool)
	for _, name := range activeSessions {
		activeMap[name] = true
	}

	// Build session status list
	var statusList []*types.SessionStatus
	for _, sess := range sessions {
		status := &types.SessionStatus{
			Session:       sess,
			TmuxActive:    activeMap[sess.Name],
			ClaudeRunning: false,
		}

		// Check Claude status if session is active
		if status.TmuxActive {
			status.ClaudeRunning = tmuxMgr.GetClaudeStatus(sess.Name)
		}

		statusList = append(statusList, status)
	}

	// Show FZF selection
	selected, err := fzf.SelectSession(statusList)
	if err != nil {
		return fmt.Errorf("session selection cancelled: %w", err)
	}

	// Switch to session
	fmt.Printf("🚀 Switching to session '%s'...\n", selected.Session.Name)

	// If session is not active, recreate it
	if !selected.TmuxActive {
		fmt.Println("⚠️  Session not active, recreating...")
		if err := tmuxMgr.CreateSession(selected.Session.Name, selected.Session.ClonePath); err != nil {
			return fmt.Errorf("failed to recreate session: %w", err)
		}

		// Create windows
		tmuxMgr.CreateWindow(selected.Session.Name, "terminal", "", selected.Session.ClonePath)

		if cfg.ClaudeBin != "" {
			claudeCmd := cfg.ClaudeBin + " " + strings.Join(cfg.ClaudeArgs, " ")
			tmuxMgr.CreateWindow(selected.Session.Name, "claude", claudeCmd, selected.Session.ClonePath)
		}
	}

	if err := tmuxMgr.SwitchToSession(selected.Session.Name); err != nil {
		fmt.Printf("⚠️  Failed to switch to session: %v\n", err)
		fmt.Printf("You can attach manually with: tmux attach -t %s\n", selected.Session.Name)
	}

	return nil
}

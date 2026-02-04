package main

import (
	"context"
	"fmt"
	"os"

	"github.com/mateimicu/tmux-claude-fleet/internal/config"
	"github.com/mateimicu/tmux-claude-fleet/internal/fzf"
	"github.com/mateimicu/tmux-claude-fleet/internal/session"
	"github.com/mateimicu/tmux-claude-fleet/internal/tmux"
	"github.com/mateimicu/tmux-claude-fleet/pkg/types"
	"github.com/spf13/cobra"
)

func deleteCmd() *cobra.Command {
	var keepClone bool

	cmd := &cobra.Command{
		Use:   "delete [session-name]",
		Short: "Delete a session",
		Long:  `Delete a managed session. Optionally delete the cloned repository.`,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var sessionName string
			if len(args) > 0 {
				sessionName = args[0]
			}
			return runDelete(cmd.Context(), sessionName, keepClone)
		},
	}

	cmd.Flags().BoolVar(&keepClone, "keep-clone", false, "Keep the cloned repository")

	return cmd
}

func runDelete(_ context.Context, sessionName string, keepClone bool) error {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	sessionMgr := session.NewManager(cfg.SessionsDir)

	// If no session name provided, show FZF selection
	if sessionName == "" {
		sessions, err := sessionMgr.List()
		if err != nil {
			return fmt.Errorf("failed to list sessions: %w", err)
		}

		if len(sessions) == 0 {
			fmt.Println("No sessions found.")
			return nil
		}

		// Get tmux status
		tmuxMgr := tmux.New()
		activeSessions, err := tmuxMgr.ListSessions()
		if err != nil {
			return fmt.Errorf("failed to list tmux sessions: %w", err)
		}
		activeMap := make(map[string]bool)
		for _, name := range activeSessions {
			activeMap[name] = true
		}

		// Build session status list
		var statusList []*types.SessionStatus
		for _, sess := range sessions {
			status := &types.SessionStatus{
				Session:    sess,
				TmuxActive: activeMap[sess.Name],
			}
			statusList = append(statusList, status)
		}

		selected, err := fzf.SelectSession(statusList)
		if err != nil {
			return fmt.Errorf("session selection cancelled: %w", err)
		}

		sessionName = selected.Session.Name
	}

	// Load session metadata
	sess, err := sessionMgr.Load(sessionName)
	if err != nil {
		return fmt.Errorf("session '%s' not found: %w", sessionName, err)
	}

	// Confirm deletion
	fmt.Printf("🗑️  Delete session '%s'?\n", sessionName)
	fmt.Printf("   Repository: %s\n", sess.RepoURL)
	fmt.Printf("   Clone path: %s\n", sess.ClonePath)
	if !keepClone {
		fmt.Println("   ⚠️  This will also DELETE the cloned repository!")
	}
	fmt.Print("\nType 'yes' to confirm: ")

	var confirmation string
	if _, err := fmt.Scanln(&confirmation); err != nil {
		fmt.Println("Deletion cancelled.")
		return nil
	}

	if confirmation != "yes" {
		fmt.Println("Deletion cancelled.")
		return nil
	}

	// Kill tmux session if active
	tmuxMgr := tmux.New()
	if tmuxMgr.SessionExists(sessionName) {
		fmt.Printf("🛑 Killing tmux session '%s'...\n", sessionName)
		if err := tmuxMgr.KillSession(sessionName); err != nil {
			fmt.Printf("⚠️  Failed to kill tmux session: %v\n", err)
		}
	}

	// Delete clone if requested
	if !keepClone {
		fmt.Printf("🗑️  Deleting clone at %s...\n", sess.ClonePath)
		if err := os.RemoveAll(sess.ClonePath); err != nil {
			fmt.Printf("⚠️  Failed to delete clone: %v\n", err)
		}
	}

	// Delete metadata
	if err := sessionMgr.Delete(sessionName); err != nil {
		return fmt.Errorf("failed to delete session metadata: %w", err)
	}

	fmt.Println("✓ Session deleted successfully!")

	return nil
}

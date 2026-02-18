package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mateimicu/tmux-claude-matrix/internal/session"
	"github.com/mateimicu/tmux-claude-matrix/internal/tmux"
)

func renameCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rename [title]",
		Short: "Rename the current session",
		Long:  `Set a new title for the current tmux session. If no title is provided, reads from stdin.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var title string
			if len(args) > 0 {
				title = strings.Join(args, " ")
			} else {
				fmt.Print("Enter new title: ")
				scanner := bufio.NewScanner(os.Stdin)
				if scanner.Scan() {
					title = strings.TrimSpace(scanner.Text())
				}
				if title == "" {
					return fmt.Errorf("no title provided")
				}
			}
			return runRename(cmd.Context(), title)
		},
	}
}

func runRename(ctx context.Context, title string) error {
	// Detect current tmux session
	sessionName, err := getCurrentTmuxSession()
	if err != nil {
		return fmt.Errorf("failed to detect current tmux session: %w", err)
	}

	cfg := configFromContext(ctx)
	log := loggerFromContext(ctx)

	// Load session metadata
	sessionMgr := session.NewManager(cfg.SessionsDir)
	sess, err := sessionMgr.Load(sessionName)
	if err != nil {
		return fmt.Errorf("session %q not found in metadata: %w", sessionName, err)
	}

	// Update title
	sess.Title = title
	if err := sessionMgr.Save(sess); err != nil {
		return fmt.Errorf("failed to save session metadata: %w", err)
	}

	// Update tmux env var
	tmuxMgr := tmux.New()
	if err := tmuxMgr.SetSessionEnv(sessionName, "@claude-matrix-title", title); err != nil {
		log.Warnf("Warning: failed to update tmux env var: %v\n", err)
	}

	// User-facing success confirmation — always visible
	fmt.Printf("Session '%s' renamed to '%s'\n", sessionName, title)
	return nil
}

func getCurrentTmuxSession() (string, error) {
	cmd := exec.Command("tmux", "display-message", "-p", "#S")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

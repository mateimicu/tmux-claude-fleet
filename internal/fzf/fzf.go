package fzf

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/mateimicu/tmux-claude-fleet/pkg/types"
)

// SelectRepository shows FZF interface for repo selection
func SelectRepository(repos []*types.Repository) (*types.Repository, error) {
	if len(repos) == 0 {
		return nil, fmt.Errorf("no repositories found")
	}

	// Format repos for display
	var lines []string
	for _, repo := range repos {
		line := formatRepoLine(repo)
		lines = append(lines, line)
	}

	// Run FZF
	selected, err := runFZF(strings.Join(lines, "\n"),
		"--prompt=📁 Select repository > ",
		"--reverse",
		"--border=rounded",
		"--header=↑↓ navigate | enter: select | ctrl-c: cancel",
		"--height=40%",
	)
	if err != nil {
		return nil, err
	}

	// Extract URL from selected line
	url := extractURL(selected)

	// Find original repo
	for _, repo := range repos {
		if repo.URL == url {
			return repo, nil
		}
	}

	return nil, fmt.Errorf("selected repo not found")
}

// SessionAction represents an action to perform on a session
type SessionAction string

const (
	// SessionActionSwitch indicates switching to a session
	SessionActionSwitch SessionAction = "switch"
	// SessionActionDelete indicates deleting a session
	SessionActionDelete SessionAction = "delete"
	// SessionActionCancel indicates cancelling the selection
	SessionActionCancel SessionAction = "cancel"
)

// SessionSelection represents the result of session selection
type SessionSelection struct {
	Session *types.SessionStatus
	Action  SessionAction
}

// SelectSession shows FZF interface for session selection
func SelectSession(sessions []*types.SessionStatus) (*types.SessionStatus, error) {
	selection, err := SelectSessionWithAction(sessions)
	if err != nil {
		return nil, err
	}
	if selection.Action == SessionActionCancel {
		return nil, fmt.Errorf("selection cancelled")
	}
	return selection.Session, nil
}

// SelectSessionWithAction shows FZF interface for session selection with action support
func SelectSessionWithAction(sessions []*types.SessionStatus) (*SessionSelection, error) {
	if len(sessions) == 0 {
		return nil, fmt.Errorf("no sessions found")
	}

	// Format sessions for display
	var lines []string
	for _, sess := range sessions {
		line := formatSessionLine(sess)
		lines = append(lines, line)
	}

	// Run FZF with action keys
	key, selected, err := runFZFWithExpect(
		strings.Join(lines, "\n"),
		[]string{"ctrl-d"},
		"--prompt=🚀 Select session > ",
		"--reverse",
		"--border=rounded",
		"--header=↑↓ navigate | enter: switch | ctrl-d: delete | ctrl-c: cancel",
		"--height=40%",
	)
	if err != nil {
		return &SessionSelection{Action: SessionActionCancel}, err
	}

	// Extract session name from selected line
	name := extractSessionName(selected)

	// Find original session
	for _, sess := range sessions {
		if sess.Session.Name == name {
			action := SessionActionSwitch
			if key == "ctrl-d" {
				action = SessionActionDelete
			}
			return &SessionSelection{
				Session: sess,
				Action:  action,
			}, nil
		}
	}

	return nil, fmt.Errorf("selected session not found")
}

func formatRepoLine(r *types.Repository) string {
	if r.Description != "" {
		return fmt.Sprintf("%s: %s - %s [%s]", r.Source, r.Name, r.Description, r.URL)
	}
	return fmt.Sprintf("%s: %s [%s]", r.Source, r.Name, r.URL)
}

func formatSessionLine(s *types.SessionStatus) string {
	status := "⚫"
	if s.TmuxActive {
		status = "🟢"
	}

	claudeStatus := ""
	if s.ClaudeRunning {
		claudeStatus = " [Claude ✓]"
	}

	return fmt.Sprintf("%s %s - %s%s [%s]",
		status, s.Session.Name, s.Session.RepoURL, claudeStatus, s.Session.Name)
}

func runFZF(input string, args ...string) (string, error) {
	cmd := exec.Command("fzf", args...)
	cmd.Stdin = strings.NewReader(input)

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", err
	}

	return strings.TrimSpace(out.String()), nil
}

// runFZFWithExpect runs FZF with --expect to capture key presses
// Returns the key pressed and the selected line
func runFZFWithExpect(input string, expectedKeys []string, args ...string) (string, string, error) {
	// Add --expect flag with keys
	expectArg := "--expect=" + strings.Join(expectedKeys, ",")
	allArgs := append([]string{expectArg}, args...)

	cmd := exec.Command("fzf", allArgs...)
	cmd.Stdin = strings.NewReader(input)

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", "", err
	}

	// Output format with --expect:
	// Line 1: key pressed (empty if Enter)
	// Line 2: selected item
	output := strings.TrimSpace(out.String())
	lines := strings.SplitN(output, "\n", 2)

	if len(lines) == 0 {
		return "", "", fmt.Errorf("no output from fzf")
	}

	if len(lines) == 1 {
		// Only one line means Enter was pressed
		return "", lines[0], nil
	}

	// Two lines: first is key, second is selection
	return lines[0], lines[1], nil
}

func extractURL(line string) string {
	// Extract URL from format: "source: name [URL]" or "source: name - desc [URL]"
	start := strings.LastIndex(line, "[")
	end := strings.LastIndex(line, "]")
	if start > 0 && end > start {
		return line[start+1 : end]
	}
	return ""
}

func extractSessionName(line string) string {
	// Extract session name from format: "status name - url [name]"
	start := strings.LastIndex(line, "[")
	end := strings.LastIndex(line, "]")
	if start > 0 && end > start {
		return line[start+1 : end]
	}
	return ""
}

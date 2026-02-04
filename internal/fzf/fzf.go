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

// SelectSession shows FZF interface for session selection
func SelectSession(sessions []*types.SessionStatus) (*types.SessionStatus, error) {
	if len(sessions) == 0 {
		return nil, fmt.Errorf("no sessions found")
	}

	// Format sessions for display
	var lines []string
	for _, sess := range sessions {
		line := formatSessionLine(sess)
		lines = append(lines, line)
	}

	// Run FZF
	selected, err := runFZF(strings.Join(lines, "\n"),
		"--prompt=🚀 Select session > ",
		"--reverse",
		"--border=rounded",
		"--header=↑↓ navigate | enter: switch | ctrl-c: cancel",
		"--height=40%",
	)
	if err != nil {
		return nil, err
	}

	// Extract session name from selected line
	name := extractSessionName(selected)

	// Find original session
	for _, sess := range sessions {
		if sess.Session.Name == name {
			return sess, nil
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

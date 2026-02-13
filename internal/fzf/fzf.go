package fzf

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/mateimicu/tmux-claude-matrix/pkg/types"
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

	// Sort sessions by creation time (newest first)
	sortedSessions := make([]*types.SessionStatus, len(sessions))
	copy(sortedSessions, sessions)
	sort.Slice(sortedSessions, func(i, j int) bool {
		return sortedSessions[i].Session.CreatedAt.After(sortedSessions[j].Session.CreatedAt)
	})

	// Format sessions for display with numbering
	var lines []string
	for idx, sess := range sortedSessions {
		line := formatSessionLine(sess, idx+1, len(sortedSessions))
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

func formatSessionLine(s *types.SessionStatus, sessionNum, totalSessions int) string {
	// Tmux session status
	status := "⚫"
	if s.TmuxActive {
		status = "🟢"
	}

	// Claude state with detailed indicators
	claudeStatus := getClaudeStatusIndicator(s.ClaudeState)
	claudeDesc := getClaudeStateDescription(s.ClaudeState)

	// Parse repo URL to get source and org/repo
	source, orgRepo := parseRepoURL(s.Session.RepoURL)

	// Calculate padding width based on total sessions
	paddingWidth := len(fmt.Sprintf("%d", totalSessions))
	sessionNumStr := fmt.Sprintf("%0*d", paddingWidth, sessionNum)

	return fmt.Sprintf("%s %s: %s - %s %s %s [%s]",
		status, source, orgRepo, sessionNumStr, claudeStatus, claudeDesc, s.Session.Name)
}

// getClaudeStatusIndicator returns the emoji indicator for Claude state
func getClaudeStatusIndicator(state types.ClaudeState) string {
	switch state {
	case types.ClaudeStateRunning:
		return "🟢"
	case types.ClaudeStateWaitingForInput:
		return "⏸️"
	case types.ClaudeStateIdle:
		return "💤"
	case types.ClaudeStateError:
		return "⚠️"
	case types.ClaudeStateStopped:
		return "⚫"
	default:
		return "❓"
	}
}

// getClaudeStateDescription returns a human-readable description
func getClaudeStateDescription(state types.ClaudeState) string {
	switch state {
	case types.ClaudeStateRunning:
		return "[Claude: Active]"
	case types.ClaudeStateWaitingForInput:
		return "[Claude: Needs Input]"
	case types.ClaudeStateIdle:
		return "[Claude: Idle]"
	case types.ClaudeStateError:
		return "[Claude: Error]"
	case types.ClaudeStateStopped:
		return "[Claude: Stopped]"
	default:
		return "[Claude: Unknown]"
	}
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

// runFZFWithExpect runs FZF with key bindings to capture special key presses.
// It uses --bind with println+accept instead of --expect to reliably override
// FZF's default key bindings (e.g., ctrl-d's default delete-char/eof action)
// and any user configuration in FZF_DEFAULT_OPTS.
// Returns the key pressed and the selected line.
func runFZFWithExpect(input string, expectedKeys []string, args ...string) (string, string, error) {
	// Build --bind args for each expected key.
	// Using --bind instead of --expect because --expect=ctrl-d can conflict
	// with FZF's built-in ctrl-d binding (delete-char/eof) or user's
	// FZF_DEFAULT_OPTS, causing the wrong item to be returned.
	// --bind explicitly overrides any previous binding for the key.
	var allArgs []string
	for _, key := range expectedKeys {
		allArgs = append(allArgs, fmt.Sprintf("--bind=%s:println(%s)+accept", key, key))
	}
	allArgs = append(allArgs, args...)

	cmd := exec.Command("fzf", allArgs...)
	cmd.Stdin = strings.NewReader(input)

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr

	// Filter out FZF_DEFAULT_OPTS to prevent user configuration from
	// interfering with our key bindings
	cmd.Env = filterFZFEnv(os.Environ())

	if err := cmd.Run(); err != nil {
		return "", "", err
	}

	return parseFZFOutput(out.String(), expectedKeys)
}

// parseFZFOutput parses FZF output to extract the key pressed and selected line.
// When an expected key is pressed (via --bind println+accept): key\nselected\n
// When Enter is pressed: selected\n
func parseFZFOutput(output string, expectedKeys []string) (string, string, error) {
	if strings.TrimSpace(output) == "" {
		return "", "", fmt.Errorf("no output from fzf")
	}

	// Split on first newline to separate potential key from selection
	parts := strings.SplitN(output, "\n", 2)

	if len(parts) >= 2 {
		potentialKey := strings.TrimSpace(parts[0])
		for _, key := range expectedKeys {
			if potentialKey == key {
				return key, strings.TrimSpace(parts[1]), nil
			}
		}
	}

	// No expected key matched - Enter was pressed
	return "", strings.TrimSpace(output), nil
}

// filterFZFEnv removes FZF configuration environment variables that could
// interfere with our key bindings
func filterFZFEnv(env []string) []string {
	filtered := make([]string, 0, len(env))
	for _, e := range env {
		if !strings.HasPrefix(e, "FZF_DEFAULT_OPTS=") &&
			!strings.HasPrefix(e, "FZF_DEFAULT_OPTS_FILE=") {
			filtered = append(filtered, e)
		}
	}
	return filtered
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
	// Extract session name from format: "status source: org/repo - 01 [name]"
	start := strings.LastIndex(line, "[")
	end := strings.LastIndex(line, "]")
	if start > 0 && end > start {
		return line[start+1 : end]
	}
	return ""
}

// parseRepoURL extracts the source type (github/local) and org/repo from a repository URL
func parseRepoURL(url string) (source, orgRepo string) {
	// Check if it's a GitHub URL
	if strings.Contains(url, "github.com") {
		source = "github"
		// Handle different GitHub URL formats
		// HTTPS: https://github.com/org/repo or https://github.com/org/repo.git
		// SSH: git@github.com:org/repo.git
		if path, found := strings.CutPrefix(url, "git@github.com:"); found {
			// SSH format: git@github.com:org/repo.git
			path = strings.TrimSuffix(path, ".git")
			orgRepo = path
		} else if strings.Contains(url, "github.com/") {
			// HTTPS format: https://github.com/org/repo or https://github.com/org/repo.git
			parts := strings.Split(url, "github.com/")
			if len(parts) >= 2 {
				path := parts[1]
				path = strings.TrimSuffix(path, ".git")
				orgRepo = path
			}
		}
	} else {
		// Assume local repository
		source = "local"
		// Extract the last two path components as org/repo
		url = strings.TrimSuffix(url, "/")
		parts := strings.Split(url, "/")
		switch {
		case len(parts) >= 2:
			orgRepo = parts[len(parts)-2] + "/" + parts[len(parts)-1]
		case len(parts) == 1:
			orgRepo = parts[0]
		default:
			orgRepo = url
		}
	}

	// Fallback if orgRepo is empty
	if orgRepo == "" {
		orgRepo = url
	}

	return source, orgRepo
}

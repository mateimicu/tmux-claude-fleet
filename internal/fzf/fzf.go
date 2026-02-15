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
		line := FormatRepoLine(repo)
		lines = append(lines, line)
	}

	// Run FZF
	selected, err := runFZF(strings.Join(lines, "\n"),
		"--prompt=📁 Select repository > ",
		"--reverse",
		"--border=rounded",
		"--header=↑↓ navigate | enter: select | ctrl-c: cancel",
		"--height=80%",
	)
	if err != nil {
		return nil, err
	}

	// Extract identifier from selected line
	identifier := extractURL(selected)

	// Check if it's a workspace selection
	if name, ok := strings.CutPrefix(identifier, "workspace:"); ok {
		for _, repo := range repos {
			if repo.IsWorkspace && repo.Name == name {
				return repo, nil
			}
		}
		return nil, fmt.Errorf("selected workspace not found: %s", name)
	}

	// Find original repo by URL
	for _, repo := range repos {
		if repo.URL == identifier {
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

	// Format sessions as aligned table
	headerLine, lines := formatSessionTable(sortedSessions)

	// Prepend header line so FZF can freeze it with --header-lines=1
	allLines := append([]string{headerLine}, lines...)

	// Run FZF with action keys
	legend := "↑↓ navigate | enter: switch | ctrl-d: delete | ctrl-c: cancel\n" +
		"Session: 🟢 active  ⚫ inactive | Claude: 🟢 Active  ❓ Waiting  💬 Ready  ⚠️ Error  ⚫ Stopped  ❔ Unknown"
	key, selected, err := runFZFWithExpect(
		strings.Join(allLines, "\n"),
		[]string{"ctrl-d"},
		"--prompt=🚀 Select session > ",
		"--reverse",
		"--border=rounded",
		"--header="+legend,
		"--header-lines=1",
		"--height=80%",
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

// FormatRepoLine formats a repository as a single line for FZF display.
func FormatRepoLine(r *types.Repository) string {
	if r.IsWorkspace {
		return fmt.Sprintf("workspace: %s - %s [workspace:%s]", r.Name, r.Description, r.Name)
	}
	if r.Description != "" {
		return fmt.Sprintf("%s: %s - %s [%s]", r.Source, r.Name, r.Description, r.URL)
	}
	return fmt.Sprintf("%s: %s [%s]", r.Source, r.Name, r.URL)
}

// formatSessionTable formats all sessions as an aligned table.
// Returns a header line and data lines with columns padded to align.
func formatSessionTable(sessions []*types.SessionStatus) (string, []string) {
	paddingWidth := len(fmt.Sprintf("%d", len(sessions)))
	if paddingWidth < 1 {
		paddingWidth = 1
	}

	type rowData struct {
		num     string
		tmux    string
		source  string
		repo    string
		claude  string
		session string
	}

	// Pre-compute row data and track max column widths
	var rows []rowData
	maxSourceW := displayWidth("SOURCE")
	maxRepoW := displayWidth("REPOSITORY")
	maxClaudeW := displayWidth("CLAUDE")

	for idx, s := range sessions {
		source, orgRepo := parseRepoURL(s.Session.RepoURL)
		claudeIndicator := getClaudeStatusIndicator(s.ClaudeState)
		claudeLabel := getClaudeStateLabel(s.ClaudeState)

		tmux := "⚫"
		if s.TmuxActive {
			tmux = "🟢"
		}

		// Use title if available, otherwise fall back to orgRepo
		displayName := orgRepo
		if s.Session.Title != "" {
			displayName = s.Session.Title
		}

		claudeCol := claudeIndicator + " " + claudeLabel

		row := rowData{
			num:     fmt.Sprintf("%0*d", paddingWidth, idx+1),
			tmux:    tmux,
			source:  source,
			repo:    displayName,
			claude:  claudeCol,
			session: s.Session.Name,
		}
		rows = append(rows, row)

		if w := displayWidth(source); w > maxSourceW {
			maxSourceW = w
		}
		if w := displayWidth(displayName); w > maxRepoW {
			maxRepoW = w
		}
		if w := displayWidth(claudeCol); w > maxClaudeW {
			maxClaudeW = w
		}
	}

	// Build header
	header := fmt.Sprintf(" %s  %s  %s  %s  %s  %s",
		padToDisplayWidth("#", paddingWidth),
		padToDisplayWidth("TMUX", 4),
		padToDisplayWidth("SOURCE", maxSourceW),
		padToDisplayWidth("REPOSITORY", maxRepoW),
		padToDisplayWidth("CLAUDE", maxClaudeW),
		"SESSION",
	)

	// Build data lines
	var lines []string
	for _, r := range rows {
		line := fmt.Sprintf(" %s  %s  %s  %s  %s  [%s]",
			r.num,
			padToDisplayWidth(r.tmux, 4),
			padToDisplayWidth(r.source, maxSourceW),
			padToDisplayWidth(r.repo, maxRepoW),
			padToDisplayWidth(r.claude, maxClaudeW),
			r.session,
		)
		lines = append(lines, line)
	}

	return header, lines
}

// getClaudeStatusIndicator returns the emoji indicator for Claude state
func getClaudeStatusIndicator(state types.ClaudeState) string {
	switch state {
	case types.ClaudeStateRunning:
		return "🟢"
	case types.ClaudeStateWaitingForInput:
		return "❓"
	case types.ClaudeStateIdle:
		return "💬"
	case types.ClaudeStateError:
		return "⚠️"
	case types.ClaudeStateStopped:
		return "⚫"
	default:
		return "❔"
	}
}

// getClaudeStateLabel returns a short label for the Claude state
func getClaudeStateLabel(state types.ClaudeState) string {
	switch state {
	case types.ClaudeStateRunning:
		return "Active"
	case types.ClaudeStateWaitingForInput:
		return "Waiting"
	case types.ClaudeStateIdle:
		return "Ready"
	case types.ClaudeStateError:
		return "Error"
	case types.ClaudeStateStopped:
		return "Stopped"
	default:
		return "Unknown"
	}
}

// displayWidth returns the display width of a string, accounting for
// wide characters like emojis that take 2 terminal cells.
func displayWidth(s string) int {
	w := 0
	for _, r := range s {
		if r == 0xFE0F { // variation selector-16, zero-width
			continue
		}
		if isWideRune(r) {
			w += 2
		} else {
			w++
		}
	}
	return w
}

// isWideRune returns true if the rune occupies 2 terminal cells.
func isWideRune(r rune) bool {
	return (r >= 0x1F300 && r <= 0x1F9FF) || // Misc Symbols, Emoticons, Transport, etc.
		(r >= 0x2600 && r <= 0x27BF) || // Misc Symbols & Dingbats
		(r >= 0x2300 && r <= 0x23FF) || // Misc Technical
		(r >= 0x2B50 && r <= 0x2B55) // Stars
}

// padToDisplayWidth pads a string with spaces to reach the target display width.
func padToDisplayWidth(s string, width int) string {
	dw := displayWidth(s)
	if dw >= width {
		return s
	}
	return s + strings.Repeat(" ", width-dw)
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

// runFZFWithExpect runs FZF with --expect to capture special key presses.
// It filters FZF_DEFAULT_OPTS to prevent user configuration from interfering
// with our expected key bindings.
// Returns the key pressed (empty string for Enter) and the selected line.
func runFZFWithExpect(input string, expectedKeys []string, args ...string) (string, string, error) {
	// Use --expect to capture special key presses.
	// FZF_DEFAULT_OPTS is filtered out to prevent user bindings (e.g.,
	// --bind=ctrl-d:delete-char) from overriding our --expect keys.
	allArgs := []string{"--expect=" + strings.Join(expectedKeys, ",")}
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
// With --expect, fzf always outputs: key\nselected\n (key is empty for Enter).
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

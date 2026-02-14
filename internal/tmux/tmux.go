package tmux

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/mateimicu/tmux-claude-matrix/pkg/types"
)

// Manager handles tmux operations
type Manager struct{}

// New creates a new tmux Manager
func New() *Manager {
	return &Manager{}
}

// CreateSession creates a new tmux session
func (m *Manager) CreateSession(name, path, command string) error {
	args := []string{"new-session", "-d", "-s", name, "-c", path}
	if command != "" {
		args = append(args, command)
	}
	cmd := exec.Command("tmux", args...)
	return cmd.Run()
}

// CreateSessionWithCommand creates a new tmux session and runs a command in the first window
func (m *Manager) CreateSessionWithCommand(name, path, command string) error {
	args := []string{"new-session", "-d", "-s", name, "-c", path}
	if command != "" {
		args = append(args, command)
	}
	cmd := exec.Command("tmux", args...)
	return cmd.Run()
}

// CreateWindow creates a window in a session
func (m *Manager) CreateWindow(session, name, command, path string) error {
	args := []string{"new-window", "-t", session + ":", "-n", name}
	if path != "" {
		args = append(args, "-c", path)
	}
	if command != "" {
		args = append(args, command)
	}

	cmd := exec.Command("tmux", args...)
	return cmd.Run()
}

// SessionExists checks if a tmux session exists
func (m *Manager) SessionExists(name string) bool {
	cmd := exec.Command("tmux", "has-session", "-t", name)
	return cmd.Run() == nil
}

// KillSession kills a tmux session
func (m *Manager) KillSession(name string) error {
	cmd := exec.Command("tmux", "kill-session", "-t", name)
	return cmd.Run()
}

// SwitchToSession attaches or switches to a session
func (m *Manager) SwitchToSession(name string) error {
	if os.Getenv("TMUX") != "" {
		// Inside tmux, switch client
		cmd := exec.Command("tmux", "switch-client", "-t", name)
		return cmd.Run()
	}
	// Outside tmux, attach
	cmd := exec.Command("tmux", "attach-session", "-t", name)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// SetSessionEnv sets a session-level environment variable
func (m *Manager) SetSessionEnv(session, key, value string) error {
	cmd := exec.Command("tmux", "set-environment", "-t", session, key, value)
	return cmd.Run()
}

// GetSessionEnv gets a session-level environment variable
func (m *Manager) GetSessionEnv(session, key string) (string, error) {
	cmd := exec.Command("tmux", "show-environment", "-t", session, key)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	// Output format: "KEY=VALUE\n"
	line := strings.TrimSpace(string(output))
	if _, value, ok := strings.Cut(line, "="); ok {
		return value, nil
	}
	return "", fmt.Errorf("unexpected format: %s", line)
}

// GetClaudeStatus checks if Claude is running in session
func (m *Manager) GetClaudeStatus(session string) bool {
	// Get pane PIDs from the first window
	cmd := exec.Command("tmux", "list-panes", "-t", session,
		"-F", "#{pane_pid}")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	// Check each PID for claude process
	pids := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, pid := range pids {
		if m.processIsClaude(pid) {
			return true
		}
	}

	return false
}

// processIsClaude checks if a PID is running Claude
func (m *Manager) processIsClaude(pid string) bool {
	if pid == "" {
		return false
	}

	// Get process tree
	cmd := exec.Command("pgrep", "-P", pid)
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	childPids := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, childPid := range childPids {
		if childPid == "" {
			continue
		}

		// Check process name
		psCmd := exec.Command("ps", "-p", childPid, "-o", "comm=")
		psOutput, err := psCmd.Output()
		if err != nil {
			continue
		}

		processName := strings.TrimSpace(string(psOutput))
		if strings.Contains(processName, "claude") {
			return true
		}
	}

	return false
}

// ListSessions returns all tmux session names
func (m *Manager) ListSessions() ([]string, error) {
	cmd := exec.Command("tmux", "list-sessions", "-F", "#{session_name}")
	output, err := cmd.Output()
	if err != nil {
		// No sessions exist
		if strings.Contains(err.Error(), "no server running") {
			return nil, nil
		}
		return nil, err
	}

	sessions := strings.Split(strings.TrimSpace(string(output)), "\n")
	var result []string
	for _, s := range sessions {
		if s != "" {
			result = append(result, s)
		}
	}

	return result, nil
}

// SelectWindow selects a window in the session
func (m *Manager) SelectWindow(session, window string) error {
	cmd := exec.Command("tmux", "select-window", "-t", fmt.Sprintf("%s:%s", session, window))
	return cmd.Run()
}

// GetDetailedClaudeState returns the detailed state of Claude in a session
func (m *Manager) GetDetailedClaudeState(session string) (types.ClaudeState, time.Time) {
	// First check if Claude window exists
	cmd := exec.Command("tmux", "list-windows", "-t", session, "-F", "#{window_name}")
	output, err := cmd.Output()
	if err != nil {
		return types.ClaudeStateStopped, time.Time{}
	}

	hasClaudeWindow := false
	windows := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, w := range windows {
		if w == "claude" {
			hasClaudeWindow = true
			break
		}
	}

	if !hasClaudeWindow {
		return types.ClaudeStateStopped, time.Time{}
	}

	// Get pane PID
	cmd = exec.Command("tmux", "list-panes", "-t", session+":claude", "-F", "#{pane_pid}")
	output, err = cmd.Output()
	if err != nil {
		return types.ClaudeStateStopped, time.Time{}
	}

	pids := strings.Split(strings.TrimSpace(string(output)), "\n")
	var claudePID string
	for _, pid := range pids {
		if m.processIsClaude(pid) {
			claudePID = pid
			break
		}
	}

	if claudePID == "" {
		return types.ClaudeStateStopped, time.Time{}
	}

	// Get process state
	processState, err := m.getProcessState(claudePID)
	if err != nil {
		return types.ClaudeStateUnknown, time.Time{}
	}

	// Capture pane content to analyze
	content, err := m.capturePaneContent(session, "claude", 50)
	if err != nil {
		return types.ClaudeStateUnknown, time.Time{}
	}

	// Get last activity time from pane
	lastActivity := m.getPaneLastActivity(session, "claude")

	// Analyze state based on process state and output
	state := m.analyzeClaudeState(processState, content)
	return state, lastActivity
}

// capturePaneContent captures the last N lines from a pane
func (m *Manager) capturePaneContent(session, window string, lines int) (string, error) {
	target := fmt.Sprintf("%s:%s", session, window)
	cmd := exec.Command("tmux", "capture-pane", "-t", target, "-p", "-S", fmt.Sprintf("-%d", lines))
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// getProcessState returns the process state (R, S, D, Z, etc.)
func (m *Manager) getProcessState(pid string) (string, error) {
	// First get child PIDs
	cmd := exec.Command("pgrep", "-P", pid)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	childPids := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, childPid := range childPids {
		if childPid == "" {
			continue
		}

		// Check if this is the claude process
		psCmd := exec.Command("ps", "-p", childPid, "-o", "comm=")
		psOutput, err := psCmd.Output()
		if err != nil {
			continue
		}

		processName := strings.TrimSpace(string(psOutput))
		if strings.Contains(processName, "claude") {
			// Get state for this process
			stateCmd := exec.Command("ps", "-p", childPid, "-o", "state=")
			stateOutput, err := stateCmd.Output()
			if err != nil {
				return "", err
			}
			return strings.TrimSpace(string(stateOutput)), nil
		}
	}

	return "", fmt.Errorf("claude process not found")
}

// getPaneLastActivity returns the last activity time for a pane
func (m *Manager) getPaneLastActivity(session, window string) time.Time {
	target := fmt.Sprintf("%s:%s", session, window)
	cmd := exec.Command("tmux", "display-message", "-t", target, "-p", "#{pane_activity}")
	output, err := cmd.Output()
	if err != nil {
		return time.Time{}
	}

	// Parse Unix timestamp
	timestamp := strings.TrimSpace(string(output))
	var unixTime int64
	fmt.Sscanf(timestamp, "%d", &unixTime) //nolint:errcheck // Best-effort timestamp parsing
	if unixTime > 0 {
		return time.Unix(unixTime, 0)
	}

	return time.Time{}
}

// analyzeClaudeState analyzes process state and output to determine Claude's state
func (m *Manager) analyzeClaudeState(processState, content string) types.ClaudeState {
	// Check for error indicators in output
	errorIndicators := []string{
		"Error:",
		"error:",
		"ERROR:",
		"Exception:",
		"Traceback",
		"panic:",
		"fatal:",
	}

	for _, indicator := range errorIndicators {
		if strings.Contains(content, indicator) {
			return types.ClaudeStateError
		}
	}

	// Check for input waiting indicators
	inputIndicators := []string{
		"Continue? (y/n)",
		"Enter your choice:",
		"Waiting for",
		"[y/N]",
		"Press any key",
		"(yes/no)",
		"Continue?",
	}

	for _, indicator := range inputIndicators {
		if strings.Contains(content, indicator) {
			return types.ClaudeStateWaitingForInput
		}
	}

	// Check process state
	// R = Running, S = Sleeping/Idle, D = Disk wait, Z = Zombie
	switch processState {
	case "R", "R+":
		return types.ClaudeStateRunning
	case "S", "S+", "I", "I+":
		// Sleeping - could be idle or waiting
		// Check if there's recent output suggesting completion
		if strings.Contains(content, "completed") ||
			strings.Contains(content, "Done") ||
			strings.Contains(content, "finished") {
			return types.ClaudeStateIdle
		}
		// If sleeping with cursor visible, likely waiting for input
		return types.ClaudeStateWaitingForInput
	case "D", "D+":
		// Disk wait - actively working
		return types.ClaudeStateRunning
	case "Z":
		// Zombie process
		return types.ClaudeStateError
	default:
		return types.ClaudeStateUnknown
	}
}

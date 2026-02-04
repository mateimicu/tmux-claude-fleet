package tmux

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Manager handles tmux operations
type Manager struct{}

// New creates a new tmux Manager
func New() *Manager {
	return &Manager{}
}

// CreateSession creates a new tmux session
func (m *Manager) CreateSession(name, path string) error {
	cmd := exec.Command("tmux", "new-session", "-d", "-s", name, "-c", path)
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

// GetClaudeStatus checks if Claude is running in session
func (m *Manager) GetClaudeStatus(session string) bool {
	// Get pane PIDs from all windows in the session
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

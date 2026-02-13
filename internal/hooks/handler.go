package hooks

import (
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/mateimicu/tmux-claude-matrix/internal/status"
	"github.com/mateimicu/tmux-claude-matrix/pkg/types"
)

// HookEvent represents a Claude Code hook event received via stdin.
type HookEvent struct {
	HookEventName    string `json:"hook_event_name"`
	NotificationType string `json:"notification_type,omitempty"`
	SessionID        string `json:"session_id"`
}

// MapEventToState maps a hook event to its corresponding ClaudeState.
func MapEventToState(event *HookEvent) types.ClaudeState {
	switch event.HookEventName {
	case "SessionStart":
		return types.ClaudeStateIdle
	case "UserPromptSubmit":
		return types.ClaudeStateRunning
	case "PreToolUse":
		return types.ClaudeStateRunning
	case "Stop":
		return types.ClaudeStateIdle
	case "Notification":
		switch event.NotificationType {
		case "permission_prompt", "elicitation_dialog":
			return types.ClaudeStateWaitingForInput
		case "idle_prompt":
			return types.ClaudeStateIdle
		default:
			return types.ClaudeStateUnknown
		}
	case "SessionEnd":
		return types.ClaudeStateStopped
	default:
		return types.ClaudeStateUnknown
	}
}

// HandleHookEvent reads a hook event from stdin and updates tmux state accordingly.
func HandleHookEvent(reader io.Reader) error {
	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}

	var event HookEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return err
	}

	state := MapEventToState(&event)

	tmuxPane := os.Getenv("TMUX_PANE")
	if tmuxPane == "" {
		return nil
	}

	// Query tmux for the session name
	out, err := exec.Command("tmux", "display-message", "-t", tmuxPane, "-p", "#{session_name}").Output()
	if err != nil {
		return err
	}
	sessionName := strings.TrimSpace(string(out))

	statusDir := status.DefaultStatusDir()

	if state == types.ClaudeStateStopped {
		// Reset window name to plain "claude" before removing state
		_ = exec.Command("tmux", "rename-window", "-t", tmuxPane, "claude").Run() //nolint:errcheck // Best-effort reset
		return status.RemoveState(statusDir, sessionName)
	}

	// Read current state to avoid unnecessary tmux rename
	current, err := status.ReadState(statusDir, sessionName)
	if err == nil && current.State == string(state) {
		return nil
	}

	if err := status.WriteState(statusDir, sessionName, state, event.SessionID); err != nil {
		return err
	}

	emoji := status.EmojiForState(state)
	return exec.Command("tmux", "rename-window", "-t", tmuxPane, emoji+"claude").Run()
}

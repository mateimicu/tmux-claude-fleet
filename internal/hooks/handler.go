package hooks

import (
	"encoding/json"
	"io"
	"os"

	"github.com/mateimicu/tmux-claude-matrix/internal/status"
	"github.com/mateimicu/tmux-claude-matrix/internal/tmux"
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
// It writes per-agent state files and recomputes the aggregate for the session.
func HandleHookEvent(reader io.Reader, mgr *tmux.Manager) error {
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

	sessionName, err := mgr.GetSessionNameFromPane(tmuxPane)
	if err != nil {
		return err
	}

	statusDir := status.DefaultStatusDir()
	agentID := event.SessionID
	if agentID == "" {
		agentID = "default"
	}

	if state == types.ClaudeStateStopped {
		// Remove this agent's state file
		if err := status.RemoveAgentState(statusDir, sessionName, agentID); err != nil {
			return err
		}
	} else {
		// Skip write if this agent's state hasn't changed
		current, readErr := status.ReadAgentState(statusDir, sessionName, agentID)
		if readErr == nil && current.State == string(state) {
			return nil
		}
		if err := status.WriteAgentState(statusDir, sessionName, agentID, state); err != nil {
			return err
		}
	}

	// Recompute aggregate from all agent files
	aggState, err := status.UpdateAggregate(statusDir, sessionName, status.DefaultStaleThreshold)
	if err != nil {
		return err
	}

	// Update tmux window name to reflect aggregate state
	if aggState == types.ClaudeStateStopped {
		_ = mgr.RenameWindowByPane(tmuxPane, "claude") //nolint:errcheck // Best-effort reset
		return nil
	}
	emoji := status.EmojiForState(aggState)
	return mgr.RenameWindowByPane(tmuxPane, emoji+"claude")
}

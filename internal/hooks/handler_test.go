package hooks

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/mateimicu/tmux-claude-matrix/pkg/types"
)

func TestMapEventToState(t *testing.T) {
	tests := []struct {
		name  string
		event HookEvent
		want  types.ClaudeState
	}{
		{
			name:  "SessionStart maps to idle",
			event: HookEvent{HookEventName: "SessionStart"},
			want:  types.ClaudeStateIdle,
		},
		{
			name:  "UserPromptSubmit maps to running",
			event: HookEvent{HookEventName: "UserPromptSubmit"},
			want:  types.ClaudeStateRunning,
		},
		{
			name:  "PreToolUse maps to running",
			event: HookEvent{HookEventName: "PreToolUse"},
			want:  types.ClaudeStateRunning,
		},
		{
			name:  "Stop maps to idle",
			event: HookEvent{HookEventName: "Stop"},
			want:  types.ClaudeStateIdle,
		},
		{
			name:  "Notification with permission_prompt maps to waiting_for_input",
			event: HookEvent{HookEventName: "Notification", NotificationType: "permission_prompt"},
			want:  types.ClaudeStateWaitingForInput,
		},
		{
			name:  "Notification with elicitation_dialog maps to waiting_for_input",
			event: HookEvent{HookEventName: "Notification", NotificationType: "elicitation_dialog"},
			want:  types.ClaudeStateWaitingForInput,
		},
		{
			name:  "Notification with idle_prompt maps to idle",
			event: HookEvent{HookEventName: "Notification", NotificationType: "idle_prompt"},
			want:  types.ClaudeStateIdle,
		},
		{
			name:  "SessionEnd maps to stopped",
			event: HookEvent{HookEventName: "SessionEnd"},
			want:  types.ClaudeStateStopped,
		},
		{
			name:  "unknown event maps to unknown",
			event: HookEvent{HookEventName: "SomethingElse"},
			want:  types.ClaudeStateUnknown,
		},
		{
			name:  "Notification with unknown type maps to unknown",
			event: HookEvent{HookEventName: "Notification", NotificationType: "something_new"},
			want:  types.ClaudeStateUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MapEventToState(&tt.event)
			if got != tt.want {
				t.Errorf("MapEventToState(%+v) = %q, want %q", tt.event, got, tt.want)
			}
		})
	}
}

func TestParseHookEvent(t *testing.T) {
	event := HookEvent{
		HookEventName:    "Notification",
		NotificationType: "permission_prompt",
		SessionID:        "sess-abc-123",
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("failed to marshal test event: %v", err)
	}

	var parsed HookEvent
	if err := json.NewDecoder(bytes.NewReader(data)).Decode(&parsed); err != nil {
		t.Fatalf("failed to decode hook event: %v", err)
	}

	if parsed.HookEventName != "Notification" {
		t.Errorf("HookEventName = %q, want %q", parsed.HookEventName, "Notification")
	}
	if parsed.NotificationType != "permission_prompt" {
		t.Errorf("NotificationType = %q, want %q", parsed.NotificationType, "permission_prompt")
	}
	if parsed.SessionID != "sess-abc-123" {
		t.Errorf("SessionID = %q, want %q", parsed.SessionID, "sess-abc-123")
	}
}

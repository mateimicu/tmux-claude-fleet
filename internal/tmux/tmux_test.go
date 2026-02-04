package tmux

import (
	"testing"

	"github.com/mateimicu/tmux-claude-matrix/pkg/types"
)

func TestAnalyzeClaudeState(t *testing.T) {
	m := &Manager{}

	tests := []struct {
		name         string
		processState string
		content      string
		expected     types.ClaudeState
	}{
		{
			name:         "Error state - error indicator",
			processState: "S",
			content:      "Error: Something went wrong",
			expected:     types.ClaudeStateError,
		},
		{
			name:         "Error state - traceback",
			processState: "S",
			content:      "Traceback (most recent call last):",
			expected:     types.ClaudeStateError,
		},
		{
			name:         "Waiting for input - yes/no prompt",
			processState: "S",
			content:      "Continue? (yes/no)",
			expected:     types.ClaudeStateWaitingForInput,
		},
		{
			name:         "Waiting for input - y/n prompt",
			processState: "S",
			content:      "Continue? [y/N]",
			expected:     types.ClaudeStateWaitingForInput,
		},
		{
			name:         "Running state",
			processState: "R",
			content:      "Processing your request...",
			expected:     types.ClaudeStateRunning,
		},
		{
			name:         "Idle state - completed",
			processState: "S",
			content:      "Task completed successfully",
			expected:     types.ClaudeStateIdle,
		},
		{
			name:         "Idle state - done",
			processState: "S",
			content:      "Done processing",
			expected:     types.ClaudeStateIdle,
		},
		{
			name:         "Sleeping with no indicators",
			processState: "S",
			content:      "Some output here",
			expected:     types.ClaudeStateWaitingForInput,
		},
		{
			name:         "Zombie process",
			processState: "Z",
			content:      "",
			expected:     types.ClaudeStateError,
		},
		{
			name:         "Disk wait state",
			processState: "D",
			content:      "Reading file...",
			expected:     types.ClaudeStateRunning,
		},
		{
			name:         "Unknown process state",
			processState: "X",
			content:      "",
			expected:     types.ClaudeStateUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := m.analyzeClaudeState(tt.processState, tt.content)
			if result != tt.expected {
				t.Errorf("analyzeClaudeState(%q, %q) = %q, expected %q",
					tt.processState, tt.content, result, tt.expected)
			}
		})
	}
}

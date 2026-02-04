package fzf

import (
	"strings"
	"testing"
	"time"

	"github.com/mateimicu/tmux-claude-fleet/pkg/types"
)

func TestParseRepoURL(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		expectedSource string
		expectedRepo   string
	}{
		{
			name:           "GitHub HTTPS URL",
			url:            "https://github.com/mateimicu/tmux-claude-fleet",
			expectedSource: "github",
			expectedRepo:   "mateimicu/tmux-claude-fleet",
		},
		{
			name:           "GitHub HTTPS URL with .git",
			url:            "https://github.com/mateimicu/tmux-claude-fleet.git",
			expectedSource: "github",
			expectedRepo:   "mateimicu/tmux-claude-fleet",
		},
		{
			name:           "GitHub SSH URL",
			url:            "git@github.com:mateimicu/tmux-claude-fleet.git",
			expectedSource: "github",
			expectedRepo:   "mateimicu/tmux-claude-fleet",
		},
		{
			name:           "Local path",
			url:            "/home/user/projects/myorg/myrepo",
			expectedSource: "local",
			expectedRepo:   "myorg/myrepo",
		},
		{
			name:           "Local path with trailing slash",
			url:            "/home/user/projects/myorg/myrepo/",
			expectedSource: "local",
			expectedRepo:   "myorg/myrepo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source, repo := parseRepoURL(tt.url)
			if source != tt.expectedSource {
				t.Errorf("parseRepoURL(%q) source = %q, want %q", tt.url, source, tt.expectedSource)
			}
			if repo != tt.expectedRepo {
				t.Errorf("parseRepoURL(%q) repo = %q, want %q", tt.url, repo, tt.expectedRepo)
			}
		})
	}
}

func TestGetClaudeStatusIndicator(t *testing.T) {
	tests := []struct {
		name     string
		state    types.ClaudeState
		expected string
	}{
		{"Running", types.ClaudeStateRunning, "🟢"},
		{"Waiting for input", types.ClaudeStateWaitingForInput, "⏸️"},
		{"Idle", types.ClaudeStateIdle, "💤"},
		{"Error", types.ClaudeStateError, "⚠️"},
		{"Stopped", types.ClaudeStateStopped, "⚫"},
		{"Unknown", types.ClaudeStateUnknown, "❓"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getClaudeStatusIndicator(tt.state)
			if result != tt.expected {
				t.Errorf("getClaudeStatusIndicator(%q) = %q, expected %q",
					tt.state, result, tt.expected)
			}
		})
	}
}

func TestGetClaudeStateDescription(t *testing.T) {
	tests := []struct {
		name     string
		state    types.ClaudeState
		expected string
	}{
		{"Running", types.ClaudeStateRunning, "[Claude: Active]"},
		{"Waiting for input", types.ClaudeStateWaitingForInput, "[Claude: Needs Input]"},
		{"Idle", types.ClaudeStateIdle, "[Claude: Idle]"},
		{"Error", types.ClaudeStateError, "[Claude: Error]"},
		{"Stopped", types.ClaudeStateStopped, "[Claude: Stopped]"},
		{"Unknown", types.ClaudeStateUnknown, "[Claude: Unknown]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getClaudeStateDescription(tt.state)
			if result != tt.expected {
				t.Errorf("getClaudeStateDescription(%q) = %q, expected %q",
					tt.state, result, tt.expected)
			}
		})
	}
}

func TestFormatSessionLine(t *testing.T) {
	tests := []struct {
		name          string
		session       *types.SessionStatus
		sessionNum    int
		totalSessions int
		wantContains  []string
	}{
		{
			name: "GitHub repo, active tmux, Claude running",
			session: &types.SessionStatus{
				Session: &types.Session{
					Name:      "test-session-1",
					RepoURL:   "https://github.com/mateimicu/tmux-claude-fleet",
					CreatedAt: time.Now(),
				},
				TmuxActive:  true,
				ClaudeState: types.ClaudeStateRunning,
			},
			sessionNum:    1,
			totalSessions: 10,
			wantContains:  []string{"🟢", "github:", "mateimicu/tmux-claude-fleet", "01", "[Claude: Active]", "[test-session-1]"},
		},
		{
			name: "Local repo, inactive",
			session: &types.SessionStatus{
				Session: &types.Session{
					Name:      "local-project",
					RepoURL:   "/home/user/projects/myorg/myrepo",
					CreatedAt: time.Now(),
				},
				TmuxActive:  false,
				ClaudeState: types.ClaudeStateStopped,
			},
			sessionNum:    5,
			totalSessions: 10,
			wantContains:  []string{"⚫", "local:", "myorg/myrepo", "05", "[Claude: Stopped]", "[local-project]"},
		},
		{
			name: "Three-digit padding with waiting state",
			session: &types.SessionStatus{
				Session: &types.Session{
					Name:      "project-100",
					RepoURL:   "git@github.com:user/repo.git",
					CreatedAt: time.Now(),
				},
				TmuxActive:  true,
				ClaudeState: types.ClaudeStateWaitingForInput,
			},
			sessionNum:    1,
			totalSessions: 150,
			wantContains:  []string{"🟢", "github:", "user/repo", "001", "⏸️", "[Claude: Needs Input]", "[project-100]"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatSessionLine(tt.session, tt.sessionNum, tt.totalSessions)
			for _, want := range tt.wantContains {
				if !strings.Contains(result, want) {
					t.Errorf("formatSessionLine() = %q, should contain %q", result, want)
				}
			}
		})
	}
}

func TestExtractSessionName(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected string
	}{
		{
			name:     "Extract from GitHub session with Claude state",
			line:     "🟢 github: mateimicu/tmux-claude-fleet - 01 🟢 [Claude: Active] [test-session-1]",
			expected: "test-session-1",
		},
		{
			name:     "Extract from local session",
			line:     "⚫ local: myorg/myrepo - 05 ⚫ [Claude: Stopped] [local-project]",
			expected: "local-project",
		},
		{
			name:     "Extract with waiting state",
			line:     "🟢 github: user/repo - 001 ⏸️ [Claude: Needs Input] [my-session]",
			expected: "my-session",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractSessionName(tt.line)
			if result != tt.expected {
				t.Errorf("extractSessionName(%q) = %q, want %q", tt.line, result, tt.expected)
			}
		})
	}
}

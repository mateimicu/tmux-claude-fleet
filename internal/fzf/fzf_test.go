package fzf

import (
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

func TestFormatSessionLine(t *testing.T) {
	tests := []struct {
		name          string
		session       *types.SessionStatus
		sessionNum    int
		totalSessions int
		expected      string
	}{
		{
			name: "GitHub repo, active tmux, Claude running",
			session: &types.SessionStatus{
				Session: &types.Session{
					Name:      "test-session-1",
					RepoURL:   "https://github.com/mateimicu/tmux-claude-fleet",
					CreatedAt: time.Now(),
				},
				TmuxActive:    true,
				ClaudeRunning: true,
			},
			sessionNum:    1,
			totalSessions: 10,
			expected:      "🟢 github: mateimicu/tmux-claude-fleet - 01 [Claude ✓] [test-session-1]",
		},
		{
			name: "Local repo, inactive",
			session: &types.SessionStatus{
				Session: &types.Session{
					Name:      "local-project",
					RepoURL:   "/home/user/projects/myorg/myrepo",
					CreatedAt: time.Now(),
				},
				TmuxActive:    false,
				ClaudeRunning: false,
			},
			sessionNum:    5,
			totalSessions: 10,
			expected:      "⚫ local: myorg/myrepo - 05 [local-project]",
		},
		{
			name: "Three-digit padding",
			session: &types.SessionStatus{
				Session: &types.Session{
					Name:      "project-100",
					RepoURL:   "git@github.com:user/repo.git",
					CreatedAt: time.Now(),
				},
				TmuxActive:    true,
				ClaudeRunning: false,
			},
			sessionNum:    1,
			totalSessions: 150,
			expected:      "🟢 github: user/repo - 001 [project-100]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatSessionLine(tt.session, tt.sessionNum, tt.totalSessions)
			if result != tt.expected {
				t.Errorf("formatSessionLine() = %q, want %q", result, tt.expected)
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
			name:     "Extract from GitHub session",
			line:     "🟢 github: mateimicu/tmux-claude-fleet - 01 [Claude ✓] [test-session-1]",
			expected: "test-session-1",
		},
		{
			name:     "Extract from local session",
			line:     "⚫ local: myorg/myrepo - 05 [local-project]",
			expected: "local-project",
		},
		{
			name:     "Extract with Claude status",
			line:     "🟢 github: user/repo - 001 [Claude ✓] [my-session]",
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

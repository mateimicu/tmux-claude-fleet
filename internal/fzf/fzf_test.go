package fzf

import (
	"strings"
	"testing"
	"time"

	"github.com/mateimicu/tmux-claude-matrix/pkg/types"
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

func TestGetClaudeStateLabel(t *testing.T) {
	tests := []struct {
		name     string
		state    types.ClaudeState
		expected string
	}{
		{"Running", types.ClaudeStateRunning, "Active"},
		{"Waiting for input", types.ClaudeStateWaitingForInput, "Needs Input"},
		{"Idle", types.ClaudeStateIdle, "Idle"},
		{"Error", types.ClaudeStateError, "Error"},
		{"Stopped", types.ClaudeStateStopped, "Stopped"},
		{"Unknown", types.ClaudeStateUnknown, "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getClaudeStateLabel(tt.state)
			if result != tt.expected {
				t.Errorf("getClaudeStateLabel(%q) = %q, expected %q",
					tt.state, result, tt.expected)
			}
		})
	}
}

func TestFormatSessionTable(t *testing.T) {
	sessions := []*types.SessionStatus{
		{
			Session: &types.Session{
				Name:      "test-session-1",
				RepoURL:   "https://github.com/mateimicu/tmux-claude-fleet",
				CreatedAt: time.Now(),
			},
			TmuxActive:  true,
			ClaudeState: types.ClaudeStateRunning,
		},
		{
			Session: &types.Session{
				Name:      "local-project",
				RepoURL:   "/home/user/projects/myorg/myrepo",
				CreatedAt: time.Now(),
			},
			TmuxActive:  false,
			ClaudeState: types.ClaudeStateStopped,
		},
	}

	header, lines := formatSessionTable(sessions)

	// Header should contain column names
	for _, col := range []string{"#", "TMUX", "SOURCE", "REPOSITORY", "CLAUDE", "SESSION"} {
		if !strings.Contains(header, col) {
			t.Errorf("header %q should contain column name %q", header, col)
		}
	}

	if len(lines) != 2 {
		t.Fatalf("expected 2 data lines, got %d", len(lines))
	}

	// First row: active GitHub session
	row1 := lines[0]
	for _, want := range []string{"1", "🟢", "github", "mateimicu/tmux-claude-fleet", "Active", "[test-session-1]"} {
		if !strings.Contains(row1, want) {
			t.Errorf("row 1 %q should contain %q", row1, want)
		}
	}

	// Second row: inactive local session
	row2 := lines[1]
	for _, want := range []string{"2", "⚫", "local", "myorg/myrepo", "Stopped", "[local-project]"} {
		if !strings.Contains(row2, want) {
			t.Errorf("row 2 %q should contain %q", row2, want)
		}
	}
}

func TestFormatSessionTableAlignment(t *testing.T) {
	sessions := []*types.SessionStatus{
		{
			Session: &types.Session{
				Name:      "short",
				RepoURL:   "https://github.com/a/b",
				CreatedAt: time.Now(),
			},
			TmuxActive:  true,
			ClaudeState: types.ClaudeStateRunning,
		},
		{
			Session: &types.Session{
				Name:      "longer-name",
				RepoURL:   "https://github.com/organization/very-long-repository-name",
				CreatedAt: time.Now(),
			},
			TmuxActive:  false,
			ClaudeState: types.ClaudeStateWaitingForInput,
		},
	}

	header, lines := formatSessionTable(sessions)

	// All lines should have the same display width up to the SESSION column.
	// The SESSION column is the last one and varies in width, so check that
	// the prefix before "[" (the session name bracket) has consistent display width.
	headerPrefixW := displayWidth(header) - displayWidth("SESSION")
	for i, line := range lines {
		bracketIdx := strings.LastIndex(line, "[")
		if bracketIdx < 0 {
			t.Fatalf("line %d missing session name bracket: %q", i, line)
		}
		prefix := line[:bracketIdx]
		prefixW := displayWidth(prefix)
		if prefixW != headerPrefixW {
			t.Errorf("line %d prefix display width = %d, want %d (header width)\nheader: %q\nline:   %q",
				i, prefixW, headerPrefixW, header, line)
		}
	}
}

func TestDisplayWidth(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"ASCII", "hello", 5},
		{"Empty", "", 0},
		{"Green circle emoji", "🟢", 2},
		{"Black circle emoji", "⚫", 2},
		{"Pause with variation selector", "⏸️", 2},
		{"Warning with variation selector", "⚠️", 2},
		{"Emoji plus text", "🟢 Active", 9},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := displayWidth(tt.input)
			if result != tt.expected {
				t.Errorf("displayWidth(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestPadToDisplayWidth(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		width         int
		expectedWidth int
	}{
		{"Pad ASCII", "hi", 6, 6},
		{"Pad emoji", "🟢", 4, 4},
		{"No pad needed", "hello", 3, 5},
		{"Pad emoji with variation selector", "⏸️", 4, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := padToDisplayWidth(tt.input, tt.width)
			resultW := displayWidth(result)
			if resultW != tt.expectedWidth {
				t.Errorf("padToDisplayWidth(%q, %d) display width = %d, want %d (result=%q)",
					tt.input, tt.width, resultW, tt.expectedWidth, result)
			}
		})
	}
}

func TestFormatRepoLine(t *testing.T) {
	tests := []struct {
		name         string
		repo         *types.Repository
		wantContains []string
	}{
		{
			name: "Regular repo with description",
			repo: &types.Repository{
				Source:      "github",
				Name:        "org/repo",
				Description: "A cool repo",
				URL:         "https://github.com/org/repo",
			},
			wantContains: []string{"github:", "org/repo", "A cool repo", "[https://github.com/org/repo]"},
		},
		{
			name: "Regular repo without description",
			repo: &types.Repository{
				Source: "local",
				Name:   "org/repo",
				URL:    "https://github.com/org/repo",
			},
			wantContains: []string{"local:", "org/repo", "[https://github.com/org/repo]"},
		},
		{
			name: "Workspace",
			repo: &types.Repository{
				Source:         "workspace",
				Name:           "my-project",
				Description:    "3 repos",
				IsWorkspace:    true,
				WorkspaceRepos: []string{"a", "b", "c"},
			},
			wantContains: []string{"workspace:", "my-project", "3 repos", "[workspace:my-project]"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatRepoLine(tt.repo)
			for _, want := range tt.wantContains {
				if !strings.Contains(result, want) {
					t.Errorf("formatRepoLine() = %q, should contain %q", result, want)
				}
			}
		})
	}
}

func TestExtractURLWorkspace(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected string
	}{
		{
			name:     "Workspace identifier",
			line:     "workspace: my-project - 3 repos [workspace:my-project]",
			expected: "workspace:my-project",
		},
		{
			name:     "Regular repo URL",
			line:     "github: org/repo - desc [https://github.com/org/repo]",
			expected: "https://github.com/org/repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractURL(tt.line)
			if result != tt.expected {
				t.Errorf("extractURL(%q) = %q, want %q", tt.line, result, tt.expected)
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
			name:     "Extract from table row with active session",
			line:     " 01  🟢  github  mateimicu/tmux-claude-fleet  🟢 Active       [test-session-1]",
			expected: "test-session-1",
		},
		{
			name:     "Extract from table row with inactive session",
			line:     " 05  ⚫  local   myorg/myrepo                 ⚫ Stopped       [local-project]",
			expected: "local-project",
		},
		{
			name:     "Extract from table row with waiting state",
			line:     " 001  🟢  github  user/repo                   ⏸️ Needs Input  [my-session]",
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

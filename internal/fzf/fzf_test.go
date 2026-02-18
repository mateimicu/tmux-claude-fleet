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
		{
			name:           "Workspace URL",
			url:            "workspace:my-project",
			expectedSource: "workspace",
			expectedRepo:   "my-project",
		},
		{
			name:           "Workspace URL with dashes",
			url:            "workspace:my-multi-repo-workspace",
			expectedSource: "workspace",
			expectedRepo:   "my-multi-repo-workspace",
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
		{"Waiting for input", types.ClaudeStateWaitingForInput, "❓"},
		{"Idle", types.ClaudeStateIdle, "💬"},
		{"Error", types.ClaudeStateError, "⚠️"},
		{"Stopped", types.ClaudeStateStopped, "⚫"},
		{"Unknown", types.ClaudeStateUnknown, "❔"},
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
		{"Waiting for input", types.ClaudeStateWaitingForInput, "Waiting"},
		{"Idle", types.ClaudeStateIdle, "Ready"},
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

	// Header should contain column names including TITLE
	for _, col := range []string{"#", "TMUX", "SOURCE", "REPOSITORY", "TITLE", "CLAUDE", "SESSION"} {
		if !strings.Contains(header, col) {
			t.Errorf("header %q should contain column name %q", header, col)
		}
	}

	if len(lines) != 2 {
		t.Fatalf("expected 2 data lines, got %d", len(lines))
	}

	// First row: active GitHub session - REPOSITORY shows orgRepo, TITLE shows session name (no Title set)
	row1 := lines[0]
	for _, want := range []string{"1", "🟢", "github", "mateimicu/tmux-claude-fleet", "test-session-1", "Active", "[test-session-1]"} {
		if !strings.Contains(row1, want) {
			t.Errorf("row 1 %q should contain %q", row1, want)
		}
	}

	// Second row: inactive local session
	row2 := lines[1]
	for _, want := range []string{"2", "⚫", "local", "myorg/myrepo", "local-project", "Stopped", "[local-project]"} {
		if !strings.Contains(row2, want) {
			t.Errorf("row 2 %q should contain %q", row2, want)
		}
	}
}

func TestFormatSessionTableWithTitle(t *testing.T) {
	sessions := []*types.SessionStatus{
		{
			Session: &types.Session{
				Name:      "test-session-1",
				Title:     "my custom title",
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

	_, lines := formatSessionTable(sessions)

	// First row: REPOSITORY shows orgRepo (not title), TITLE shows the custom title
	if !strings.Contains(lines[0], "mateimicu/tmux-claude-fleet") {
		t.Errorf("REPOSITORY column should show orgRepo, got %q", lines[0])
	}
	if !strings.Contains(lines[0], "my custom title") {
		t.Errorf("TITLE column should show custom title, got %q", lines[0])
	}

	// Second row: REPOSITORY shows orgRepo, TITLE falls back to session name
	if !strings.Contains(lines[1], "myorg/myrepo") {
		t.Errorf("REPOSITORY column should show orgRepo, got %q", lines[1])
	}
	if !strings.Contains(lines[1], "local-project") {
		t.Errorf("TITLE column should fall back to session name, got %q", lines[1])
	}
}

func TestFormatSessionTableTitleColumn(t *testing.T) {
	tests := []struct {
		name      string
		title     string
		sessName  string
		wantTitle string
	}{
		{
			name:      "title set shows title in TITLE column",
			title:     "my-feature-work",
			sessName:  "mateimicu-repo-1",
			wantTitle: "my-feature-work",
		},
		{
			name:      "empty title falls back to session name",
			title:     "",
			sessName:  "mateimicu-repo-1",
			wantTitle: "mateimicu-repo-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sessions := []*types.SessionStatus{
				{
					Session: &types.Session{
						Name:      tt.sessName,
						Title:     tt.title,
						RepoURL:   "https://github.com/org/repo",
						CreatedAt: time.Now(),
					},
					TmuxActive:  true,
					ClaudeState: types.ClaudeStateRunning,
				},
			}

			_, lines := formatSessionTable(sessions)
			if len(lines) != 1 {
				t.Fatalf("expected 1 line, got %d", len(lines))
			}

			// REPOSITORY should always show orgRepo
			if !strings.Contains(lines[0], "org/repo") {
				t.Errorf("REPOSITORY column should show orgRepo, got %q", lines[0])
			}

			// TITLE should show expected value
			if !strings.Contains(lines[0], tt.wantTitle) {
				t.Errorf("TITLE column should contain %q, got %q", tt.wantTitle, lines[0])
			}
		})
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

func TestRepoTypeLabel(t *testing.T) {
	tests := []struct {
		name     string
		repo     *types.Repository
		expected string
	}{
		{
			name:     "workspace repo",
			repo:     &types.Repository{Source: "workspace", IsWorkspace: true},
			expected: "📂 workspace",
		},
		{
			name:     "github repo",
			repo:     &types.Repository{Source: "github"},
			expected: "🐙 github",
		},
		{
			name:     "local repo",
			repo:     &types.Repository{Source: "local"},
			expected: "💻 local",
		},
		{
			name:     "unknown source falls back to raw source",
			repo:     &types.Repository{Source: "custom"},
			expected: "custom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := repoTypeLabel(tt.repo)
			if result != tt.expected {
				t.Errorf("repoTypeLabel() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestFormatRepoTable(t *testing.T) {
	repos := []*types.Repository{
		{
			Source:      "github",
			Name:        "org/repo",
			Description: "A cool repo",
			URL:         "https://github.com/org/repo",
		},
		{
			Source: "local",
			Name:   "org/localrepo",
			URL:    "/home/user/org/localrepo",
		},
		{
			Source:         "workspace",
			Name:           "my-project",
			Description:    "3 repos",
			IsWorkspace:    true,
			WorkspaceRepos: []string{"a", "b", "c"},
		},
	}

	header, lines := FormatRepoTable(repos)

	// Header should contain column names
	for _, col := range []string{"TYPE", "ORG/REPO", "DESCRIPTION"} {
		if !strings.Contains(header, col) {
			t.Errorf("header %q should contain column name %q", header, col)
		}
	}

	if len(lines) != 3 {
		t.Fatalf("expected 3 data lines, got %d", len(lines))
	}

	// Row 1: github repo with description
	for _, want := range []string{"🐙 github", "org/repo", "A cool repo", "[https://github.com/org/repo]"} {
		if !strings.Contains(lines[0], want) {
			t.Errorf("row 1 %q should contain %q", lines[0], want)
		}
	}

	// Row 2: local repo without description
	for _, want := range []string{"💻 local", "org/localrepo", "[/home/user/org/localrepo]"} {
		if !strings.Contains(lines[1], want) {
			t.Errorf("row 2 %q should contain %q", lines[1], want)
		}
	}

	// Row 3: workspace
	for _, want := range []string{"📂 workspace", "my-project", "3 repos", "[workspace:my-project]"} {
		if !strings.Contains(lines[2], want) {
			t.Errorf("row 3 %q should contain %q", lines[2], want)
		}
	}
}

func TestFormatRepoTableAlignment(t *testing.T) {
	repos := []*types.Repository{
		{
			Source:      "github",
			Name:        "a/b",
			Description: "short",
			URL:         "https://github.com/a/b",
		},
		{
			Source:      "github",
			Name:        "organization/very-long-repository-name",
			Description: "A much longer description for alignment testing",
			URL:         "https://github.com/organization/very-long-repository-name",
		},
		{
			Source:      "local",
			Name:        "myorg/myrepo",
			Description: "",
			URL:         "/home/user/myorg/myrepo",
		},
	}

	header, lines := FormatRepoTable(repos)

	// All data lines should have the same display width up to the "[identifier]" bracket.
	// The header has no bracket so measure its full width.
	headerW := displayWidth(header)
	for i, line := range lines {
		bracketIdx := strings.LastIndex(line, "[")
		if bracketIdx < 0 {
			t.Fatalf("line %d missing identifier bracket: %q", i, line)
		}
		prefix := line[:bracketIdx]
		prefixW := displayWidth(prefix)
		if prefixW != headerW {
			t.Errorf("line %d prefix display width = %d, want %d (header width)\nheader: %q\nline:   %q",
				i, prefixW, headerW, header, line)
		}
	}
}

func TestFormatRepoTableEmojiIndicators(t *testing.T) {
	repos := []*types.Repository{
		{Source: "github", Name: "org/repo", URL: "https://github.com/org/repo"},
		{Source: "local", Name: "org/local", URL: "/path/to/repo"},
		{Source: "workspace", Name: "ws", IsWorkspace: true},
	}

	_, lines := FormatRepoTable(repos)

	if !strings.Contains(lines[0], "🐙 github") {
		t.Errorf("github repo should show 🐙 github indicator, got %q", lines[0])
	}
	if !strings.Contains(lines[1], "💻 local") {
		t.Errorf("local repo should show 💻 local indicator, got %q", lines[1])
	}
	if !strings.Contains(lines[2], "📂 workspace") {
		t.Errorf("workspace repo should show 📂 workspace indicator, got %q", lines[2])
	}
}

func TestExtractURLWorkspace(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected string
	}{
		{
			name:     "Workspace identifier from table row",
			line:     " 📂 workspace  my-project  3 repos [workspace:my-project]",
			expected: "workspace:my-project",
		},
		{
			name:     "Regular repo URL from table row",
			line:     " 🐙 github  org/repo  A cool repo [https://github.com/org/repo]",
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

func TestParseFZFOutput(t *testing.T) {
	expectedKeys := []string{"ctrl-d"}

	tests := []struct {
		name         string
		output       string
		expectedKeys []string
		wantKey      string
		wantSelected string
		wantErr      bool
	}{
		{
			name:         "Enter pressed - returns selection without key",
			output:       "🟢 github: org/repo - 01 🟢 [Claude: Active] [my-session]\n",
			expectedKeys: expectedKeys,
			wantKey:      "",
			wantSelected: "🟢 github: org/repo - 01 🟢 [Claude: Active] [my-session]",
		},
		{
			name:         "ctrl-d pressed - returns key and selection",
			output:       "ctrl-d\n🟢 github: org/repo - 01 🟢 [Claude: Active] [my-session]\n",
			expectedKeys: expectedKeys,
			wantKey:      "ctrl-d",
			wantSelected: "🟢 github: org/repo - 01 🟢 [Claude: Active] [my-session]",
		},
		{
			name:         "ctrl-d pressed without trailing newline",
			output:       "ctrl-d\n🟢 github: org/repo - 01 🟢 [Claude: Active] [my-session]",
			expectedKeys: expectedKeys,
			wantKey:      "ctrl-d",
			wantSelected: "🟢 github: org/repo - 01 🟢 [Claude: Active] [my-session]",
		},
		{
			name:         "Empty output returns error",
			output:       "",
			expectedKeys: expectedKeys,
			wantErr:      true,
		},
		{
			name:         "Whitespace only output returns error",
			output:       "  \n\n  ",
			expectedKeys: expectedKeys,
			wantErr:      true,
		},
		{
			name:         "ctrl-d with empty selection",
			output:       "ctrl-d\n\n",
			expectedKeys: expectedKeys,
			wantKey:      "ctrl-d",
			wantSelected: "",
		},
		{
			name:         "Selection line starting with key name is not confused as key",
			output:       "ctrl-d-session-name\n",
			expectedKeys: expectedKeys,
			wantKey:      "",
			wantSelected: "ctrl-d-session-name",
		},
		{
			name:         "ctrl-t pressed returns toggle key",
			output:       "ctrl-t\n 01  🟢  github  org/repo  🟢 Active  [my-session]\n",
			expectedKeys: []string{"ctrl-d", "ctrl-t"},
			wantKey:      "ctrl-t",
			wantSelected: "01  🟢  github  org/repo  🟢 Active  [my-session]",
		},
		{
			name:         "ctrl-t with empty selection",
			output:       "ctrl-t\n\n",
			expectedKeys: []string{"ctrl-d", "ctrl-t"},
			wantKey:      "ctrl-t",
			wantSelected: "",
		},
		{
			name:         "ctrl-r pressed returns rename key",
			output:       "ctrl-r\n 01  🟢  github  org/repo  my-session  🟢 Active  [my-session]\n",
			expectedKeys: []string{"ctrl-d", "ctrl-t", "ctrl-r"},
			wantKey:      "ctrl-r",
			wantSelected: "01  🟢  github  org/repo  my-session  🟢 Active  [my-session]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, selected, err := parseFZFOutput(tt.output, tt.expectedKeys)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseFZFOutput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			if key != tt.wantKey {
				t.Errorf("parseFZFOutput() key = %q, want %q", key, tt.wantKey)
			}
			if selected != tt.wantSelected {
				t.Errorf("parseFZFOutput() selected = %q, want %q", selected, tt.wantSelected)
			}
		})
	}
}

func TestSessionActions_NoDuplicateValues(t *testing.T) {
	// Guard against duplicate action values that could cause switch
	// cases to silently fall through to the wrong handler.
	// SelectSession uses a switch on SessionAction; every value must
	// be distinct so that toggle, cancel, delete, and switch are
	// each routed correctly.
	values := map[SessionAction]bool{}
	for _, action := range []SessionAction{
		SessionActionSwitch,
		SessionActionDelete,
		SessionActionCancel,
		SessionActionToggleFilter,
		SessionActionRename,
	} {
		if values[action] {
			t.Errorf("duplicate SessionAction value: %q", action)
		}
		values[action] = true
	}
}

func TestFilterActiveSessions(t *testing.T) {
	tests := []struct {
		name     string
		sessions []*types.SessionStatus
		wantLen  int
	}{
		{
			name:     "empty input",
			sessions: nil,
			wantLen:  0,
		},
		{
			name: "all active",
			sessions: []*types.SessionStatus{
				{Session: &types.Session{Name: "s1"}, TmuxActive: true},
				{Session: &types.Session{Name: "s2"}, TmuxActive: true},
			},
			wantLen: 2,
		},
		{
			name: "all inactive",
			sessions: []*types.SessionStatus{
				{Session: &types.Session{Name: "s1"}, TmuxActive: false},
				{Session: &types.Session{Name: "s2"}, TmuxActive: false},
			},
			wantLen: 0,
		},
		{
			name: "mixed active and inactive",
			sessions: []*types.SessionStatus{
				{Session: &types.Session{Name: "active1"}, TmuxActive: true},
				{Session: &types.Session{Name: "inactive1"}, TmuxActive: false},
				{Session: &types.Session{Name: "active2"}, TmuxActive: true},
			},
			wantLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FilterActiveSessions(tt.sessions)
			if len(result) != tt.wantLen {
				t.Errorf("FilterActiveSessions() returned %d sessions, want %d", len(result), tt.wantLen)
			}
			for _, s := range result {
				if !s.TmuxActive {
					t.Errorf("FilterActiveSessions() returned inactive session %q", s.Session.Name)
				}
			}
		})
	}
}

func TestFilterActiveSessionsPreservesOrder(t *testing.T) {
	sessions := []*types.SessionStatus{
		{Session: &types.Session{Name: "a"}, TmuxActive: true},
		{Session: &types.Session{Name: "b"}, TmuxActive: false},
		{Session: &types.Session{Name: "c"}, TmuxActive: true},
		{Session: &types.Session{Name: "d"}, TmuxActive: false},
		{Session: &types.Session{Name: "e"}, TmuxActive: true},
	}

	result := FilterActiveSessions(sessions)
	if len(result) != 3 {
		t.Fatalf("expected 3 sessions, got %d", len(result))
	}
	wantNames := []string{"a", "c", "e"}
	for i, want := range wantNames {
		if result[i].Session.Name != want {
			t.Errorf("result[%d].Name = %q, want %q", i, result[i].Session.Name, want)
		}
	}
}

func TestSessionLegend(t *testing.T) {
	tests := []struct {
		name           string
		showActiveOnly bool
		wantContains   []string
		wantNotContain []string
	}{
		{
			name:           "default view shows hide inactive hint",
			showActiveOnly: false,
			wantContains:   []string{"ctrl-t: hide inactive", "enter: switch", "ctrl-d: delete", "ctrl-r: rename"},
			wantNotContain: []string{"ctrl-t: show all"},
		},
		{
			name:           "filtered view shows show all hint",
			showActiveOnly: true,
			wantContains:   []string{"ctrl-t: show all", "enter: switch", "ctrl-d: delete", "ctrl-r: rename"},
			wantNotContain: []string{"ctrl-t: hide inactive"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			legend := sessionLegend(tt.showActiveOnly)
			for _, want := range tt.wantContains {
				if !strings.Contains(legend, want) {
					t.Errorf("sessionLegend(%v) should contain %q, got %q", tt.showActiveOnly, want, legend)
				}
			}
			for _, notWant := range tt.wantNotContain {
				if strings.Contains(legend, notWant) {
					t.Errorf("sessionLegend(%v) should NOT contain %q, got %q", tt.showActiveOnly, notWant, legend)
				}
			}
		})
	}
}

func TestSessionLegendAlwaysContainsEmojiLegend(t *testing.T) {
	for _, showActiveOnly := range []bool{true, false} {
		legend := sessionLegend(showActiveOnly)
		for _, want := range []string{"🟢 active", "⚫ inactive", "🟢 Active", "❓ Waiting", "💬 Ready", "⚠️ Error", "⚫ Stopped", "❔ Unknown"} {
			if !strings.Contains(legend, want) {
				t.Errorf("sessionLegend(%v) should always contain %q", showActiveOnly, want)
			}
		}
	}
}

func TestFilterFZFEnv(t *testing.T) {
	env := []string{
		"HOME=/home/user",
		"FZF_DEFAULT_OPTS=--bind ctrl-d:delete-char",
		"PATH=/usr/bin:/bin",
		"FZF_DEFAULT_OPTS_FILE=/home/user/.fzfrc",
		"SHELL=/bin/bash",
	}

	filtered := filterFZFEnv(env)

	for _, e := range filtered {
		if strings.HasPrefix(e, "FZF_DEFAULT_OPTS=") || strings.HasPrefix(e, "FZF_DEFAULT_OPTS_FILE=") {
			t.Errorf("filterFZFEnv() should have removed %q", e)
		}
	}

	if len(filtered) != 3 {
		t.Errorf("filterFZFEnv() returned %d items, want 3", len(filtered))
	}
}

func TestBuildRepoFZFArgs(t *testing.T) {
	t.Run("SimplePath", func(t *testing.T) {
		args := buildRepoFZFArgs("/usr/local/bin/claude-matrix")

		hasReload := false
		hasHeader := false
		hasHeaderLines := false
		for _, arg := range args {
			if strings.Contains(arg, "ctrl-r:reload") {
				hasReload = true
				if !strings.Contains(arg, "/usr/local/bin/claude-matrix") {
					t.Errorf("reload binding should contain binary path, got %q", arg)
				}
				if !strings.Contains(arg, "--force-refresh") {
					t.Errorf("reload binding should contain --force-refresh, got %q", arg)
				}
				if !strings.Contains(arg, "'/usr/local/bin/claude-matrix'") {
					t.Errorf("binary path should be single-quoted, got %q", arg)
				}
			}
			if strings.Contains(arg, "ctrl-r") && strings.Contains(arg, "refresh") && strings.HasPrefix(arg, "--header=") {
				hasHeader = true
			}
			if arg == "--header-lines=1" {
				hasHeaderLines = true
			}
		}
		if !hasReload {
			t.Error("FZF args should contain ctrl-r reload binding")
		}
		if !hasHeader {
			t.Error("FZF header should mention ctrl-r refresh")
		}
		if !hasHeaderLines {
			t.Error("FZF args should contain --header-lines=1 for frozen column header")
		}
	})

	t.Run("PathWithSpaces", func(t *testing.T) {
		args := buildRepoFZFArgs("/Users/First Last/bin/claude-matrix")

		for _, arg := range args {
			if strings.Contains(arg, "ctrl-r:reload") {
				if !strings.Contains(arg, "'/Users/First Last/bin/claude-matrix'") {
					t.Errorf("path with spaces should be single-quoted, got %q", arg)
				}
			}
		}
	})

	t.Run("PathWithSingleQuote", func(t *testing.T) {
		args := buildRepoFZFArgs("/Users/O'Brien/bin/claude-matrix")

		for _, arg := range args {
			if strings.Contains(arg, "ctrl-r:reload") {
				// The quote should be escaped as '\''
				if !strings.Contains(arg, "'/Users/O'\\''Brien/bin/claude-matrix'") {
					t.Errorf("single quote in path should be escaped, got %q", arg)
				}
			}
		}
	})
}

func TestExtractSessionName(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected string
	}{
		{
			name:     "Extract from table row with active session",
			line:     " 01  🟢  github  mateimicu/tmux-claude-fleet  test-session-1  🟢 Active       [test-session-1]",
			expected: "test-session-1",
		},
		{
			name:     "Extract from table row with inactive session",
			line:     " 05  ⚫  local   myorg/myrepo                 local-project  ⚫ Stopped       [local-project]",
			expected: "local-project",
		},
		{
			name:     "Extract from table row with waiting state",
			line:     " 001  🟢  github  user/repo                   my-session  ❓ Waiting  [my-session]",
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

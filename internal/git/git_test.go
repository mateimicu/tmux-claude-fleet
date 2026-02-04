package git

import "testing"

func TestExtractRepoName(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "HTTPS URL",
			url:      "https://github.com/mateimicu/tmux-claude-fleet",
			expected: "mateimicu/tmux-claude-fleet",
		},
		{
			name:     "HTTPS URL with .git",
			url:      "https://github.com/mateimicu/tmux-claude-fleet.git",
			expected: "mateimicu/tmux-claude-fleet",
		},
		{
			name:     "SSH URL",
			url:      "git@github.com:mateimicu/tmux-claude-fleet.git",
			expected: "mateimicu/tmux-claude-fleet",
		},
		{
			name:     "SSH URL without .git",
			url:      "git@github.com:mateimicu/tmux-claude-fleet",
			expected: "mateimicu/tmux-claude-fleet",
		},
		{
			name:     "URL with trailing slash",
			url:      "https://github.com/mateimicu/tmux-claude-fleet/",
			expected: "mateimicu/tmux-claude-fleet",
		},
		{
			name:     "Simple path",
			url:      "/path/to/repo",
			expected: "to/repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractRepoName(tt.url)
			if result != tt.expected {
				t.Errorf("ExtractRepoName(%q) = %q, expected %q", tt.url, result, tt.expected)
			}
		})
	}
}

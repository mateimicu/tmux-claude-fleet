package git

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetMirrorPath(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		cacheDir string
		expected string
	}{
		{
			name:     "HTTPS URL",
			url:      "https://github.com/org/repo.git",
			cacheDir: "/cache",
			expected: "/cache/mirrors/org-repo",
		},
		{
			name:     "SSH URL",
			url:      "git@github.com:org/repo",
			cacheDir: "/cache",
			expected: "/cache/mirrors/org-repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Manager{}
			result := m.GetMirrorPath(tt.url, tt.cacheDir)
			if result != tt.expected {
				t.Errorf("GetMirrorPath(%q, %q) = %q, expected %q", tt.url, tt.cacheDir, result, tt.expected)
			}
		})
	}
}

func TestMirrorExists(t *testing.T) {
	tmpDir := t.TempDir()
	mirrorPath := filepath.Join(tmpDir, "test-mirror")

	m := &Manager{}

	// Should not exist initially
	if m.MirrorExists(mirrorPath) {
		t.Error("MirrorExists should return false for non-existent path")
	}

	// Create a directory
	if err := os.MkdirAll(mirrorPath, 0755); err != nil {
		t.Fatal(err)
	}

	// Should exist now
	if !m.MirrorExists(mirrorPath) {
		t.Error("MirrorExists should return true for existing path")
	}
}

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

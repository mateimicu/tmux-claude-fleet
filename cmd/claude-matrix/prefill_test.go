package main

import (
	"testing"

	"github.com/mateimicu/tmux-claude-matrix/pkg/types"
)

func TestFlattenRepoURLs(t *testing.T) {
	tests := []struct {
		name     string
		repos    []*types.Repository
		expected []string
	}{
		{
			name:     "empty input",
			repos:    nil,
			expected: nil,
		},
		{
			name: "single repo",
			repos: []*types.Repository{
				{URL: "https://github.com/org/repo1"},
			},
			expected: []string{"https://github.com/org/repo1"},
		},
		{
			name: "workspace with sub-repos",
			repos: []*types.Repository{
				{
					IsWorkspace:    true,
					WorkspaceRepos: []string{"https://github.com/org/a", "https://github.com/org/b"},
				},
			},
			expected: []string{"https://github.com/org/a", "https://github.com/org/b"},
		},
		{
			name: "workspace with empty sub-repos falls back to URL",
			repos: []*types.Repository{
				{
					URL:         "https://github.com/org/ws",
					IsWorkspace: true,
				},
			},
			expected: []string{"https://github.com/org/ws"},
		},
		{
			name: "deduplicates URLs",
			repos: []*types.Repository{
				{URL: "https://github.com/org/repo1"},
				{
					IsWorkspace:    true,
					WorkspaceRepos: []string{"https://github.com/org/repo1", "https://github.com/org/repo2"},
				},
			},
			expected: []string{"https://github.com/org/repo1", "https://github.com/org/repo2"},
		},
		{
			name: "skips repos with empty URL",
			repos: []*types.Repository{
				{URL: ""},
				{URL: "https://github.com/org/repo1"},
			},
			expected: []string{"https://github.com/org/repo1"},
		},
		{
			name: "mixed repos and workspaces",
			repos: []*types.Repository{
				{URL: "https://github.com/org/standalone"},
				{
					IsWorkspace:    true,
					WorkspaceRepos: []string{"https://github.com/org/ws-a", "https://github.com/org/standalone"},
				},
				{URL: "https://github.com/org/another"},
			},
			expected: []string{
				"https://github.com/org/standalone",
				"https://github.com/org/ws-a",
				"https://github.com/org/another",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := flattenRepoURLs(tt.repos)
			if len(result) != len(tt.expected) {
				t.Fatalf("flattenRepoURLs() returned %d URLs, want %d\ngot:  %v\nwant: %v",
					len(result), len(tt.expected), result, tt.expected)
			}
			for i, want := range tt.expected {
				if result[i] != want {
					t.Errorf("flattenRepoURLs()[%d] = %q, want %q", i, result[i], want)
				}
			}
		})
	}
}

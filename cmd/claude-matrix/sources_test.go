package main

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/mateimicu/tmux-claude-matrix/pkg/types"
)

func TestBuildSources(t *testing.T) {
	tests := []struct {
		name      string
		cfg       *types.Config
		wantCount int
		wantErr   bool
		wantName  string // checked only when wantCount == 1
		errSubstr string // checked only when wantErr == true
	}{
		{
			name: "local only",
			cfg: &types.Config{
				LocalConfigEnabled: true,
				LocalReposFile:     "/tmp/nonexistent-repos.txt",
			},
			wantCount: 1,
			wantName:  "local",
		},
		{
			name:      "no sources configured",
			cfg:       &types.Config{},
			wantErr:   true,
			errSubstr: "no repository sources configured",
		},
		{
			name: "workspaces enabled",
			cfg: &types.Config{
				WorkspacesEnabled: true,
				WorkspacesFile:    "/tmp/nonexistent-workspaces.yaml",
			},
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sources, err := buildSources(context.Background(), tt.cfg, io.Discard)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errSubstr != "" && !strings.Contains(err.Error(), tt.errSubstr) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(sources) != tt.wantCount {
				t.Fatalf("expected %d source(s), got %d", tt.wantCount, len(sources))
			}
			if tt.wantName != "" && sources[0].Name() != tt.wantName {
				t.Errorf("expected source name %q, got %q", tt.wantName, sources[0].Name())
			}
		})
	}

	// Separate test: GitHub enabled but no token should return a specific error.
	// This requires isolating from the host environment's gh CLI auth.
	t.Run("GitHub enabled but no token returns specific error", func(t *testing.T) {
		t.Setenv("GITHUB_TOKEN", "")
		t.Setenv("GH_TOKEN", "")
		t.Setenv("PATH", "") // prevent gh CLI discovery

		cfg := &types.Config{
			GitHubEnabled: true,
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // immediately cancel so gh CLI subprocess can't run

		_, err := buildSources(ctx, cfg, io.Discard)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "no authentication token found") {
			t.Errorf("error %q should contain 'no authentication token found'", err.Error())
		}
	})
}

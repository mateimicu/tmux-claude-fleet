package main

import (
	"context"
	"io"
	"testing"

	"github.com/mateimicu/tmux-claude-matrix/internal/logging"
	"github.com/mateimicu/tmux-claude-matrix/pkg/types"
)

func TestBuildSources_LocalOnly(t *testing.T) {
	cfg := &types.Config{
		LocalConfigEnabled: true,
		LocalReposFile:     "/tmp/nonexistent-repos.txt",
		GitHubEnabled:      false,
		WorkspacesEnabled:  false,
	}

	log := &logging.Logger{DebugW: io.Discard, WarnW: io.Discard}
	sources, err := buildSources(context.Background(), cfg, log)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(sources))
	}

	if sources[0].Name() != "local" {
		t.Errorf("expected source name 'local', got '%s'", sources[0].Name())
	}
}

func TestBuildSources_NoSourcesConfigured(t *testing.T) {
	cfg := &types.Config{
		LocalConfigEnabled: false,
		GitHubEnabled:      false,
		WorkspacesEnabled:  false,
	}

	log := &logging.Logger{DebugW: io.Discard, WarnW: io.Discard}
	_, err := buildSources(context.Background(), cfg, log)
	if err == nil {
		t.Fatal("expected error when no sources configured")
	}
}

func TestBuildSources_WorkspacesEnabled(t *testing.T) {
	cfg := &types.Config{
		WorkspacesEnabled:  true,
		WorkspacesFile:     "/tmp/nonexistent-workspaces.yaml",
		LocalConfigEnabled: false,
		GitHubEnabled:      false,
	}

	log := &logging.Logger{DebugW: io.Discard, WarnW: io.Discard}
	sources, err := buildSources(context.Background(), cfg, log)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(sources))
	}
}

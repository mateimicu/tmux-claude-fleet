package repos

import (
	"context"
	"testing"
	"time"
)

func TestGetGitHubToken_EnvVarBypassesGH(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "test-token-123")

	start := time.Now()
	token, source := GetGitHubToken(context.Background())
	elapsed := time.Since(start)

	if token != "test-token-123" {
		t.Errorf("Expected token 'test-token-123', got %q", token)
	}
	if source != "environment variable" {
		t.Errorf("Expected source 'environment variable', got %q", source)
	}
	if elapsed > 100*time.Millisecond {
		t.Errorf("Env var path should be fast, took %v", elapsed)
	}
}

func TestGetGitHubToken_ContextPassthrough(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	token, source := GetGitHubToken(ctx)

	if token != "" {
		t.Errorf("Expected empty token with cancelled context, got %q", token)
	}
	if source != "" {
		t.Errorf("Expected empty source with cancelled context, got %q", source)
	}
}

func TestGetGHToken_RespectsContextCancellation(t *testing.T) {
	if !commandExists("gh") {
		t.Skip("gh CLI not installed; context cancellation cannot reach exec.CommandContext path")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	_, err := getGHToken(ctx)
	elapsed := time.Since(start)

	if err == nil {
		t.Error("Expected error with cancelled context, got nil")
	}
	if elapsed > 500*time.Millisecond {
		t.Errorf("Cancelled context should fail fast, took %v", elapsed)
	}
}

package repos

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// GetGitHubToken returns a GitHub token from either:
// 1. GITHUB_TOKEN environment variable
// 2. gh CLI (if installed and authenticated)
func GetGitHubToken(ctx context.Context) (string, string) {
	// First, check environment variable
	token := os.Getenv("GITHUB_TOKEN")
	if token != "" {
		return token, "environment variable"
	}

	// Second, try gh CLI
	token, err := getGHToken(ctx)
	if err == nil && token != "" {
		return token, "gh CLI"
	}

	return "", ""
}

// getGHToken gets the token from gh CLI
func getGHToken(ctx context.Context) (string, error) {
	// Check if gh is installed
	if !commandExists("gh") {
		return "", fmt.Errorf("gh not installed")
	}

	// Get token from gh with a 3-second timeout
	tokenCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	cmd := exec.CommandContext(tokenCtx, "gh", "auth", "token")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	token := strings.TrimSpace(string(output))
	if token == "" {
		return "", fmt.Errorf("gh not authenticated")
	}

	return token, nil
}

// commandExists checks if a command is available in PATH
func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

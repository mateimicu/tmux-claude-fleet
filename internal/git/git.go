package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Manager struct{}

func New() *Manager {
	return &Manager{}
}

// Clone clones a repository to the specified path
func (m *Manager) Clone(url, path string) error {
	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	cmd := exec.Command("git", "clone", url, path)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// ExtractRepoName extracts org/repo from a git URL
func ExtractRepoName(url string) string {
	// Remove .git suffix
	clean := strings.TrimSuffix(url, ".git")

	// Remove trailing slash
	clean = strings.TrimSuffix(clean, "/")

	// Handle SSH URLs (git@github.com:org/repo)
	if strings.Contains(clean, ":") && strings.Contains(clean, "@") {
		parts := strings.Split(clean, ":")
		if len(parts) >= 2 {
			clean = parts[len(parts)-1]
		}
	}

	// Extract last two path components
	parts := strings.Split(clean, "/")
	if len(parts) >= 2 {
		return parts[len(parts)-2] + "/" + parts[len(parts)-1]
	}

	return filepath.Base(clean)
}

package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Manager handles git operations
type Manager struct{}

// New creates a new git Manager
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

// CloneWithCache clones a repository using a local mirror cache for faster cloning
func (m *Manager) CloneWithCache(url, path, cacheDir string) error {
	mirrorPath := m.GetMirrorPath(url, cacheDir)

	if !m.MirrorExists(mirrorPath) {
		// Create new mirror
		if err := m.createMirror(url, mirrorPath); err != nil {
			return err
		}
	} else {
		// Update existing mirror with latest commits
		if err := m.updateMirror(mirrorPath); err != nil {
			return err
		}
	}

	// Clone using the mirror as reference
	return m.cloneWithReference(url, path, mirrorPath)
}

// GetMirrorPath returns the path where the mirror cache should be stored
func (m *Manager) GetMirrorPath(url, cacheDir string) string {
	// Extract org/repo and convert to filesystem-safe name
	repoName := ExtractRepoName(url)
	safeName := strings.ReplaceAll(repoName, "/", "-")
	return filepath.Join(cacheDir, "mirrors", safeName)
}

// MirrorExists checks if a mirror cache exists at the given path
func (m *Manager) MirrorExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// createMirror creates a new mirror cache of the repository
func (m *Manager) createMirror(url, path string) error {
	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	cmd := exec.Command("git", "clone", "--mirror", url, path)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// updateMirror fetches the latest objects into an existing mirror
func (m *Manager) updateMirror(path string) error {
	cmd := exec.Command("git", "-C", path, "fetch", "--prune")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// cloneWithReference clones using an existing mirror as reference
func (m *Manager) cloneWithReference(url, path, reference string) error {
	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	cmd := exec.Command("git", "clone", "--reference", reference, "--dissociate", url, path)
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

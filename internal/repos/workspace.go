package repos

import (
	"context"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/mateimicu/tmux-claude-matrix/pkg/types"
)

// WorkspaceSource discovers workspaces from a YAML config file
type WorkspaceSource struct {
	filePath string
}

// workspacesFile is the top-level structure of workspaces.yaml
type workspacesFile struct {
	Workspaces map[string]workspaceEntry `yaml:"workspaces"`
}

// workspaceEntry is a single workspace definition
type workspaceEntry struct {
	Repos       []string `yaml:"repos"`
	Description string   `yaml:"description"`
}

// NewWorkspaceSource creates a new workspace source
func NewWorkspaceSource(filePath string) *WorkspaceSource {
	return &WorkspaceSource{filePath: filePath}
}

// Name returns the source name
func (w *WorkspaceSource) Name() string {
	return "workspace"
}

// List returns all workspaces as Repository entries
func (w *WorkspaceSource) List(_ context.Context) ([]*types.Repository, error) {
	data, err := os.ReadFile(w.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read workspaces file: %w", err)
	}

	var file workspacesFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("failed to parse workspaces file: %w", err)
	}

	var repos []*types.Repository
	for name, entry := range file.Workspaces {
		if len(entry.Repos) == 0 {
			continue
		}

		desc := entry.Description
		if desc == "" {
			desc = fmt.Sprintf("%d repos", len(entry.Repos))
		}

		repos = append(repos, &types.Repository{
			Source:         "workspace",
			URL:            "", // Workspaces don't have a single URL
			Name:           name,
			Description:    desc,
			IsWorkspace:    true,
			WorkspaceRepos: entry.Repos,
		})
	}

	return repos, nil
}

// ParseWorkspacesFile parses a workspaces YAML file and returns the entries.
// Exported for testing.
func ParseWorkspacesFile(data []byte) (map[string][]string, error) {
	var file workspacesFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		return nil, err
	}

	result := make(map[string][]string, len(file.Workspaces))
	for name, entry := range file.Workspaces {
		result[name] = entry.Repos
	}

	return result, nil
}

// FormatWorkspaceRepoList returns a human-readable list of repos in a workspace
func FormatWorkspaceRepoList(urls []string) string {
	var names []string
	for _, url := range urls {
		// Extract short name from URL
		name := url
		// Remove .git suffix
		name = strings.TrimSuffix(name, ".git")
		// Get last path component
		parts := strings.Split(name, "/")
		if len(parts) > 0 {
			name = parts[len(parts)-1]
		}
		names = append(names, name)
	}
	return strings.Join(names, ", ")
}

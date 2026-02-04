package types

import "time"

// Repository represents a discovered repository
type Repository struct {
	Source      string `json:"source"`      // "local", "github"
	URL         string `json:"url"`         // Clone URL
	Name        string `json:"name"`        // Display name (org/repo)
	Description string `json:"description"` // Optional description
}

// Session represents a tmux session managed by fleet
type Session struct {
	CreatedAt time.Time `json:"created_at"`
	Name      string    `json:"name"`
	RepoURL   string    `json:"repo_url"`
	ClonePath string    `json:"clone_path"`
}

// SessionStatus represents runtime session information
type SessionStatus struct {
	Session       *Session
	TmuxActive    bool
	ClaudeRunning bool
}

// Config represents plugin configuration
type Config struct {
	GitHubOrgs         []string // Filter repos by organization
	ClaudeArgs         []string
	CloneDir           string
	LocalReposFile     string
	ClaudeBin          string
	CacheDir           string
	SessionsDir        string
	CacheTTL           time.Duration
	GitHubEnabled      bool
	LocalConfigEnabled bool
}

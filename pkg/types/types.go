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
	Name      string    `json:"name"`
	RepoURL   string    `json:"repo_url"`
	ClonePath string    `json:"clone_path"`
	CreatedAt time.Time `json:"created_at"`
}

// SessionStatus represents runtime session information
type SessionStatus struct {
	Session       *Session
	TmuxActive    bool
	ClaudeRunning bool
}

// Config represents plugin configuration
type Config struct {
	CloneDir           string
	GitHubEnabled      bool
	GitHubOrgs         []string // Filter repos by organization
	LocalConfigEnabled bool
	LocalReposFile     string
	ClaudeBin          string
	ClaudeArgs         []string
	CacheDir           string
	CacheTTL           time.Duration
	SessionsDir        string
}

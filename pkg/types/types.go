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

// ClaudeState represents the detailed state of a Claude process
type ClaudeState string

const (
	// ClaudeStateUnknown indicates state cannot be determined
	ClaudeStateUnknown ClaudeState = "unknown"
	// ClaudeStateStopped indicates Claude is not running
	ClaudeStateStopped ClaudeState = "stopped"
	// ClaudeStateRunning indicates Claude is actively processing
	ClaudeStateRunning ClaudeState = "running"
	// ClaudeStateWaitingForInput indicates Claude is waiting for user input
	ClaudeStateWaitingForInput ClaudeState = "waiting_for_input"
	// ClaudeStateIdle indicates Claude finished and is idle
	ClaudeStateIdle ClaudeState = "idle"
	// ClaudeStateError indicates Claude encountered an error
	ClaudeStateError ClaudeState = "error"
)

// SessionStatus represents runtime session information
type SessionStatus struct {
	Session       *Session
	TmuxActive    bool
	ClaudeRunning bool
	ClaudeState   ClaudeState
	LastActivity  time.Time
}

// Config represents plugin configuration
type Config struct {
	CloneDir           string
	LocalReposFile     string
	ClaudeBin          string
	CacheDir           string
	SessionsDir        string
	GitHubOrgs         []string
	ClaudeArgs         []string
	CacheTTL           time.Duration
	GitHubEnabled      bool
	LocalConfigEnabled bool
}

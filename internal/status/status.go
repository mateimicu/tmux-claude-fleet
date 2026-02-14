package status

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/mateimicu/tmux-claude-matrix/pkg/types"
)

// StateFile represents the persisted state of a Claude Code session.
type StateFile struct {
	State     string    `json:"state"`
	UpdatedAt time.Time `json:"updated_at"`
	SessionID string    `json:"session_id,omitempty"`
}

// DefaultStatusDir returns the default directory for state files.
func DefaultStatusDir() string {
	return filepath.Join(os.Getenv("HOME"), ".tmux-claude-matrix/status")
}

// WriteState atomically writes a state file for the given session.
// It creates statusDir if it doesn't exist, writes to a temp file, then renames.
func WriteState(statusDir, sessionName string, state types.ClaudeState, claudeSessionID string) error {
	if err := os.MkdirAll(statusDir, 0o755); err != nil {
		return err
	}

	sf := StateFile{
		State:     string(state),
		UpdatedAt: time.Now(),
		SessionID: claudeSessionID,
	}

	data, err := json.Marshal(sf)
	if err != nil {
		return err
	}

	target := stateFilePath(statusDir, sessionName)

	tmpFile, err := os.CreateTemp(statusDir, sessionName+"*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()    //nolint:errcheck // Best-effort cleanup on write failure
		os.Remove(tmpPath) //nolint:errcheck // Best-effort cleanup on write failure
		return err
	}
	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpPath) //nolint:errcheck // Best-effort cleanup on close failure
		return err
	}
	if err := os.Rename(tmpPath, target); err != nil {
		os.Remove(tmpPath) //nolint:errcheck // Best-effort cleanup on rename failure
		return err
	}
	return nil
}

// ReadState reads and parses the state file for the given session.
func ReadState(statusDir, sessionName string) (*StateFile, error) {
	data, err := os.ReadFile(stateFilePath(statusDir, sessionName))
	if err != nil {
		return nil, err
	}

	var sf StateFile
	if err := json.Unmarshal(data, &sf); err != nil {
		return nil, err
	}

	return &sf, nil
}

// RemoveState deletes the state file for the given session. Returns nil if the file doesn't exist.
func RemoveState(statusDir, sessionName string) error {
	err := os.Remove(stateFilePath(statusDir, sessionName))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

// IsStale returns true if the state file's UpdatedAt exceeds maxAge.
func IsStale(sf *StateFile, maxAge time.Duration) bool {
	return time.Since(sf.UpdatedAt) > maxAge
}

// EmojiForState maps a ClaudeState to a display emoji.
func EmojiForState(state types.ClaudeState) string {
	switch state {
	case types.ClaudeStateRunning:
		return "\U0001f7e2" // green circle
	case types.ClaudeStateWaitingForInput:
		return "\u2753" // question mark
	case types.ClaudeStateIdle:
		return "\U0001f4ac" // speech balloon
	case types.ClaudeStateStopped:
		return "\u26ab" // black circle
	case types.ClaudeStateError:
		return "\u26a0\ufe0f" // warning sign
	case types.ClaudeStateUnknown:
		return "\u2754" // white question mark (unknown)
	default:
		return "\u2754" // white question mark (unknown)
	}
}

func stateFilePath(statusDir, sessionName string) string {
	return filepath.Join(statusDir, sessionName+".state")
}

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

// DefaultStaleThreshold is the duration after which an agent state file is considered stale.
const DefaultStaleThreshold = 10 * time.Minute

// WriteState atomically writes a state file for the given session.
// It creates statusDir if it doesn't exist, writes to a temp file, then renames.
func WriteState(statusDir, sessionName string, state types.ClaudeState, claudeSessionID string) error {
	sf := StateFile{
		State:     string(state),
		UpdatedAt: time.Now(),
		SessionID: claudeSessionID,
	}
	return atomicWriteJSON(statusDir, stateFilePath(statusDir, sessionName), sf)
}

// ReadState reads and parses the state file for the given session.
func ReadState(statusDir, sessionName string) (*StateFile, error) {
	return readStateFromPath(stateFilePath(statusDir, sessionName))
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

// StatePriority returns the attention-priority rank for a ClaudeState.
// Higher values indicate states that need more attention.
func StatePriority(state types.ClaudeState) int {
	switch state {
	case types.ClaudeStateError:
		return 6
	case types.ClaudeStateWaitingForInput:
		return 5
	case types.ClaudeStateRunning:
		return 4
	case types.ClaudeStateIdle:
		return 3
	case types.ClaudeStateStopped:
		return 2
	case types.ClaudeStateUnknown:
		return 1
	default:
		return 0
	}
}

// WriteAgentState atomically writes a per-agent state file.
func WriteAgentState(statusDir, sessionName, agentSessionID string, state types.ClaudeState) error {
	sf := StateFile{
		State:     string(state),
		UpdatedAt: time.Now(),
		SessionID: agentSessionID,
	}
	return atomicWriteJSON(statusDir, agentStateFilePath(statusDir, sessionName, agentSessionID), sf)
}

// ReadAgentState reads a per-agent state file.
func ReadAgentState(statusDir, sessionName, agentSessionID string) (*StateFile, error) {
	return readStateFromPath(agentStateFilePath(statusDir, sessionName, agentSessionID))
}

// RemoveAgentState deletes a per-agent state file. Returns nil if it doesn't exist.
func RemoveAgentState(statusDir, sessionName, agentSessionID string) error {
	err := os.Remove(agentStateFilePath(statusDir, sessionName, agentSessionID))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

// RemoveAllAgentStates deletes all per-agent state files for a session.
func RemoveAllAgentStates(statusDir, sessionName string) error {
	files, err := listAgentStateFiles(statusDir, sessionName)
	if err != nil {
		return err
	}
	for _, f := range files {
		os.Remove(f) //nolint:errcheck // Best-effort cleanup
	}
	return nil
}

// UpdateAggregate recomputes the aggregate state from all per-agent files,
// cleans up stale agent files, and writes the aggregate {sessionName}.state file.
// Returns the computed aggregate state.
func UpdateAggregate(statusDir, sessionName string, staleThreshold time.Duration) (types.ClaudeState, error) {
	files, err := listAgentStateFiles(statusDir, sessionName)
	if err != nil {
		return types.ClaudeStateStopped, err
	}

	bestState := types.ClaudeStateStopped

	for _, f := range files {
		sf, readErr := readStateFromPath(f)
		if readErr != nil {
			os.Remove(f) //nolint:errcheck // Best-effort cleanup of unreadable file
			continue
		}
		if IsStale(sf, staleThreshold) {
			os.Remove(f) //nolint:errcheck // Best-effort cleanup of stale file
			continue
		}
		state := types.ClaudeState(sf.State)
		if StatePriority(state) > StatePriority(bestState) {
			bestState = state
		}
	}

	if bestState == types.ClaudeStateStopped {
		return bestState, RemoveState(statusDir, sessionName)
	}
	return bestState, WriteState(statusDir, sessionName, bestState, "")
}

func stateFilePath(statusDir, sessionName string) string {
	return filepath.Join(statusDir, sessionName+".state")
}

func agentStateFilePath(statusDir, sessionName, agentSessionID string) string {
	safe := sanitizeAgentID(agentSessionID)
	return filepath.Join(statusDir, sessionName+".agent."+safe+".state")
}

// sanitizeAgentID strips path components to prevent directory traversal.
func sanitizeAgentID(id string) string {
	id = filepath.Base(id)
	if id == "." || id == ".." || id == "" {
		return "_"
	}
	return id
}

func listAgentStateFiles(statusDir, sessionName string) ([]string, error) {
	pattern := filepath.Join(statusDir, sessionName+".agent.*.state")
	return filepath.Glob(pattern)
}

func readStateFromPath(path string) (*StateFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var sf StateFile
	if err := json.Unmarshal(data, &sf); err != nil {
		return nil, err
	}
	return &sf, nil
}

// atomicWriteJSON atomically writes JSON data to targetPath via temp file + rename.
func atomicWriteJSON(statusDir, targetPath string, v any) error {
	if err := os.MkdirAll(statusDir, 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	tmpFile, err := os.CreateTemp(statusDir, "*.tmp")
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
	if err := os.Rename(tmpPath, targetPath); err != nil {
		os.Remove(tmpPath) //nolint:errcheck // Best-effort cleanup on rename failure
		return err
	}
	return nil
}

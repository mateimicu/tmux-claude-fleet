package status

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mateimicu/tmux-claude-matrix/pkg/types"
)

func TestWriteAndReadState(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "status-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	sessionName := "test-session"
	state := types.ClaudeStateRunning
	claudeSessionID := "sess-abc-123"

	if err := WriteState(tmpDir, sessionName, state, claudeSessionID); err != nil {
		t.Fatalf("WriteState failed: %v", err)
	}

	sf, err := ReadState(tmpDir, sessionName)
	if err != nil {
		t.Fatalf("ReadState failed: %v", err)
	}

	if sf.State != string(state) {
		t.Errorf("State = %q, want %q", sf.State, state)
	}
	if sf.SessionID != claudeSessionID {
		t.Errorf("SessionID = %q, want %q", sf.SessionID, claudeSessionID)
	}
	if time.Since(sf.UpdatedAt) > 5*time.Second {
		t.Errorf("UpdatedAt too old: %v", sf.UpdatedAt)
	}
}

func TestReadState_NotFound(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "status-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	_, err = ReadState(tmpDir, "nonexistent")
	if err == nil {
		t.Fatal("expected error for missing state file, got nil")
	}
}

func TestRemoveState(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "status-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Write a state file, then remove it
	if err := WriteState(tmpDir, "to-remove", types.ClaudeStateIdle, ""); err != nil {
		t.Fatalf("WriteState failed: %v", err)
	}

	if err := RemoveState(tmpDir, "to-remove"); err != nil {
		t.Fatalf("RemoveState failed: %v", err)
	}

	// Verify it's gone
	path := filepath.Join(tmpDir, "to-remove.state")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("expected file to be removed, stat err: %v", err)
	}

	// Removing a non-existent file should not error
	if err := RemoveState(tmpDir, "never-existed"); err != nil {
		t.Errorf("RemoveState on non-existent file should not error, got: %v", err)
	}
}

func TestIsStale(t *testing.T) {
	threshold := 5 * time.Minute

	t.Run("fresh entry is not stale", func(t *testing.T) {
		sf := &StateFile{UpdatedAt: time.Now().Add(-1 * time.Second)}
		if IsStale(sf, threshold) {
			t.Error("expected fresh entry to not be stale")
		}
	})

	t.Run("old entry is stale", func(t *testing.T) {
		sf := &StateFile{UpdatedAt: time.Now().Add(-10 * time.Minute)}
		if !IsStale(sf, threshold) {
			t.Error("expected old entry to be stale")
		}
	})
}

func TestEmojiForState(t *testing.T) {
	tests := []struct {
		state types.ClaudeState
		emoji string
	}{
		{types.ClaudeStateRunning, "\U0001f7e2"},
		{types.ClaudeStateWaitingForInput, "\u2753"},
		{types.ClaudeStateIdle, "\U0001f4ac"},
		{types.ClaudeStateStopped, "\u26ab"},
		{types.ClaudeStateError, "\u26a0\ufe0f"},
		{types.ClaudeStateUnknown, "\u2754"},
		{types.ClaudeState("something-else"), "\u2754"},
	}

	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			got := EmojiForState(tt.state)
			if got != tt.emoji {
				t.Errorf("EmojiForState(%q) = %q, want %q", tt.state, got, tt.emoji)
			}
		})
	}
}

func TestWriteState_AtomicCreate(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "status-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Use a nested path that doesn't exist yet
	statusDir := filepath.Join(tmpDir, "nested", "status")

	if err := WriteState(statusDir, "auto-created", types.ClaudeStateRunning, ""); err != nil {
		t.Fatalf("WriteState should create directory, got: %v", err)
	}

	// Verify the file exists and is valid JSON
	data, err := os.ReadFile(filepath.Join(statusDir, "auto-created.state"))
	if err != nil {
		t.Fatalf("failed to read state file: %v", err)
	}

	var sf StateFile
	if err := json.Unmarshal(data, &sf); err != nil {
		t.Fatalf("state file is not valid JSON: %v", err)
	}

	if sf.State != string(types.ClaudeStateRunning) {
		t.Errorf("State = %q, want %q", sf.State, types.ClaudeStateRunning)
	}
}

func TestStatePriority(t *testing.T) {
	// error > waiting_for_input > running > idle > stopped > unknown
	ordered := []types.ClaudeState{
		types.ClaudeStateUnknown,
		types.ClaudeStateStopped,
		types.ClaudeStateIdle,
		types.ClaudeStateRunning,
		types.ClaudeStateWaitingForInput,
		types.ClaudeStateError,
	}

	for i := 1; i < len(ordered); i++ {
		if StatePriority(ordered[i]) <= StatePriority(ordered[i-1]) {
			t.Errorf("expected %q priority > %q priority, got %d <= %d",
				ordered[i], ordered[i-1],
				StatePriority(ordered[i]), StatePriority(ordered[i-1]))
		}
	}
}

func TestWriteAndReadAgentState(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "status-agent-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	sessionName := "my-session"
	agentID := "sess-abc-123"
	state := types.ClaudeStateRunning

	if err := WriteAgentState(tmpDir, sessionName, agentID, state); err != nil {
		t.Fatalf("WriteAgentState failed: %v", err)
	}

	sf, err := ReadAgentState(tmpDir, sessionName, agentID)
	if err != nil {
		t.Fatalf("ReadAgentState failed: %v", err)
	}

	if sf.State != string(state) {
		t.Errorf("State = %q, want %q", sf.State, state)
	}
	if sf.SessionID != agentID {
		t.Errorf("SessionID = %q, want %q", sf.SessionID, agentID)
	}

	// Verify file is at the expected path
	expected := filepath.Join(tmpDir, sessionName+".agent."+agentID+".state")
	if _, err := os.Stat(expected); err != nil {
		t.Errorf("expected agent state file at %s, got err: %v", expected, err)
	}
}

func TestRemoveAgentState(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "status-agent-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	sessionName := "my-session"
	agentID := "sess-abc-123"

	if err := WriteAgentState(tmpDir, sessionName, agentID, types.ClaudeStateRunning); err != nil {
		t.Fatal(err)
	}

	if err := RemoveAgentState(tmpDir, sessionName, agentID); err != nil {
		t.Fatalf("RemoveAgentState failed: %v", err)
	}

	// File should be gone
	_, err = ReadAgentState(tmpDir, sessionName, agentID)
	if err == nil {
		t.Error("expected error reading removed agent state, got nil")
	}

	// Removing non-existent agent should not error
	if err := RemoveAgentState(tmpDir, sessionName, "nonexistent"); err != nil {
		t.Errorf("RemoveAgentState on non-existent should not error, got: %v", err)
	}
}

func TestUpdateAggregate_SingleAgent(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "status-agg-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	sessionName := "my-session"
	if err := WriteAgentState(tmpDir, sessionName, "agent-1", types.ClaudeStateRunning); err != nil {
		t.Fatal(err)
	}

	aggState, err := UpdateAggregate(tmpDir, sessionName, DefaultStaleThreshold)
	if err != nil {
		t.Fatalf("UpdateAggregate failed: %v", err)
	}
	if aggState != types.ClaudeStateRunning {
		t.Errorf("aggregate = %q, want %q", aggState, types.ClaudeStateRunning)
	}

	// Verify aggregate file was written
	sf, err := ReadState(tmpDir, sessionName)
	if err != nil {
		t.Fatalf("ReadState (aggregate) failed: %v", err)
	}
	if sf.State != string(types.ClaudeStateRunning) {
		t.Errorf("aggregate file state = %q, want %q", sf.State, types.ClaudeStateRunning)
	}
}

func TestUpdateAggregate_HighestPriorityWins(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "status-agg-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	sessionName := "my-session"

	// Agent 1 is idle, Agent 2 is running, Agent 3 is waiting_for_input
	for _, tc := range []struct {
		id    string
		state types.ClaudeState
	}{
		{"agent-1", types.ClaudeStateIdle},
		{"agent-2", types.ClaudeStateRunning},
		{"agent-3", types.ClaudeStateWaitingForInput},
	} {
		if err := WriteAgentState(tmpDir, sessionName, tc.id, tc.state); err != nil {
			t.Fatal(err)
		}
	}

	aggState, err := UpdateAggregate(tmpDir, sessionName, DefaultStaleThreshold)
	if err != nil {
		t.Fatalf("UpdateAggregate failed: %v", err)
	}

	// waiting_for_input has highest priority among these three
	if aggState != types.ClaudeStateWaitingForInput {
		t.Errorf("aggregate = %q, want %q", aggState, types.ClaudeStateWaitingForInput)
	}
}

func TestUpdateAggregate_NoAgents_ReturnsStopped(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "status-agg-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	aggState, err := UpdateAggregate(tmpDir, "empty-session", DefaultStaleThreshold)
	if err != nil {
		t.Fatalf("UpdateAggregate failed: %v", err)
	}
	if aggState != types.ClaudeStateStopped {
		t.Errorf("aggregate = %q, want %q", aggState, types.ClaudeStateStopped)
	}

	// Aggregate file should not exist
	_, err = ReadState(tmpDir, "empty-session")
	if err == nil {
		t.Error("expected no aggregate file for empty session")
	}
}

func TestUpdateAggregate_StaleAgentsIgnored(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "status-agg-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	sessionName := "my-session"

	// Write a stale agent file manually (updated 15 minutes ago)
	staleSF := StateFile{
		State:     string(types.ClaudeStateError),
		UpdatedAt: time.Now().Add(-15 * time.Minute),
		SessionID: "stale-agent",
	}
	staleData, _ := json.Marshal(staleSF)
	stalePath := filepath.Join(tmpDir, sessionName+".agent.stale-agent.state")
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stalePath, staleData, 0o644); err != nil {
		t.Fatal(err)
	}

	// Write a fresh agent
	if err := WriteAgentState(tmpDir, sessionName, "fresh-agent", types.ClaudeStateIdle); err != nil {
		t.Fatal(err)
	}

	aggState, err := UpdateAggregate(tmpDir, sessionName, DefaultStaleThreshold)
	if err != nil {
		t.Fatalf("UpdateAggregate failed: %v", err)
	}

	// Stale error agent should be ignored; only fresh idle agent counts
	if aggState != types.ClaudeStateIdle {
		t.Errorf("aggregate = %q, want %q", aggState, types.ClaudeStateIdle)
	}

	// Stale file should have been removed
	if _, err := os.Stat(stalePath); !os.IsNotExist(err) {
		t.Error("expected stale agent file to be removed")
	}
}

func TestUpdateAggregate_AllStale_ReturnsStopped(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "status-agg-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	sessionName := "my-session"

	// Write only stale agent files
	staleSF := StateFile{
		State:     string(types.ClaudeStateRunning),
		UpdatedAt: time.Now().Add(-15 * time.Minute),
		SessionID: "stale-agent",
	}
	staleData, _ := json.Marshal(staleSF)
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, sessionName+".agent.stale-agent.state"), staleData, 0o644); err != nil {
		t.Fatal(err)
	}

	aggState, err := UpdateAggregate(tmpDir, sessionName, DefaultStaleThreshold)
	if err != nil {
		t.Fatalf("UpdateAggregate failed: %v", err)
	}
	if aggState != types.ClaudeStateStopped {
		t.Errorf("aggregate = %q, want %q", aggState, types.ClaudeStateStopped)
	}
}

func TestUpdateAggregate_SessionEndFlow(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "status-agg-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	sessionName := "my-session"

	// Two agents running
	if err := WriteAgentState(tmpDir, sessionName, "agent-1", types.ClaudeStateRunning); err != nil {
		t.Fatal(err)
	}
	if err := WriteAgentState(tmpDir, sessionName, "agent-2", types.ClaudeStateIdle); err != nil {
		t.Fatal(err)
	}

	// Agent-1 ends: remove its file, recompute
	if err := RemoveAgentState(tmpDir, sessionName, "agent-1"); err != nil {
		t.Fatal(err)
	}
	aggState, err := UpdateAggregate(tmpDir, sessionName, DefaultStaleThreshold)
	if err != nil {
		t.Fatal(err)
	}
	if aggState != types.ClaudeStateIdle {
		t.Errorf("after agent-1 end: aggregate = %q, want %q", aggState, types.ClaudeStateIdle)
	}

	// Agent-2 ends: remove its file, recompute
	if err := RemoveAgentState(tmpDir, sessionName, "agent-2"); err != nil {
		t.Fatal(err)
	}
	aggState, err = UpdateAggregate(tmpDir, sessionName, DefaultStaleThreshold)
	if err != nil {
		t.Fatal(err)
	}
	if aggState != types.ClaudeStateStopped {
		t.Errorf("after all agents end: aggregate = %q, want %q", aggState, types.ClaudeStateStopped)
	}

	// Aggregate file should be removed when all agents gone
	if _, err := os.Stat(filepath.Join(tmpDir, sessionName+".state")); !os.IsNotExist(err) {
		t.Error("expected aggregate file to be removed when no agents remain")
	}
}

func TestSanitizeAgentID(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"normal ID", "sess-abc-123", "sess-abc-123"},
		{"path traversal", "../../etc/passwd", "passwd"},
		{"slash in ID", "foo/bar", "bar"},
		{"backslash in ID (unix safe)", `foo\bar`, `foo\bar`},
		{"dot-dot only", "..", "_"},
		{"single dot", ".", "_"},
		{"empty string", "", "_"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeAgentID(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeAgentID(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSanitizeSessionName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"normal name", "my-session", "my-session"},
		{"path traversal", "../../etc/passwd", "passwd"},
		{"slash in name", "foo/bar", "bar"},
		{"dot-dot only", "..", "_"},
		{"single dot", ".", "_"},
		{"empty string", "", "_"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeSessionName(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeSessionName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestWriteAgentState_PathTraversal(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "status-traversal-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Attempt path traversal via agentSessionID
	maliciousID := "../../etc/evil"
	if err := WriteAgentState(tmpDir, "my-session", maliciousID, types.ClaudeStateRunning); err != nil {
		t.Fatalf("WriteAgentState failed: %v", err)
	}

	// File must be inside tmpDir, not escaped
	entries, err := filepath.Glob(filepath.Join(tmpDir, "*.state"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 state file in tmpDir, got %d", len(entries))
	}

	// The sanitized name should be used
	expected := filepath.Join(tmpDir, "my-session.agent.evil.state")
	if entries[0] != expected {
		t.Errorf("file path = %q, want %q", entries[0], expected)
	}

	// Must be readable back with the same malicious ID (sanitized consistently)
	sf, err := ReadAgentState(tmpDir, "my-session", maliciousID)
	if err != nil {
		t.Fatalf("ReadAgentState with malicious ID failed: %v", err)
	}
	if sf.State != string(types.ClaudeStateRunning) {
		t.Errorf("State = %q, want %q", sf.State, types.ClaudeStateRunning)
	}
}

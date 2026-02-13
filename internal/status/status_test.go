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
		{types.ClaudeStateUnknown, "\u2753"},
		{types.ClaudeState("something-else"), "\u2753"},
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

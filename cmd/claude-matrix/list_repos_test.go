package main

import (
	"testing"
)

func TestListReposCmd_Structure(t *testing.T) {
	cmd := listReposCmd()

	if cmd.Use != "list-repos" {
		t.Errorf("expected Use 'list-repos', got %q", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("expected Short description to be set")
	}

	// Verify --force-refresh flag exists with correct default
	flag := cmd.Flags().Lookup("force-refresh")
	if flag == nil {
		t.Fatal("expected --force-refresh flag")
		return
	}
	if flag.DefValue != "false" {
		t.Errorf("expected --force-refresh default 'false', got %q", flag.DefValue)
	}
}

func TestListReposCmd_ForceRefreshFlagParsing(t *testing.T) {
	cmd := listReposCmd()

	// Simulate passing --force-refresh
	cmd.SetArgs([]string{"--force-refresh"})
	if err := cmd.ParseFlags([]string{"--force-refresh"}); err != nil {
		t.Fatalf("failed to parse --force-refresh flag: %v", err)
	}

	val, err := cmd.Flags().GetBool("force-refresh")
	if err != nil {
		t.Fatalf("failed to get force-refresh flag: %v", err)
	}
	if !val {
		t.Error("expected force-refresh to be true after parsing --force-refresh")
	}
}

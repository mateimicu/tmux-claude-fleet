package main

import (
	"strings"
	"testing"

	"github.com/mateimicu/tmux-claude-matrix/internal/hooks"
)

// TestHookHandlerCmd_AcceptsRegisteredMarker is a contract test that verifies
// the cobra command can parse whatever marker the registration side
// (hooks.HookMarker) generates. This prevents the two sides from drifting
// apart — the exact bug that caused all hook invocations to fail silently
// when --from=tmux-claude-matrix was rejected as an unknown flag.
func TestHookHandlerCmd_AcceptsRegisteredMarker(t *testing.T) {
	cmd := hookHandlerCmd()

	// hooks.HookMarker is the source of truth for what gets appended to the
	// hook-handler command in Claude's settings.json. Split it into args
	// the same way the shell would.
	args := strings.Fields(hooks.HookMarker)

	if err := cmd.ParseFlags(args); err != nil {
		t.Fatalf("hook-handler cobra command must accept the registered marker %q, but got: %v\n"+
			"This means every hook invocation from Claude Code will fail.\n"+
			"Fix: ensure hookHandlerCmd() defines flags for all tokens in hooks.HookMarker.",
			hooks.HookMarker, err)
	}
}

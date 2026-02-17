# PRD: Fix List View Counter and Tmux Status Variable

## Goal

Fix two related bugs in the session display:
1. The REPOSITORY column in the FZF list view incorrectly shows a counter suffix (e.g., `mateimicu/tmux-claude-matrix #1`). The repo name should be displayed without any counter — only the session name should carry a counter.
2. The tmux environment variable `@claude-matrix-title` is set to the session Title (which includes the repo name with counter) instead of the tmux session name. It should be set to the session name (e.g., `tmux-claude-matrix-1`).

## Requirements

1. The REPOSITORY column in the FZF list view (`internal/fzf/fzf.go`) must display the plain `org/repo` string extracted from the repo URL, without any `#N` counter suffix.
2. The tmux environment variable `@claude-matrix-title` must be set to the tmux session name (e.g., `tmux-claude-matrix-1`) instead of the Title (e.g., `mateimicu/tmux-claude-matrix #1`) in all places where it is set:
   - `cmd/claude-matrix/create.go` (session creation)
   - `cmd/claude-matrix/list.go` (session switch/recreation)
   - `cmd/claude-matrix/rename.go` (session rename)
3. The `Session.Title` field and `GenerateTitle()` function may still be used internally, but must not leak into the REPOSITORY display column or the `@claude-matrix-title` tmux variable.

## Acceptance Criteria

- [ ] FZF list view REPOSITORY column shows `org/repo` (e.g., `mateimicu/tmux-claude-matrix`) without any `#N` suffix
- [ ] `@claude-matrix-title` tmux variable is set to the session name (e.g., `tmux-claude-matrix-1`) in `create.go`
- [ ] `@claude-matrix-title` tmux variable is set to the session name in `list.go` (handleSwitchAction)
- [ ] `@claude-matrix-title` tmux variable is set to the session name in `rename.go`
- [ ] Existing tests pass (`make test`)
- [ ] New or updated tests cover the changed display logic

## Out of Scope

- Changing the `GenerateTitle()` or `GenerateUniqueName()` logic itself
- Changing the session name format or sanitization
- Modifying the tmux status bar template configuration
- Changing how Claude state indicators are displayed

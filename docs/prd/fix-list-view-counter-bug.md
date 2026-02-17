# PRD: Fix List View Display and Tmux Status Variable

## Goal

Fix the list view so that the REPOSITORY column shows the plain repo name (no counter), add a separate TITLE column for the customizable display name, and change `@claude-matrix-title` to default to the session name on creation (while remaining overridable via rename). Add a Ctrl+R shortcut in the list view to rename sessions inline.

## Background

The FZF list view currently shows the session `Title` (e.g., `mateimicu/tmux-claude-matrix #1`) in the REPOSITORY column. This is confusing because:
- The counter makes it look like the repo itself is numbered
- The repo name and the customizable title are conflated into one column

Additionally, `@claude-matrix-title` is set to the session title (e.g., `mateimicu/tmux-claude-matrix #1`). It should default to the tmux session name (e.g., `mateimicu-tmux-claude-matrix-1`) so status bar integrations reference the session by default. Users can override this via the rename command.

## Requirements

1. **REPOSITORY column shows plain repo name** - Display `orgRepo` from `parseRepoURL()` without any `#N` counter suffix. Do not fall back to `Session.Title` in `formatSessionTable()` (`internal/fzf/fzf.go:239-243`).

2. **Add a TITLE column to the list view** - A new column in the FZF table that displays `Session.Title`. This column shows the customizable display name (which defaults to the session name on creation, and can be changed via rename).

3. **`@claude-matrix-title` defaults to session name on create** - In `cmd/claude-matrix/create.go`, set `@claude-matrix-title` to the session name (e.g., `mateimicu-tmux-claude-matrix-1`) instead of the generated title. Also set `Session.Title` to the session name as the default.

4. **`@claude-matrix-title` reflects the current title on switch** - In `cmd/claude-matrix/list.go` (`handleSwitchAction`), set `@claude-matrix-title` to `Session.Title` (which is either the default session name or a user-renamed value).

5. **Rename command updates both `Session.Title` and `@claude-matrix-title`** - The existing rename command in `cmd/claude-matrix/rename.go` should continue to work as-is: it updates `Session.Title` and sets `@claude-matrix-title` to the user-provided title.

6. **Add Ctrl+R shortcut in list view for inline rename** - In the FZF interactive UI (`internal/fzf/fzf.go`), add a `ctrl-r` key binding that triggers a rename action for the selected session. This should prompt for a new title and update both `Session.Title` and `@claude-matrix-title`.

7. **Remove `GenerateTitle()` usage for default titles** - On session creation, use the session name as the default title instead of calling `GenerateTitle()`. The `GenerateTitle()` function can be removed if no longer needed elsewhere.

## Acceptance Criteria

- [ ] REPOSITORY column in list view shows `org/repo` (e.g., `mateimicu/tmux-claude-matrix`) without any `#N` suffix
- [ ] A new TITLE column appears in the list view showing the session's customizable title
- [ ] On session creation, `@claude-matrix-title` is set to the session name
- [ ] On session creation, `Session.Title` defaults to the session name
- [ ] On session switch (list view), `@claude-matrix-title` is set to `Session.Title`
- [ ] Rename command continues to update both `Session.Title` and `@claude-matrix-title`
- [ ] Ctrl+R in list view triggers rename for the selected session
- [ ] Existing tests pass; new/updated tests cover the changed display logic
- [ ] `make check` (lint + test) passes cleanly

## Out of Scope

- Changing `GenerateUniqueName()` or `sanitizeName()` logic
- Changing the tmux status bar template configuration
- Changing how Claude state indicators are displayed
- Migrating existing session metadata JSON files
- Changing the `@claude-matrix-title` variable name itself

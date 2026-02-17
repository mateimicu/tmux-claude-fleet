# PRD: Fix List View Counter Bug

## Goal

Fix the list view so that the REPOSITORY column shows the plain repo name without a numeric counter suffix, and change the `@claude-matrix-title` tmux variable to store the session name instead of the display title. Remove the now-unused `Title` field and `GenerateTitle()` function.

## Background

The FZF list view currently displays the session `Title` (e.g., `mateimicu/tmux-claude-matrix #1`) in the REPOSITORY column. This is confusing because the counter makes it look like the repo itself is numbered. Only the session (shown in the SESSION column as `[org-repo-1]`) should carry a distinguishing counter.

Additionally, the `@claude-matrix-title` tmux user variable is set to the session title (e.g., `mateimicu/tmux-claude-matrix #1`). It should store the tmux session name (e.g., `mateimicu-tmux-claude-matrix-1`) so that status bar integrations reference the session, not the repo.

## Requirements

1. **List view REPOSITORY column must show plain repo name** — Display `orgRepo` from `parseRepoURL()` without any counter suffix. Remove the fallback to `Session.Title` in `formatSessionTable()` (`internal/fzf/fzf.go:239-243`).

2. **`@claude-matrix-title` must store the session name** — All `SetSessionEnv` calls that set `@claude-matrix-title` must pass the session name (e.g., `mateimicu-tmux-claude-matrix-1`), not the title. Affected call sites:
   - `cmd/claude-matrix/create.go:131` (regular repo creation)
   - `cmd/claude-matrix/create.go:201` (workspace creation)
   - `cmd/claude-matrix/list.go:193` (session switch/recreation)
   - `cmd/claude-matrix/rename.go:69` (rename command)

3. **Remove `Title` field from `Session` struct** — Delete `Title string` from `types.Session` in `pkg/types/types.go:19`. Existing JSON metadata files with a `title` key should deserialize without error (Go ignores unknown JSON fields by default).

4. **Remove `GenerateTitle()` function** — Delete `GenerateTitle()` from `internal/session/session.go:91-105` and its tests.

5. **Remove all `Title` references in create and list commands** — Delete the `title`/`wsTitle` variables and their usage in `create.go` and `list.go`.

6. **Remove the `rename` command** — The `rename` command currently sets the `Title` field. Since `Title` is being removed, the rename command should be removed entirely. Users can identify sessions by the SESSION column name.

## Acceptance Criteria

- [ ] `formatSessionTable()` REPOSITORY column shows `orgRepo` from `parseRepoURL()` only (no `#N` suffix)
- [ ] `@claude-matrix-title` tmux variable is set to the session name (e.g., `mateimicu-tmux-claude-matrix-1`) at all call sites
- [ ] `Session.Title` field is removed from `pkg/types/types.go`
- [ ] `GenerateTitle()` is removed from `internal/session/session.go`
- [ ] Existing session metadata JSON files with a `title` key still load without error
- [ ] `rename` command is removed from CLI
- [ ] All existing tests pass; new tests cover the updated display and variable behavior
- [ ] `make check` (lint + test) passes cleanly

## Out of Scope

- Changing the session naming scheme (`GenerateUniqueName` / `sanitizeName`)
- Changing the FZF column layout or adding new columns
- Migrating existing session metadata files (removing the `title` key from on-disk JSON)
- Changing the `@claude-matrix-title` variable name itself

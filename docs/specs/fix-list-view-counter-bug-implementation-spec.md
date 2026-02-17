# Implementation Spec: Fix List View Display and Tmux Status Variable

## Context

### Background

The FZF list view conflates two concerns into the REPOSITORY column: it shows `Session.Title` (e.g., `mateimicu/tmux-claude-matrix #1`) instead of the plain repo name from `parseRepoURL()`. The `#N` counter suffix comes from `GenerateTitle()` which is called at session creation time. Additionally, `@claude-matrix-title` is set to this generated title rather than the session name, making status bar integrations confusing.

### Current Architecture

**Session creation flow** (`cmd/claude-matrix/create.go`):
1. `GenerateUniqueName(repoName)` produces a tmux session name (e.g., `mateimicu-tmux-claude-matrix-1`).
2. `GenerateTitle(selected.Name)` produces a title like `mateimicu/tmux-claude-matrix #1`.
3. `Session.Title` is set to the generated title.
4. `@claude-matrix-title` is set to the generated title.

**List view display** (`internal/fzf/fzf.go:239-243`):
- `formatSessionTable()` uses `Session.Title` (if non-empty) as the REPOSITORY column display name, falling back to `orgRepo` from `parseRepoURL()`.
- There is no separate TITLE column.

**Session switch** (`cmd/claude-matrix/list.go:192`):
- `handleSwitchAction()` sets `@claude-matrix-title` to `Session.Title`.

**Rename** (`cmd/claude-matrix/rename.go`):
- Updates `Session.Title` and `@claude-matrix-title` to the user-provided value.

### Constraints

- Must not change `GenerateUniqueName()` or `sanitizeName()` logic.
- Must not change tmux status bar template configuration.
- Must not migrate existing session metadata JSON files.
- Must maintain backward compatibility: old sessions with missing `Title` field (empty string) should degrade gracefully.
- `make check` (lint + test) must pass.

### Related Resources

| File | Lines | Description |
|------|-------|-------------|
| `internal/fzf/fzf.go` | 206-293 | `formatSessionTable()` -- table layout and column logic |
| `internal/fzf/fzf.go` | 72-84 | `SessionAction` enum and constants |
| `internal/fzf/fzf.go` | 86-90 | `SessionSelection` struct |
| `internal/fzf/fzf.go` | 134-193 | `SelectSessionWithAction()` -- FZF invocation with `--expect` keys |
| `internal/fzf/fzf.go` | 103-112 | `sessionLegend()` -- header legend text |
| `internal/fzf/fzf.go` | 455-463 | `extractSessionName()` -- parses `[name]` from selected line |
| `internal/session/session.go` | 91-105 | `GenerateTitle()` -- produces `"displayName #N"` titles |
| `cmd/claude-matrix/create.go` | 87-143 | `createRepoSession()` -- sets Title and `@claude-matrix-title` |
| `cmd/claude-matrix/create.go` | 145-213 | `createWorkspaceSession()` -- same pattern for workspaces |
| `cmd/claude-matrix/list.go` | 173-204 | `handleSwitchAction()` -- sets `@claude-matrix-title` on switch |
| `cmd/claude-matrix/rename.go` | 41-75 | `runRename()` -- updates Title + env var |
| `pkg/types/types.go` | 16-23 | `Session` struct definition |
| `internal/fzf/fzf_test.go` | 113-198 | `TestFormatSessionTable`, `TestFormatSessionTableWithTitle` |

## Architecture Decisions

### AD-1: REPOSITORY column always shows `orgRepo` from `parseRepoURL()`

**Approach:** Remove the Title-to-REPOSITORY fallback in `formatSessionTable()`. The REPOSITORY column always displays the `orgRepo` value derived from `Session.RepoURL`.

**Rationale:** The REPOSITORY column should reflect the actual repository, not a user-customizable title. Conflating the two causes the `#N` counter to appear in a column labeled "REPOSITORY."

**Impact:** `internal/fzf/fzf.go` -- `formatSessionTable()` only. The `displayName` variable is replaced with `orgRepo` directly.

### AD-2: Add a TITLE column between REPOSITORY and CLAUDE

**Approach:** Insert a new TITLE column into the `formatSessionTable()` table layout. Position it after REPOSITORY and before CLAUDE, since it contextualizes the repo. The column shows `Session.Title` (which defaults to the session name). For backward-compatible empty titles, display `"-"` or the session name as a fallback.

**Rationale:** The customizable display name needs its own column to avoid conflation with the immutable repo identifier.

**Impact:** `internal/fzf/fzf.go` -- `formatSessionTable()` column definitions, `rowData` struct, width tracking, header format string, and row format string. The `sessionLegend()` hint text should be updated to mention `ctrl-r: rename`.

### AD-3: Default Title = session name (not GenerateTitle output)

**Approach:** In `createRepoSession()` and `createWorkspaceSession()`, set both `Session.Title` and `@claude-matrix-title` to `sessionName` (the tmux session name, e.g., `mateimicu-tmux-claude-matrix-1`) instead of calling `GenerateTitle()`.

**Rationale:** The session name is stable, predictable, and matches what users see in tmux. `GenerateTitle()` produces a counter-based string that conflates repo identity with instance numbering.

**Impact:** `cmd/claude-matrix/create.go` -- two call sites (repo and workspace). `GenerateTitle()` in `internal/session/session.go` becomes unused and should be removed.

### AD-4: Remove `GenerateTitle()`

**Approach:** Delete `GenerateTitle()` from `internal/session/session.go` and remove its test (`TestGenerateTitle`) from `internal/session/session_test.go`.

**Rationale:** No callers remain after AD-3. Dead code removal.

**Impact:** `internal/session/session.go`, `internal/session/session_test.go`.

### AD-5: Add `SessionActionRename` and `ctrl-r` binding

**Approach:** Add a new `SessionActionRename` constant to the `SessionAction` enum. Register `ctrl-r` in the `--expect` key list passed to `runFZFWithExpect()` within `SelectSessionWithAction()`. When `ctrl-r` is pressed, return `SessionActionRename` with the selected session.

**Rationale:** Using the existing `--expect` mechanism keeps the implementation consistent with delete (`ctrl-d`) and toggle (`ctrl-t`). The rename prompt and persistence logic lives in the caller (`list.go`), not in the FZF package, maintaining separation of concerns.

**Impact:**
- `internal/fzf/fzf.go` -- new constant, updated `--expect` list, updated key-to-action mapping, updated `sessionLegend()`.
- `cmd/claude-matrix/list.go` -- new `case fzf.SessionActionRename` in the action switch, plus a new `handleRenameAction()` function.

### AD-6: Rename action handler in list.go

**Approach:** Add `handleRenameAction(sessionMgr, tmuxMgr, selected)` to `cmd/claude-matrix/list.go`. This function prompts for a new title (via stdin), updates `Session.Title` via `sessionMgr.Save()`, and sets `@claude-matrix-title` via `tmuxMgr.SetSessionEnv()`. After rename, the list loop continues (re-displays the list with the updated title).

**Rationale:** Reuses the same rename logic as `cmd/claude-matrix/rename.go` but triggered from the list view. We do not call `runRename()` directly because it detects the current session via `tmux display-message`, which would return the session running the list command, not the selected session. Instead, the handler receives the selected session explicitly.

**Impact:** `cmd/claude-matrix/list.go` -- new function and new case branch in the main loop.

## Component Structure & Interfaces

### Modified Components

#### 1. `internal/fzf/fzf.go`

**SessionAction enum:**
```go
// Add:
SessionActionRename SessionAction = "rename"
```

**`sessionLegend(showActiveOnly bool) string`:**
- Add `ctrl-r: rename` to the legend string.

**`SelectSessionWithAction(sessions, showActiveOnly) (*SessionSelection, error)`:**
- Add `"ctrl-r"` to the `expectedKeys` slice passed to `runFZFWithExpect()`.
- Add a key-to-action mapping: `"ctrl-r"` returns `SessionActionRename`.

**`formatSessionTable(sessions) (string, []string)`:**
- Remove the Title-fallback logic (lines 239-243). Always use `orgRepo` for the repo column.
- Add a `title` field to the `rowData` struct.
- Add `maxTitleW` width tracking.
- Add TITLE column to the header format string (after REPOSITORY, before CLAUDE).
- Add title column to each data row.
- For empty `Session.Title`, display the session name as fallback.

#### 2. `internal/session/session.go`

- Remove `GenerateTitle()` method.

#### 3. `cmd/claude-matrix/create.go`

**`createRepoSession()`:**
- Replace `sessionMgr.GenerateTitle(selected.Name)` with `sessionName`.
- Set `Session.Title = sessionName`.
- Set `@claude-matrix-title` to `sessionName`.

**`createWorkspaceSession()`:**
- Same changes as `createRepoSession()`.

#### 4. `cmd/claude-matrix/list.go`

**`runList()`:**
- Add `case fzf.SessionActionRename:` to the action switch.
- Call `handleRenameAction(sessionMgr, tmuxMgr, selection.Session)`.
- Continue loop after rename (like delete).

**New function `handleRenameAction(sessionMgr, tmuxMgr, selected)`:**
- Prompt user for new title via `bufio.Scanner` on stdin.
- If empty input, cancel rename.
- Update `selected.Session.Title` and save via `sessionMgr.Save()`.
- Set `@claude-matrix-title` via `tmuxMgr.SetSessionEnv()`.
- Print confirmation message.

## Integration Points

1. **FZF key binding registration** (`internal/fzf/fzf.go`): The `ctrl-r` key is added to the `--expect` list alongside `ctrl-d` and `ctrl-t`. FZF returns the pressed key in its output, which `parseFZFOutput()` already handles correctly for any registered key.

2. **List command action dispatch** (`cmd/claude-matrix/list.go`): The existing switch on `selection.Action` is extended with `SessionActionRename`. The handler follows the same pattern as `handleDeleteAction`: perform the operation, then continue the loop to re-display the list.

3. **Session metadata persistence** (`internal/session/session.go`): `Save()` and `Load()` remain unchanged. The `Title` field in the JSON already exists. The only change is what value is written at creation time (session name instead of generated title).

4. **Tmux env var** (`internal/tmux/tmux.go`): `SetSessionEnv()` is unchanged. Call sites in `create.go` and `list.go` pass different values than before.

## Data Flow & State Management

### Session Creation (changed)
```
User selects repo
  -> GenerateUniqueName(repoName) -> sessionName  (unchanged)
  -> Session.Title = sessionName                   (changed: was GenerateTitle())
  -> @claude-matrix-title = sessionName            (changed: was GenerateTitle())
  -> sessionMgr.Save(session)
```

### List View Display (changed)
```
sessionMgr.List()
  -> formatSessionTable(sessions)
     -> For each session:
        parseRepoURL(RepoURL) -> (source, orgRepo)  (unchanged)
        REPOSITORY column = orgRepo                   (changed: was Session.Title fallback)
        TITLE column = Session.Title                  (new column)
           if Title == "" -> display session Name     (backward compat fallback)
```

### Rename from List View (new)
```
User presses ctrl-r on selected session
  -> SessionActionRename returned to list.go
  -> handleRenameAction prompts for new title
  -> sessionMgr.Load(name) -> session
  -> session.Title = newTitle
  -> sessionMgr.Save(session)
  -> tmuxMgr.SetSessionEnv(name, "@claude-matrix-title", newTitle)
  -> Loop continues, list re-displays with updated title
```

### Session Switch (minor change)
```
User selects session (Enter)
  -> handleSwitchAction sets @claude-matrix-title = Session.Title  (unchanged logic)
     Value is now session name by default (instead of "org/repo #N")
```

## Test Strategy

### Tests to Update

1. **`TestFormatSessionTable`** (`internal/fzf/fzf_test.go`):
   - Verify REPOSITORY column shows `orgRepo` (not Title).
   - Verify new TITLE column appears in header.
   - Verify TITLE column displays `Session.Title` for sessions that have one.
   - Verify TITLE column displays session name for sessions with empty Title (backward compat).

2. **`TestFormatSessionTableWithTitle`** (`internal/fzf/fzf_test.go`):
   - Rewrite: Title should appear in the TITLE column, not the REPOSITORY column. REPOSITORY column should show `orgRepo` regardless of whether Title is set.

3. **`TestFormatSessionTableAlignment`** (`internal/fzf/fzf_test.go`):
   - Update to account for the additional TITLE column in alignment checks.

4. **`TestSessionActions_NoDuplicateValues`** (`internal/fzf/fzf_test.go`):
   - Add `SessionActionRename` to the duplicate-check list.

5. **`TestSessionLegend`** and **`TestSessionLegendAlwaysContainsEmojiLegend`** (`internal/fzf/fzf_test.go`):
   - Update to verify `ctrl-r: rename` appears in legend.

6. **`TestExtractSessionName`** (`internal/fzf/fzf_test.go`):
   - Update test input lines to include the TITLE column so `extractSessionName()` still correctly parses `[name]` from the new format.

### Tests to Remove

1. **`TestGenerateTitle`** (`internal/session/session_test.go`):
   - Remove entirely (function is deleted).

### Tests to Add

1. **`TestFormatSessionTableTitleColumn`** (`internal/fzf/fzf_test.go`):
   - Table-driven test verifying:
     - Session with Title set -> TITLE column shows Title, REPOSITORY shows orgRepo.
     - Session with empty Title -> TITLE column shows session Name.
     - Both columns are independently correct for each row.

2. **`TestSessionActionRenameValue`** (covered by existing `TestSessionActions_NoDuplicateValues` update):
   - Ensure `SessionActionRename` has a unique string value.

3. **`TestParseFZFOutput_CtrlR`** (`internal/fzf/fzf_test.go`):
   - Add a test case to the existing `TestParseFZFOutput` table for `ctrl-r` key parsing.

### Key Scenarios

- **New session creation**: Title defaults to session name, `@claude-matrix-title` set to session name.
- **List view with mixed sessions**: Old sessions (empty Title) show session name in TITLE column; new sessions show the session name (or renamed value). REPOSITORY always shows `orgRepo`.
- **Rename from list**: ctrl-r prompts, updates metadata and env var, list refreshes.
- **Rename via command**: Existing `rename` command continues to work identically.
- **Switch after rename**: `@claude-matrix-title` reflects the renamed value.

### Coverage

Focus unit tests on `formatSessionTable()` (the display logic is the core bug fix). The rename action handler in `list.go` is integration-level (requires stdin interaction and tmux), so verify the metadata + env var update logic via the existing `TestRenameFlow` pattern in `session_test.go`. The `SessionAction` enum guard test covers the new constant.

## Coding Expert Assignment

### Expert 1: Full-Stack Implementation

**Scope:** All changes across all files.

**Files to modify:**
- `internal/fzf/fzf.go` -- REPOSITORY column fix, TITLE column addition, `SessionActionRename`, `ctrl-r` binding, legend update
- `internal/fzf/fzf_test.go` -- Update existing tests, add new test cases
- `internal/session/session.go` -- Remove `GenerateTitle()`
- `internal/session/session_test.go` -- Remove `TestGenerateTitle`
- `cmd/claude-matrix/create.go` -- Use session name as default title
- `cmd/claude-matrix/list.go` -- Add rename action handler

**Rationale:** The changes are tightly coupled (column layout change affects tests, title default change affects creation and display). A single expert avoids coordination overhead and can ensure consistency across all touch points.

**Order of implementation:**
1. Remove `GenerateTitle()` from session package and update tests.
2. Update `create.go` to use session name as default title.
3. Fix `formatSessionTable()`: remove Title fallback, add TITLE column.
4. Add `SessionActionRename` enum value and `ctrl-r` binding.
5. Add `handleRenameAction()` in `list.go`.
6. Update legend text.
7. Update all affected tests.
8. Run `make check` to verify.

## Acceptance Criteria (Technical)

- [ ] `formatSessionTable()` REPOSITORY column always displays `orgRepo` from `parseRepoURL()`, never `Session.Title`.
- [ ] `formatSessionTable()` includes a TITLE column header and corresponding data in each row.
- [ ] TITLE column displays `Session.Title` when non-empty; falls back to `Session.Name` when empty.
- [ ] `SessionActionRename` constant exists with a unique value in the `SessionAction` enum.
- [ ] `SelectSessionWithAction()` registers `ctrl-r` in the `--expect` key list and maps it to `SessionActionRename`.
- [ ] `sessionLegend()` includes `ctrl-r: rename` in the legend text.
- [ ] `createRepoSession()` sets `Session.Title` and `@claude-matrix-title` to `sessionName`.
- [ ] `createWorkspaceSession()` sets `Session.Title` and `@claude-matrix-title` to `sessionName`.
- [ ] `GenerateTitle()` is removed from `internal/session/session.go`.
- [ ] `handleRenameAction()` exists in `list.go`, prompts for title, updates `Session.Title` via `sessionMgr.Save()`, and sets `@claude-matrix-title` via `tmuxMgr.SetSessionEnv()`.
- [ ] `runList()` dispatches `SessionActionRename` to `handleRenameAction()` and continues the list loop.
- [ ] All existing tests pass after updates; no regressions.
- [ ] `make check` (golangci-lint + tests with race detector) passes cleanly.

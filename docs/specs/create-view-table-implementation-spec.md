# Implementation Spec: Create View Table Layout

## Context

### Background

The Create View (repository selection during `claude-matrix create`) uses `FormatRepoLine()` to format each repository as a flat string like `github: org/repo - A cool repo [URL]`. This per-item approach prevents column alignment and makes it hard to scan large repo lists. The List View already solves this with `formatSessionTable()`, which uses batch formatting with dynamic column widths.

This spec converts the Create View to a table layout matching the List View's approach: batch formatting with aligned columns, emoji indicators, and a frozen header row.

### Current Architecture

**Per-item formatting** (`internal/fzf/fzf.go:202-211`):
- `FormatRepoLine(r *types.Repository) string` formats one repo at a time.
- Called in a loop by `SelectRepository()` and `runListRepos()`.
- No column alignment — relies on string concatenation.

**Batch formatting reference** (`internal/fzf/fzf.go:215-308`):
- `formatSessionTable(sessions) (string, []string)` computes max widths across all rows, then pads each column.
- Returns `(headerLine, dataLines)`.
- `SelectSessionWithAction()` prepends the header and passes `--header-lines=1` to FZF.

**Call sites for `FormatRepoLine`:**
1. `SelectRepository()` in `internal/fzf/fzf.go:38-40` — loops over repos, joins lines.
2. `runListRepos()` in `cmd/claude-matrix/list_repos.go:62-63` — prints one line per repo for FZF ctrl-r reload.

**Selection extraction:**
- `extractURL()` in `internal/fzf/fzf.go:460-468` parses `[identifier]` from the last `[...]` bracket on the line. This works regardless of column layout.

### Constraints

- Must not change the `Repository` struct or `Source` interface (PRD out of scope).
- Must preserve the `[identifier]` bracket format at the end of each line so `extractURL()` continues to work unchanged.
- The `list-repos` command is internal-only, so changing its output format is safe.
- `make check` (lint + test) must pass.

### Related Resources

| File | Lines | Description |
|------|-------|-------------|
| `internal/fzf/fzf.go` | 202-211 | `FormatRepoLine()` — current per-item formatting |
| `internal/fzf/fzf.go` | 215-308 | `formatSessionTable()` — reference batch pattern |
| `internal/fzf/fzf.go` | 14-28 | `buildRepoFZFArgs()` — FZF args for repo selection |
| `internal/fzf/fzf.go` | 30-70 | `SelectRepository()` — repo selection flow |
| `internal/fzf/fzf.go` | 346-378 | `displayWidth`, `padToDisplayWidth` — alignment helpers |
| `internal/fzf/fzf.go` | 460-468 | `extractURL()` — bracket identifier extraction |
| `cmd/claude-matrix/list_repos.go` | 33-67 | `runListRepos()` — outputs formatted lines for FZF reload |
| `pkg/types/types.go` | 6-13 | `Repository` struct — Source, Name, Description, IsWorkspace, URL |
| `internal/fzf/fzf_test.go` | 365-413 | `TestFormatRepoLine` — tests for current formatting |
| `internal/fzf/fzf_test.go` | 415-441 | `TestExtractURLWorkspace` — bracket extraction tests |

## Architecture Decisions

### AD-1: Replace per-item `FormatRepoLine` with batch `FormatRepoTable`

**Approach:** Introduce a new exported function `FormatRepoTable(repos []*types.Repository) (string, []string)` that follows the exact same pattern as `formatSessionTable()`. It receives all repos, computes dynamic column widths, and returns a header line plus padded data lines. Remove `FormatRepoLine`.

**Rationale:** Column alignment requires knowing the max width of each column across all rows. This is the same shift `formatSessionTable` already made for sessions. Reusing the same `(header, lines)` return pattern keeps the codebase consistent.

**Impact:** `internal/fzf/fzf.go` — new function replaces `FormatRepoLine`. Both call sites (`SelectRepository`, `runListRepos`) updated.

### AD-2: Three visible columns plus hidden identifier

**Approach:** The table has three visible columns:
- **TYPE**: Source type with emoji indicator and label.
- **ORG/REPO**: Repository display name (`repo.Name`).
- **DESCRIPTION**: Repository description text.

Plus a hidden `[identifier]` at the end of each line (URL or `workspace:<name>`), not represented as a column header.

**Rationale:** Matches the PRD requirements. The TYPE column replaces the plain text source prefix. ORG/REPO is the primary identifier users scan for. DESCRIPTION provides context. The hidden identifier preserves the existing `extractURL()` contract.

### AD-3: Source type emoji mapping via helper function

**Approach:** Add an unexported helper `repoTypeLabel(repo *types.Repository) string` that returns the emoji+label string:
- `repo.IsWorkspace == true` → `"📂 workspace"`
- `repo.Source == "github"` → `"🐙 github"`
- `repo.Source == "local"` → `"💻 local"`
- Fallback: `repo.Source` (no emoji, just the raw source string)

Check `IsWorkspace` first since workspace repos have `Source: "workspace"` but the `IsWorkspace` flag is the canonical indicator.

**Rationale:** Isolating the emoji mapping into a helper keeps `FormatRepoTable` focused on layout. The workspace check is ordered first because it's the most specific.

**Impact:** `internal/fzf/fzf.go` — new unexported function.

### AD-4: Workspace description includes repo count

**Approach:** In `FormatRepoTable`, the DESCRIPTION column for workspace entries includes the repo count prefix from `repo.Description` as-is. The PRD example shows `"3 repos - My project workspace"` — this is already the format produced by the workspace discovery code which sets `Description` to `"N repos"` or `"N repos - user description"`.

**Rationale:** No special formatting needed in the table function. The workspace source already computes the description with repo count. The table just displays `repo.Description` verbatim for all repo types.

### AD-5: Frozen header row via `--header-lines=1`

**Approach:** Add `"--header-lines=1"` to `buildRepoFZFArgs()`. In `SelectRepository()`, prepend the header line to the data lines before passing to FZF (same pattern as `SelectSessionWithAction`).

For `list-repos` output, print the header line first, then data lines. FZF's reload mechanism replaces the full input, and `--header-lines=1` will freeze the first line of the reloaded output as the column header.

**Rationale:** Follows the exact pattern already used by the List View. The `--header` flag (legend/keybinding hints) and `--header-lines=1` (frozen first-line column headers) coexist in FZF.

**Impact:** `internal/fzf/fzf.go` — `buildRepoFZFArgs()` and `SelectRepository()`. `cmd/claude-matrix/list_repos.go` — `runListRepos()`.

### AD-6: `extractURL` remains unchanged

**Approach:** No changes to `extractURL()`. The `[identifier]` bracket format at the end of each data line is preserved in the new table layout, exactly as it was in the flat format.

**Rationale:** `extractURL()` uses `strings.LastIndex` for `[` and `]`, which works regardless of preceding column content.

## Component Structure & Interfaces

### Modified Components

#### 1. `internal/fzf/fzf.go`

**Remove `FormatRepoLine(r *types.Repository) string`:**
- Delete entirely. Replaced by `FormatRepoTable`.

**New function `repoTypeLabel(repo *types.Repository) string`:**
- Returns emoji+label string for the TYPE column.
- Mapping: `IsWorkspace` → `"📂 workspace"`, `"github"` → `"🐙 github"`, `"local"` → `"💻 local"`, fallback → raw `Source`.

**New function `FormatRepoTable(repos []*types.Repository) (string, []string)`:**
- Signature mirrors `formatSessionTable`.
- Exported (used by `list_repos.go` in a different package).
- Computes per-row data and tracks max column widths for TYPE, ORG/REPO, DESCRIPTION.
- Returns `(headerLine, dataLines)`.
- Each data line ends with ` [identifier]` where identifier is `repo.URL` for git repos or `workspace:<name>` for workspaces.
- Header line contains column names: `TYPE`, `ORG/REPO`, `DESCRIPTION`.

**Update `SelectRepository(repos, binaryPath) (*types.Repository, error)`:**
- Replace the `FormatRepoLine` loop with a single call to `FormatRepoTable(repos)`.
- Prepend header line to data lines (same as `SelectSessionWithAction` pattern at line 154).
- Pass the combined lines to `runFZF`.

**Update `buildRepoFZFArgs(binaryPath string) []string`:**
- Add `"--header-lines=1"` to the returned args slice.

#### 2. `cmd/claude-matrix/list_repos.go`

**Update `runListRepos(ctx, forceRefresh) error`:**
- Replace the `FormatRepoLine` loop with a single call to `fzf.FormatRepoTable(repoList)`.
- Print header line first (`fmt.Println(header)`).
- Print each data line (`fmt.Println(line)`).

## Integration Points

1. **FZF header mechanism** (`internal/fzf/fzf.go`): `--header-lines=1` freezes the first input line as column headers. The existing `--header=...` legend text is separate and continues to display keybinding hints. Both flags coexist — this is the same pattern as `SelectSessionWithAction`.

2. **FZF reload binding** (`buildRepoFZFArgs` → `list-repos --force-refresh`): When ctrl-r triggers reload, `list-repos` outputs header + data lines. FZF replaces its input and `--header-lines=1` freezes the new first line as column headers. The reload output format must match the initial format exactly.

3. **Selection extraction** (`extractURL`): Unchanged. The `[identifier]` bracket at the end of each data line is preserved. `extractURL` uses `strings.LastIndex` which works regardless of preceding table content.

4. **Workspace detection** (`SelectRepository` lines 52-59): After `extractURL` returns the identifier, the existing `strings.CutPrefix(identifier, "workspace:")` check continues to work because the bracket content is unchanged.

## Data Flow

### Create View Selection (changed)
```
repos.ListAll() -> []*Repository
  -> FormatRepoTable(repos)                         (NEW: was FormatRepoLine per repo)
     -> For each repo:
        repoTypeLabel(repo) -> "🐙 github"          (NEW: emoji + label)
        repo.Name -> "org/repo"                       (unchanged)
        repo.Description -> "A cool repo"             (unchanged)
        repo.URL or "workspace:<name>" -> identifier  (unchanged)
     -> Compute max column widths                     (NEW: batch width computation)
     -> Return (headerLine, dataLines)                (NEW: header + padded lines)
  -> Prepend header to data lines                     (NEW)
  -> runFZF(allLines, args with --header-lines=1)     (CHANGED: added --header-lines=1)
  -> extractURL(selected) -> identifier               (unchanged)
  -> Match repo by URL or workspace name              (unchanged)
```

### list-repos Reload (changed)
```
runListRepos(ctx, forceRefresh)
  -> repos.ListAll() -> []*Repository
  -> FormatRepoTable(repos)                           (NEW: was FormatRepoLine per repo)
  -> fmt.Println(header)                              (NEW: header line first)
  -> for each line: fmt.Println(line)                 (data lines)
```

## Test Strategy

### Tests to Update

1. **`TestFormatRepoLine`** (`internal/fzf/fzf_test.go:365-413`):
   - Rename to `TestFormatRepoTable`.
   - Change from testing individual line output to testing `(header, lines)` return value.
   - Verify header contains column names: `TYPE`, `ORG/REPO`, `DESCRIPTION`.
   - Verify each data line contains the expected TYPE emoji, ORG/REPO, DESCRIPTION, and `[identifier]`.
   - Use the same test repos (github with description, local without description, workspace).

2. **`TestExtractURLWorkspace`** (`internal/fzf/fzf_test.go:415-441`):
   - Update test input lines to use the new table format (padded columns ending with `[identifier]`).
   - `extractURL` behavior should be identical since it uses `LastIndex`.

3. **`TestBuildRepoFZFArgs`** (`internal/fzf/fzf_test.go:710-765`):
   - Add assertion that `--header-lines=1` is present in the returned args.

### Tests to Add

1. **`TestFormatRepoTableAlignment`** (`internal/fzf/fzf_test.go`):
   - Analogous to `TestFormatSessionTableAlignment`.
   - Verify all data lines have consistent display width up to the `[identifier]` bracket.
   - Use repos with varying name/description lengths to exercise padding.

2. **`TestFormatRepoTableEmojiIndicators`** (`internal/fzf/fzf_test.go`):
   - Verify `🐙 github` appears for github repos.
   - Verify `💻 local` appears for local repos.
   - Verify `📂 workspace` appears for workspace repos.

3. **`TestRepoTypeLabel`** (`internal/fzf/fzf_test.go`):
   - Table-driven test for the emoji mapping helper.
   - Cases: github, local, workspace, unknown source fallback.

### Key Scenarios

- **Mixed source types**: Table with github, local, and workspace repos aligns correctly despite different emoji widths.
- **Empty descriptions**: Repos without descriptions show empty DESCRIPTION column; padding still aligns.
- **Workspace description with repo count**: Workspace description like `"3 repos"` displays verbatim in DESCRIPTION column.
- **FZF reload consistency**: `list-repos --force-refresh` output matches `SelectRepository` initial format (header + data).
- **Selection extraction**: `extractURL` parses `[identifier]` from table-formatted lines identically to flat-formatted lines.

### Coverage

Focus unit tests on `FormatRepoTable` (the core formatting function) and `repoTypeLabel` (the emoji mapping). The `extractURL` tests confirm bracket extraction still works with the new format. Alignment tests catch display width miscalculations. FZF args tests verify `--header-lines=1` is included.

## Coding Expert Assignment

### Expert 1: Full Implementation

**Scope:** All changes across all files.

**Files to modify:**
- `internal/fzf/fzf.go` — Remove `FormatRepoLine`, add `repoTypeLabel`, add `FormatRepoTable`, update `SelectRepository`, update `buildRepoFZFArgs`
- `internal/fzf/fzf_test.go` — Update `TestFormatRepoLine` → `TestFormatRepoTable`, update `TestExtractURLWorkspace`, update `TestBuildRepoFZFArgs`, add alignment and emoji tests
- `cmd/claude-matrix/list_repos.go` — Update `runListRepos` to use `FormatRepoTable`

**Rationale:** The changes are tightly coupled — the new function is used by both `SelectRepository` and `runListRepos`, and all tests relate to the same formatting logic. A single expert avoids coordination overhead.

**Order of implementation:**
1. Add `repoTypeLabel` helper with tests.
2. Add `FormatRepoTable` with tests (alignment, column content, emoji indicators).
3. Update `SelectRepository` to use `FormatRepoTable` + prepend header.
4. Update `buildRepoFZFArgs` to include `--header-lines=1`.
5. Update `runListRepos` to use `FormatRepoTable`.
6. Remove `FormatRepoLine`.
7. Update `TestExtractURLWorkspace` input lines for new format.
8. Update `TestBuildRepoFZFArgs` to check for `--header-lines=1`.
9. Run `make check` to verify.

## Acceptance Criteria (Technical)

- [ ] `FormatRepoTable(repos)` returns `(headerLine, dataLines)` with dynamically padded columns.
- [ ] Header line contains column names: `TYPE`, `ORG/REPO`, `DESCRIPTION`.
- [ ] TYPE column shows `🐙 github` for github repos, `💻 local` for local repos, `📂 workspace` for workspace repos.
- [ ] ORG/REPO column shows `repo.Name`.
- [ ] DESCRIPTION column shows `repo.Description` (empty string if none).
- [ ] Each data line ends with `[identifier]` where identifier is URL or `workspace:<name>`.
- [ ] Columns are padded using `displayWidth`/`padToDisplayWidth` helpers.
- [ ] `SelectRepository` prepends header to data lines and passes to FZF.
- [ ] `buildRepoFZFArgs` includes `--header-lines=1`.
- [ ] `list-repos` command outputs header line first, then data lines.
- [ ] `FormatRepoLine` is removed.
- [ ] `extractURL` works unchanged with the new line format.
- [ ] All existing tests pass after updates; new tests cover alignment and emoji mapping.
- [ ] `make check` (golangci-lint + tests with race detector) passes cleanly.

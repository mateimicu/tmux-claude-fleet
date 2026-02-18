# PRD: Create View Table Layout

## Goal

Improve the Create View (repository selection during `claude-matrix create`) to use an aligned table layout matching the List View's structure. Currently, the Create View uses a flat format (`github: org/repo - A cool repo [URL]`) that is hard to scan. The new layout should present repositories in clearly aligned columns with emoji source indicators, making it easy to distinguish source types and scan large repo lists.

## Background

- **List View** (`formatSessionTable` in `internal/fzf/fzf.go:190-277`): Uses a properly aligned table with columns `#`, `TMUX`, `SOURCE`, `REPOSITORY`, `CLAUDE`, `SESSION`. Each column is padded to align using `displayWidth`/`padToDisplayWidth` helpers that handle emoji widths correctly.
- **Create View** (`FormatRepoLine` in `internal/fzf/fzf.go:179-188`): Outputs a flat string like `github: org/repo - A cool repo [https://github.com/org/repo]` with no column alignment.
- **`list-repos` command** (`cmd/claude-matrix/list_repos.go`): Outputs `FormatRepoLine` per repo for FZF's ctrl-r reload binding. This is documented as internal-only (`"Used internally by the FZF reload binding"`), so changing its output format is safe.

## Requirements

1. Replace `FormatRepoLine` with a new `formatRepoTable` function (analogous to `formatSessionTable`) that returns a header line and aligned data lines.
2. The table must have three visible columns, plus a hidden identifier:
   - **TYPE**: Source type with emoji indicator and label (see R3 for mapping).
   - **ORG/REPO**: Repository display name (`org/repo` for git repos, workspace name for workspaces).
   - **DESCRIPTION**: Repository description text. For workspaces, include the repo count (e.g., `3 repos - My project workspace`). Empty string if no description.
   - **[identifier]**: Hidden at end of line in `[brackets]` for selection extraction. Contains the URL for git repos or `workspace:<name>` for workspaces. Not a visible column header.
3. Source type emoji mapping:
   - `github` source: `🐙 github`
   - `local` source: `💻 local`
   - `workspace` source (when `IsWorkspace == true`): `📂 workspace`
4. Columns must be padded to align using the existing `displayWidth`/`padToDisplayWidth` helpers, matching the List View's alignment approach. Column widths are computed dynamically from the data.
5. A frozen header row (`--header-lines=1` in FZF args) must display column names (`TYPE`, `ORG/REPO`, `DESCRIPTION`) and remain visible while scrolling.
6. `SelectRepository` must pass the header line + data lines to FZF (prepending the header to the data, same pattern as `SelectSessionWithAction`).
7. The `list-repos` command must output the new table format (header line first, then data lines) so that ctrl-r reload produces consistent output.
8. `extractURL` continues to work unchanged since the `[identifier]` bracket format at the end of each line is preserved.
9. The existing `FormatRepoLine` function signature may change or be replaced. Update all call sites (`SelectRepository`, `list_repos.go`).
10. All existing tests for `FormatRepoLine`, `extractURL`, and related helpers must be updated to reflect the new table format.

## Acceptance Criteria

- [ ] Create View displays repos in an aligned table with columns: TYPE, ORG/REPO, DESCRIPTION.
- [ ] TYPE column shows emoji + label: `🐙 github`, `💻 local`, `📂 workspace`.
- [ ] Columns are dynamically padded so all rows align regardless of content width.
- [ ] A frozen header row with column names is visible at all times in FZF.
- [ ] Workspace entries show `📂 workspace` in the TYPE column.
- [ ] Workspace description includes repo count (e.g., `3 repos`).
- [ ] `[identifier]` bracket at end of each line is preserved for selection extraction.
- [ ] `list-repos --force-refresh` outputs the same table format (header + data lines).
- [ ] ctrl-r reload in the Create View FZF picker refreshes correctly with the new format.
- [ ] All existing tests pass after updates; new tests cover `formatRepoTable` alignment, column content, and emoji indicators.
- [ ] `extractURL` and workspace selection logic remain functional.

## Out of Scope

- Changes to the List View (`formatSessionTable` / session selection UI).
- Adding new source types beyond `github`, `local`, `workspace`.
- Sorting or filtering repos in the Create View (existing behavior preserved).
- Changes to the `Repository` type or `Source` interface.
- Adding a `--format` flag to `list-repos` for backward compatibility (confirmed internal-only).

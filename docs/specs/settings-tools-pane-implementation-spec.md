# Implementation Spec: Settings/Tools Pane with Cache Pre-fill Action

**PRD**: `docs/prd/settings-tools-pane.md`
**Status**: Draft

## Overview

This feature adds a `ctrl-s` keybinding to the session list view that opens a tools sub-menu. The first (and only) tool action is "Pre-fill mirror cache", which bulk-creates/updates git mirror caches for all configured repositories.

The design follows existing patterns: `SessionAction` enum + `runFZFWithExpect` keybinding pattern for UI, `repos.Discoverer.ListAll()` for discovery, and `git.Manager` for mirror operations.

## Architecture

### Data Flow

```
User presses ctrl-s in session list
  -> runList() receives SessionActionTools
  -> Show tools FZF sub-menu (SelectToolAction)
  -> User selects "Pre-fill mirror cache"
  -> Discover all repos via buildSources + Discoverer.ListAll
  -> Flatten workspace repos into individual clone URLs
  -> For each URL: EnsureMirror (create or update)
  -> Print summary
  -> Return to session list loop
```

### Component Changes

Four areas of the codebase are modified. No new packages are introduced.

---

## 1. `internal/fzf/fzf.go` — New Action + Tools Menu

### 1a. New `SessionActionTools` constant

Add to the existing `SessionAction` const block:

```go
SessionActionTools SessionAction = "tools"
```

Update the guard test in `fzf_test.go` (`TestSessionActions_NoDuplicateValues`) to include `SessionActionTools` in the checked list.

### 1b. Add `ctrl-s` to session list keybindings

In `SelectSessionWithAction`, add `"ctrl-s"` to the `expectedKeys` slice passed to `runFZFWithExpect` (currently `[]string{"ctrl-d", "ctrl-t"}`).

Add a handler for `key == "ctrl-s"` that returns `SessionActionTools` (no session selection needed, similar to `ctrl-t` / `SessionActionToggleFilter`).

### 1c. Update `sessionLegend`

Add `ctrl-s: tools` to the legend string. Place it after the toggle filter hint on the first line.

### 1d. New `ToolAction` type and `SelectToolAction` function

Define a new type and function for the tools sub-menu:

```go
type ToolAction string

const (
    ToolActionPrefillCache ToolAction = "prefill_cache"
    ToolActionCancel       ToolAction = "cancel"
)

func SelectToolAction() (ToolAction, error)
```

**Contract**: `SelectToolAction` displays a simple FZF menu with available tool actions. For now, the only entry is "Pre-fill mirror cache". Uses `runFZF` (not `runFZFWithExpect` — no special keys needed). Returns `ToolActionPrefillCache` on selection, `ToolActionCancel` on FZF exit/cancel error.

FZF styling should match existing patterns:
- `--prompt=🔧 Select tool > `
- `--reverse`, `--border=rounded`, `--height=80%`
- `--header=` with a simple legend: `enter: select | ctrl-c: back`

---

## 2. `internal/git/git.go` — New `EnsureMirror` method

Add a new exported method to `Manager`:

```go
func (m *Manager) EnsureMirror(url, cacheDir string) (created bool, err error)
```

**Contract**: Creates a new mirror if one doesn't exist, or updates (fetch --prune) an existing mirror. Returns `created=true` if a new mirror was created, `created=false` if an existing mirror was updated. This method encapsulates the existing `createMirror`/`updateMirror` logic without the final clone step that `CloneWithCache` performs.

Implementation reuses the existing unexported `GetMirrorPath`, `MirrorExists`, `createMirror`, and `updateMirror` methods — no need to export them.

---

## 3. `cmd/claude-matrix/list.go` — Tools action handler

### 3a. New switch case in `runList`

Add a `case fzf.SessionActionTools:` branch in the action switch (alongside `SessionActionToggleFilter`, `SessionActionDelete`, `SessionActionSwitch`). This case calls a new `handleToolsAction` function and then `continue`s the loop to re-show the session list.

### 3b. `handleToolsAction` function

```go
func handleToolsAction(ctx context.Context) error
```

**Contract**: Calls `fzf.SelectToolAction()`. On `ToolActionPrefillCache`, calls `runPrefillCache(ctx)`. On cancel, returns nil (back to session list).

Defined in `cmd/claude-matrix/list.go` alongside the existing `handleDeleteAction` and `handleSwitchAction` functions.

---

## 4. `cmd/claude-matrix/prefill.go` — New file: Pre-fill cache logic

New file containing the pre-fill orchestration. This is placed in `cmd/claude-matrix/` (same package as the other command handlers) because it coordinates between config, sources, and git — it is command-level orchestration, not reusable library logic.

### 4a. `runPrefillCache` function

```go
func runPrefillCache(ctx context.Context) error
```

**Contract and data flow**:

1. **Load config**: `config.Load()` to get `CacheDir` and source settings.
2. **Build sources**: `buildSources(ctx, cfg, io.Discard)` — suppress source log output (the user doesn't need to see GitHub auth status during pre-fill).
3. **Discover repos**: `repos.NewDiscoverer(sources...).ListAll(ctx)` to get all repositories.
4. **Flatten to clone URLs**: Iterate repos. For each:
   - If `repo.IsWorkspace && len(repo.WorkspaceRepos) > 0`: add each URL from `repo.WorkspaceRepos`.
   - If `repo.URL != ""`: add `repo.URL`.
   - Deduplicate URLs (a repo may appear both as a workspace sub-repo and as a standalone entry).
5. **Mirror loop**: For each unique URL (sequentially):
   - Print `[X/N] Cloning mirror: org/repo...` or `[X/N] Updating mirror: org/repo...` based on whether mirror exists.
   - Call `gitMgr.EnsureMirror(url, cfg.CacheDir)`.
   - On success: print `[X/N] ✓ org/repo`. Track as "new" or "updated".
   - On error: print `[X/N] ✗ org/repo: <error>`. Track as "failed". Continue to next.
   - Check `ctx.Done()` between repos for cancellation support.
6. **Summary**: Print totals: `Total: N | New: X | Updated: Y | Failed: Z`.

Use `git.ExtractRepoName(url)` for display names (already exported).

### 4b. Context cancellation

`runList` must pass a cancellable context. The `runPrefillCache` function checks `ctx.Err()` between each repo. On cancellation, it prints a partial summary and returns nil (not an error — the user intentionally cancelled).

Note: `runList` currently receives `context.Context` from Cobra (`cmd.Context()`). This should be passed through to `handleToolsAction` and `runPrefillCache`. The existing `runList(_ context.Context)` signature already accepts it but discards it; it should use it instead.

---

## Test Boundaries

### Unit tests (no mocking, pure logic)

| What | Where | Tests |
|------|-------|-------|
| `SessionActionTools` uniqueness | `internal/fzf/fzf_test.go` | Update existing `TestSessionActions_NoDuplicateValues` |
| `sessionLegend` includes `ctrl-s: tools` | `internal/fzf/fzf_test.go` | Update existing `TestSessionLegend` |
| `ToolAction` values are distinct | `internal/fzf/fzf_test.go` | New test |
| URL deduplication in flatten logic | `cmd/claude-matrix/prefill_test.go` | Test the flatten+dedup helper |

### Integration-level tests (may need temp dirs)

| What | Where | Tests |
|------|-------|-------|
| `EnsureMirror` creates mirror when missing | `internal/git/git_test.go` | New test with temp dir (uses real git) |
| `EnsureMirror` updates mirror when exists | `internal/git/git_test.go` | New test with temp dir (uses real git) |

### Not tested (UI/FZF interactions)

- FZF `ctrl-s` key binding (requires interactive terminal)
- Tools menu display and selection (requires FZF binary)
- Full end-to-end pre-fill flow (requires repos + network)

---

## Files Changed Summary

| File | Change |
|------|--------|
| `internal/fzf/fzf.go` | Add `SessionActionTools` const, `ToolAction` type, `SelectToolAction()` func, update `SelectSessionWithAction` expected keys, update `sessionLegend` |
| `internal/fzf/fzf_test.go` | Update guard test, update legend test, add `ToolAction` guard test |
| `internal/git/git.go` | Add `EnsureMirror()` method |
| `internal/git/git_test.go` | Add `EnsureMirror` tests |
| `cmd/claude-matrix/list.go` | Add `SessionActionTools` case, `handleToolsAction()`, pass context through |
| `cmd/claude-matrix/prefill.go` | New file: `runPrefillCache()`, URL flatten+dedup helper |
| `cmd/claude-matrix/prefill_test.go` | New file: dedup/flatten tests |

## Out of Scope (per PRD)

- Parallel mirror creation
- Selective pre-fill (specific repos/sources)
- Cache eviction
- CLI subcommand (`claude-matrix prefill`)
- Additional tool actions beyond pre-fill

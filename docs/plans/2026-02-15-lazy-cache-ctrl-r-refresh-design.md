# Lazy Cache Load + Ctrl+R Refresh

## Problem

Currently, the GitHub repo cache has a 30-minute TTL. Every time the user runs `create` and the cache is stale, they wait for a fresh API fetch. The user wants:

1. Repos load on first `create` when cache is empty (already works)
2. Ctrl+R in the FZF repo picker triggers a manual refresh
3. Otherwise, don't auto-refresh (use a 24h TTL as safety net)

## Approach

Use FZF's native `--bind reload` feature with a new `list-repos` CLI subcommand.

## Changes

### 1. New `list-repos` subcommand (`cmd/claude-matrix/list_repos.go`)

- Outputs FZF-formatted repo lines to stdout (one per line)
- `--force-refresh` flag clears GitHub cache before fetching
- Suppresses all logging (uses `io.Discard`) so only data lines are emitted
- Reuses shared source-building logic

### 2. Shared source builder (`cmd/claude-matrix/sources.go`)

Extract the repeated "build sources list" logic from `create.go`, `refresh.go`, and the new `list_repos.go` into a shared `buildSources(ctx, cfg, logger)` function.

### 3. FZF `SelectRepository` changes (`internal/fzf/fzf.go`)

- Accept binary path parameter to construct reload command
- Add `--bind 'ctrl-r:reload(claude-matrix list-repos --force-refresh)'`
- Update header: `"ctrl-r: refresh"` added to keybinding hints
- FZF updates the list in-place after reload completes

### 4. Cache TTL default change (`internal/config/config.go`)

- Change default from `30 * time.Minute` to `24 * time.Hour`

### 5. Update `create.go`

- Use shared `buildSources()` helper
- Pass binary path to `fzf.SelectRepository()`

### 6. Update `refresh.go`

- Use shared `buildSources()` helper

## Testing

- Unit test for `list-repos` output format
- Unit test for shared `buildSources()` function
- Unit test for FZF args construction (verify `--bind` is present)
- Update existing tests that depend on the 30m default TTL

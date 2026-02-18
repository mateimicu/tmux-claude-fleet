# Implementation Spec: Debug Logging Configuration

**PRD**: `docs/prd/debug-logging.md` (PR #56)
**Branch**: `spec/debug-logging` (stacked on `prd/debug-logging`)

## Overview

Suppress progress/status output by default. Add a `Debug` flag (config file,
env var, CLI flag) that restores verbose output. Warnings and errors always
print regardless of debug mode.

## Architecture

### Approach: `io.Writer`-based debug logger

The codebase already uses the `io.Writer` pattern in `GitHubSource.SetLogger()`
and `buildSources()` — passing `os.Stdout` for verbose output or `io.Discard`
to suppress. This spec extends that pattern globally through a shared debug
writer rather than adding `if cfg.Debug` checks at every call site.

A single `io.Writer` is resolved once at startup: `os.Stdout` when debug is on,
`io.Discard` when off. All progress messages write to this writer. Warnings and
errors continue writing directly to `os.Stdout`/`os.Stderr`.

## Component Changes

### 1. Config struct — `pkg/types/types.go`

Add one field to `Config`:

```go
type Config struct {
    // ... existing fields ...
    Debug bool
}
```

Default: `false`.

### 2. Config loading — `internal/config/config.go`

Three additions following existing patterns:

| Layer | Key / Var | Values | Precedence |
|-------|-----------|--------|------------|
| Config file | `DEBUG` | `1`, `true` | Lowest |
| Env var | `TMUX_CLAUDE_MATRIX_DEBUG` | `1`, `true` | Middle |
| CLI flag | `--debug` / `-d` | boolean | Highest |

- `defaults()` — `Debug: false` (already zero-value, but explicit).
- `applyConfigValue()` — add `case "DEBUG"` using the existing
  `value == "1" || value == "true"` pattern.
- `applyEnvOverrides()` — add `TMUX_CLAUDE_MATRIX_DEBUG` check using the
  same boolean pattern as `GITHUB_ENABLED`.

CLI flag override happens in the root command (see section 4), not in
`config.Load()`, because Cobra flags are not available at config-load time.

### 3. Debug writer helper — `internal/logging/logging.go` (new package)

Minimal package — one function that resolves the writer:

```
package logging

func DebugWriter(debug bool) io.Writer
```

**Contract**: Returns `os.Stdout` if `debug` is `true`, `io.Discard` otherwise.

This keeps the decision in one place. Commands call it once and pass the
resulting `io.Writer` to `fmt.Fprintf(debugW, ...)` for progress messages.

No other functions needed — `fmt.Fprintf` with `io.Writer` is the logging
mechanism. No custom logger struct, no log levels.

### 4. Root command — `cmd/claude-matrix/main.go`

Add a persistent flag and `PersistentPreRunE` to the root command:

```
rootCmd.PersistentFlags().BoolP("debug", "d", false, "Enable debug logging")
```

**`PersistentPreRunE` responsibilities**:
1. Load config via `config.Load()`.
2. If `--debug` flag was explicitly set by the user, override `cfg.Debug`.
3. Store `cfg` in the command context (using a context key) so subcommands
   can retrieve it without calling `config.Load()` again.

**Context key**: Define an unexported context key type in `main.go`:

```
type ctxKey string
const configKey ctxKey = "config"
```

Helper functions in `main.go`:
- `configFromContext(ctx context.Context) *types.Config` — retrieves config.

Subcommands switch from calling `config.Load()` directly to reading from
context.

### 5. Command changes — `cmd/claude-matrix/*.go`

Each command's `run*` function changes:

1. **Get config from context** instead of calling `config.Load()`.
2. **Resolve debug writer** via `logging.DebugWriter(cfg.Debug)`.
3. **Replace progress `fmt.Print*` calls** with `fmt.Fprintf(debugW, ...)`.
4. **Keep warning/error prints unchanged** — they stay as direct
   `fmt.Printf` / `fmt.Fprintln(os.Stderr, ...)`.

#### `create.go`

Progress messages to route through debug writer:
- `"🔍 Discovering repositories..."`
- `"✓ Found %d repositories"`
- `"📦 Repository already exists at %s"`
- `"📦 Cloning %s (using cache for faster cloning)..."`
- `"✓ Clone complete"`
- `"🚀 Creating tmux session '%s'..."`
- `"✓ Session created: %s"`
- Workspace equivalents: `"📦 Setting up workspace..."`, clone status lines

Warning/error messages that remain always-visible:
- `"⚠️  Failed to save session metadata: %v"`
- `"⚠️  Failed to set session title env: %v"`
- `"⚠️  Failed to switch to session: %v"`
- `"You can attach manually with: tmux attach -t %s"`

#### `sources.go`

`buildSources()` already accepts an `io.Writer` parameter. No signature change.
Callers pass the debug writer instead of hardcoded `os.Stdout`/`io.Discard`.

The `"⚠️  GitHub authentication not found"` message is a warning — change its
write target from `logger` to `os.Stdout` (always visible). The remaining
GitHub integration status lines stay on `logger` (debug-only).

#### `diagnose.go`

`diagnose` is a diagnostic command — all its output is intentional and
informational. Its messages stay always-visible (no change to existing prints).

One addition: report the effective debug mode in the configuration section:
```
  Debug mode: true/false
```

#### `refresh.go`

Progress messages to route through debug writer:
- `"🔄 Refreshing repository cache..."`
- `"✓ Cache refreshed with %d repositories"`
- `"📁 Cache location: %s"`
- `"⏰ Cache TTL: %s"`

Warning that remains always-visible:
- `"⚠️  Failed to clear cache: %v"`

#### `list.go`

Most output here is interactive UI (prompts, confirmation, FZF). These are
user-facing and stay always-visible.

Progress messages to route through debug writer:
- `"🚀 Switching to session '%s'..."`
- `"✓ Session '%s' deleted successfully!"`

Warning/error messages that remain always-visible:
- `"⚠️  Failed to delete session: %v"`
- `"⚠️  Session not active, recreating..."`
- `"⚠️  Failed to switch to session: %v"`
- `"⚠️  Failed to set session title: %v"`
- `"You can attach manually with: tmux attach -t %s"`

#### `list_repos.go`

Already uses `io.Discard` for `buildSources()`. No change needed — this
command outputs FZF-formatted lines only.

### 6. `GitHubSource` logger — `internal/repos/github.go`

No changes to `GitHubSource` itself. The `SetLogger()` / `io.Writer` pattern
already works. The debug writer is passed through `buildSources()` which calls
`ghSource.SetLogger(logger)`.

## Data Flow

```
CLI startup
  │
  ├── rootCmd.PersistentPreRunE
  │     ├── config.Load()          → cfg.Debug from file + env
  │     ├── --debug flag check     → override cfg.Debug if set
  │     └── store cfg in context
  │
  └── subcommand.RunE
        ├── cfg := configFromContext(ctx)
        ├── debugW := logging.DebugWriter(cfg.Debug)
        ├── fmt.Fprintf(debugW, "progress...")   → visible only if debug
        ├── fmt.Printf("⚠️ warning...")          → always visible
        └── buildSources(ctx, cfg, debugW)
              └── ghSource.SetLogger(debugW)
```

## Precedence Resolution

Precedence is handled in two stages:

1. **`config.Load()`** — file value loaded first, then env var overwrites it.
   This matches the existing pattern (`applyEnvOverrides` runs after
   `loadFromFile`).
2. **`PersistentPreRunE`** — CLI flag overwrites whatever `config.Load()`
   returned. Only override if the flag was explicitly passed (use Cobra's
   `cmd.Flags().Changed("debug")` to distinguish "not passed" from "passed
   as false").

## Test Boundaries

### Config tests — `internal/config/config_test.go` (new file)

Table-driven tests covering:
- Default `Debug` is `false`.
- Config file `DEBUG=1` sets `Debug` to `true`.
- Env var `TMUX_CLAUDE_MATRIX_DEBUG=true` overrides config file `DEBUG=0`.
- Env var takes effect when no config file is present.

### Logging tests — `internal/logging/logging_test.go` (new file)

- `DebugWriter(true)` returns `os.Stdout`.
- `DebugWriter(false)` returns `io.Discard`.

### CLI flag tests — `cmd/claude-matrix/main_test.go` or integration test

- `--debug` flag sets `cfg.Debug` to `true`.
- `--debug` overrides env var and config file.
- `-d` shorthand works.

### Behavior verification

- With debug off: progress messages do not appear on stdout.
- With debug on: progress messages appear on stdout.
- Warnings/errors always appear regardless of debug mode.

## Files Changed (summary)

| File | Change |
|------|--------|
| `pkg/types/types.go` | Add `Debug bool` to `Config` |
| `internal/config/config.go` | Add `DEBUG` config key + env var |
| `internal/logging/logging.go` | New — `DebugWriter()` function |
| `internal/logging/logging_test.go` | New — tests for `DebugWriter()` |
| `internal/config/config_test.go` | New — tests for debug config loading |
| `cmd/claude-matrix/main.go` | Add `--debug` flag, `PersistentPreRunE`, context helpers |
| `cmd/claude-matrix/create.go` | Use debug writer for progress messages |
| `cmd/claude-matrix/sources.go` | Move warning to always-visible; callers pass debug writer |
| `cmd/claude-matrix/diagnose.go` | Report debug mode in output |
| `cmd/claude-matrix/refresh.go` | Use debug writer for progress messages |
| `cmd/claude-matrix/list.go` | Use debug writer for progress messages |

## Out of Scope

Per PRD: no structured logging, no log-to-file, no multi-level verbosity, no
timestamps/caller info, no changes to git command passthrough.

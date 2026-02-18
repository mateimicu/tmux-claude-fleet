# Implementation Spec: Debug Logging Configuration

**PRD**: `docs/prd/debug-logging.md` (PR #56)
**Status**: Draft

## Overview

Suppress progress/status output by default. Add a `Debug` flag (config file,
env var, CLI flag) that restores verbose output. Warnings and errors always
print regardless of debug mode.

The codebase already uses an `io.Writer` pattern in `buildSources()` and
`GitHubSource.SetLogger()` — passing `os.Stdout` for verbose output or
`io.Discard` to suppress. This spec extends that pattern globally through a
shared logging helper.

## 1. Config Layer

### 1.1 Add `Debug` field — `pkg/types/types.go`

Add `Debug bool` to the `Config` struct. Default: `false`.

### 1.2 Config file loading — `internal/config/config.go`

- `defaults()`: set `Debug: false` (explicit, matches zero-value).
- `applyConfigValue()`: add `case "DEBUG"` using the existing
  `value == "1" || value == "true"` pattern.

### 1.3 Env var loading — `internal/config/config.go`

- `applyEnvOverrides()`: read `TMUX_CLAUDE_MATRIX_DEBUG`, apply same
  boolean parsing as `GITHUB_ENABLED`.

### 1.4 Precedence

| Layer | Key / Var | Values | Precedence |
|-------|-----------|--------|------------|
| Config file | `DEBUG` | `1`, `true` | Lowest |
| Env var | `TMUX_CLAUDE_MATRIX_DEBUG` | `1`, `true` | Middle |
| CLI flag | `--debug` / `-d` | boolean | Highest |

The config file and env var are handled by `Load()` (file first, then env
overrides). The CLI flag override happens at the Cobra level (Section 3).

## 2. Logging Helper — `internal/logging/logging.go` (new package)

Minimal package providing a `Logger` with two writers:

```go
package logging

type Logger struct {
    Debug io.Writer   // writes only when debug enabled; io.Discard otherwise
    Warn  io.Writer   // always writes (os.Stderr)
}

func New(debug bool) *Logger
```

- `Debug` → `os.Stdout` when debug is true, `io.Discard` when false.
- `Warn` → `os.Stderr` always.
- Callers use `fmt.Fprintf(log.Debug, ...)` for progress messages and
  `fmt.Fprintf(log.Warn, ...)` for warnings/errors.
- No interfaces, no levels, no structured logging. Two writers.

The coding expert may optionally add `Debugf`/`Warnf` convenience methods
if it reduces boilerplate.

## 3. CLI Flag Integration — `cmd/claude-matrix/main.go`

### 3.1 Persistent flag

```
rootCmd.PersistentFlags().BoolP("debug", "d", false, "Enable debug logging")
```

### 3.2 `PersistentPreRunE`

Add a `PersistentPreRunE` on `rootCmd` that:
1. Loads config via `config.Load()`.
2. If `--debug` flag was explicitly set (`cmd.Flags().Changed("debug")`),
   overrides `cfg.Debug` with the flag value.
3. Creates `logging.New(cfg.Debug)`.
4. Stores `cfg` and the logger in the command's context using context key
   types.

### 3.3 Context helpers

Define unexported context key types and helper functions (in `main.go` or
a separate `context.go` — coding expert decides):

```go
func configFromContext(ctx context.Context) *types.Config
func loggerFromContext(ctx context.Context) *logging.Logger
```

Subcommands retrieve config and logger from context instead of calling
`config.Load()` directly.

## 4. Command-Level Changes

Each command switches from direct `fmt.Print*` to using the logger's writers.

### 4.1 `create.go`

- Get config and logger from context (remove `config.Load()` call).
- Pass `log.Debug` to `buildSources()`.
- Progress messages → `log.Debug`:
  - `"🔍 Discovering repositories..."`
  - `"✓ Found %d repositories"`
  - `"📦 Repository already exists at %s"`
  - `"📦 Cloning %s (using cache for faster cloning)..."`
  - `"✓ Clone complete"`
  - `"🚀 Creating tmux session '%s'..."`
  - `"✓ Session created: %s"`
  - Workspace equivalents
- Warnings → `log.Warn` (always visible):
  - `"⚠️  Failed to save session metadata: %v"`
  - `"⚠️  Failed to set session title env: %v"`
  - `"⚠️  Failed to switch to session: %v"`
  - `"You can attach manually with: tmux attach -t %s"`

### 4.2 `sources.go`

`buildSources()` already accepts an `io.Writer`. No signature change needed.
Callers pass the debug writer.

The `"⚠️ GitHub authentication not found"` message is a warning — the coding
expert should ensure it always prints (either change its write target from
`logger` to stderr, or have `buildSources()` accept the full `Logger`).

### 4.3 `refresh.go`

- Get config and logger from context.
- Progress → `log.Debug`: `"🔄 Refreshing..."`, `"✓ Cache refreshed..."`,
  cache location/TTL.
- Warning → `log.Warn`: `"⚠️ Failed to clear cache"`.
- Pass `log.Debug` to `buildSources()`.

### 4.4 `diagnose.go`

- `diagnose` is a diagnostic command — all its output is intentional.
  Existing messages stay always-visible (no routing through debug writer).
- **Add one line** to the configuration section: `"  Debug mode: true/false"`.

### 4.5 `list.go`

- Get config and logger from context.
- Progress messages → `log.Debug`.
- Warnings → `log.Warn`.
- Interactive UI prompts stay always-visible.

### 4.6 `list_repos.go`

- Already uses `io.Discard` for `buildSources()`. No functional change.
- Get config from context instead of calling `config.Load()`.

### 4.7 Other commands

`rename.go`, `hook_handler.go`, `setup_hooks.go`, `remove_hooks.go`,
`version.go` — retrieve config from context. Minimal or no logging changes.

## 5. Data Flow

```
CLI invocation
  │
  ▼
rootCmd.PersistentPreRunE
  ├── config.Load()           → cfg.Debug from file + env
  ├── --debug flag check      → override cfg.Debug if set
  ├── logging.New(cfg.Debug)  → Logger{Debug: stdout|discard, Warn: stderr}
  └── store cfg + logger in context
  │
  ▼
subcommand.RunE
  ├── cfg := configFromContext(ctx)
  ├── log := loggerFromContext(ctx)
  ├── fmt.Fprintf(log.Debug, "progress...")  → visible only if debug
  ├── fmt.Fprintf(log.Warn,  "warning...")   → always visible
  └── buildSources(ctx, cfg, log.Debug)
        └── ghSource.SetLogger(log.Debug)
```

## 6. Test Boundaries

### Config tests — `internal/config/config_test.go`

Table-driven tests:
- Default `Debug` is `false`.
- Config file `DEBUG=1` → `Debug` is `true`.
- Env var `TMUX_CLAUDE_MATRIX_DEBUG=true` overrides config file.
- Env var works when no config file present.

### Logging tests — `internal/logging/logging_test.go`

- `New(false).Debug` discards output; `New(false).Warn` writes.
- `New(true).Debug` writes; `New(true).Warn` writes.

### CLI flag tests

- `--debug` flag sets `cfg.Debug` to `true`.
- `--debug` overrides env var and config file.
- `-d` shorthand works.

### Behavior verification

- Debug off: progress messages absent from stdout.
- Debug on: progress messages present on stdout.
- Warnings/errors always present regardless of debug mode.

## 7. Files Changed

| File | Change |
|------|--------|
| `pkg/types/types.go` | Add `Debug bool` to `Config` |
| `internal/config/config.go` | Add `DEBUG` config key + env var |
| `internal/config/config_test.go` | New — tests for debug config loading |
| `internal/logging/logging.go` | New — `Logger` with `Debug`/`Warn` writers |
| `internal/logging/logging_test.go` | New — tests for `Logger` |
| `cmd/claude-matrix/main.go` | `--debug` flag, `PersistentPreRunE`, context helpers |
| `cmd/claude-matrix/create.go` | Use logger for progress/warnings |
| `cmd/claude-matrix/sources.go` | Callers pass debug writer; warning always visible |
| `cmd/claude-matrix/diagnose.go` | Report debug mode in output |
| `cmd/claude-matrix/refresh.go` | Use logger for progress/warnings |
| `cmd/claude-matrix/list.go` | Use logger for progress/warnings |
| `cmd/claude-matrix/list_repos.go` | Get config from context |

## 8. Out of Scope

Per PRD: no structured logging, no log-to-file, no multi-level verbosity,
no timestamps/caller info, no changes to git command passthrough.

# Implementation Spec: Debug Logging Configuration

**PRD**: `docs/prd/debug-logging.md` (PR #56)
**Branch**: `spec/debug-logging` (stacked on `prd/debug-logging`)

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

Minimal package providing a `Logger` struct with two `io.Writer` fields:

```go
package logging

type Logger struct {
    DebugW io.Writer   // writes only when debug enabled; io.Discard otherwise
    WarnW  io.Writer   // always writes (os.Stderr)
}

func New(debug bool) *Logger
```

- `DebugW` → `os.Stdout` when debug is true, `io.Discard` when false.
- `WarnW` → `os.Stderr` always.
- Callers use `fmt.Fprintf(log.DebugW, ...)` for progress messages and
  `fmt.Fprintf(log.WarnW, ...)` for warnings/errors.
- No interfaces, no levels, no structured logging. Two writers.

**Naming note**: Fields are suffixed with `W` (for "writer") to avoid
ambiguity with potential method names (e.g., a `Debug()` convenience method).

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

**Chaining caveat**: Cobra's `PersistentPreRunE` does not chain — if a
subcommand defines its own `PersistentPreRunE`, it shadows the parent's.
Currently no subcommands define one. If a future subcommand needs its own
`PersistentPreRunE`, it must explicitly call the parent's first (or the root
command's logic must be factored into a reusable function). The coding expert
should add a code comment noting this.

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
- Pass `log.DebugW` to `buildSources()`.
- Progress messages → `log.DebugW`:
  - `"🔍 Discovering repositories..."`
  - `"✓ Found %d repositories"`
  - `"📦 Repository already exists at %s"`
  - `"📦 Cloning %s (using cache for faster cloning)..."`
  - `"✓ Clone complete"`
  - `"🚀 Creating tmux session '%s'..."`
  - `"✓ Session created: %s"`
  - Workspace equivalents
- Warnings → `log.WarnW` (always visible):
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
- Progress → `log.DebugW`: `"🔄 Refreshing..."`, `"✓ Cache refreshed..."`,
  cache location/TTL.
- Warning → `log.WarnW`: `"⚠️ Failed to clear cache"`.
- Pass `log.DebugW` to `buildSources()`.

### 4.4 `diagnose.go`

- `diagnose` is a diagnostic command — all its output is intentional.
  Existing messages stay always-visible (no routing through debug writer).
- **Add one line** to the configuration section: `"  Debug mode: true/false"`.

### 4.5 `list.go`

- Get config and logger from context.
- Progress messages → `log.DebugW`.
- Warnings → `log.WarnW`.
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
  ├── logging.New(cfg.Debug)  → Logger{DebugW: stdout|discard, WarnW: stderr}
  └── store cfg + logger in context
  │
  ▼
subcommand.RunE
  ├── cfg := configFromContext(ctx)
  ├── log := loggerFromContext(ctx)
  ├── fmt.Fprintf(log.DebugW, "progress...")  → visible only if debug
  ├── fmt.Fprintf(log.WarnW,  "warning...")   → always visible
  └── buildSources(ctx, cfg, log.DebugW)
        └── ghSource.SetLogger(log.DebugW)
```

## 6. Test Strategy

### 6.1 Test harness approach

- **Unit tests** (`internal/logging`, `internal/config`): Use `bytes.Buffer`
  as the `io.Writer` target. Write to the buffer via the logger, then assert
  on `buf.String()`. No subprocess needed.
- **CLI integration tests** (`cmd/claude-matrix`): Use `exec.Command` to run
  the compiled binary as a subprocess, capture stdout/stderr, and assert on
  output. This tests the full flag → config → logger → output pipeline.

### 6.2 Coverage targets

- `internal/logging` — ~90% (small package, easy to cover fully).
- `internal/config` additions — ~90% (table-driven tests for each
  config/env/default path).
- CLI integration — ~70–80% (subprocess tests cover the main flag paths;
  full command workflows may not all be feasible in CI).

### 6.3 Config tests — `internal/config/config_test.go`

Table-driven tests using temp config files and `t.Setenv()`:

| # | Test case | Setup | Expected |
|---|-----------|-------|----------|
| 1 | Default debug is false | No config file, no env var | `cfg.Debug == false` |
| 2 | Config file `DEBUG=1` | Temp file with `DEBUG=1` | `cfg.Debug == true` |
| 3 | Config file `DEBUG=true` | Temp file with `DEBUG=true` | `cfg.Debug == true` |
| 4 | Config file `DEBUG=0` explicit disable | Temp file with `DEBUG=0` | `cfg.Debug == false` |
| 5 | Env var overrides config file | File `DEBUG=0` + env `TMUX_CLAUDE_MATRIX_DEBUG=1` | `cfg.Debug == true` |
| 6 | Env var alone (no config file) | Env `TMUX_CLAUDE_MATRIX_DEBUG=true` | `cfg.Debug == true` |
| 7 | Empty env var is ignored | Env `TMUX_CLAUDE_MATRIX_DEBUG=""` | `cfg.Debug == false` (the `val != ""` guard in `applyEnvOverrides` skips empty strings, preserving file/default value) |

### 6.4 Logging tests — `internal/logging/logging_test.go`

Use `bytes.Buffer` to verify writer behavior:

| # | Test case | Expected |
|---|-----------|----------|
| 1 | `New(false).DebugW` discards | Write to `DebugW` → buffer empty |
| 2 | `New(false).WarnW` writes | Write to `WarnW` → buffer has content |
| 3 | `New(true).DebugW` writes | Write to `DebugW` → buffer has content |
| 4 | `New(true).WarnW` writes | Write to `WarnW` → buffer has content |

Note: For testability, the coding expert should allow injecting writers into
`New()` (e.g., via functional options or a `NewWithWriters` variant) so tests
can capture output in `bytes.Buffer` instead of relying on `os.Stdout`.

### 6.5 CLI flag tests — subprocess integration

Build the binary, run as subprocess with `exec.Command`, capture output:

| # | Test case | Invocation | Expected |
|---|-----------|------------|----------|
| 1 | `--debug` enables verbose | `claude-matrix create --debug` | stdout contains progress messages |
| 2 | `-d` shorthand works | `claude-matrix create -d` | same as `--debug` |
| 3 | `--debug` overrides `DEBUG=0` in config | Config file `DEBUG=0` + `--debug` flag | `cfg.Debug == true`, progress visible (PRD criterion #5) |
| 4 | No flag = silent by default | `claude-matrix create` | stdout has no progress messages |

### 6.6 Behavior verification tests

| # | Test case | Expected |
|---|-----------|----------|
| 1 | Debug off: progress suppressed | Run `create` without `--debug`: no "Discovering repositories" / "Found N" / "Cloning" messages on stdout |
| 2 | Debug on: progress visible | Run `create --debug`: all progress messages present on stdout |
| 3 | Warnings always visible (debug off) | Run `create` without `--debug`: warning messages (`⚠️`) still appear |
| 4 | GitHub auth warning always visible | With `GITHUB_ENABLED=true`, no token, debug off: `"⚠️  GitHub authentication not found"` appears on stdout/stderr (PRD requirement #6) |
| 5 | `diagnose` reports debug mode | Run `diagnose`: output contains `"Debug mode: false"` (or `true` with `--debug`) (PRD criterion #8) |

## 7. Files Changed

| File | Change |
|------|--------|
| `pkg/types/types.go` | Add `Debug bool` to `Config` |
| `internal/config/config.go` | Add `DEBUG` config key + env var |
| `internal/config/config_test.go` | New — tests for debug config loading |
| `internal/logging/logging.go` | New — `Logger` struct with `DebugW`/`WarnW` writers |
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

# PRD: Debug Logging Configuration

## Goal

Reduce UI clutter in the `claude-matrix` CLI by suppressing progress and status
log lines by default. Introduce a debug mode that, when enabled, restores
verbose output for troubleshooting. Debug mode is configurable via config file,
environment variable, or per-command CLI flag.

## Background

The CLI currently prints progress messages (repository discovery, clone status,
session creation) directly to stdout on every invocation. These messages are
useful during development and troubleshooting but create noise during normal
interactive use, especially with the FZF selection flow where the user only
cares about picking a repo and getting a session.

## Requirements

1. Add a `Debug` boolean field to the `Config` struct (default: `false`).
2. Support setting debug mode via config file key `DEBUG` (`1`/`true` to
   enable).
3. Support setting debug mode via environment variable
   `TMUX_CLAUDE_MATRIX_DEBUG` (`1`/`true` to enable).
4. Add a global `--debug` / `-d` persistent flag on the root Cobra command.
   The CLI flag takes precedence over the config file and env var.
5. When debug mode is **off** (default), suppress all progress/status messages
   across all commands. Progress messages are informational lines such as:
   - "Discovering repositories..."
   - "Found N repositories"
   - "Cloning ... (using cache for faster cloning)..."
   - "Clone complete"
   - "Creating tmux session '...'..."
   - "Session created: ..."
   - "Setting up workspace '...' with N repos..."
   - GitHub source status lines ("Using cached GitHub repos", "Fetching
     GitHub repos from API", "Cached N repos for future use")
6. When debug mode is **off**, **always** show warnings and errors. Lines that
   must remain visible regardless of debug mode include:
   - "Failed to save session metadata: ..."
   - "Failed to set session title env: ..."
   - "Failed to switch to session: ..."
   - "You can attach manually with: tmux attach -t ..."
   - "GitHub authentication not found, skipping GitHub repositories"
   - Any fatal error output
7. When debug mode is **on**, print all messages (current behaviour preserved).
8. Implement a shared logging helper (e.g., a `debugf`/`debugln` writer or
   function) that checks the debug flag and writes to stdout only when enabled.
   Avoid scattering `if cfg.Debug` checks across every call site.
9. Precedence order: CLI flag > environment variable > config file > default
   (`false`).

## Acceptance Criteria

- [ ] Running `claude-matrix create` with default config produces no
      progress/status output (only FZF picker and any warnings/errors).
- [ ] Running `claude-matrix create --debug` shows all progress/status output
      matching current behaviour.
- [ ] Setting `DEBUG=1` in the config file enables verbose output without the
      CLI flag.
- [ ] Setting `TMUX_CLAUDE_MATRIX_DEBUG=1` enables verbose output without the
      CLI flag or config file entry.
- [ ] The `--debug` CLI flag overrides a `DEBUG=0` config file setting.
- [ ] Warning and error messages are always printed regardless of debug mode.
- [ ] The `--debug` flag is available on all commands (global persistent flag).
- [ ] The `diagnose` command reports the effective debug mode in its output.
- [ ] Existing tests pass; new tests cover the debug config loading
      (config file, env var, flag precedence).

## Out of Scope

- Structured logging (e.g., JSON log output).
- Log-to-file support.
- Multiple verbosity levels (e.g., `--verbose`, `-vvv`).
- Timestamps or caller information in log output.
- Changes to git command stdout/stderr passthrough behaviour.

# tmux-claude-matrix

Go CLI tool for managing tmux development sessions with Claude AI state tracking.
Manages tmux sessions tied to git repo clones. Three repo sources (GitHub API,
local file, YAML workspaces). Tracks Claude Code state per session via hooks
(running, idle, waiting, stopped, error). Interactive FZF UI for selection.

## Build & Test
* `make test` - run tests with race detector and coverage
* `make lint` - run golangci-lint (must be installed)
* `make fmt` - format with gofmt + goimports
* `make check` - lint + test
* `make ci` - fmt + lint + test + build (full CI locally)
* `make build` - build binary to ./bin/claude-matrix

## Project Structure
* `cmd/claude-matrix/` - Cobra CLI commands (create, list, rename, refresh, diagnose, hooks)
* `internal/config/` - Configuration loading (env > files > defaults)
* `internal/repos/` - Repository discovery (GitHub API, local file, workspaces)
* `internal/session/` - Session metadata persistence (JSON)
* `internal/git/` - Git clone operations with mirror cache
* `internal/tmux/` - tmux session/window management
* `internal/fzf/` - Interactive selection UI
* `internal/hooks/` - Claude event-to-state mapping (handles multi-agent via per-agent state files)
* `internal/status/` - Session state tracking with per-agent files and attention-priority aggregate
* `pkg/types/` - Exported types (Repository, Session, Config, ClaudeState)

## Code Patterns
* Go module: `github.com/mateimicu/tmux-claude-matrix`
* Go 1.23; CI tests on 1.21, 1.22, 1.23 (macOS + Linux)
* Table-driven tests with `t.Run()` subtests
* Temp files/dirs for test fixtures (`os.CreateTemp`, `os.MkdirTemp`)
* Interface-based repo discovery (`Source` interface in internal/repos/)
* golangci-lint config in `.golangci.yml` — errcheck, revive, gocritic enabled
* goimports local prefix: `github.com/mateimicu/tmux-claude-matrix`
* Version injected via LDFLAGS from `git describe --tags`

## Multi-Agent Status Tracking
* In-process Claude teammates share a tmux pane but have unique `session_id` values
* Per-agent state files: `{sessionName}.agent.{sessionID}.state`
* Aggregate state file: `{sessionName}.state` (backward-compatible, read by old binaries)
* Attention-priority order: `error` > `waiting_for_input` > `running` > `idle` > `stopped` > `unknown`
* `DefaultStaleThreshold` = 10 minutes — stale per-agent files are excluded and cleaned up
* `sanitizeAgentID()` prevents path traversal via `filepath.Base()`
* Sub-agents (Task tool) share parent's `session_id` — transparent, no special handling
* Tmux-mode teammates have separate panes — already handled by existing per-session state files

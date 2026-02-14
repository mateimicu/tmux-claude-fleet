# Session Titles Design

## Problem

Users managing multiple tmux-claude-matrix sessions cannot easily identify which session they are currently in. The tmux session name is an auto-generated sanitized slug (e.g., `mateimicu-tmux-claude-matrix`) which is not always immediately recognizable.

## Solution

Add a human-readable `Title` field to each session that is displayed in the tmux status bar via a session-level environment variable.

## Data Model

Add `Title` to `types.Session`:

```go
type Session struct {
    CreatedAt time.Time `json:"created_at"`
    Name      string    `json:"name"`
    Title     string    `json:"title"`
    RepoURL   string    `json:"repo_url"`
    ClonePath string    `json:"clone_path"`
}
```

## Auto-Generated Title Format

Default: `org/repo #N` where N is the session count for that repo at creation time.

Example: `mateimicu/tmux-claude-matrix #1`

Fallback for sessions without a title: use session `Name`.

## Tmux Integration

The plugin sets a session-level environment variable:

```
tmux set-environment -t <session> @claude-matrix-title "<title>"
```

Users add `#{@claude-matrix-title}` to their tmux `status-right` (or `status-left`) to display it.

## Commands

### Create (modified)

After repo selection, compute default title and set the tmux env var.

### Rename (new)

`claude-matrix rename [new-title]` -- updates title in metadata and tmux env var for the current session. If no argument, prompts interactively.

## FZF List Display

The session list will show the title alongside existing status indicators.

## Backwards Compatibility

Old sessions with empty `Title` fall back to displaying the session `Name`. No migration needed -- Go's JSON unmarshalling leaves missing fields as zero values.

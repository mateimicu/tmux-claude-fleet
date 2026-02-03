#!/usr/bin/env bash
set -e

# Get plugin directory
PLUGIN_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Source all required libraries
source "$PLUGIN_DIR/scripts/lib/common.sh"
source "$PLUGIN_DIR/scripts/lib/config.sh"
source "$PLUGIN_DIR/scripts/lib/session.sh"
source "$PLUGIN_DIR/scripts/lib/repos.sh"
source "$PLUGIN_DIR/scripts/lib/fzf.sh"
source "$PLUGIN_DIR/scripts/lib/git.sh"
source "$PLUGIN_DIR/scripts/lib/tmux.sh"

# Cleanup on failure
cleanup_on_failure() {
    if [ -n "$SESSION_NAME" ]; then
        tmux_kill_session "$SESSION_NAME" 2>/dev/null || true
    fi
    if [ -n "$CLONE_PATH" ] && [ -d "$CLONE_PATH" ]; then
        rm -rf "$CLONE_PATH"
    fi
    if [ -n "$META_FILE" ] && [ -f "$META_FILE" ]; then
        rm -f "$META_FILE"
    fi
}
trap cleanup_on_failure ERR

# Acquire lock to prevent concurrent creates
LOCK_FILE="$HOME/.tmux-claude-fleet/.lock"
if ! acquire_lock "$LOCK_FILE"; then
    echo "Error: Another operation is in progress" >&2
    exit 1
fi
trap "release_lock '$LOCK_FILE'" EXIT

# Load configuration
if ! config_load; then
    echo "Error: Failed to load configuration" >&2
    exit 1
fi

# Check dependencies
if ! check_dependencies; then
    exit 1
fi

# Gather repos from all enabled sources
echo "Loading repositories..."
repo_list=$(repos_list_all) || {
    echo "Error: No repositories found" >&2
    echo "Please configure repositories in $HOME/.tmux-claude-fleet/repos.txt" >&2
    echo "or enable GitHub integration" >&2
    exit 1
}

# Present fzf for selection
selected=$(echo "$repo_list" | repos_format_for_fzf | fzf_select_repo) || {
    echo "No repository selected"
    exit 0
}

# Parse selected repo URL
repo_url=$(echo "$selected" | cut -d'|' -f1)
repo_name=$(git_get_repo_name "$repo_url")

if [ -z "$repo_name" ]; then
    echo "Error: Could not determine repository name" >&2
    exit 1
fi

# Generate unique session name
base_name=$(sanitize_name "$repo_name")
SESSION_NAME=$(session_generate_unique_name "$base_name")
CLONE_DIR=$(config_get "CLONE_DIR")
CLONE_PATH="$CLONE_DIR/$SESSION_NAME"

echo "Creating session: $SESSION_NAME"
echo "Cloning to: $CLONE_PATH"

# Clone repository
if ! git_clone "$repo_url" "$CLONE_PATH"; then
    echo "Error: Failed to clone repository" >&2
    exit 1
fi

# Create tmux session
echo "Creating tmux session..."
if ! tmux_create_session "$SESSION_NAME" "$CLONE_PATH"; then
    echo "Error: Failed to create tmux session" >&2
    exit 1
fi

# Kill the default first window
tmux kill-window -t "$SESSION_NAME:0" 2>/dev/null || true

# Create terminal window
echo "Creating terminal window..."
if ! tmux_create_window "$SESSION_NAME" "terminal" "" "$CLONE_PATH"; then
    echo "Error: Failed to create terminal window" >&2
    exit 1
fi

# Create Claude window if Claude is available
CLAUDE_BIN=$(config_get "CLAUDE_BIN")
if [ -n "$CLAUDE_BIN" ] && [ -x "$CLAUDE_BIN" ]; then
    echo "Creating Claude window..."
    CLAUDE_ARGS=$(config_get "CLAUDE_ARGS")
    if ! tmux_create_window "$SESSION_NAME" "claude" "$CLAUDE_BIN $CLAUDE_ARGS" "$CLONE_PATH"; then
        echo "Warning: Failed to create Claude window" >&2
    fi
else
    echo "Warning: Claude CLI not found, skipping Claude window"
fi

# Save session metadata
META_FILE=$(_get_metadata_file "$SESSION_NAME")
if ! session_save_metadata "$SESSION_NAME" "$repo_url" "$CLONE_PATH"; then
    echo "Error: Failed to save session metadata" >&2
    exit 1
fi

echo "Session created successfully: $SESSION_NAME"

# Switch to session
echo "Switching to session..."
tmux_switch_to_session "$SESSION_NAME"

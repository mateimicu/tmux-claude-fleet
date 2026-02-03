#!/usr/bin/env bash
set -e

# Get plugin directory
PLUGIN_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Source all required libraries
source "$PLUGIN_DIR/scripts/lib/common.sh"
source "$PLUGIN_DIR/scripts/lib/config.sh"
source "$PLUGIN_DIR/scripts/lib/session.sh"
source "$PLUGIN_DIR/scripts/lib/fzf.sh"
source "$PLUGIN_DIR/scripts/lib/tmux.sh"

# Load configuration
config_load || exit 1

# Get all fleet sessions
sessions=$(session_list)

if [ -z "$sessions" ]; then
    echo "No sessions found"
    echo ""
    echo "Create a new session with: tmux-claude-fleet create"
    echo "or press your configured keybinding (default: prefix + C)"
    exit 0
fi

# Format for fzf: session_name|status|created|path
formatted=""
for session in $sessions; do
    # Get session status
    if tmux_session_exists "$session"; then
        claude_st=$(claude_status "$session")
        if [ "$claude_st" = "running" ]; then
            status="✓ Active"
        else
            status="⚠ Claude stopped"
        fi
    else
        status="✗ Stopped"
    fi

    # Load metadata
    meta=$(session_load_metadata "$session")
    created=$(echo "$meta" | grep '^CREATED_AT=' | cut -d'=' -f2- | tr -d '"')
    path=$(echo "$meta" | grep '^CLONE_PATH=' | cut -d'=' -f2- | tr -d '"')

    if [ -z "$formatted" ]; then
        formatted="$session|$status|$created|$path"
    else
        formatted="$formatted"$'\n'"$session|$status|$created|$path"
    fi
done

# Present fzf browser
selected=$(echo "$formatted" | fzf_select_session) || {
    echo "No session selected"
    exit 0
}

# Handle actions
if [[ "$selected" == delete:* ]]; then
    # Ctrl-D pressed - delete session
    session_name=$(echo "$selected" | sed 's/^delete://' | cut -d'|' -f1)
    exec "$PLUGIN_DIR/scripts/delete-session.sh" "$session_name"
else
    # Normal selection - switch to session
    session_name=$(echo "$selected" | cut -d'|' -f1)
    echo "Switching to session: $session_name"
    tmux_switch_to_session "$session_name"
fi

#!/usr/bin/env bash
set -e

# Get plugin directory
PLUGIN_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Source all required libraries
source "$PLUGIN_DIR/scripts/lib/common.sh"
source "$PLUGIN_DIR/scripts/lib/config.sh"
source "$PLUGIN_DIR/scripts/lib/session.sh"
source "$PLUGIN_DIR/scripts/lib/tmux.sh"

# Load configuration
config_load || exit 1

# Get session name from argument
session_name="$1"

if [ -z "$session_name" ]; then
    echo "Error: Session name required" >&2
    echo "Usage: $0 SESSION_NAME" >&2
    exit 1
fi

# Check if session exists
if ! session_exists "$session_name"; then
    echo "Error: Session not found: $session_name" >&2
    exit 1
fi

# Confirm deletion
echo "Delete session '$session_name'?"
read -p "Continue? (y/n) " -n 1 -r
echo

if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Cancelled"
    exit 0
fi

# Kill tmux session if running
if tmux_session_exists "$session_name"; then
    echo "Killing tmux session..."
    tmux_kill_session "$session_name"
fi

# Load metadata to get clone path
meta=$(session_load_metadata "$session_name")
clone_path=$(echo "$meta" | grep '^CLONE_PATH=' | cut -d'=' -f2- | tr -d '"')

# Ask about repository cleanup
if [ -d "$clone_path" ]; then
    echo ""
    echo "Delete cloned repository at: $clone_path"
    read -p "Continue? (y/n) " -n 1 -r
    echo

    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo "Deleting repository..."
        rm -rf "$clone_path"
        echo "Repository deleted"
    fi
fi

# Remove session metadata
echo "Removing session metadata..."
session_delete "$session_name"

echo "Session deleted successfully: $session_name"

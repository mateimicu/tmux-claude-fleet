#!/usr/bin/env bash
# Tmux Claude Fleet Plugin
# Entry point for tmux plugin manager

CURRENT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
BINARY="$CURRENT_DIR/bin/claude-fleet"

# Get tmux option value
get_tmux_option() {
    local option="$1"
    local default_value="$2"
    local option_value=$(tmux show-option -gqv "$option")
    if [ -z "$option_value" ]; then
        echo "$default_value"
    else
        echo "$option_value"
    fi
}

# Check if binary exists
if [ ! -x "$BINARY" ]; then
    tmux display-message "claude-fleet: Binary not found. Run: make install"
    exit 1
fi

# Get keybindings
create_key=$(get_tmux_option "@claude-fleet-create-key" "C")
list_key=$(get_tmux_option "@claude-fleet-list-key" "L")
delete_key=$(get_tmux_option "@claude-fleet-delete-key" "D")

# Bind keys using new-window
tmux bind-key "$create_key" new-window "$BINARY create"
tmux bind-key "$list_key" new-window "$BINARY list"
tmux bind-key "$delete_key" new-window "$BINARY delete"

tmux display-message "claude-fleet: Plugin loaded (Go version)"

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

# Auto-build binary if it doesn't exist
if [ ! -x "$BINARY" ]; then
    # Check if Go is installed
    if ! command -v go >/dev/null 2>&1; then
        tmux display-message "claude-fleet: Go not found. Install Go or download binary from releases."
        exit 1
    fi

    # Check if we have the source code
    if [ ! -f "$CURRENT_DIR/go.mod" ]; then
        tmux display-message "claude-fleet: Source code not found. Clone the full repository."
        exit 1
    fi

    tmux display-message "claude-fleet: Building binary (first time only)..."

    # Build the binary in the background
    (
        cd "$CURRENT_DIR" && make build >/dev/null 2>&1
        if [ $? -eq 0 ]; then
            tmux display-message "claude-fleet: Build successful! Reload tmux config to activate."
        else
            tmux display-message "claude-fleet: Build failed. Run 'make build' manually to see errors."
        fi
    ) &

    exit 0
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

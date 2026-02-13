#!/usr/bin/env bash
# Tmux Claude Matrix Plugin
# Entry point for tmux plugin manager

CURRENT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
BINARY="$CURRENT_DIR/bin/claude-matrix"

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
        tmux display-message "claude-matrix: Go not found. Install Go or download binary from releases."
        exit 1
    fi

    # Check if we have the source code
    if [ ! -f "$CURRENT_DIR/go.mod" ]; then
        tmux display-message "claude-matrix: Source code not found. Clone the full repository."
        exit 1
    fi

    tmux display-message "claude-matrix: Building binary (first time only)..."

    # Build the binary in the background
    (
        cd "$CURRENT_DIR" && make build >/dev/null 2>&1
        if [ $? -eq 0 ]; then
            tmux display-message "claude-matrix: Build successful! Reload tmux config to activate."
        else
            tmux display-message "claude-matrix: Build failed. Run 'make build' manually to see errors."
        fi
    ) &

    exit 0
fi

# Get keybindings
create_key=$(get_tmux_option "@claude-matrix-create-key" "a")
list_key=$(get_tmux_option "@claude-matrix-list-key" "A")
delete_key=$(get_tmux_option "@claude-matrix-delete-key" "D")
use_popup=$(get_tmux_option "@claude-matrix-use-popup" "true")

# Bind keys using popup or new-window
if [ "$use_popup" = "true" ]; then
    tmux bind-key "$create_key" display-popup -w 80% -h 80% -E "$BINARY create"
    tmux bind-key "$list_key" display-popup -w 80% -h 80% -E "$BINARY list"
    tmux bind-key "$delete_key" display-popup -w 80% -h 80% -E "$BINARY delete"
else
    tmux bind-key "$create_key" new-window "$BINARY create"
    tmux bind-key "$list_key" new-window "$BINARY list"
    tmux bind-key "$delete_key" new-window "$BINARY delete"
fi

tmux display-message "claude-matrix: Plugin loaded (Go version)"

#!/usr/bin/env bash
# Tmux Claude Fleet Plugin
# Entry point for tmux plugin manager

CURRENT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

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

# Check tmux version
check_tmux_version() {
    local version=$(tmux -V | cut -d' ' -f2 | cut -d'-' -f1)
    local major=$(echo "$version" | cut -d'.' -f1)
    local minor=$(echo "$version" | cut -d'.' -f2)

    # Return 0 if version >= 3.2, 1 otherwise
    if [ "$major" -gt 3 ] || ([ "$major" -eq 3 ] && [ "$minor" -ge 2 ]); then
        return 0
    else
        return 1
    fi
}

# Get keybindings from options
create_key=$(get_tmux_option "@claude-fleet-create-key" "C")
list_key=$(get_tmux_option "@claude-fleet-list-key" "L")

# Check if tmux supports display-popup (3.2+)
if check_tmux_version; then
    # Use display-popup for tmux 3.2+
    tmux bind-key "$create_key" display-popup -E -w 80% -h 80% \
        "$CURRENT_DIR/scripts/create-session.sh"

    tmux bind-key "$list_key" display-popup -E -w 80% -h 80% \
        "$CURRENT_DIR/scripts/list-sessions.sh"
else
    # Fall back to new-window for older tmux versions
    tmux bind-key "$create_key" new-window \
        "$CURRENT_DIR/scripts/create-session.sh"

    tmux bind-key "$list_key" new-window \
        "$CURRENT_DIR/scripts/list-sessions.sh"

    # Show warning message
    tmux display-message "tmux-claude-fleet: Using new-window mode (tmux 3.2+ recommended for popup support)"
fi

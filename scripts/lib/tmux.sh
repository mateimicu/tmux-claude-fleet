#!/usr/bin/env bash
# Tmux operations for tmux-claude-fleet

# Source common utilities
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

# Check tmux version
# Usage: tmux_version_check
tmux_version_check() {
    local version=$(tmux -V | cut -d' ' -f2 | cut -d'-' -f1)
    local major=$(echo "$version" | cut -d'.' -f1)
    local minor=$(echo "$version" | cut -d'.' -f2)

    # Return version as comparable number (e.g., 3.2 -> 302)
    echo $(( major * 100 + minor ))
}

# Check if tmux supports display-popup (version >= 3.2)
# Usage: tmux_has_popup
tmux_has_popup() {
    local version=$(tmux_version_check)
    [ "$version" -ge 302 ]
}

# Check if tmux session exists
# Usage: tmux_session_exists NAME
tmux_session_exists() {
    local name="$1"

    if [ -z "$name" ]; then
        return 1
    fi

    tmux has-session -t "$name" 2>/dev/null
}

# Create new tmux session
# Usage: tmux_create_session NAME PATH
tmux_create_session() {
    local name="$1"
    local path="$2"

    if [ -z "$name" ] || [ -z "$path" ]; then
        log "$LOG_ERROR" "tmux_create_session: name and path are required"
        return 1
    fi

    if tmux_session_exists "$name"; then
        log "$LOG_ERROR" "tmux_create_session: session already exists: $name"
        return 1
    fi

    if [ ! -d "$path" ]; then
        log "$LOG_ERROR" "tmux_create_session: path does not exist: $path"
        return 1
    fi

    log "$LOG_INFO" "Creating tmux session: $name"
    tmux new-session -d -s "$name" -c "$path"
    return $?
}

# Create window in tmux session
# Usage: tmux_create_window SESSION NAME COMMAND PATH
tmux_create_window() {
    local session="$1"
    local name="$2"
    local command="$3"
    local path="$4"

    if [ -z "$session" ] || [ -z "$name" ]; then
        log "$LOG_ERROR" "tmux_create_window: session and name are required"
        return 1
    fi

    if ! tmux_session_exists "$session"; then
        log "$LOG_ERROR" "tmux_create_window: session does not exist: $session"
        return 1
    fi

    log "$LOG_INFO" "Creating window '$name' in session $session"

    # Build tmux command
    local tmux_cmd="tmux new-window -t $session: -n $name"

    if [ -n "$path" ] && [ -d "$path" ]; then
        tmux_cmd="$tmux_cmd -c $path"
    fi

    if [ -n "$command" ]; then
        tmux_cmd="$tmux_cmd '$command'"
    fi

    eval "$tmux_cmd"
    return $?
}

# Get PIDs of all panes in a window
# Usage: tmux_get_pane_pids SESSION WINDOW
tmux_get_pane_pids() {
    local session="$1"
    local window="$2"

    if [ -z "$session" ] || [ -z "$window" ]; then
        return 1
    fi

    if ! tmux_session_exists "$session"; then
        return 1
    fi

    tmux list-panes -t "$session:$window" -F "#{pane_pid}" 2>/dev/null
}

# Kill tmux session
# Usage: tmux_kill_session NAME
tmux_kill_session() {
    local name="$1"

    if [ -z "$name" ]; then
        log "$LOG_ERROR" "tmux_kill_session: name is required"
        return 1
    fi

    if ! tmux_session_exists "$name"; then
        log "$LOG_WARN" "tmux_kill_session: session does not exist: $name"
        return 0
    fi

    log "$LOG_INFO" "Killing tmux session: $name"
    tmux kill-session -t "$name"
    return $?
}

# Switch to tmux session (or attach if outside tmux)
# Usage: tmux_switch_to_session NAME
tmux_switch_to_session() {
    local name="$1"

    if [ -z "$name" ]; then
        log "$LOG_ERROR" "tmux_switch_to_session: name is required"
        return 1
    fi

    if ! tmux_session_exists "$name"; then
        log "$LOG_ERROR" "tmux_switch_to_session: session does not exist: $name"
        return 1
    fi

    if [ -n "$TMUX" ]; then
        # Inside tmux, switch to session
        tmux switch-client -t "$name"
    else
        # Outside tmux, attach to session
        tmux attach-session -t "$name"
    fi

    return $?
}

# Check if Claude is running in session
# Usage: claude_status SESSION_NAME
claude_status() {
    local session="$1"

    if [ -z "$session" ]; then
        echo "unknown"
        return 1
    fi

    if ! tmux_session_exists "$session"; then
        echo "stopped"
        return 1
    fi

    # Get PIDs from claude window (window 1 by default)
    local pids=$(tmux_get_pane_pids "$session" "claude" 2>/dev/null)

    if [ -z "$pids" ]; then
        echo "stopped"
        return 1
    fi

    # Check if any child process is claude
    for pid in $pids; do
        # Get child processes
        local children=$(pgrep -P "$pid" 2>/dev/null)
        for child_pid in $children; do
            local cmd=$(ps -p "$child_pid" -o comm= 2>/dev/null)
            if [[ "$cmd" =~ claude ]]; then
                echo "running"
                return 0
            fi
        done

        # Also check the pane's own process
        local cmd=$(ps -p "$pid" -o comm= 2>/dev/null)
        if [[ "$cmd" =~ claude ]]; then
            echo "running"
            return 0
        fi
    done

    echo "stopped"
    return 1
}

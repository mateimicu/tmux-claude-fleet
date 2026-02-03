#!/usr/bin/env bash
# Common utilities for tmux-claude-fleet

# Log levels
readonly LOG_ERROR="ERROR"
readonly LOG_WARN="WARN"
readonly LOG_INFO="INFO"
readonly LOG_DEBUG="DEBUG"

# Get log file path
_get_log_file() {
    echo "${TMUX_CLAUDE_FLEET_LOG_FILE:-$HOME/.tmux-claude-fleet/plugin.log}"
}

# Log message to file and optionally to tmux status
# Usage: log LEVEL MESSAGE
log() {
    local level="$1"
    shift
    local message="$*"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    local log_file=$(_get_log_file)

    # Ensure log directory exists
    mkdir -p "$(dirname "$log_file")"

    # Write to log file
    echo "[$timestamp] [$level] $message" >> "$log_file"

    # Show errors in tmux status
    if [ "$level" = "$LOG_ERROR" ] && [ -n "$TMUX" ]; then
        tmux display-message "tmux-claude-fleet: $message" 2>/dev/null || true
    fi
}

# Ensure directory exists with proper error handling
# Usage: ensure_dir PATH
ensure_dir() {
    local path="$1"

    if [ -z "$path" ]; then
        log "$LOG_ERROR" "ensure_dir: path is required"
        return 1
    fi

    if [ -e "$path" ] && [ ! -d "$path" ]; then
        log "$LOG_ERROR" "ensure_dir: path exists but is not a directory: $path"
        return 1
    fi

    if ! mkdir -p "$path" 2>/dev/null; then
        log "$LOG_ERROR" "ensure_dir: failed to create directory: $path"
        return 1
    fi

    return 0
}

# Sanitize name for tmux/filesystem use
# - Convert to lowercase
# - Replace non-alphanumeric with dash
# - Remove leading/trailing dashes
# - Collapse multiple dashes
# - Limit to 50 characters
# Usage: sanitize_name INPUT
sanitize_name() {
    local input="$1"

    if [ -z "$input" ]; then
        echo "unnamed"
        return 0
    fi

    # Convert to lowercase, replace non-alphanumeric with dash
    local sanitized=$(echo "$input" | tr '[:upper:]' '[:lower:]' | sed 's/[^a-z0-9]/-/g')

    # Remove leading/trailing dashes
    sanitized=$(echo "$sanitized" | sed 's/^-*//' | sed 's/-*$//')

    # Collapse multiple dashes
    sanitized=$(echo "$sanitized" | sed 's/-\+/-/g')

    # Limit to 50 characters
    sanitized=$(echo "$sanitized" | cut -c1-50)

    # Remove trailing dashes that might appear after truncation
    sanitized=$(echo "$sanitized" | sed 's/-*$//')

    # Fallback if empty after sanitization
    if [ -z "$sanitized" ]; then
        echo "unnamed"
    else
        echo "$sanitized"
    fi
}

# Acquire file-based lock with timeout
# Usage: acquire_lock LOCKFILE [TIMEOUT_SECONDS]
acquire_lock() {
    local lockfile="$1"
    local timeout="${2:-10}"
    local waited=0

    if [ -z "$lockfile" ]; then
        log "$LOG_ERROR" "acquire_lock: lockfile is required"
        return 1
    fi

    # Ensure lock directory exists
    ensure_dir "$(dirname "$lockfile")" || return 1

    while [ $waited -lt $timeout ]; do
        # Try to create lock file exclusively
        if mkdir "$lockfile" 2>/dev/null; then
            echo "$$" > "$lockfile/pid"
            log "$LOG_DEBUG" "Lock acquired: $lockfile"
            return 0
        fi

        # Check if lock is stale
        if [ -f "$lockfile/pid" ]; then
            local lock_pid=$(cat "$lockfile/pid" 2>/dev/null)
            if [ -n "$lock_pid" ] && ! kill -0 "$lock_pid" 2>/dev/null; then
                log "$LOG_WARN" "Removing stale lock from pid $lock_pid"
                rm -rf "$lockfile"
                continue
            fi
        fi

        sleep 1
        waited=$((waited + 1))
    done

    log "$LOG_ERROR" "Failed to acquire lock: $lockfile (timeout after ${timeout}s)"
    return 1
}

# Release file-based lock
# Usage: release_lock LOCKFILE
release_lock() {
    local lockfile="$1"

    if [ -z "$lockfile" ]; then
        log "$LOG_ERROR" "release_lock: lockfile is required"
        return 1
    fi

    if [ -d "$lockfile" ]; then
        # Verify we own the lock
        local lock_pid=$(cat "$lockfile/pid" 2>/dev/null)
        if [ "$lock_pid" = "$$" ]; then
            rm -rf "$lockfile"
            log "$LOG_DEBUG" "Lock released: $lockfile"
            return 0
        else
            log "$LOG_WARN" "Lock not owned by current process: $lockfile (owner: $lock_pid, current: $$)"
            return 1
        fi
    fi

    return 0
}

# Check for required dependencies
# Usage: check_dependencies
check_dependencies() {
    local missing=()

    for cmd in tmux fzf git; do
        if ! command -v "$cmd" &>/dev/null; then
            missing+=("$cmd")
        fi
    done

    if [ ${#missing[@]} -gt 0 ]; then
        log "$LOG_ERROR" "Missing required dependencies: ${missing[*]}"
        echo "Error: Missing required dependencies: ${missing[*]}" >&2
        echo "Please install them and try again." >&2
        return 1
    fi

    # Claude is optional but warn if missing
    if ! command -v claude &>/dev/null; then
        log "$LOG_WARN" "Claude CLI not found in PATH"
        echo "Warning: Claude CLI not found. Sessions will be created without Claude window." >&2
    fi

    return 0
}

#!/usr/bin/env bash
# Session operations for tmux-claude-fleet

# Guard against multiple sourcing
if [ -n "$TMUX_CLAUDE_FLEET_SESSION_LOADED" ]; then
    return 0
fi
readonly TMUX_CLAUDE_FLEET_SESSION_LOADED=1

# Source common utilities
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

# Get sessions directory
_get_sessions_dir() {
    echo "$HOME/.tmux-claude-fleet/sessions"
}

# Get metadata file path for session
_get_metadata_file() {
    local session_name="$1"
    echo "$(_get_sessions_dir)/$session_name.meta"
}

# Save session metadata
# Usage: session_save_metadata NAME REPO_URL CLONE_PATH
session_save_metadata() {
    local name="$1"
    local repo_url="$2"
    local clone_path="$3"

    if [ -z "$name" ] || [ -z "$repo_url" ] || [ -z "$clone_path" ]; then
        log "$LOG_ERROR" "session_save_metadata: all parameters required"
        return 1
    fi

    local sessions_dir=$(_get_sessions_dir)
    ensure_dir "$sessions_dir" || return 1

    local meta_file=$(_get_metadata_file "$name")
    local timestamp=$(date -u '+%Y-%m-%dT%H:%M:%S')

    cat > "$meta_file" << EOF
REPO_URL="$repo_url"
CLONE_PATH="$clone_path"
CREATED_AT="$timestamp"
EOF

    log "$LOG_INFO" "Session metadata saved: $name"
    return 0
}

# Load session metadata
# Usage: session_load_metadata NAME
session_load_metadata() {
    local name="$1"

    if [ -z "$name" ]; then
        log "$LOG_ERROR" "session_load_metadata: name is required"
        return 1
    fi

    local meta_file=$(_get_metadata_file "$name")

    if [ ! -f "$meta_file" ]; then
        log "$LOG_ERROR" "session_load_metadata: metadata not found for session: $name"
        return 1
    fi

    cat "$meta_file"
    return 0
}

# Check if session metadata exists
# Usage: session_exists NAME
session_exists() {
    local name="$1"

    if [ -z "$name" ]; then
        return 1
    fi

    local meta_file=$(_get_metadata_file "$name")
    [ -f "$meta_file" ]
}

# List all fleet sessions
# Usage: session_list
session_list() {
    local sessions_dir=$(_get_sessions_dir)

    if [ ! -d "$sessions_dir" ]; then
        return 0
    fi

    # List all .meta files, strip extension
    find "$sessions_dir" -maxdepth 1 -name "*.meta" -type f 2>/dev/null | \
        sed 's/\.meta$//' | \
        xargs -n 1 basename 2>/dev/null | \
        sort
}

# Delete session metadata
# Usage: session_delete NAME
session_delete() {
    local name="$1"

    if [ -z "$name" ]; then
        log "$LOG_ERROR" "session_delete: name is required"
        return 1
    fi

    local meta_file=$(_get_metadata_file "$name")

    if [ ! -f "$meta_file" ]; then
        log "$LOG_WARN" "session_delete: metadata not found for session: $name"
        return 0
    fi

    rm -f "$meta_file"
    log "$LOG_INFO" "Session metadata deleted: $name"
    return 0
}

# Generate unique session name
# Usage: session_generate_unique_name BASE_NAME
session_generate_unique_name() {
    local base_name="$1"

    if [ -z "$base_name" ]; then
        log "$LOG_ERROR" "session_generate_unique_name: base_name is required"
        return 1
    fi

    # Start with base name
    local candidate="$base_name"
    local counter=1

    # Check both metadata and tmux session existence
    while session_exists "$candidate" || ( command -v tmux &>/dev/null && tmux has-session -t "$candidate" 2>/dev/null ); do
        candidate="${base_name}-${counter}"
        counter=$((counter + 1))

        # Prevent infinite loop
        if [ $counter -gt 100 ]; then
            log "$LOG_ERROR" "session_generate_unique_name: too many sessions with base name: $base_name"
            return 1
        fi
    done

    echo "$candidate"
    return 0
}

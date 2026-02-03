#!/usr/bin/env bash
# Git operations for tmux-claude-fleet

# Guard against multiple sourcing
if [ -n "$TMUX_CLAUDE_FLEET_GIT_LOADED" ]; then
    return 0
fi
readonly TMUX_CLAUDE_FLEET_GIT_LOADED=1

# Source common utilities
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

# Clone git repository
# Usage: git_clone URL PATH
git_clone() {
    local url="$1"
    local path="$2"

    if [ -z "$url" ] || [ -z "$path" ]; then
        log "$LOG_ERROR" "git_clone: url and path are required"
        return 1
    fi

    # Validate URL
    if ! git_is_valid_url "$url"; then
        log "$LOG_ERROR" "git_clone: invalid git URL: $url"
        return 1
    fi

    # Check if path already exists
    if [ -e "$path" ]; then
        log "$LOG_ERROR" "git_clone: path already exists: $path"
        return 1
    fi

    # Ensure parent directory exists
    ensure_dir "$(dirname "$path")" || return 1

    # Clone repository
    log "$LOG_INFO" "Cloning $url to $path"
    if ! git clone "$url" "$path" 2>&1 | tee -a "$(_get_log_file)"; then
        log "$LOG_ERROR" "git_clone: failed to clone $url"
        # Clean up failed clone
        rm -rf "$path"
        return 1
    fi

    log "$LOG_INFO" "Successfully cloned $url"
    return 0
}

# Extract repository name from URL
# Usage: git_get_repo_name URL
git_get_repo_name() {
    local url="$1"

    if [ -z "$url" ]; then
        log "$LOG_ERROR" "git_get_repo_name: url is required"
        return 1
    fi

    # Remove trailing slash
    url="${url%/}"

    # Remove .git suffix
    url="${url%.git}"

    # Extract last path component
    local repo_name=$(basename "$url")

    if [ -z "$repo_name" ] || [ "$repo_name" = "." ] || [ "$repo_name" = ".." ]; then
        log "$LOG_ERROR" "git_get_repo_name: could not extract repo name from: $url"
        return 1
    fi

    echo "$repo_name"
    return 0
}

# Validate git URL format
# Usage: git_is_valid_url URL
git_is_valid_url() {
    local url="$1"

    if [ -z "$url" ]; then
        return 1
    fi

    # Check for HTTPS URLs
    if [[ "$url" =~ ^https?://[a-zA-Z0-9.-]+/[a-zA-Z0-9._/-]+ ]]; then
        return 0
    fi

    # Check for SSH URLs (git@host:path)
    if [[ "$url" =~ ^[a-zA-Z0-9._-]+@[a-zA-Z0-9.-]+:[a-zA-Z0-9._/-]+ ]]; then
        return 0
    fi

    # Check for git:// URLs
    if [[ "$url" =~ ^git://[a-zA-Z0-9.-]+/[a-zA-Z0-9._/-]+ ]]; then
        return 0
    fi

    # Check for local paths (starts with / or ./)
    if [[ "$url" =~ ^(/|\./) ]]; then
        return 0
    fi

    return 1
}

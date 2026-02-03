#!/usr/bin/env bash
# Repository discovery for tmux-claude-fleet

# Guard against multiple sourcing
if [ -n "$TMUX_CLAUDE_FLEET_REPOS_LOADED" ]; then
    return 0
fi
readonly TMUX_CLAUDE_FLEET_REPOS_LOADED=1

# Source common utilities and config
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"
source "$SCRIPT_DIR/config.sh"

# Read repositories from local file
# Usage: repos_from_local
repos_from_local() {
    local repos_file=$(config_get "LOCAL_REPOS_FILE")

    if [ ! -f "$repos_file" ]; then
        log "$LOG_DEBUG" "Local repos file not found: $repos_file"
        return 0
    fi

    log "$LOG_DEBUG" "Reading repos from: $repos_file"

    # Read file, skip comments and empty lines
    while IFS= read -r line; do
        # Skip comments and empty lines
        [[ "$line" =~ ^[[:space:]]*# ]] && continue
        [[ -z "$line" ]] && continue

        # Extract URL and optional description
        local url=$(echo "$line" | awk '{print $1}')
        local description=$(echo "$line" | sed -n 's/^[^#]*#[[:space:]]*//p')

        if [ -n "$url" ]; then
            echo "$url|$description"
        fi
    done < <(grep -v '^[[:space:]]*$' "$repos_file" 2>/dev/null)
}

# Fetch repositories from GitHub API
# Usage: repos_from_github
repos_from_github() {
    local cache_dir=$(config_get "CACHE_DIR")
    local cache_ttl=$(config_get "CACHE_TTL")
    local cache_file="$cache_dir/github-repos"

    ensure_dir "$cache_dir" || return 1

    # Check cache first
    if [ -f "$cache_file" ]; then
        # Get file age in seconds
        local file_time=$(stat -f%m "$cache_file" 2>/dev/null || stat -c%Y "$cache_file" 2>/dev/null)
        local current_time=$(date +%s)
        local age=$((current_time - file_time))

        if [ $age -lt $cache_ttl ]; then
            log "$LOG_DEBUG" "Using cached GitHub repos (age: ${age}s)"
            cat "$cache_file"
            return 0
        fi
    fi

    log "$LOG_INFO" "Fetching GitHub repositories"

    # Try gh CLI first
    if command -v gh &>/dev/null; then
        log "$LOG_DEBUG" "Using gh CLI for GitHub API"
        if gh repo list --limit 100 --json nameWithOwner,description 2>/dev/null | \
            jq -r '.[] | "\(.nameWithOwner)|\(.description // "")"' > "$cache_file" 2>/dev/null; then
            cat "$cache_file"
            return 0
        else
            log "$LOG_WARN" "gh CLI failed, trying direct API"
        fi
    fi

    # Fall back to curl with GITHUB_TOKEN
    if [ -n "$GITHUB_TOKEN" ]; then
        log "$LOG_DEBUG" "Using GITHUB_TOKEN for API"
        if curl -s -H "Authorization: token $GITHUB_TOKEN" \
            "https://api.github.com/user/repos?per_page=100" 2>/dev/null | \
            jq -r '.[] | "\(.full_name)|\(.description // "")"' > "$cache_file" 2>/dev/null; then
            cat "$cache_file"
            return 0
        else
            log "$LOG_WARN" "GitHub API request failed"
        fi
    fi

    log "$LOG_WARN" "GitHub integration requires 'gh' CLI or GITHUB_TOKEN environment variable"
    return 1
}

# List all repositories from enabled sources
# Usage: repos_list_all
repos_list_all() {
    local all_repos=""
    local has_repos=false

    # Gather from local config if enabled
    if [ "$(config_get "LOCAL_CONFIG_ENABLED")" = "true" ]; then
        local local_repos=$(repos_from_local)
        if [ -n "$local_repos" ]; then
            all_repos="$local_repos"
            has_repos=true
        fi
    fi

    # Gather from GitHub if enabled
    if [ "$(config_get "GITHUB_ENABLED")" = "true" ]; then
        local github_repos=$(repos_from_github)
        if [ -n "$github_repos" ]; then
            if [ -n "$all_repos" ]; then
                all_repos="$all_repos"$'\n'"$github_repos"
            else
                all_repos="$github_repos"
            fi
            has_repos=true
        fi
    fi

    if [ "$has_repos" = false ]; then
        log "$LOG_ERROR" "No repositories found from any source"
        echo "Error: No repositories configured" >&2
        echo "Please configure LOCAL_REPOS_FILE or enable GitHub integration" >&2
        return 1
    fi

    # Remove duplicates while preserving order
    echo "$all_repos" | awk '!seen[$0]++'
    return 0
}

# Format repository list for fzf
# Usage: repos_format_for_fzf
repos_format_for_fzf() {
    while IFS='|' read -r url description; do
        if [ -n "$description" ]; then
            echo "$url|$description"
        else
            echo "$url|No description"
        fi
    done
}

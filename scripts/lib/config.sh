#!/usr/bin/env bash
# Configuration management for tmux-claude-fleet

# Source common utilities
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

# Default configuration values
declare -gA CONFIG_DEFAULTS=(
    [CLONE_DIR]="$HOME/.tmux-claude-fleet/repos"
    [GITHUB_ENABLED]="true"
    [LOCAL_CONFIG_ENABLED]="true"
    [LOCAL_REPOS_FILE]="$HOME/.tmux-claude-fleet/repos.txt"
    [CLAUDE_BIN]="$(command -v claude)"
    [CLAUDE_ARGS]="--dangerously-skip-permissions"
    [LOG_FILE]="$HOME/.tmux-claude-fleet/plugin.log"
    [CACHE_DIR]="$HOME/.tmux-claude-fleet/.cache"
    [CACHE_TTL]="300"
)

# Current configuration (loaded values)
declare -gA CONFIG

# Configuration file search paths (in order of priority)
readonly CONFIG_PATHS=(
    "$HOME/.config/tmux-claude-fleet/config"
    "$HOME/.tmux-claude-fleet/config"
)

# Load configuration from file
# Usage: _config_load_file PATH
_config_load_file() {
    local config_file="$1"

    if [ ! -f "$config_file" ]; then
        return 1
    fi

    log "$LOG_DEBUG" "Loading config from: $config_file"

    # Read config file, skip comments and empty lines
    while IFS='=' read -r key value; do
        # Skip comments and empty lines
        [[ "$key" =~ ^[[:space:]]*# ]] && continue
        [[ -z "$key" ]] && continue

        # Trim whitespace
        key=$(echo "$key" | xargs)
        value=$(echo "$value" | xargs)

        # Remove quotes from value
        value=$(echo "$value" | sed 's/^"\(.*\)"$/\1/' | sed "s/^'\(.*\)'$/\1/")

        CONFIG["$key"]="$value"
    done < <(grep -v '^[[:space:]]*$' "$config_file")

    return 0
}

# Load environment variable overrides
# Usage: _config_load_env_overrides
_config_load_env_overrides() {
    local prefix="TMUX_CLAUDE_FLEET_"

    for key in "${!CONFIG_DEFAULTS[@]}"; do
        local env_var="${prefix}${key}"
        if [ -n "${!env_var}" ]; then
            CONFIG["$key"]="${!env_var}"
            log "$LOG_DEBUG" "Config override from env: $key=${!env_var}"
        fi
    done
}

# Load configuration with priority: env vars > config files > defaults
# Usage: config_load
config_load() {
    # Start with defaults
    for key in "${!CONFIG_DEFAULTS[@]}"; do
        CONFIG["$key"]="${CONFIG_DEFAULTS[$key]}"
    done

    # Load from config files (later files override earlier ones)
    local loaded_from=""
    for config_path in "${CONFIG_PATHS[@]}"; do
        if _config_load_file "$config_path"; then
            loaded_from="$config_path"
            break
        fi
    done

    # Apply environment variable overrides
    _config_load_env_overrides

    # Log loaded configuration
    if [ -n "$loaded_from" ]; then
        log "$LOG_INFO" "Configuration loaded from: $loaded_from"
    else
        log "$LOG_INFO" "Using default configuration"
    fi

    # Validate configuration
    if ! config_validate; then
        log "$LOG_ERROR" "Configuration validation failed"
        return 1
    fi

    return 0
}

# Get configuration value with fallback
# Usage: config_get KEY [DEFAULT]
config_get() {
    local key="$1"
    local default="${2:-}"

    if [ -n "${CONFIG[$key]}" ]; then
        echo "${CONFIG[$key]}"
    elif [ -n "$default" ]; then
        echo "$default"
    else
        echo "${CONFIG_DEFAULTS[$key]}"
    fi
}

# Validate configuration
# Usage: config_validate
config_validate() {
    local errors=()

    # Validate CLONE_DIR
    local clone_dir=$(config_get "CLONE_DIR")
    if [ -z "$clone_dir" ]; then
        errors+=("CLONE_DIR cannot be empty")
    else
        # Create if doesn't exist
        if ! ensure_dir "$clone_dir"; then
            errors+=("Failed to create CLONE_DIR: $clone_dir")
        fi
    fi

    # Validate CACHE_DIR
    local cache_dir=$(config_get "CACHE_DIR")
    if [ -n "$cache_dir" ]; then
        if ! ensure_dir "$cache_dir"; then
            errors+=("Failed to create CACHE_DIR: $cache_dir")
        fi
    fi

    # Validate CACHE_TTL is a number
    local cache_ttl=$(config_get "CACHE_TTL")
    if ! [[ "$cache_ttl" =~ ^[0-9]+$ ]]; then
        errors+=("CACHE_TTL must be a positive integer: $cache_ttl")
    fi

    # Validate boolean values
    for bool_key in GITHUB_ENABLED LOCAL_CONFIG_ENABLED; do
        local value=$(config_get "$bool_key")
        if [[ ! "$value" =~ ^(true|false)$ ]]; then
            errors+=("$bool_key must be 'true' or 'false': $value")
        fi
    done

    # Validate LOCAL_REPOS_FILE if LOCAL_CONFIG_ENABLED
    if [ "$(config_get "LOCAL_CONFIG_ENABLED")" = "true" ]; then
        local repos_file=$(config_get "LOCAL_REPOS_FILE")
        if [ -n "$repos_file" ] && [ ! -f "$repos_file" ]; then
            log "$LOG_WARN" "LOCAL_REPOS_FILE not found: $repos_file"
        fi
    fi

    # Validate CLAUDE_BIN if set
    local claude_bin=$(config_get "CLAUDE_BIN")
    if [ -n "$claude_bin" ] && [ ! -x "$claude_bin" ]; then
        log "$LOG_WARN" "CLAUDE_BIN not executable: $claude_bin"
    fi

    # Report errors
    if [ ${#errors[@]} -gt 0 ]; then
        for error in "${errors[@]}"; do
            log "$LOG_ERROR" "Config validation: $error"
        done
        return 1
    fi

    return 0
}

#!/usr/bin/env bash
# Configuration management for tmux-claude-fleet

# Guard against multiple sourcing
if [ -n "$TMUX_CLAUDE_FLEET_CONFIG_LOADED" ]; then
    return 0
fi
readonly TMUX_CLAUDE_FLEET_CONFIG_LOADED=1

# Source common utilities
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

# Default configuration values (bash 3.2 compatible)
_CONFIG_CLONE_DIR="$HOME/.tmux-claude-fleet/repos"
_CONFIG_GITHUB_ENABLED="true"
_CONFIG_LOCAL_CONFIG_ENABLED="true"
_CONFIG_LOCAL_REPOS_FILE="$HOME/.tmux-claude-fleet/repos.txt"
_CONFIG_CLAUDE_BIN="$(command -v claude 2>/dev/null || echo "")"
_CONFIG_CLAUDE_ARGS="--dangerously-skip-permissions"
_CONFIG_LOG_FILE="$HOME/.tmux-claude-fleet/plugin.log"
_CONFIG_CACHE_DIR="$HOME/.tmux-claude-fleet/.cache"
_CONFIG_CACHE_TTL="300"

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

        # Set configuration variable
        case "$key" in
            CLONE_DIR)
                _CONFIG_CLONE_DIR="$value"
                ;;
            GITHUB_ENABLED)
                _CONFIG_GITHUB_ENABLED="$value"
                ;;
            LOCAL_CONFIG_ENABLED)
                _CONFIG_LOCAL_CONFIG_ENABLED="$value"
                ;;
            LOCAL_REPOS_FILE)
                _CONFIG_LOCAL_REPOS_FILE="$value"
                ;;
            CLAUDE_BIN)
                _CONFIG_CLAUDE_BIN="$value"
                ;;
            CLAUDE_ARGS)
                _CONFIG_CLAUDE_ARGS="$value"
                ;;
            LOG_FILE)
                _CONFIG_LOG_FILE="$value"
                ;;
            CACHE_DIR)
                _CONFIG_CACHE_DIR="$value"
                ;;
            CACHE_TTL)
                _CONFIG_CACHE_TTL="$value"
                ;;
        esac
    done < <(grep -v '^[[:space:]]*$' "$config_file")

    return 0
}

# Load environment variable overrides
# Usage: _config_load_env_overrides
_config_load_env_overrides() {
    local prefix="TMUX_CLAUDE_FLEET_"

    [ -n "${TMUX_CLAUDE_FLEET_CLONE_DIR}" ] && _CONFIG_CLONE_DIR="${TMUX_CLAUDE_FLEET_CLONE_DIR}"
    [ -n "${TMUX_CLAUDE_FLEET_GITHUB_ENABLED}" ] && _CONFIG_GITHUB_ENABLED="${TMUX_CLAUDE_FLEET_GITHUB_ENABLED}"
    [ -n "${TMUX_CLAUDE_FLEET_LOCAL_CONFIG_ENABLED}" ] && _CONFIG_LOCAL_CONFIG_ENABLED="${TMUX_CLAUDE_FLEET_LOCAL_CONFIG_ENABLED}"
    [ -n "${TMUX_CLAUDE_FLEET_LOCAL_REPOS_FILE}" ] && _CONFIG_LOCAL_REPOS_FILE="${TMUX_CLAUDE_FLEET_LOCAL_REPOS_FILE}"
    [ -n "${TMUX_CLAUDE_FLEET_CLAUDE_BIN}" ] && _CONFIG_CLAUDE_BIN="${TMUX_CLAUDE_FLEET_CLAUDE_BIN}"
    [ -n "${TMUX_CLAUDE_FLEET_CLAUDE_ARGS}" ] && _CONFIG_CLAUDE_ARGS="${TMUX_CLAUDE_FLEET_CLAUDE_ARGS}"
    [ -n "${TMUX_CLAUDE_FLEET_LOG_FILE}" ] && _CONFIG_LOG_FILE="${TMUX_CLAUDE_FLEET_LOG_FILE}"
    [ -n "${TMUX_CLAUDE_FLEET_CACHE_DIR}" ] && _CONFIG_CACHE_DIR="${TMUX_CLAUDE_FLEET_CACHE_DIR}"
    [ -n "${TMUX_CLAUDE_FLEET_CACHE_TTL}" ] && _CONFIG_CACHE_TTL="${TMUX_CLAUDE_FLEET_CACHE_TTL}"
}

# Load configuration with priority: env vars > config files > defaults
# Usage: config_load
config_load() {
    # Load from config files (first file found wins)
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

    case "$key" in
        CLONE_DIR)
            echo "${_CONFIG_CLONE_DIR}"
            ;;
        GITHUB_ENABLED)
            echo "${_CONFIG_GITHUB_ENABLED}"
            ;;
        LOCAL_CONFIG_ENABLED)
            echo "${_CONFIG_LOCAL_CONFIG_ENABLED}"
            ;;
        LOCAL_REPOS_FILE)
            echo "${_CONFIG_LOCAL_REPOS_FILE}"
            ;;
        CLAUDE_BIN)
            echo "${_CONFIG_CLAUDE_BIN}"
            ;;
        CLAUDE_ARGS)
            echo "${_CONFIG_CLAUDE_ARGS}"
            ;;
        LOG_FILE)
            echo "${_CONFIG_LOG_FILE}"
            ;;
        CACHE_DIR)
            echo "${_CONFIG_CACHE_DIR}"
            ;;
        CACHE_TTL)
            echo "${_CONFIG_CACHE_TTL}"
            ;;
        *)
            echo "$default"
            ;;
    esac
}

# Validate configuration
# Usage: config_validate
config_validate() {
    local errors=()

    # Validate CLONE_DIR
    if [ -z "$_CONFIG_CLONE_DIR" ]; then
        errors+=("CLONE_DIR cannot be empty")
    else
        if ! ensure_dir "$_CONFIG_CLONE_DIR"; then
            errors+=("Failed to create CLONE_DIR: $_CONFIG_CLONE_DIR")
        fi
    fi

    # Validate CACHE_DIR
    if [ -n "$_CONFIG_CACHE_DIR" ]; then
        if ! ensure_dir "$_CONFIG_CACHE_DIR"; then
            errors+=("Failed to create CACHE_DIR: $_CONFIG_CACHE_DIR")
        fi
    fi

    # Validate CACHE_TTL is a number
    if ! [[ "$_CONFIG_CACHE_TTL" =~ ^[0-9]+$ ]]; then
        errors+=("CACHE_TTL must be a positive integer: $_CONFIG_CACHE_TTL")
    fi

    # Validate boolean values
    if [[ ! "$_CONFIG_GITHUB_ENABLED" =~ ^(true|false)$ ]]; then
        errors+=("GITHUB_ENABLED must be 'true' or 'false': $_CONFIG_GITHUB_ENABLED")
    fi

    if [[ ! "$_CONFIG_LOCAL_CONFIG_ENABLED" =~ ^(true|false)$ ]]; then
        errors+=("LOCAL_CONFIG_ENABLED must be 'true' or 'false': $_CONFIG_LOCAL_CONFIG_ENABLED")
    fi

    # Validate LOCAL_REPOS_FILE if LOCAL_CONFIG_ENABLED
    if [ "$_CONFIG_LOCAL_CONFIG_ENABLED" = "true" ]; then
        if [ -n "$_CONFIG_LOCAL_REPOS_FILE" ] && [ ! -f "$_CONFIG_LOCAL_REPOS_FILE" ]; then
            log "$LOG_WARN" "LOCAL_REPOS_FILE not found: $_CONFIG_LOCAL_REPOS_FILE"
        fi
    fi

    # Validate CLAUDE_BIN if set
    if [ -n "$_CONFIG_CLAUDE_BIN" ] && [ ! -x "$_CONFIG_CLAUDE_BIN" ]; then
        log "$LOG_WARN" "CLAUDE_BIN not executable: $_CONFIG_CLAUDE_BIN"
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

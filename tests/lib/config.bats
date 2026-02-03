#!/usr/bin/env bats

load '../helpers/test_helper'

setup() {
    setup_test_env
    source "$BATS_TEST_DIRNAME/../../scripts/lib/config.sh"
}

teardown() {
    teardown_test_env
}

@test "config_load: uses defaults when no config file exists" {
    config_load
    [ "$(config_get "CACHE_TTL")" = "300" ]
}

@test "config_load: loads from ~/.config/tmux-claude-fleet/config" {
    mkdir -p "$HOME/.config/tmux-claude-fleet"
    echo "CACHE_TTL=600" > "$HOME/.config/tmux-claude-fleet/config"

    config_load
    [ "$(config_get "CACHE_TTL")" = "600" ]
}

@test "config_load: loads from ~/.tmux-claude-fleet/config" {
    mkdir -p "$HOME/.tmux-claude-fleet"
    echo "CACHE_TTL=900" > "$HOME/.tmux-claude-fleet/config"

    config_load
    [ "$(config_get "CACHE_TTL")" = "900" ]
}

@test "config_load: ~/.config has priority over ~/.tmux-claude-fleet" {
    mkdir -p "$HOME/.config/tmux-claude-fleet"
    mkdir -p "$HOME/.tmux-claude-fleet"
    echo "CACHE_TTL=600" > "$HOME/.config/tmux-claude-fleet/config"
    echo "CACHE_TTL=900" > "$HOME/.tmux-claude-fleet/config"

    config_load
    [ "$(config_get "CACHE_TTL")" = "600" ]
}

@test "config_load: environment variables override config file" {
    mkdir -p "$HOME/.config/tmux-claude-fleet"
    echo "CACHE_TTL=600" > "$HOME/.config/tmux-claude-fleet/config"
    export TMUX_CLAUDE_FLEET_CACHE_TTL=1200

    config_load
    [ "$(config_get "CACHE_TTL")" = "1200" ]
}

@test "config_load: skips comment lines" {
    mkdir -p "$HOME/.config/tmux-claude-fleet"
    cat > "$HOME/.config/tmux-claude-fleet/config" << EOF
# This is a comment
CACHE_TTL=600
# Another comment
EOF

    config_load
    [ "$(config_get "CACHE_TTL")" = "600" ]
}

@test "config_load: skips empty lines" {
    mkdir -p "$HOME/.config/tmux-claude-fleet"
    cat > "$HOME/.config/tmux-claude-fleet/config" << EOF

CACHE_TTL=600

EOF

    config_load
    [ "$(config_get "CACHE_TTL")" = "600" ]
}

@test "config_load: removes quotes from values" {
    mkdir -p "$HOME/.config/tmux-claude-fleet"
    cat > "$HOME/.config/tmux-claude-fleet/config" << EOF
CLONE_DIR="$HOME/test"
LOCAL_REPOS_FILE='$HOME/repos.txt'
EOF

    config_load
    [ "$(config_get "CLONE_DIR")" = "$HOME/test" ]
    [ "$(config_get "LOCAL_REPOS_FILE")" = "$HOME/repos.txt" ]
}

@test "config_get: returns configured value" {
    CONFIG["TEST_KEY"]="test_value"
    [ "$(config_get "TEST_KEY")" = "test_value" ]
}

@test "config_get: returns default when key not set" {
    result=$(config_get "NONEXISTENT" "default_value")
    [ "$result" = "default_value" ]
}

@test "config_get: returns empty when key not set and no default" {
    result=$(config_get "NONEXISTENT")
    [ -z "$result" ]
}

@test "config_validate: succeeds with valid configuration" {
    mkdir -p "$HOME/.config/tmux-claude-fleet"
    create_test_config_fixture "$HOME/.config/tmux-claude-fleet/config"

    config_load
    run config_validate
    [ "$status" -eq 0 ]
}

@test "config_validate: fails when CACHE_TTL is not a number" {
    CONFIG["CACHE_TTL"]="not_a_number"
    run config_validate
    [ "$status" -eq 1 ]
}

@test "config_validate: fails when boolean values are invalid" {
    CONFIG["GITHUB_ENABLED"]="maybe"
    run config_validate
    [ "$status" -eq 1 ]
}

@test "config_validate: creates CLONE_DIR if missing" {
    local test_clone_dir="$TEST_TEMP_DIR/new-clone-dir"
    CONFIG["CLONE_DIR"]="$test_clone_dir"

    config_validate
    [ -d "$test_clone_dir" ]
}

@test "config_validate: creates CACHE_DIR if missing" {
    local test_cache_dir="$TEST_TEMP_DIR/new-cache-dir"
    CONFIG["CACHE_DIR"]="$test_cache_dir"

    config_validate
    [ -d "$test_cache_dir" ]
}

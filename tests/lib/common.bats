#!/usr/bin/env bats

load '../helpers/test_helper'

setup() {
    setup_test_env
    source "$BATS_TEST_DIRNAME/../../scripts/lib/common.sh"
}

teardown() {
    teardown_test_env
}

@test "log: writes to log file" {
    log "INFO" "test message"
    [ -f "$TMUX_CLAUDE_FLEET_LOG_FILE" ]
    grep -q "test message" "$TMUX_CLAUDE_FLEET_LOG_FILE"
}

@test "log: includes timestamp and level" {
    log "ERROR" "error message"
    grep -qE '\[[0-9]{4}-[0-9]{2}-[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2}\] \[ERROR\] error message' "$TMUX_CLAUDE_FLEET_LOG_FILE"
}

@test "log: creates log directory if missing" {
    rm -rf "$(dirname "$TMUX_CLAUDE_FLEET_LOG_FILE")"
    log "INFO" "test"
    [ -f "$TMUX_CLAUDE_FLEET_LOG_FILE" ]
}

@test "ensure_dir: creates directory" {
    local test_dir="$TEST_TEMP_DIR/new-dir"
    ensure_dir "$test_dir"
    [ -d "$test_dir" ]
}

@test "ensure_dir: succeeds if directory exists" {
    local test_dir="$TEST_TEMP_DIR/existing-dir"
    mkdir -p "$test_dir"
    ensure_dir "$test_dir"
    [ -d "$test_dir" ]
}

@test "ensure_dir: fails if path is a file" {
    local test_file="$TEST_TEMP_DIR/existing-file"
    touch "$test_file"
    run ensure_dir "$test_file"
    [ "$status" -eq 1 ]
}

@test "ensure_dir: fails if path is empty" {
    run ensure_dir ""
    [ "$status" -eq 1 ]
}

@test "ensure_dir: creates nested directories" {
    local test_dir="$TEST_TEMP_DIR/a/b/c/d"
    ensure_dir "$test_dir"
    [ -d "$test_dir" ]
}

@test "sanitize_name: converts to lowercase" {
    result=$(sanitize_name "MyProject")
    [ "$result" = "myproject" ]
}

@test "sanitize_name: replaces spaces with dashes" {
    result=$(sanitize_name "my project name")
    [ "$result" = "my-project-name" ]
}

@test "sanitize_name: replaces special characters with dashes" {
    result=$(sanitize_name "my@project#name!")
    [ "$result" = "my-project-name" ]
}

@test "sanitize_name: removes leading dashes" {
    result=$(sanitize_name "--myproject")
    [ "$result" = "myproject" ]
}

@test "sanitize_name: removes trailing dashes" {
    result=$(sanitize_name "myproject--")
    [ "$result" = "myproject" ]
}

@test "sanitize_name: collapses multiple dashes" {
    result=$(sanitize_name "my---project---name")
    [ "$result" = "my-project-name" ]
}

@test "sanitize_name: handles unicode characters" {
    result=$(sanitize_name "my-プロジェクト-name")
    [ "$result" = "my-name" ]
}

@test "sanitize_name: limits to 50 characters" {
    result=$(sanitize_name "this-is-a-very-long-project-name-that-exceeds-fifty-characters-in-length")
    [ ${#result} -le 50 ]
}

@test "sanitize_name: returns 'unnamed' for empty input" {
    result=$(sanitize_name "")
    [ "$result" = "unnamed" ]
}

@test "sanitize_name: returns 'unnamed' for only special characters" {
    result=$(sanitize_name "###@@@!!!")
    [ "$result" = "unnamed" ]
}

@test "acquire_lock: creates lock file" {
    local lockfile="$TEST_TEMP_DIR/test.lock"
    acquire_lock "$lockfile"
    [ -d "$lockfile" ]
    [ -f "$lockfile/pid" ]
    release_lock "$lockfile"
}

@test "acquire_lock: stores current PID in lock" {
    local lockfile="$TEST_TEMP_DIR/test.lock"
    acquire_lock "$lockfile"
    [ "$(cat "$lockfile/pid")" = "$$" ]
    release_lock "$lockfile"
}

@test "acquire_lock: fails if lock already held" {
    local lockfile="$TEST_TEMP_DIR/test.lock"
    acquire_lock "$lockfile" 2
    run acquire_lock "$lockfile" 2
    [ "$status" -eq 1 ]
    release_lock "$lockfile"
}

@test "acquire_lock: removes stale lock" {
    local lockfile="$TEST_TEMP_DIR/test.lock"
    mkdir -p "$lockfile"
    echo "999999" > "$lockfile/pid"

    acquire_lock "$lockfile"
    [ "$status" -eq 0 ]
    [ "$(cat "$lockfile/pid")" = "$$" ]
    release_lock "$lockfile"
}

@test "acquire_lock: fails if lockfile is empty" {
    run acquire_lock ""
    [ "$status" -eq 1 ]
}

@test "release_lock: removes lock file" {
    local lockfile="$TEST_TEMP_DIR/test.lock"
    acquire_lock "$lockfile"
    release_lock "$lockfile"
    [ ! -e "$lockfile" ]
}

@test "release_lock: succeeds if lock doesn't exist" {
    local lockfile="$TEST_TEMP_DIR/nonexistent.lock"
    run release_lock "$lockfile"
    [ "$status" -eq 0 ]
}

@test "release_lock: fails if lock owned by different process" {
    local lockfile="$TEST_TEMP_DIR/test.lock"
    mkdir -p "$lockfile"
    echo "999999" > "$lockfile/pid"

    run release_lock "$lockfile"
    [ "$status" -eq 1 ]
    rm -rf "$lockfile"
}

@test "check_dependencies: succeeds when all dependencies present" {
    mock_git
    mock_tmux
    mock_claude

    run check_dependencies
    [ "$status" -eq 0 ]
}

@test "check_dependencies: fails when git missing" {
    mock_tmux
    export PATH="/nonexistent:$PATH"

    run check_dependencies
    [ "$status" -eq 1 ]
    [[ "$output" =~ "git" ]]
}

@test "check_dependencies: warns when claude missing" {
    mock_git
    mock_tmux

    # Remove claude from PATH
    export PATH="$TEST_TEMP_DIR:$PATH"

    run check_dependencies
    [[ "$output" =~ "Claude CLI not found" ]]
}

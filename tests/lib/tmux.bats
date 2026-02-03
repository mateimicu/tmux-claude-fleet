#!/usr/bin/env bats

load '../helpers/test_helper'

setup() {
    setup_test_env
    mock_tmux
    source "$BATS_TEST_DIRNAME/../../scripts/lib/tmux.sh"
}

teardown() {
    teardown_test_env
}

@test "tmux_session_exists: returns true for existing session" {
    tmux new-session -d -s "test-session"

    run tmux_session_exists "test-session"
    [ "$status" -eq 0 ]
}

@test "tmux_session_exists: returns false for non-existing session" {
    run tmux_session_exists "nonexistent-session"
    [ "$status" -eq 1 ]
}

@test "tmux_session_exists: returns false for empty name" {
    run tmux_session_exists ""
    [ "$status" -eq 1 ]
}

@test "tmux_create_session: creates new session" {
    tmux_create_session "new-session" "$TEST_TEMP_DIR"

    run tmux_session_exists "new-session"
    [ "$status" -eq 0 ]
}

@test "tmux_create_session: fails if session exists" {
    tmux new-session -d -s "existing-session"

    run tmux_create_session "existing-session" "$TEST_TEMP_DIR"
    [ "$status" -eq 1 ]
}

@test "tmux_create_session: fails if path doesn't exist" {
    run tmux_create_session "new-session" "/nonexistent/path"
    [ "$status" -eq 1 ]
}

@test "tmux_create_session: fails without required parameters" {
    run tmux_create_session "new-session"
    [ "$status" -eq 1 ]

    run tmux_create_session
    [ "$status" -eq 1 ]
}

@test "tmux_kill_session: removes session" {
    tmux new-session -d -s "test-session"
    tmux_kill_session "test-session"

    run tmux_session_exists "test-session"
    [ "$status" -eq 1 ]
}

@test "tmux_kill_session: succeeds if session doesn't exist" {
    run tmux_kill_session "nonexistent-session"
    [ "$status" -eq 0 ]
}

@test "tmux_kill_session: fails without name" {
    run tmux_kill_session
    [ "$status" -eq 1 ]
}

@test "claude_status: returns stopped for non-existing session" {
    result=$(claude_status "nonexistent")
    [ "$result" = "stopped" ]
}

@test "claude_status: returns unknown for empty session name" {
    result=$(claude_status "")
    [ "$result" = "unknown" ]
}

#!/usr/bin/env bats

load '../helpers/test_helper'

setup() {
    setup_test_env
    mock_tmux
    source "$BATS_TEST_DIRNAME/../../scripts/lib/session.sh"
}

teardown() {
    teardown_test_env
}

@test "session_save_metadata: creates metadata file" {
    session_save_metadata "test-session" "https://github.com/user/repo" "/path/to/clone"

    meta_file="$HOME/.tmux-claude-fleet/sessions/test-session.meta"
    assert_file_exists "$meta_file"
}

@test "session_save_metadata: stores correct data" {
    session_save_metadata "test-session" "https://github.com/user/repo" "/path/to/clone"

    meta_file="$HOME/.tmux-claude-fleet/sessions/test-session.meta"
    assert_file_contains "$meta_file" 'REPO_URL="https://github.com/user/repo"'
    assert_file_contains "$meta_file" 'CLONE_PATH="/path/to/clone"'
    assert_file_contains "$meta_file" 'CREATED_AT='
}

@test "session_save_metadata: fails without required parameters" {
    run session_save_metadata "test-session" "https://github.com/user/repo"
    [ "$status" -eq 1 ]

    run session_save_metadata "test-session"
    [ "$status" -eq 1 ]

    run session_save_metadata
    [ "$status" -eq 1 ]
}

@test "session_load_metadata: returns metadata content" {
    session_save_metadata "test-session" "https://github.com/user/repo" "/path/to/clone"

    result=$(session_load_metadata "test-session")
    [[ "$result" =~ REPO_URL=\"https://github.com/user/repo\" ]]
    [[ "$result" =~ CLONE_PATH=\"/path/to/clone\" ]]
}

@test "session_load_metadata: fails if metadata doesn't exist" {
    run session_load_metadata "nonexistent-session"
    [ "$status" -eq 1 ]
}

@test "session_load_metadata: fails without name" {
    run session_load_metadata
    [ "$status" -eq 1 ]
}

@test "session_exists: returns true when metadata exists" {
    session_save_metadata "test-session" "https://github.com/user/repo" "/path/to/clone"

    run session_exists "test-session"
    [ "$status" -eq 0 ]
}

@test "session_exists: returns false when metadata doesn't exist" {
    run session_exists "nonexistent-session"
    [ "$status" -eq 1 ]
}

@test "session_exists: returns false for empty name" {
    run session_exists ""
    [ "$status" -eq 1 ]
}

@test "session_list: returns empty when no sessions" {
    result=$(session_list)
    [ -z "$result" ]
}

@test "session_list: lists all sessions" {
    session_save_metadata "session-1" "https://github.com/user/repo1" "/path/1"
    session_save_metadata "session-2" "https://github.com/user/repo2" "/path/2"
    session_save_metadata "session-3" "https://github.com/user/repo3" "/path/3"

    result=$(session_list)
    echo "$result" | grep -q "session-1"
    echo "$result" | grep -q "session-2"
    echo "$result" | grep -q "session-3"
}

@test "session_list: returns sorted list" {
    session_save_metadata "zebra" "https://github.com/user/zebra" "/path/z"
    session_save_metadata "alpha" "https://github.com/user/alpha" "/path/a"
    session_save_metadata "beta" "https://github.com/user/beta" "/path/b"

    result=$(session_list)
    first=$(echo "$result" | head -n 1)
    [ "$first" = "alpha" ]
}

@test "session_delete: removes metadata file" {
    session_save_metadata "test-session" "https://github.com/user/repo" "/path/to/clone"
    session_delete "test-session"

    meta_file="$HOME/.tmux-claude-fleet/sessions/test-session.meta"
    [ ! -f "$meta_file" ]
}

@test "session_delete: succeeds if metadata doesn't exist" {
    run session_delete "nonexistent-session"
    [ "$status" -eq 0 ]
}

@test "session_delete: fails without name" {
    run session_delete
    [ "$status" -eq 1 ]
}

@test "session_generate_unique_name: returns base name if available" {
    result=$(session_generate_unique_name "myproject")
    [ "$result" = "myproject" ]
}

@test "session_generate_unique_name: appends counter if base name exists" {
    session_save_metadata "myproject" "https://github.com/user/repo" "/path/to/clone"

    result=$(session_generate_unique_name "myproject")
    [ "$result" = "myproject-1" ]
}

@test "session_generate_unique_name: increments counter until unique" {
    session_save_metadata "myproject" "https://github.com/user/repo1" "/path/1"
    session_save_metadata "myproject-1" "https://github.com/user/repo2" "/path/2"
    session_save_metadata "myproject-2" "https://github.com/user/repo3" "/path/3"

    result=$(session_generate_unique_name "myproject")
    [ "$result" = "myproject-3" ]
}

@test "session_generate_unique_name: checks both metadata and tmux sessions" {
    # Create tmux session without metadata
    tmux new-session -d -s "myproject"

    result=$(session_generate_unique_name "myproject")
    [ "$result" = "myproject-1" ]
}

@test "session_generate_unique_name: fails without base name" {
    run session_generate_unique_name
    [ "$status" -eq 1 ]
}

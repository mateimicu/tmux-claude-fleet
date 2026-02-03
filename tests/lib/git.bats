#!/usr/bin/env bats

load '../helpers/test_helper'

setup() {
    setup_test_env
    mock_git
    source "$BATS_TEST_DIRNAME/../../scripts/lib/git.sh"
}

teardown() {
    teardown_test_env
}

@test "git_clone: clones repository" {
    local clone_path="$TEST_TEMP_DIR/test-repo"
    git_clone "https://github.com/user/repo" "$clone_path"

    [ -d "$clone_path" ]
}

@test "git_clone: fails with invalid URL" {
    local clone_path="$TEST_TEMP_DIR/test-repo"
    run git_clone "invalid-url" "$clone_path"
    [ "$status" -eq 1 ]
    [ ! -d "$clone_path" ]
}

@test "git_clone: fails if path exists" {
    local clone_path="$TEST_TEMP_DIR/existing"
    mkdir -p "$clone_path"

    run git_clone "https://github.com/user/repo" "$clone_path"
    [ "$status" -eq 1 ]
}

@test "git_clone: cleans up on failure" {
    local clone_path="$TEST_TEMP_DIR/test-repo"
    # Mock git will fail on "invalid" in URL
    git_clone "https://github.com/invalid/repo" "$clone_path" || true

    [ ! -d "$clone_path" ]
}

@test "git_clone: fails without required parameters" {
    run git_clone "https://github.com/user/repo"
    [ "$status" -eq 1 ]

    run git_clone
    [ "$status" -eq 1 ]
}

@test "git_get_repo_name: extracts name from HTTPS URL" {
    result=$(git_get_repo_name "https://github.com/user/myrepo")
    [ "$result" = "myrepo" ]
}

@test "git_get_repo_name: extracts name from SSH URL" {
    result=$(git_get_repo_name "git@github.com:user/myrepo.git")
    [ "$result" = "myrepo" ]
}

@test "git_get_repo_name: removes .git suffix" {
    result=$(git_get_repo_name "https://github.com/user/myrepo.git")
    [ "$result" = "myrepo" ]
}

@test "git_get_repo_name: handles trailing slash" {
    result=$(git_get_repo_name "https://github.com/user/myrepo/")
    [ "$result" = "myrepo" ]
}

@test "git_get_repo_name: handles complex paths" {
    result=$(git_get_repo_name "https://gitlab.com/group/subgroup/myrepo")
    [ "$result" = "myrepo" ]
}

@test "git_get_repo_name: fails with empty URL" {
    run git_get_repo_name ""
    [ "$status" -eq 1 ]
}

@test "git_get_repo_name: fails with invalid URL" {
    run git_get_repo_name "/"
    [ "$status" -eq 1 ]
}

@test "git_is_valid_url: accepts HTTPS URLs" {
    run git_is_valid_url "https://github.com/user/repo"
    [ "$status" -eq 0 ]

    run git_is_valid_url "http://github.com/user/repo"
    [ "$status" -eq 0 ]
}

@test "git_is_valid_url: accepts SSH URLs" {
    run git_is_valid_url "git@github.com:user/repo.git"
    [ "$status" -eq 0 ]

    run git_is_valid_url "ssh@gitlab.com:group/project.git"
    [ "$status" -eq 0 ]
}

@test "git_is_valid_url: accepts git:// URLs" {
    run git_is_valid_url "git://github.com/user/repo"
    [ "$status" -eq 0 ]
}

@test "git_is_valid_url: accepts local paths" {
    run git_is_valid_url "/absolute/path/to/repo"
    [ "$status" -eq 0 ]

    run git_is_valid_url "./relative/path"
    [ "$status" -eq 0 ]
}

@test "git_is_valid_url: rejects invalid URLs" {
    run git_is_valid_url "not-a-url"
    [ "$status" -eq 1 ]

    run git_is_valid_url ""
    [ "$status" -eq 1 ]

    run git_is_valid_url "ftp://invalid.com/repo"
    [ "$status" -eq 1 ]
}

@test "git_is_valid_url: handles URLs with .git suffix" {
    run git_is_valid_url "https://github.com/user/repo.git"
    [ "$status" -eq 0 ]
}

@test "git_is_valid_url: handles URLs with dashes and underscores" {
    run git_is_valid_url "https://github.com/my-user/my_repo-name"
    [ "$status" -eq 0 ]
}

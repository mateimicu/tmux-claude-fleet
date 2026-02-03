#!/usr/bin/env bats

load '../helpers/test_helper'

setup() {
    setup_test_env
    source "$BATS_TEST_DIRNAME/../../scripts/lib/repos.sh"
}

teardown() {
    teardown_test_env
}

@test "repos_from_local: reads repositories from file" {
    create_test_repos_fixture "$HOME/.tmux-claude-fleet/repos.txt"

    result=$(repos_from_local)
    echo "$result" | grep -q "https://github.com/user/repo1"
    echo "$result" | grep -q "git@github.com:user/repo2.git"
}

@test "repos_from_local: includes descriptions" {
    create_test_repos_fixture "$HOME/.tmux-claude-fleet/repos.txt"

    result=$(repos_from_local)
    echo "$result" | grep -q "First test repo"
    echo "$result" | grep -q "Second test repo"
}

@test "repos_from_local: skips comments" {
    create_test_repos_fixture "$HOME/.tmux-claude-fleet/repos.txt"

    result=$(repos_from_local)
    ! echo "$result" | grep -q "# Test repositories"
}

@test "repos_from_local: skips empty lines" {
    cat > "$HOME/.tmux-claude-fleet/repos.txt" << 'EOF'
https://github.com/user/repo1

https://github.com/user/repo2

EOF

    result=$(repos_from_local)
    [ $(echo "$result" | wc -l) -eq 2 ]
}

@test "repos_from_local: returns empty if file doesn't exist" {
    result=$(repos_from_local)
    [ -z "$result" ]
}

@test "repos_from_github: uses cache if fresh" {
    mkdir -p "$TMUX_CLAUDE_FLEET_CACHE_DIR"
    echo "user/repo|Description" > "$TMUX_CLAUDE_FLEET_CACHE_DIR/github-repos"

    result=$(repos_from_github)
    [ "$result" = "user/repo|Description" ]
}

@test "repos_from_github: warns if gh CLI and token missing" {
    export PATH="/nonexistent:$PATH"
    unset GITHUB_TOKEN

    run repos_from_github 2>&1
    [ "$status" -eq 1 ]
}

@test "repos_list_all: combines local and github sources" {
    # Setup local repos
    create_test_repos_fixture "$HOME/.tmux-claude-fleet/repos.txt"

    # Setup GitHub cache
    mkdir -p "$TMUX_CLAUDE_FLEET_CACHE_DIR"
    echo "githubuser/repo|GitHub repo" > "$TMUX_CLAUDE_FLEET_CACHE_DIR/github-repos"

    # Load config first
    config_load

    result=$(repos_list_all)
    echo "$result" | grep -q "github.com/user/repo1"
    echo "$result" | grep -q "githubuser/repo"
}

@test "repos_list_all: removes duplicates" {
    # Create duplicate entries
    cat > "$HOME/.tmux-claude-fleet/repos.txt" << 'EOF'
https://github.com/user/repo
https://github.com/user/repo
EOF

    config_load

    result=$(repos_list_all)
    count=$(echo "$result" | grep -c "github.com/user/repo")
    [ "$count" -eq 1 ]
}

@test "repos_list_all: fails if no sources configured" {
    CONFIG["LOCAL_CONFIG_ENABLED"]="false"
    CONFIG["GITHUB_ENABLED"]="false"

    run repos_list_all
    [ "$status" -eq 1 ]
}

@test "repos_format_for_fzf: adds 'No description' for empty descriptions" {
    echo "https://github.com/user/repo|" | repos_format_for_fzf | grep -q "No description"
}

@test "repos_format_for_fzf: preserves existing descriptions" {
    result=$(echo "https://github.com/user/repo|My project" | repos_format_for_fzf)
    echo "$result" | grep -q "My project"
}

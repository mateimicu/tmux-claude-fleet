#!/usr/bin/env bash
# Test helper functions for BATS tests

# Setup test environment
setup_test_env() {
    export TEST_TEMP_DIR="${BATS_TEST_TMPDIR}/tmux-claude-fleet-$$"
    mkdir -p "$TEST_TEMP_DIR"

    export HOME="$TEST_TEMP_DIR/home"
    mkdir -p "$HOME"

    export TMUX_CLAUDE_FLEET_CLONE_DIR="$TEST_TEMP_DIR/repos"
    export TMUX_CLAUDE_FLEET_LOG_FILE="$TEST_TEMP_DIR/test.log"
    export TMUX_CLAUDE_FLEET_CACHE_DIR="$TEST_TEMP_DIR/.cache"

    mkdir -p "$HOME/.tmux-claude-fleet/sessions"
    mkdir -p "$TMUX_CLAUDE_FLEET_CLONE_DIR"
    mkdir -p "$TMUX_CLAUDE_FLEET_CACHE_DIR"
}

# Cleanup test environment
teardown_test_env() {
    if [ -n "$TEST_TEMP_DIR" ] && [ -d "$TEST_TEMP_DIR" ]; then
        rm -rf "$TEST_TEMP_DIR"
    fi
}

# Mock git command
mock_git() {
    cat > "$TEST_TEMP_DIR/git" << 'EOF'
#!/usr/bin/env bash
case "$1" in
    clone)
        shift
        url="$1"
        path="$2"
        if [[ "$url" == *"invalid"* ]]; then
            echo "fatal: repository not found" >&2
            exit 128
        fi
        mkdir -p "$path"
        cd "$path" && git init -q
        ;;
    *)
        command git "$@"
        ;;
esac
EOF
    chmod +x "$TEST_TEMP_DIR/git"
    export PATH="$TEST_TEMP_DIR:$PATH"
}

# Mock tmux command
mock_tmux() {
    cat > "$TEST_TEMP_DIR/tmux" << 'EOF'
#!/usr/bin/env bash
case "$1" in
    has-session)
        session="$3"
        [ -f "$HOME/.tmux-claude-fleet/.mock-sessions/$session" ] && exit 0
        exit 1
        ;;
    new-session)
        session=$(echo "$@" | grep -oP '(?<=-s )[^ ]+')
        mkdir -p "$HOME/.tmux-claude-fleet/.mock-sessions"
        touch "$HOME/.tmux-claude-fleet/.mock-sessions/$session"
        ;;
    kill-session)
        session=$(echo "$@" | grep -oP '(?<=-t )[^ ]+')
        rm -f "$HOME/.tmux-claude-fleet/.mock-sessions/$session"
        ;;
    list-sessions)
        if [ -d "$HOME/.tmux-claude-fleet/.mock-sessions" ]; then
            ls "$HOME/.tmux-claude-fleet/.mock-sessions" 2>/dev/null
        fi
        ;;
    *)
        echo "Mock tmux: $*" >&2
        ;;
esac
EOF
    chmod +x "$TEST_TEMP_DIR/tmux"
    export PATH="$TEST_TEMP_DIR:$PATH"
}

# Mock claude command
mock_claude() {
    cat > "$TEST_TEMP_DIR/claude" << 'EOF'
#!/usr/bin/env bash
echo "Mock Claude CLI"
sleep infinity
EOF
    chmod +x "$TEST_TEMP_DIR/claude"
    export PATH="$TEST_TEMP_DIR:$PATH"
}

# Create a test repo list fixture
create_test_repos_fixture() {
    cat > "$1" << 'EOF'
# Test repositories
https://github.com/user/repo1 # First test repo
git@github.com:user/repo2.git # Second test repo
https://github.com/org/project # Third test repo
EOF
}

# Create a test config fixture
create_test_config_fixture() {
    cat > "$1" << EOF
CLONE_DIR="$TMUX_CLAUDE_FLEET_CLONE_DIR"
GITHUB_ENABLED=false
LOCAL_CONFIG_ENABLED=true
LOCAL_REPOS_FILE="$HOME/.tmux-claude-fleet/repos.txt"
CLAUDE_BIN="$TEST_TEMP_DIR/claude"
CLAUDE_ARGS="--dangerously-skip-permissions"
LOG_FILE="$TMUX_CLAUDE_FLEET_LOG_FILE"
CACHE_DIR="$TMUX_CLAUDE_FLEET_CACHE_DIR"
CACHE_TTL=300
EOF
}

# Assert file exists
assert_file_exists() {
    [ -f "$1" ] || {
        echo "Expected file to exist: $1" >&2
        return 1
    }
}

# Assert file contains
assert_file_contains() {
    local file="$1"
    local pattern="$2"
    grep -q "$pattern" "$file" || {
        echo "Expected file $file to contain: $pattern" >&2
        echo "File contents:" >&2
        cat "$file" >&2
        return 1
    }
}

# Assert directory exists
assert_dir_exists() {
    [ -d "$1" ] || {
        echo "Expected directory to exist: $1" >&2
        return 1
    }
}

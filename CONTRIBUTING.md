# Contributing to Tmux Claude Fleet

Thank you for your interest in contributing! This guide will help you get started.

## Code of Conduct

- Be respectful and inclusive
- Focus on constructive feedback
- Help create a welcoming environment

## Development Setup

### Prerequisites

1. Install dependencies:
```bash
# macOS
brew install tmux fzf git bats-core

# Linux (Ubuntu/Debian)
sudo apt-get install tmux fzf git bats
```

2. Install Claude CLI (optional):
```bash
# Follow installation instructions at:
# https://github.com/anthropics/claude-cli
```

3. Clone the repository:
```bash
git clone https://github.com/mateimicu/tmux-claude-fleet.git
cd tmux-claude-fleet
```

4. Install for development:
```bash
make install
```

### Project Structure

```
tmux-claude-fleet/
├── claude-fleet.tmux          # Plugin entry point
├── scripts/
│   ├── lib/                   # Reusable library functions
│   │   ├── common.sh          # Utilities (logging, locking, sanitization)
│   │   ├── config.sh          # Configuration management
│   │   ├── session.sh         # Session CRUD operations
│   │   ├── tmux.sh            # Tmux operations
│   │   ├── git.sh             # Git operations
│   │   ├── repos.sh           # Repository discovery
│   │   └── fzf.sh             # FZF interfaces
│   ├── create-session.sh      # Create command (thin orchestrator)
│   ├── list-sessions.sh       # List command
│   └── delete-session.sh      # Delete command
├── config/
│   └── repos.example.txt      # Example configuration
└── tests/                     # Test suite
```

## Development Workflow

### 1. Create a Feature Branch

```bash
git checkout -b feature/your-feature-name
```

### 2. Follow TDD Principles

This project follows Test-Driven Development:

1. **Write tests first** for new functionality
2. **Run tests** to ensure they fail (red)
3. **Implement** the minimum code to pass (green)
4. **Refactor** while keeping tests green

### 3. Coding Guidelines

#### Shell Script Best Practices

- Use `#!/usr/bin/env bash` shebang
- Enable strict mode: `set -e` in command scripts
- Quote all variables: `"$variable"`
- Use `local` for function variables
- Add error handling for all operations
- Use meaningful function and variable names

#### Library Functions

- **common.sh**: Shared utilities, no business logic
- **Domain libraries**: Single responsibility (config, session, git, etc.)
- **Command scripts**: Thin orchestrators, source libraries and call functions

#### Function Design

```bash
# Good: Clear purpose, error handling, logging
function_name() {
    local arg1="$1"
    local arg2="$2"

    if [ -z "$arg1" ]; then
        log "$LOG_ERROR" "function_name: arg1 is required"
        return 1
    fi

    # Do work
    log "$LOG_INFO" "function_name: doing work with $arg1"

    return 0
}
```

#### Comments

- Document complex logic
- Explain "why", not "what"
- Keep comments up to date with code changes

### 4. Writing Tests

Create test file in `tests/lib/`:

```bash
#!/usr/bin/env bats

load '../helpers/test_helper'

setup() {
    setup_test_env
    source "$BATS_TEST_DIRNAME/../../scripts/lib/yourfile.sh"
}

teardown() {
    teardown_test_env
}

@test "function_name: should do something" {
    result=$(function_name "input")
    [ "$result" = "expected" ]
}

@test "function_name: should fail with invalid input" {
    run function_name ""
    [ "$status" -eq 1 ]
}
```

### 5. Running Tests

```bash
# Run all tests
make test

# Run specific test file
bats tests/lib/common.bats

# Watch mode (requires entr)
find tests -name "*.bats" | entr -c make test
```

### 6. Testing Manually

```bash
# Create test repos file
echo "https://github.com/tmux-plugins/tpm # Test repo" > ~/.tmux-claude-fleet/repos.txt

# In tmux, test keybindings:
# - prefix + C (create session)
# - prefix + L (list sessions)

# Check logs
tail -f ~/.tmux-claude-fleet/plugin.log
```

### 7. Commit Guidelines

Use conventional commits:

```
feat: add new feature
fix: fix bug in session creation
docs: update README
test: add tests for git operations
refactor: simplify config loading
chore: update dependencies
```

### 8. Submit Pull Request

1. Ensure all tests pass:
```bash
make test
```

2. Update documentation if needed

3. Push your branch:
```bash
git push origin feature/your-feature-name
```

4. Create pull request on GitHub

## What to Contribute

### Good First Issues

- Add support for new git hosting providers
- Improve error messages
- Add more configuration options
- Enhance fzf preview content
- Improve documentation

### Feature Requests

Before implementing:
1. Open an issue to discuss the feature
2. Wait for maintainer feedback
3. Implement with tests and documentation

### Bug Reports

Include:
- Tmux version: `tmux -V`
- OS and version
- Steps to reproduce
- Expected vs actual behavior
- Relevant logs from `~/.tmux-claude-fleet/plugin.log`

## Architecture Guidelines

### Clean Shell Scripts Pattern

1. **Libraries** (`scripts/lib/*.sh`):
   - Single responsibility
   - Pure functions where possible
   - No side effects in utility functions
   - Comprehensive error handling

2. **Command Scripts** (`scripts/*.sh`):
   - Thin orchestrators
   - Source required libraries
   - Handle top-level errors with traps
   - Acquire/release locks for concurrent safety

3. **Plugin Entry** (`claude-fleet.tmux`):
   - Minimal logic
   - Register keybindings
   - Check compatibility

### Error Handling

```bash
# In libraries
function_name() {
    if [ condition ]; then
        log "$LOG_ERROR" "function_name: error description"
        return 1
    fi
}

# In commands
set -e  # Exit on error

cleanup() {
    # Cleanup resources
}
trap cleanup ERR EXIT

# Acquire lock
if ! acquire_lock "$LOCK_FILE"; then
    echo "Error: Operation in progress" >&2
    exit 1
fi
```

### Logging

```bash
# Use appropriate log levels
log "$LOG_DEBUG" "Detailed debugging info"
log "$LOG_INFO" "Normal operation"
log "$LOG_WARN" "Warning, but continuing"
log "$LOG_ERROR" "Error, operation failed"
```

## Testing Strategy

### Unit Tests

- Test individual functions in isolation
- Mock external dependencies (tmux, git, etc.)
- Use test fixtures for file operations

### Integration Tests

- Test full workflows
- Use temporary tmux server
- Clean up after tests

### Manual Testing Checklist

- [ ] Create session from local repo
- [ ] Create session from GitHub
- [ ] List sessions with preview
- [ ] Switch to session
- [ ] Delete session with repo cleanup
- [ ] Handle errors (invalid repo, network issues)
- [ ] Concurrent operation locking
- [ ] Tmux version compatibility

## Release Process

1. Update version in README
2. Update CHANGELOG
3. Tag release: `git tag v1.0.0`
4. Push tag: `git push origin v1.0.0`
5. Create GitHub release with notes

## Questions?

- Open an issue for questions
- Check existing issues and PRs
- Review project documentation

Thank you for contributing! 🚀

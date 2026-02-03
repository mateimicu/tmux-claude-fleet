# Tests

This directory contains tests for the tmux-claude-fleet plugin.

## Setup

Install test dependencies:

```bash
# macOS with Homebrew
brew install bats-core

# Linux with package manager
# Ubuntu/Debian
sudo apt-get install bats

# Fedora
sudo dnf install bats
```

## Running Tests

```bash
# Run all tests
make test

# Run specific test file
make test-file FILE=tests/lib/common.bats

# Or run directly with bats
bats tests/lib/common.bats
```

## Test Structure

```
tests/
├── helpers/          # Shared test utilities
│   └── test_helper.bash
├── fixtures/         # Test data
├── lib/              # Unit tests for library modules
│   ├── common.bats
│   ├── config.bats
│   ├── session.bats
│   ├── git.bats
│   ├── tmux.bats
│   └── repos.bats
└── integration/      # Integration tests
```

## Writing Tests

Tests use the BATS (Bash Automated Testing System) framework.

Example test:

```bash
#!/usr/bin/env bats

load '../helpers/test_helper'

setup() {
    setup_test_env
    source "$BATS_TEST_DIRNAME/../../scripts/lib/common.sh"
}

teardown() {
    teardown_test_env
}

@test "function_name: test description" {
    result=$(function_name "arg1" "arg2")
    [ "$result" = "expected" ]
}
```

## Test Coverage

Current test coverage:
- ✓ common.sh - Logging, locking, sanitization
- ✓ config.sh - Configuration loading and validation
- ✓ session.sh - Session CRUD operations
- ✓ git.sh - Git operations
- ✓ tmux.sh - Tmux operations
- ✓ repos.sh - Repository discovery

## CI/CD Integration

Tests can be run in CI/CD pipelines:

```yaml
# GitHub Actions example
- name: Run tests
  run: |
    brew install bats-core
    make test
```

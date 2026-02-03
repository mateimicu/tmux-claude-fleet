# Tmux Claude Fleet

A tmux plugin that streamlines development workflows by managing sessions with integrated Claude AI assistance. Each session clones a repository and runs both a terminal and Claude CLI in separate windows.

## Features

- **Interactive Repository Selection**: Browse and select repositories using fzf
- **Multi-source Discovery**: Support for local config files and GitHub API
- **Automated Setup**: Clone repos and create tmux sessions with one command
- **Claude Integration**: Dedicated window running Claude CLI for each project
- **Session Management**: List, switch, and delete sessions with ease
- **Persistent Storage**: Repos stored in `~/.tmux-claude-fleet/repos`
- **Status Monitoring**: Track Claude process status and session activity

## Requirements

- `tmux` (version 2.0+, 3.2+ recommended for popup support)
- `fzf` (for interactive selection)
- `git` (for cloning repositories)
- `claude` CLI (optional, for Claude integration)
- `gh` CLI or `GITHUB_TOKEN` (optional, for GitHub integration)
- `jq` (optional, for JSON parsing)

## Installation

### Using TPM (Tmux Plugin Manager)

1. Add to your `~/.tmux.conf`:

```bash
set -g @plugin 'mateimicu/tmux-claude-fleet'
```

2. Install with `prefix + I`

### Manual Installation

1. Clone the repository:

```bash
git clone https://github.com/mateimicu/tmux-claude-fleet ~/.tmux/plugins/tmux-claude-fleet
```

2. Add to your `~/.tmux.conf`:

```bash
run-shell ~/.tmux/plugins/tmux-claude-fleet/claude-fleet.tmux
```

3. Reload tmux configuration:

```bash
tmux source-file ~/.tmux.conf
```

## Usage

### Default Keybindings

- `prefix + C`: Create new session (select repository)
- `prefix + L`: List and manage sessions

### Creating a Session

1. Press `prefix + C`
2. Browse repositories using fzf (arrow keys or fuzzy search)
3. Press Enter to select
4. Repository will be cloned and tmux session created with:
   - Window 0: Terminal in project directory
   - Window 1: Claude CLI in project directory

### Listing Sessions

1. Press `prefix + L`
2. Browse sessions with preview showing:
   - Session status (active/stopped)
   - Claude status (running/stopped)
   - Recent git commits
3. Press Enter to switch to session
4. Press `Ctrl+D` to delete session

### Deleting Sessions

When deleting a session:
1. Confirm session deletion
2. Choose whether to delete the cloned repository

## Configuration

### Default Settings

Configuration is loaded from (in order of priority):
1. Environment variables (`TMUX_CLAUDE_FLEET_*`)
2. `~/.config/tmux-claude-fleet/config`
3. `~/.tmux-claude-fleet/config`
4. Plugin defaults

### Configuration File Format

Create `~/.config/tmux-claude-fleet/config`:

```bash
CLONE_DIR="$HOME/.tmux-claude-fleet/repos"
GITHUB_ENABLED=true
LOCAL_CONFIG_ENABLED=true
LOCAL_REPOS_FILE="$HOME/.tmux-claude-fleet/repos.txt"
CLAUDE_BIN="/path/to/claude"
CLAUDE_ARGS="--dangerously-skip-permissions"
LOG_FILE="$HOME/.tmux-claude-fleet/plugin.log"
CACHE_DIR="$HOME/.tmux-claude-fleet/.cache"
CACHE_TTL=300
```

### Configuration Options

| Option | Default | Description |
|--------|---------|-------------|
| `CLONE_DIR` | `~/.tmux-claude-fleet/repos` | Directory for cloned repositories |
| `GITHUB_ENABLED` | `true` | Enable GitHub API repository discovery |
| `LOCAL_CONFIG_ENABLED` | `true` | Enable local repository config file |
| `LOCAL_REPOS_FILE` | `~/.tmux-claude-fleet/repos.txt` | Path to local repository list |
| `CLAUDE_BIN` | `$(command -v claude)` | Path to Claude CLI binary |
| `CLAUDE_ARGS` | `--dangerously-skip-permissions` | Arguments passed to Claude CLI |
| `LOG_FILE` | `~/.tmux-claude-fleet/plugin.log` | Path to log file |
| `CACHE_DIR` | `~/.tmux-claude-fleet/.cache` | Directory for cache files |
| `CACHE_TTL` | `300` | Cache time-to-live in seconds |

### Environment Variable Overrides

Override any setting using environment variables:

```bash
export TMUX_CLAUDE_FLEET_CLONE_DIR="/tmp/test-repos"
export TMUX_CLAUDE_FLEET_CACHE_TTL=600
```

### Custom Keybindings

Change keybindings in `~/.tmux.conf`:

```bash
set -g @claude-fleet-create-key "F"
set -g @claude-fleet-list-key "S"
```

### Local Repository List

Create `~/.tmux-claude-fleet/repos.txt`:

```
# One repository per line
# Format: URL # Optional description

https://github.com/user/project1 # My main project
git@github.com:user/project2.git # SSH clone
https://github.com/org/project3
```

Supported URL formats:
- HTTPS: `https://github.com/user/repo`
- SSH: `git@github.com:user/repo.git`
- Git protocol: `git://github.com/user/repo`
- Local paths: `/absolute/path` or `./relative/path`

### GitHub Integration

#### Using gh CLI (Recommended)

Install and authenticate the GitHub CLI:

```bash
brew install gh
gh auth login
```

#### Using Personal Access Token

Set the `GITHUB_TOKEN` environment variable:

```bash
export GITHUB_TOKEN="ghp_your_token_here"
```

Add to your shell profile (`~/.bashrc`, `~/.zshrc`, etc.) to persist.

## Architecture

### Directory Structure

```
~/.tmux-claude-fleet/
├── repos/              # Cloned repositories
│   ├── session-1/
│   └── session-2/
├── sessions/           # Session metadata
│   ├── session-1.meta
│   └── session-2.meta
├── .cache/             # Temporary cache
│   └── github-repos
├── config              # User configuration (optional)
├── repos.txt           # Local repository list
└── plugin.log          # Plugin log file
```

### Session Metadata

Each session stores metadata in `~/.tmux-claude-fleet/sessions/<name>.meta`:

```
REPO_URL="https://github.com/user/repo"
CLONE_PATH="/Users/user/.tmux-claude-fleet/repos/session-name"
CREATED_AT="2026-02-03T10:30:00"
```

## Troubleshooting

### Common Issues

#### Claude window not created

**Cause**: Claude CLI not found in PATH

**Solution**: Ensure Claude is installed and accessible:
```bash
which claude
```

Install Claude CLI if missing or configure `CLAUDE_BIN`:
```bash
export TMUX_CLAUDE_FLEET_CLAUDE_BIN="/path/to/claude"
```

#### No repositories found

**Cause**: No sources configured

**Solution**: Either:
1. Create `~/.tmux-claude-fleet/repos.txt` with repository URLs
2. Enable GitHub integration with `gh` CLI or `GITHUB_TOKEN`

#### GitHub rate limiting

**Cause**: Too many API requests without authentication

**Solution**: Authenticate with `gh` CLI or set `GITHUB_TOKEN`

#### Popup not showing (tmux < 3.2)

**Cause**: Older tmux version doesn't support `display-popup`

**Solution**: Plugin automatically falls back to `new-window` mode. Upgrade tmux for better UX:
```bash
brew upgrade tmux  # macOS
```

#### Clone fails with permission denied

**Cause**: SSH key not configured or invalid credentials

**Solution**:
1. For SSH URLs, ensure SSH key is added to GitHub/GitLab
2. For HTTPS URLs, configure Git credentials:
```bash
git config --global credential.helper cache
```

#### Stale lock file

**Cause**: Previous operation crashed without releasing lock

**Solution**: Remove lock file:
```bash
rm -rf ~/.tmux-claude-fleet/.lock
```

### Debug Logging

Check the log file for errors:

```bash
tail -f ~/.tmux-claude-fleet/plugin.log
```

Enable verbose logging:

```bash
export TMUX_CLAUDE_FLEET_LOG_LEVEL=DEBUG
```

### Testing Installation

Run tests (requires `bats`):

```bash
# Install bats
brew install bats-core  # macOS

# Run all tests
bats tests/lib/*.bats

# Run specific test
bats tests/lib/common.bats
```

## Development

### Running Tests

```bash
# Install test dependencies
brew install bats-core

# Run all tests
make test

# Run specific test file
bats tests/lib/session.bats
```

### Project Structure

```
tmux-claude-fleet/
├── claude-fleet.tmux          # Plugin entry point
├── scripts/
│   ├── lib/                   # Shared libraries
│   │   ├── common.sh          # Utilities, logging, locking
│   │   ├── config.sh          # Configuration management
│   │   ├── session.sh         # Session CRUD operations
│   │   ├── tmux.sh            # Tmux operations
│   │   ├── git.sh             # Git operations
│   │   ├── repos.sh           # Repository discovery
│   │   └── fzf.sh             # FZF interfaces
│   ├── create-session.sh      # Create command
│   ├── list-sessions.sh       # List command
│   └── delete-session.sh      # Delete command
├── config/
│   └── repos.example.txt      # Example config
└── tests/
    ├── helpers/               # Test utilities
    ├── fixtures/              # Test data
    ├── lib/                   # Unit tests
    └── integration/           # Integration tests
```

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## License

MIT License

## Acknowledgments

- Inspired by tmux plugin ecosystem
- Built with Claude AI assistance

## Related Projects

- [tmux-plugins/tpm](https://github.com/tmux-plugins/tpm) - Tmux Plugin Manager
- [junegunn/fzf](https://github.com/junegunn/fzf) - Command-line fuzzy finder
- [anthropics/claude-cli](https://github.com/anthropics/claude-cli) - Claude CLI tool

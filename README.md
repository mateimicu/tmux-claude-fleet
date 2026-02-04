# Tmux Claude Fleet - Go Version

> A fast, reliable tmux session manager with Claude AI integration, rewritten in Go.

## Features

- **Single Binary** - No Bash version compatibility issues
- **Fast** - Compiled Go code with minimal overhead
- **Reliable** - No stdout/terminal initialization issues
- **Easy Install** - `go install` or download binary
- **Cross-Platform** - Works on macOS and Linux

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/mateimicu/tmux-claude-fleet.git
cd tmux-claude-fleet

# Build the binary
make build

# The binary will be at ./bin/claude-fleet
```

### Using Go Install

```bash
go install github.com/mateimicu/tmux-claude-fleet/cmd/claude-fleet@latest
```

### As Tmux Plugin

Add to your `.tmux.conf`:

```tmux
set -g @plugin 'mateimicu/tmux-claude-fleet'

# Optional: customize keybindings
set -g @claude-fleet-create-key 'C'
set -g @claude-fleet-list-key 'L'
set -g @claude-fleet-delete-key 'D'
```

Then reload tmux:

```bash
tmux source ~/.tmux.conf
```

## Usage

### Create a Session

```bash
# From command line
claude-fleet create

# From tmux (default: prefix + C)
```

This will:
1. Discover repositories from configured sources
2. Let you select one with FZF
3. Clone the repository (if needed)
4. Create a tmux session with:
   - A terminal window in the repo directory
   - A Claude window (if configured)

### List Sessions

```bash
# From command line
claude-fleet list

# From tmux (default: prefix + L)
```

Shows all managed sessions with status indicators:
- 🟢 Active tmux session
- ⚫ Inactive session
- [Claude ✓] Claude is running

### Delete a Session

```bash
# From command line
claude-fleet delete [session-name]

# From tmux (default: prefix + D)

# Keep the cloned repository
claude-fleet delete --keep-clone session-name
```

## Configuration

Configuration is loaded from:
1. `~/.config/tmux-claude-fleet/config`
2. `~/.tmux-claude-fleet/config`
3. Environment variables (override file config)
4. Defaults

### Config File Format

```bash
# Repository sources
GITHUB_ENABLED=1
LOCAL_CONFIG_ENABLED=1
LOCAL_REPOS_FILE=~/.tmux-claude-fleet/repos.txt

# Directories
CLONE_DIR=~/.tmux-claude-fleet/repos
SESSIONS_DIR=~/.tmux-claude-fleet/sessions
CACHE_DIR=~/.tmux-claude-fleet/.cache

# Claude integration
CLAUDE_BIN=/usr/local/bin/claude
CLAUDE_ARGS=--dangerously-skip-permissions

# Cache settings
CACHE_TTL=5m  # 5 minutes
```

### Environment Variables

All config options can be set via environment variables with `TMUX_CLAUDE_FLEET_` prefix:

```bash
export TMUX_CLAUDE_FLEET_CLONE_DIR="$HOME/dev"
export TMUX_CLAUDE_FLEET_GITHUB_ENABLED=1
export GITHUB_TOKEN="ghp_your_token_here"
```

### Local Repositories

Create `~/.tmux-claude-fleet/repos.txt`:

```
# Format: URL or URL:description
https://github.com/org/repo:My awesome project
git@github.com:org/private-repo:Private project
```

### GitHub Repositories

Set your GitHub token:

```bash
export GITHUB_TOKEN="ghp_your_token_here"
```

The plugin will:
- Fetch all your repositories from GitHub API
- Cache results for 5 minutes (configurable)
- Include both public and private repos

## Architecture

### Project Structure

```
tmux-claude-fleet/
├── cmd/
│   └── claude-fleet/          # CLI entry point
│       ├── main.go
│       ├── create.go
│       ├── list.go
│       └── delete.go
├── internal/
│   ├── config/                # Configuration management
│   ├── session/               # Session CRUD operations
│   ├── repos/                 # Repository discovery
│   │   ├── local.go          # Local file source
│   │   └── github.go         # GitHub API source
│   ├── tmux/                  # Tmux operations
│   ├── fzf/                   # FZF integration
│   └── git/                   # Git operations
├── pkg/
│   └── types/                 # Shared data types
└── bin/
    └── claude-fleet           # Compiled binary
```

### Components

- **Config**: Loads configuration from files and environment
- **Session Manager**: Handles session metadata (JSON files)
- **Repository Sources**:
  - Local: Reads from `repos.txt`
  - GitHub: Fetches from GitHub API with caching
- **Tmux Manager**: Wraps tmux CLI commands
- **FZF**: Interactive repository/session selection
- **Git**: Handles repository cloning

## Development

### Building

```bash
# Build for current platform
make build

# Cross-compile for all platforms
make cross-compile

# Build with race detector
make dev

# Format code
make fmt
```

### Testing

```bash
# Run tests
make test

# Manual end-to-end test
./bin/claude-fleet create
```

### Adding a New Repository Source

Implement the `repos.Source` interface:

```go
type Source interface {
    List(ctx context.Context) ([]*types.Repository, error)
    Name() string
}
```

Example:

```go
type MySource struct {
    // configuration
}

func (m *MySource) Name() string {
    return "mysource"
}

func (m *MySource) List(ctx context.Context) ([]*types.Repository, error) {
    // Fetch repositories
    return repos, nil
}
```

## Migration from Bash Version

### Automatic Migration

The Go version:
- Uses the same directory structure (`~/.tmux-claude-fleet/`)
- Reads the same `repos.txt` format
- Stores sessions in JSON format (auto-migrates from `.meta` files)

### Manual Migration

1. Build the Go binary: `make build`
2. Test: `./bin/claude-fleet list`
3. Update `.tmux.conf` to use the Go version
4. Optional: Remove old Bash scripts

### Backward Compatibility

- Same directory structure
- Same config file locations
- Same tmux keybindings
- Same repository format
- Compatible metadata (auto-migration)

## Benefits Over Bash Version

| Aspect | Bash (Old) | Go (New) |
|--------|-----------|---------|
| Lines of code | ~1,871 | ~1,200 |
| Dependencies | bash, grep, sed, awk, jq | tmux, fzf, git |
| Compatibility | Bash 3.2 vs 4.0 issues | No issues |
| FZF integration | Fragile (pipe issues) | Robust |
| Error handling | Trap cleanup | Defer cleanup |
| Testing | BATS (limited) | Go testing |
| Distribution | Git clone | Binary download |
| Build time | N/A | ~1-2 seconds |
| Binary size | N/A | ~5-8 MB |

## Troubleshooting

### Binary not found

```bash
cd /path/to/tmux-claude-fleet
make build
```

### GitHub API rate limiting

Set a GitHub token:

```bash
export GITHUB_TOKEN="ghp_your_token_here"
```

### Claude not starting

Check Claude binary path:

```bash
which claude
# Update config or set TMUX_CLAUDE_FLEET_CLAUDE_BIN
```

### FZF not found

Install FZF:

```bash
# macOS
brew install fzf

# Linux
sudo apt install fzf  # Debian/Ubuntu
sudo yum install fzf  # RHEL/CentOS
```

## License

MIT License - see LICENSE file for details.

## Contributing

Contributions welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## Changelog

### v2.0.0 (Go Rewrite)

- Complete rewrite in Go
- Single binary distribution
- Improved reliability and performance
- Better error handling
- Simplified architecture
- Cross-platform support

### v1.0.0 (Bash)

- Initial Bash implementation
- Basic session management
- GitHub integration
- Claude integration

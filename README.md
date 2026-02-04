# Tmux Claude Fleet

A tmux session manager with Claude AI integration.

## Installation

```bash
git clone https://github.com/mateimicu/tmux-claude-fleet.git
cd tmux-claude-fleet
./install.sh
```

Or as a tmux plugin via TPM:

```tmux
set -g @plugin 'mateimicu/tmux-claude-fleet'
```

## Usage

```bash
# Create a session
claude-fleet create

# List sessions
claude-fleet list

# Delete a session
claude-fleet delete [session-name]

# Refresh repository cache
claude-fleet refresh
```

## Configuration

Create `~/.config/tmux-claude-fleet/config`:

```bash
# Repository sources
GITHUB_ENABLED=1
LOCAL_CONFIG_ENABLED=1
LOCAL_REPOS_FILE=~/.tmux-claude-fleet/repos.txt

# Directories
CLONE_DIR=~/.tmux-claude-fleet/repos
SESSIONS_DIR=~/.tmux-claude-fleet/sessions

# Claude integration
CLAUDE_BIN=/usr/local/bin/claude
```

## License

MIT

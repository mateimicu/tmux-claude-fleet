# Tmux Claude Matrix

A tmux session manager with Claude AI integration.

## Installation

```bash
git clone https://github.com/mateimicu/tmux-claude-matrix.git
cd tmux-claude-matrix
./install.sh
```

Or as a tmux plugin via TPM:

```tmux
set -g @plugin 'mateimicu/tmux-claude-matrix'
```

## Usage

```bash
# Create a session
claude-matrix create

# List sessions
claude-matrix list

# Delete a session
claude-matrix delete [session-name]

# Refresh repository cache
claude-matrix refresh
```

## Configuration

Create `~/.config/tmux-claude-matrix/config`:

```bash
# Repository sources
GITHUB_ENABLED=1
LOCAL_CONFIG_ENABLED=1
LOCAL_REPOS_FILE=~/.tmux-claude-matrix/repos.txt

# Directories
CLONE_DIR=~/.tmux-claude-matrix/repos
SESSIONS_DIR=~/.tmux-claude-matrix/sessions

# Claude integration
CLAUDE_BIN=/usr/local/bin/claude
```

## License

MIT

# Quick Start Guide

Get started with Tmux Claude Fleet in 5 minutes.

## Step 1: Install Dependencies

```bash
# macOS
brew install tmux fzf git

# Linux (Ubuntu/Debian)
sudo apt-get install tmux fzf git

# Install Claude CLI (optional)
# Follow: https://docs.anthropic.com/claude/docs/claude-cli
```

## Step 2: Install Plugin

### Using TPM (Recommended)

Add to `~/.tmux.conf`:

```bash
# Add to plugin list
set -g @plugin 'mateimicu/tmux-claude-fleet'

# Initialize TPM (keep at bottom)
run '~/.tmux/plugins/tpm/tpm'
```

Reload config and install:
```bash
tmux source-file ~/.tmux.conf
# Then press: prefix + I
```

### Manual Install

```bash
git clone https://github.com/mateimicu/tmux-claude-fleet \
    ~/.tmux/plugins/tmux-claude-fleet

echo "run-shell ~/.tmux/plugins/tmux-claude-fleet/claude-fleet.tmux" \
    >> ~/.tmux.conf

tmux source-file ~/.tmux.conf
```

## Step 3: Configure Repositories

Create a repository list:

```bash
mkdir -p ~/.tmux-claude-fleet
cat > ~/.tmux-claude-fleet/repos.txt << 'EOF'
# Your repositories
https://github.com/yourusername/project1 # Main project
https://github.com/yourusername/project2 # Another project
EOF
```

## Step 4: Create Your First Session

1. Open tmux: `tmux`
2. Press: `prefix + C` (default: `Ctrl+b C`)
3. Select a repository using arrow keys or fuzzy search
4. Press Enter

The plugin will:
- Clone the repository
- Create a tmux session
- Open a terminal window
- Start Claude in a separate window

## Step 5: Manage Sessions

### List Sessions
- Press: `prefix + L` (default: `Ctrl+b L`)
- See all sessions with status and preview
- Press Enter to switch to a session
- Press `Ctrl+D` to delete a session

### Manual Commands

```bash
# Create session
~/.tmux/plugins/tmux-claude-fleet/scripts/create-session.sh

# List sessions
~/.tmux/plugins/tmux-claude-fleet/scripts/list-sessions.sh

# Delete session
~/.tmux/plugins/tmux-claude-fleet/scripts/delete-session.sh session-name
```

## Optional: GitHub Integration

### Using gh CLI (Easiest)

```bash
brew install gh
gh auth login
```

### Using Personal Access Token

```bash
# Create token at: https://github.com/settings/tokens
export GITHUB_TOKEN="ghp_your_token_here"

# Add to shell profile to persist
echo 'export GITHUB_TOKEN="ghp_your_token_here"' >> ~/.zshrc  # or ~/.bashrc
```

## Customization

### Change Keybindings

Add to `~/.tmux.conf`:

```bash
set -g @claude-fleet-create-key "F"  # prefix + F
set -g @claude-fleet-list-key "S"    # prefix + S
```

### Configure Clone Directory

```bash
# In ~/.config/tmux-claude-fleet/config
CLONE_DIR="$HOME/dev/fleet-projects"
```

### Disable GitHub Integration

```bash
# In ~/.config/tmux-claude-fleet/config
GITHUB_ENABLED=false
```

## Common Workflows

### Start Working on a Project

```bash
tmux                    # Start tmux
prefix + C              # Create session
# Select repo, press Enter
prefix + 1              # Switch to Claude window
```

### Switch Between Projects

```bash
prefix + L              # List sessions
# Use arrows to select
# Press Enter to switch
```

### Clean Up Old Sessions

```bash
prefix + L              # List sessions
# Navigate to session
Ctrl + D                # Delete
# Confirm and optionally delete repo
```

## Troubleshooting

### Claude Not Starting

Check if Claude CLI is installed:
```bash
which claude
```

If not found, install it or configure path:
```bash
export TMUX_CLAUDE_FLEET_CLAUDE_BIN="/path/to/claude"
```

### No Repositories Found

Create `~/.tmux-claude-fleet/repos.txt` with at least one repository:
```bash
echo "https://github.com/tmux-plugins/tpm" > ~/.tmux-claude-fleet/repos.txt
```

### Keybindings Not Working

1. Check if plugin is loaded:
```bash
tmux show-option -g @plugin
```

2. Reload tmux config:
```bash
tmux source-file ~/.tmux.conf
```

3. Check for conflicts with existing keybindings

### View Logs

```bash
tail -f ~/.tmux-claude-fleet/plugin.log
```

## Next Steps

- Read the [full documentation](../README.md)
- Configure [advanced options](../README.md#configuration)
- Join the community and contribute

## Getting Help

- [Issues](https://github.com/mateimicu/tmux-claude-fleet/issues)
- [Discussions](https://github.com/mateimicu/tmux-claude-fleet/discussions)
- Check existing [documentation](../README.md)

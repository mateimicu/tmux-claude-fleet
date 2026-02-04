# Changelog

## [Unreleased]

### Breaking Changes - Fleet to Matrix Migration

**IMPORTANT: This release contains breaking changes. Read the migration guide below.**

The project has been renamed from `tmux-claude-fleet` to `tmux-claude-matrix` to better reflect its functionality.

#### What Changed

- **Binary name**: `claude-fleet` → `claude-matrix`
- **Go module**: `github.com/mateimicu/tmux-claude-fleet` → `github.com/mateimicu/tmux-claude-matrix`
- **Config directory**: `~/.tmux-claude-fleet/` → `~/.tmux-claude-matrix/`
- **Config file**: `~/.config/tmux-claude-fleet/config` → `~/.config/tmux-claude-matrix/config`
- **Environment variables**: `TMUX_CLAUDE_FLEET_*` → `TMUX_CLAUDE_MATRIX_*`
- **Tmux plugin file**: `claude-fleet.tmux` → `claude-matrix.tmux`
- **Tmux options**: `@claude-fleet-*-key` → `@claude-matrix-*-key`

#### Migration Steps

1. **Update your tmux configuration** (if using TPM):
   ```tmux
   # Old:
   set -g @plugin 'mateimicu/tmux-claude-fleet'

   # New:
   set -g @plugin 'mateimicu/tmux-claude-matrix'
   ```

2. **Update custom keybindings** (if configured):
   ```tmux
   # Old:
   set -g @claude-fleet-create-key "a"

   # New:
   set -g @claude-matrix-create-key "a"
   ```

3. **Migrate your configuration files** (optional):
   ```bash
   # Move configuration
   mv ~/.tmux-claude-fleet ~/.tmux-claude-matrix
   mv ~/.config/tmux-claude-fleet ~/.config/tmux-claude-matrix
   ```

4. **Update environment variables** (if set):
   ```bash
   # Old:
   export TMUX_CLAUDE_FLEET_CLONE_DIR="$HOME/.tmux-claude-fleet/repos"

   # New:
   export TMUX_CLAUDE_MATRIX_CLONE_DIR="$HOME/.tmux-claude-matrix/repos"
   ```

5. **Update any scripts or aliases**:
   ```bash
   # Old:
   alias cf='claude-fleet'

   # New:
   alias cm='claude-matrix'
   ```

#### Why This Change?

The "fleet" terminology was confusing. "Matrix" better represents the tool's purpose as a grid of development environments powered by tmux and Claude AI.

### Added
- Session management with Claude AI integration
- GitHub repository discovery
- Local repository configuration
- Caching with 30-minute TTL

## [1.0.0]

- Initial release

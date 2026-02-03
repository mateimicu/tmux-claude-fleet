#!/usr/bin/env bash
# Verification script for tmux-claude-fleet installation

set -e

PLUGIN_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo "Tmux Claude Fleet - Installation Verification"
echo "=============================================="
echo ""

# Check dependencies
echo "Checking dependencies..."
DEPS_OK=true

check_command() {
    local cmd="$1"
    local required="$2"

    if command -v "$cmd" &>/dev/null; then
        local version=$($cmd --version 2>&1 | head -n1 || echo "unknown")
        echo "✓ $cmd: $version"
    else
        if [ "$required" = "required" ]; then
            echo "✗ $cmd: NOT FOUND (required)"
            DEPS_OK=false
        else
            echo "⚠ $cmd: NOT FOUND (optional)"
        fi
    fi
}

check_command tmux required
check_command fzf required
check_command git required
check_command claude optional
check_command gh optional
check_command jq optional

echo ""

# Check plugin files
echo "Checking plugin files..."
FILES_OK=true

check_file() {
    local file="$1"
    if [ -f "$file" ]; then
        echo "✓ $file"
    else
        echo "✗ $file: NOT FOUND"
        FILES_OK=false
    fi
}

check_file "$PLUGIN_DIR/claude-fleet.tmux"
check_file "$PLUGIN_DIR/scripts/create-session.sh"
check_file "$PLUGIN_DIR/scripts/list-sessions.sh"
check_file "$PLUGIN_DIR/scripts/delete-session.sh"
check_file "$PLUGIN_DIR/scripts/lib/common.sh"
check_file "$PLUGIN_DIR/scripts/lib/config.sh"
check_file "$PLUGIN_DIR/scripts/lib/session.sh"
check_file "$PLUGIN_DIR/scripts/lib/tmux.sh"
check_file "$PLUGIN_DIR/scripts/lib/git.sh"
check_file "$PLUGIN_DIR/scripts/lib/repos.sh"
check_file "$PLUGIN_DIR/scripts/lib/fzf.sh"

echo ""

# Check file permissions
echo "Checking file permissions..."
PERMS_OK=true

check_executable() {
    local file="$1"
    if [ -x "$file" ]; then
        echo "✓ $file: executable"
    else
        echo "✗ $file: NOT executable"
        PERMS_OK=false
    fi
}

check_executable "$PLUGIN_DIR/claude-fleet.tmux"
check_executable "$PLUGIN_DIR/scripts/create-session.sh"
check_executable "$PLUGIN_DIR/scripts/list-sessions.sh"
check_executable "$PLUGIN_DIR/scripts/delete-session.sh"
check_executable "$PLUGIN_DIR/scripts/lib/fzf.sh"

echo ""

# Check configuration
echo "Checking configuration..."
CONFIG_OK=true

if [ -f "$HOME/.config/tmux-claude-fleet/config" ]; then
    echo "✓ Config found: ~/.config/tmux-claude-fleet/config"
elif [ -f "$HOME/.tmux-claude-fleet/config" ]; then
    echo "✓ Config found: ~/.tmux-claude-fleet/config"
else
    echo "ℹ No custom config (using defaults)"
fi

if [ -f "$HOME/.tmux-claude-fleet/repos.txt" ]; then
    local count=$(grep -v '^#' "$HOME/.tmux-claude-fleet/repos.txt" | grep -c '^http' || echo 0)
    echo "✓ Repos file: $count repositories configured"
else
    echo "⚠ No repos file found: ~/.tmux-claude-fleet/repos.txt"
    CONFIG_OK=false
fi

echo ""

# Check tmux configuration
echo "Checking tmux integration..."
TMUX_OK=true

if [ -f "$HOME/.tmux.conf" ]; then
    if grep -q "tmux-claude-fleet" "$HOME/.tmux.conf"; then
        echo "✓ Plugin loaded in ~/.tmux.conf"
    else
        echo "⚠ Plugin not found in ~/.tmux.conf"
        echo "  Add: run-shell $PLUGIN_DIR/claude-fleet.tmux"
        TMUX_OK=false
    fi
else
    echo "⚠ No ~/.tmux.conf found"
    TMUX_OK=false
fi

echo ""

# Check tmux version
echo "Checking tmux version..."
if command -v tmux &>/dev/null; then
    TMUX_VERSION=$(tmux -V | cut -d' ' -f2 | cut -d'-' -f1)
    MAJOR=$(echo "$TMUX_VERSION" | cut -d'.' -f1)
    MINOR=$(echo "$TMUX_VERSION" | cut -d'.' -f2)

    if [ "$MAJOR" -gt 3 ] || ([ "$MAJOR" -eq 3 ] && [ "$MINOR" -ge 2 ]); then
        echo "✓ tmux $TMUX_VERSION (popup support available)"
    else
        echo "⚠ tmux $TMUX_VERSION (consider upgrading to 3.2+ for popup support)"
    fi
fi

echo ""
echo "=============================================="
echo "Summary:"
echo ""

if [ "$DEPS_OK" = true ] && [ "$FILES_OK" = true ] && [ "$PERMS_OK" = true ]; then
    echo "✓ Installation looks good!"
    echo ""

    if [ "$CONFIG_OK" = false ]; then
        echo "⚠ Configuration incomplete:"
        echo "  1. Create ~/.tmux-claude-fleet/repos.txt with your repositories"
        echo "  2. Or enable GitHub integration"
        echo ""
    fi

    if [ "$TMUX_OK" = false ]; then
        echo "⚠ Tmux integration not set up:"
        echo "  Add to ~/.tmux.conf:"
        echo "  run-shell $PLUGIN_DIR/claude-fleet.tmux"
        echo ""
    fi

    echo "Get started:"
    echo "  1. Start tmux: tmux"
    echo "  2. Create session: prefix + C"
    echo "  3. List sessions: prefix + L"
    echo ""
    echo "See docs/QUICKSTART.md for detailed instructions"
    exit 0
else
    echo "✗ Installation has issues"
    echo ""

    if [ "$DEPS_OK" = false ]; then
        echo "- Install missing required dependencies"
    fi

    if [ "$FILES_OK" = false ]; then
        echo "- Some plugin files are missing"
        echo "  Try reinstalling the plugin"
    fi

    if [ "$PERMS_OK" = false ]; then
        echo "- Fix file permissions:"
        echo "  chmod +x $PLUGIN_DIR/claude-fleet.tmux"
        echo "  chmod +x $PLUGIN_DIR/scripts/*.sh"
        echo "  chmod +x $PLUGIN_DIR/scripts/lib/fzf.sh"
    fi

    echo ""
    echo "See README.md for installation instructions"
    exit 1
fi

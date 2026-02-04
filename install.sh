#!/usr/bin/env bash
# Installation script for tmux-claude-fleet

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

echo "🚀 Installing tmux-claude-fleet..."

# Check for Go
if ! command -v go >/dev/null 2>&1; then
    echo "❌ Error: Go is not installed."
    echo ""
    echo "Please install Go first:"
    echo "  macOS:   brew install go"
    echo "  Linux:   sudo apt install golang-go  (Debian/Ubuntu)"
    echo "           sudo yum install golang      (RHEL/CentOS)"
    echo ""
    echo "Or download from: https://go.dev/dl/"
    exit 1
fi

echo "✓ Go found: $(go version)"

# Build the binary
echo "📦 Building binary..."
cd "$SCRIPT_DIR"
make build

if [ ! -x "$SCRIPT_DIR/bin/claude-fleet" ]; then
    echo "❌ Build failed!"
    exit 1
fi

echo "✓ Binary built successfully"

# Test the binary
echo "🧪 Testing binary..."
if ! "$SCRIPT_DIR/bin/claude-fleet" --help >/dev/null 2>&1; then
    echo "❌ Binary test failed!"
    exit 1
fi

echo "✓ Binary works correctly"

# Create config directory
CONFIG_DIR="$HOME/.tmux-claude-fleet"
if [ ! -d "$CONFIG_DIR" ]; then
    echo "📁 Creating config directory: $CONFIG_DIR"
    mkdir -p "$CONFIG_DIR"
fi

# Create example repos file if it doesn't exist
if [ ! -f "$CONFIG_DIR/repos.txt" ]; then
    echo "📝 Creating example repos.txt..."
    cat > "$CONFIG_DIR/repos.txt" << 'EOF'
# Example repositories
# Format: URL or URL:description
# https://github.com/user/repo
# https://github.com/user/repo:My awesome project
# git@github.com:user/private-repo:Private project
EOF
    echo "✓ Created: $CONFIG_DIR/repos.txt"
    echo "  (Edit this file to add your repositories)"
fi

echo ""
echo "✅ Installation complete!"
echo ""
echo "Next steps:"
echo "1. Add repositories to: $CONFIG_DIR/repos.txt"
echo "2. (Optional) Set GITHUB_TOKEN for GitHub repo discovery:"
echo "   export GITHUB_TOKEN=\"your_token_here\""
echo "3. Reload tmux config:"
echo "   tmux source ~/.tmux.conf"
echo "4. Use keybindings:"
echo "   prefix + C  - Create new session"
echo "   prefix + L  - List sessions"
echo "   prefix + D  - Delete session"
echo ""
echo "Or use directly:"
echo "   $SCRIPT_DIR/bin/claude-fleet create"
echo ""

#!/usr/bin/env bash
# Installation script for tmux-claude-matrix

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
BINARY="$SCRIPT_DIR/bin/claude-matrix"
REPO="${CLAUDE_MATRIX_REPO:-mateimicu/tmux-claude-matrix}"

echo "Installing tmux-claude-matrix..."

# Detect OS and architecture
detect_platform() {
    local os arch
    os="$(uname -s | tr '[:upper:]' '[:lower:]')"
    arch="$(uname -m)"

    case "$os" in
        darwin|linux) ;;
        *) echo ""; return ;;
    esac

    case "$arch" in
        x86_64|amd64) arch="amd64" ;;
        arm64|aarch64) arch="arm64" ;;
        *) echo ""; return ;;
    esac

    echo "${os}-${arch}"
}

# Verify checksum of downloaded binary
verify_checksum() {
    local file="$1"
    local checksums_file="$2"
    local filename
    filename="$(basename "$file")"

    if [ ! -f "$checksums_file" ]; then
        return 1
    fi

    local expected
    expected="$(grep "$filename" "$checksums_file" | awk '{print $1}')"
    if [ -z "$expected" ]; then
        return 1
    fi

    local actual
    if command -v sha256sum >/dev/null 2>&1; then
        actual="$(sha256sum "$file" | awk '{print $1}')"
    elif command -v shasum >/dev/null 2>&1; then
        actual="$(shasum -a 256 "$file" | awk '{print $1}')"
    else
        return 1
    fi

    [ "$expected" = "$actual" ]
}

# Query GitHub API for the latest release tag
get_latest_tag() {
    local api_url="https://api.github.com/repos/${REPO}/releases/latest"
    local release_json

    if command -v curl >/dev/null 2>&1; then
        release_json="$(curl -sf --connect-timeout 10 "$api_url")"
    elif command -v wget >/dev/null 2>&1; then
        release_json="$(wget -qO- --timeout=10 "$api_url")"
    else
        return 1
    fi

    if [ -z "$release_json" ]; then
        return 1
    fi

    local tag
    tag="$(echo "$release_json" | grep '"tag_name"' | head -1 | sed 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')"
    if [ -z "$tag" ]; then
        return 1
    fi

    echo "$tag"
}

# Download pre-built binary from GitHub Releases for a specific tag
download_binary() {
    local platform="$1"
    local tag="$2"

    echo "Downloading pre-built binary for ${platform} (${tag})..."

    local binary_name="claude-matrix-${platform}"
    local base_url="https://github.com/${REPO}/releases/download/${tag}"
    local binary_url="${base_url}/${binary_name}"
    local checksums_url="${base_url}/checksums.txt"

    local tmpdir
    tmpdir="$(mktemp -d)"
    trap 'rm -rf "$tmpdir"' RETURN

    # Download binary and checksums
    echo "  Downloading binary..."
    if command -v curl >/dev/null 2>&1; then
        curl -sfL --connect-timeout 10 -o "$tmpdir/$binary_name" "$binary_url" || { echo "  Binary download failed"; return 1; }
        curl -sfL --connect-timeout 10 -o "$tmpdir/checksums.txt" "$checksums_url" || { echo "  Checksums download failed"; return 1; }
    elif command -v wget >/dev/null 2>&1; then
        wget -q --timeout=10 -O "$tmpdir/$binary_name" "$binary_url" || { echo "  Binary download failed"; return 1; }
        wget -q --timeout=10 -O "$tmpdir/checksums.txt" "$checksums_url" || { echo "  Checksums download failed"; return 1; }
    fi

    # Verify checksum
    echo "  Verifying checksum..."
    if ! verify_checksum "$tmpdir/$binary_name" "$tmpdir/checksums.txt"; then
        echo "  Checksum verification failed"
        return 1
    fi

    # Install binary
    mkdir -p "$(dirname "$BINARY")"
    mv "$tmpdir/$binary_name" "$BINARY"
    chmod +x "$BINARY"
    echo "  Installed pre-built binary (${tag})"
}

# Build from source using make
build_from_source() {
    echo "Building from source..."

    if ! command -v go >/dev/null 2>&1; then
        echo "  Go is not installed"
        return 1
    fi

    echo "  Go found: $(go version)"

    if [ ! -f "$SCRIPT_DIR/go.mod" ]; then
        echo "  Source code not found (no go.mod)"
        return 1
    fi

    (cd "$SCRIPT_DIR" && make build)

    if [ ! -x "$BINARY" ]; then
        echo "  Build produced no binary"
        return 1
    fi

    echo "  Built from source"
}

# --- Install binary ---

platform="$(detect_platform)"
tag=""
if [ -n "$platform" ]; then
    tag="$(get_latest_tag 2>/dev/null)" || true
fi

if [ -n "$platform" ] && [ -n "$tag" ] && download_binary "$platform" "$tag"; then
    : # success
elif build_from_source; then
    : # success
else
    echo ""
    echo "Installation failed."
    echo "  Download a binary from: https://github.com/${REPO}/releases"
    echo "  Or install Go and run: make build"
    exit 1
fi

# Test the binary
echo "Testing binary..."
if ! "$BINARY" version >/dev/null 2>&1; then
    echo "Binary test failed!"
    exit 1
fi

echo "Binary works correctly"

# --- Setup config ---

CONFIG_DIR="$HOME/.tmux-claude-matrix"
if [ ! -d "$CONFIG_DIR" ]; then
    echo "Creating config directory: $CONFIG_DIR"
    mkdir -p "$CONFIG_DIR"
fi

# Create example repos file if it doesn't exist
if [ ! -f "$CONFIG_DIR/repos.txt" ]; then
    echo "Creating example repos.txt..."
    cat > "$CONFIG_DIR/repos.txt" << 'EOF'
# Example repositories
# Format: URL or URL:description
# https://github.com/user/repo
# https://github.com/user/repo:My awesome project
# git@github.com:user/private-repo:Private project
EOF
    echo "  Created: $CONFIG_DIR/repos.txt"
    echo "  (Edit this file to add your repositories)"
fi

echo ""
echo "Installation complete!"
echo ""
echo "Next steps:"
echo "1. Add repositories to: $CONFIG_DIR/repos.txt"
echo "2. (Optional) Set GITHUB_TOKEN for GitHub repo discovery:"
echo "   export GITHUB_TOKEN=\"your_token_here\""
echo "3. Reload tmux config:"
echo "   tmux source ~/.tmux.conf"
echo "4. Use keybindings:"
echo "   prefix + a  - Create new session"
echo "   prefix + A  - List sessions (Shift+a)"
echo "   prefix + D  - Delete session (Shift+d)"
echo ""
echo "Or use directly:"
echo "   $SCRIPT_DIR/bin/claude-matrix create"
echo ""

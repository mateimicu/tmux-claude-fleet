#!/usr/bin/env bash
# FZF interfaces for tmux-claude-fleet

# Source common utilities
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"
source "$SCRIPT_DIR/session.sh"
source "$SCRIPT_DIR/tmux.sh"

# Interactive repository selection
# Usage: fzf_select_repo
fzf_select_repo() {
    fzf --delimiter='|' \
        --with-nth=1 \
        --preview 'echo "Repository: {1}"; echo ""; echo "Description:"; echo {2..}' \
        --preview-window=up:5:wrap \
        --prompt="Select repository: " \
        --height=80% \
        --border \
        --ansi
}

# Interactive session browser
# Usage: fzf_select_session
fzf_select_session() {
    local plugin_dir="$(cd "$SCRIPT_DIR/.." && pwd)"

    fzf --delimiter='|' \
        --with-nth=1,2 \
        --preview "$plugin_dir/scripts/lib/fzf.sh preview_session {1}" \
        --preview-window=right:60%:wrap \
        --prompt="Select session: " \
        --height=80% \
        --border \
        --ansi \
        --bind="ctrl-d:execute(echo delete:{})+abort"
}

# Generate preview content for a session
# Usage: fzf_preview_session SESSION_NAME
fzf_preview_session() {
    local session_name="$1"

    if [ -z "$session_name" ]; then
        echo "Error: Session name required"
        return 1
    fi

    # Load session metadata
    if ! session_exists "$session_name"; then
        echo "Session: $session_name"
        echo "Status: ✗ Not found"
        return 0
    fi

    local meta=$(session_load_metadata "$session_name")
    local repo_url=$(echo "$meta" | grep '^REPO_URL=' | cut -d'=' -f2- | tr -d '"')
    local clone_path=$(echo "$meta" | grep '^CLONE_PATH=' | cut -d'=' -f2- | tr -d '"')
    local created_at=$(echo "$meta" | grep '^CREATED_AT=' | cut -d'=' -f2- | tr -d '"')

    echo "Session: $session_name"
    echo "Repository: $repo_url"
    echo "Created: $created_at"
    echo ""

    # Check tmux session status
    if tmux_session_exists "$session_name"; then
        echo "Status: ✓ Active"

        # Check Claude status
        local claude_st=$(claude_status "$session_name")
        if [ "$claude_st" = "running" ]; then
            echo "Claude: ✓ Running"
        else
            echo "Claude: ⚠ Stopped"
        fi
    else
        echo "Status: ✗ Stopped"
    fi

    echo ""

    # Show recent git activity if clone exists
    if [ -d "$clone_path" ]; then
        echo "Recent activity:"
        cd "$clone_path" && git log -5 --oneline --decorate 2>/dev/null || echo "No git history"
    else
        echo "Clone path not found: $clone_path"
    fi
}

# Main entry point for preview (used by fzf)
if [ "${BASH_SOURCE[0]}" = "${0}" ]; then
    case "$1" in
        preview_session)
            fzf_preview_session "$2"
            ;;
        *)
            echo "Usage: $0 {preview_session SESSION_NAME}"
            exit 1
            ;;
    esac
fi

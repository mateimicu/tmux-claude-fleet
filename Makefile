.PHONY: test test-unit test-lib clean install check-deps verify

# Run all tests
test: check-deps test-unit

# Run unit tests only
test-unit:
	@echo "Running unit tests..."
	@bats tests/lib/*.bats

# Run specific test file
test-file:
	@if [ -z "$(FILE)" ]; then \
		echo "Usage: make test-file FILE=tests/lib/common.bats"; \
		exit 1; \
	fi
	@bats $(FILE)

# Check for required dependencies
check-deps:
	@echo "Checking dependencies..."
	@command -v tmux >/dev/null 2>&1 || { echo "Error: tmux not found"; exit 1; }
	@command -v fzf >/dev/null 2>&1 || { echo "Error: fzf not found"; exit 1; }
	@command -v git >/dev/null 2>&1 || { echo "Error: git not found"; exit 1; }
	@command -v bats >/dev/null 2>&1 || { echo "Error: bats not found. Install with: brew install bats-core"; exit 1; }
	@echo "All dependencies found!"

# Clean temporary files
clean:
	@echo "Cleaning temporary files..."
	@rm -rf ~/.tmux-claude-fleet/.cache/*
	@find . -name "*.log" -delete
	@echo "Clean complete"

# Install plugin (for development)
install:
	@echo "Installing plugin to ~/.tmux/plugins/tmux-claude-fleet"
	@mkdir -p ~/.tmux/plugins
	@ln -sf $(PWD) ~/.tmux/plugins/tmux-claude-fleet
	@echo "Plugin installed! Add to .tmux.conf:"
	@echo "  run-shell ~/.tmux/plugins/tmux-claude-fleet/claude-fleet.tmux"

# Verify installation
verify:
	@./scripts/verify-install.sh

# Show help
help:
	@echo "Tmux Claude Fleet - Makefile targets:"
	@echo ""
	@echo "  make test          - Run all tests"
	@echo "  make test-unit     - Run unit tests only"
	@echo "  make test-file FILE=path/to/test.bats - Run specific test file"
	@echo "  make check-deps    - Check for required dependencies"
	@echo "  make clean         - Clean temporary files"
	@echo "  make install       - Install plugin for development"
	@echo "  make verify        - Verify installation"
	@echo "  make help          - Show this help message"

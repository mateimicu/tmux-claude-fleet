.PHONY: build install test clean cross-compile

BINARY_NAME=claude-fleet
BUILD_DIR=./bin
CMD_DIR=./cmd/claude-fleet
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X main.Version=$(VERSION)"

# Build for current platform
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)
	@echo "✓ Built: $(BUILD_DIR)/$(BINARY_NAME)"

# Install to bin directory
install: build
	@echo "✓ Installed to $(BUILD_DIR)/$(BINARY_NAME)"

# Run tests
test:
	@echo "Running tests..."
	go test -v -race -cover ./...

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	go clean

# Cross-compile for macOS and Linux
cross-compile:
	@echo "Cross-compiling for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	@echo "Building for macOS (amd64)..."
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(CMD_DIR)
	@echo "Building for macOS (arm64)..."
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(CMD_DIR)
	@echo "Building for Linux (amd64)..."
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_DIR)
	@echo "Building for Linux (arm64)..."
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(CMD_DIR)
	@echo "✓ Cross-compilation complete!"
	@ls -lh $(BUILD_DIR)

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Run the binary
run: build
	$(BUILD_DIR)/$(BINARY_NAME)

# Development build with race detector
dev:
	@echo "Building with race detector..."
	@mkdir -p $(BUILD_DIR)
	go build -race $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)

# Show help
help:
	@echo "Available targets:"
	@echo "  build          - Build for current platform"
	@echo "  install        - Build and install to bin directory"
	@echo "  test           - Run tests"
	@echo "  clean          - Clean build artifacts"
	@echo "  cross-compile  - Build for macOS and Linux (amd64/arm64)"
	@echo "  fmt            - Format code"
	@echo "  run            - Build and run"
	@echo "  dev            - Build with race detector"
	@echo "  help           - Show this help"

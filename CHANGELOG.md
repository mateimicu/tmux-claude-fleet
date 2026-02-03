# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial release of tmux-claude-fleet plugin
- Interactive repository selection with fzf
- Multi-source repository discovery (local config + GitHub API)
- Automated session creation with terminal and Claude windows
- Session management (list, switch, delete)
- Claude status monitoring
- Persistent session metadata storage
- Configuration management with multiple sources
- File-based locking for concurrent operations
- Comprehensive error handling and logging
- Test suite with BATS
- Full documentation and examples

### Features
- Create sessions: Clone repos and start tmux with Claude integration
- List sessions: Browse with rich preview showing status and git history
- Delete sessions: Clean up sessions and optionally remove cloned repos
- GitHub integration: Fetch repos via gh CLI or GITHUB_TOKEN
- Local config: Manage repository list in text file
- Caching: Cache GitHub API responses for performance
- Compatibility: Support tmux 2.0+ with fallback for older versions

### Technical Highlights
- Clean shell script architecture with separation of concerns
- TDD approach with comprehensive unit tests
- Defensive programming with error handling and validation
- Lock-based concurrency control
- Modular library design for maintainability

## [1.0.0] - YYYY-MM-DD

### Added
- Initial public release

[Unreleased]: https://github.com/mateimicu/tmux-claude-fleet/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/mateimicu/tmux-claude-fleet/releases/tag/v1.0.0

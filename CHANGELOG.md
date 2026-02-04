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
- `refresh` command to manually update repository cache
- Cache age display when using cached data
- Progress feedback during GitHub API pagination

### Changed
- **PERFORMANCE**: Increased default cache TTL from 5 minutes to 30 minutes
- Improved cache feedback showing age and source (cached vs API fetch)
- Enhanced GitHub API fetch with pagination progress indicators

### Features
- Create sessions: Clone repos and start tmux with Claude integration
- List sessions: Browse with rich preview showing status and git history
- Delete sessions: Clean up sessions and optionally remove cloned repos
- Refresh cache: Force update cached GitHub repositories
- GitHub integration: Fetch repos via gh CLI or GITHUB_TOKEN
- Local config: Manage repository list in text file
- Intelligent caching: Cache GitHub API responses with 30-minute TTL
  - Shows cache age when using cached data (e.g., "age: 5.2m")
  - Automatic cache expiration and refresh
  - Manual refresh with `claude-fleet refresh` command
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

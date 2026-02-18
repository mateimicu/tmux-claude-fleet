package main

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/mateimicu/tmux-claude-matrix/internal/logging"
	"github.com/mateimicu/tmux-claude-matrix/internal/repos"
	"github.com/mateimicu/tmux-claude-matrix/pkg/types"
)

// buildSources creates the list of repository sources based on config.
// The log parameter controls where status and warning messages are written.
// Debug messages use log.DebugW; the GitHub auth warning uses log.WarnW
// so it is always visible regardless of debug mode.
func buildSources(ctx context.Context, cfg *types.Config, log *logging.Logger) ([]repos.Source, error) {
	var sources []repos.Source

	if cfg.WorkspacesEnabled && cfg.WorkspacesFile != "" {
		sources = append(sources, repos.NewWorkspaceSource(cfg.WorkspacesFile))
	}

	if cfg.LocalConfigEnabled && cfg.LocalReposFile != "" {
		sources = append(sources, repos.NewLocalSource(cfg.LocalReposFile))
	}

	if cfg.GitHubEnabled {
		token, source := repos.GetGitHubToken(ctx)
		if token == "" {
			fmt.Fprintln(log.WarnW, "⚠️  GitHub authentication not found, skipping GitHub repositories") //nolint:errcheck // Logging output is non-critical
		} else {
			fmt.Fprintf(log.DebugW, "✓ GitHub integration enabled (using %s)\n", source) //nolint:errcheck // Logging output is non-critical
			if len(cfg.GitHubOrgs) > 0 {
				fmt.Fprintf(log.DebugW, "  Filtering by organizations: %s\n", strings.Join(cfg.GitHubOrgs, ", ")) //nolint:errcheck // Logging output is non-critical
			}
			ghSource := repos.NewGitHubSource(token, cfg.CacheDir, cfg.CacheTTL, cfg.GitHubOrgs)
			ghSource.SetLogger(log.DebugW)
			sources = append(sources, ghSource)
		}
	}

	if len(sources) == 0 {
		return nil, fmt.Errorf("no repository sources configured")
	}

	return sources, nil
}

// buildSourcesWithWriter creates the list of repository sources for callers
// that need an io.Writer interface (e.g., list-repos which always discards).
func buildSourcesWithWriter(ctx context.Context, cfg *types.Config, w io.Writer) ([]repos.Source, error) {
	log := &logging.Logger{DebugW: w, WarnW: w}
	return buildSources(ctx, cfg, log)
}

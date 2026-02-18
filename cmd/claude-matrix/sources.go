package main

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/mateimicu/tmux-claude-matrix/internal/repos"
	"github.com/mateimicu/tmux-claude-matrix/pkg/types"
)

// buildSources creates the list of repository sources based on config.
// The logger parameter controls where status messages are written (use io.Discard to suppress).
func buildSources(ctx context.Context, cfg *types.Config, logger io.Writer) ([]repos.Source, error) {
	var sources []repos.Source
	var githubSkippedForAuth bool

	if cfg.WorkspacesEnabled && cfg.WorkspacesFile != "" {
		sources = append(sources, repos.NewWorkspaceSource(cfg.WorkspacesFile))
	}

	if cfg.LocalConfigEnabled && cfg.LocalReposFile != "" {
		sources = append(sources, repos.NewLocalSource(cfg.LocalReposFile))
	}

	if cfg.GitHubEnabled {
		token, source := repos.GetGitHubToken(ctx)
		if token == "" {
			fmt.Fprintln(logger, "⚠️  GitHub authentication not found, skipping GitHub repositories") //nolint:errcheck // Logging output is non-critical
			githubSkippedForAuth = true
		} else {
			fmt.Fprintf(logger, "✓ GitHub integration enabled (using %s)\n", source) //nolint:errcheck // Logging output is non-critical
			if len(cfg.GitHubOrgs) > 0 {
				fmt.Fprintf(logger, "  Filtering by organizations: %s\n", strings.Join(cfg.GitHubOrgs, ", ")) //nolint:errcheck // Logging output is non-critical
			}
			ghSource := repos.NewGitHubSource(token, cfg.CacheDir, cfg.CacheTTL, cfg.GitHubOrgs)
			ghSource.SetLogger(logger)
			sources = append(sources, ghSource)
		}
	}

	if len(sources) == 0 {
		if githubSkippedForAuth {
			return nil, fmt.Errorf("GitHub source enabled but no authentication token found (set GITHUB_TOKEN or authenticate with gh)")
		}
		return nil, fmt.Errorf("no repository sources configured")
	}

	return sources, nil
}

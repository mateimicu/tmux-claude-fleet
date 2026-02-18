package main

import (
	"context"
	"fmt"
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
			log.Warnf("⚠️  GitHub authentication not found, skipping GitHub repositories\n")
		} else {
			log.Debugf("✓ GitHub integration enabled (using %s)\n", source)
			if len(cfg.GitHubOrgs) > 0 {
				log.Debugf("  Filtering by organizations: %s\n", strings.Join(cfg.GitHubOrgs, ", "))
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

package main

import (
	"context"
	"fmt"
	"io"

	"github.com/mateimicu/tmux-claude-matrix/internal/git"
	"github.com/mateimicu/tmux-claude-matrix/internal/repos"
	"github.com/mateimicu/tmux-claude-matrix/pkg/types"
)

// flattenRepoURLs extracts all unique clone URLs from a list of repositories.
// Workspace repos are expanded into their individual sub-repo URLs.
func flattenRepoURLs(repoList []*types.Repository) []string {
	seen := make(map[string]bool)
	var urls []string

	for _, repo := range repoList {
		if repo.IsWorkspace && len(repo.WorkspaceRepos) > 0 {
			for _, u := range repo.WorkspaceRepos {
				if u != "" && !seen[u] {
					urls = append(urls, u)
					seen[u] = true
				}
			}
			continue
		}
		if repo.URL != "" && !seen[repo.URL] {
			urls = append(urls, repo.URL)
			seen[repo.URL] = true
		}
	}

	return urls
}

// runPrefillCache discovers all configured repositories and creates/updates
// mirror caches for each one.
func runPrefillCache(ctx context.Context, cfg *types.Config) error {
	// Build sources (suppress log output during pre-fill)
	sources, err := buildSources(ctx, cfg, io.Discard)
	if err != nil {
		return err
	}

	// Discover repos
	discoverer := repos.NewDiscoverer(sources...)
	fmt.Println("🔍 Discovering repositories...")

	repoList, err := discoverer.ListAll(ctx)
	if err != nil {
		return fmt.Errorf("failed to discover repositories: %w", err)
	}

	urls := flattenRepoURLs(repoList)
	if len(urls) == 0 {
		fmt.Println("No repositories found to cache.")
		return nil
	}

	fmt.Printf("Found %d repositories to cache.\n\n", len(urls))

	gitMgr := git.New()
	var newCount, updatedCount, failedCount int
	total := len(urls)

	for i, url := range urls {
		// Check for cancellation between repos
		select {
		case <-ctx.Done():
			fmt.Printf("\n⚠️  Cancelled. Partial summary: Total: %d | New: %d | Updated: %d | Failed: %d\n",
				i, newCount, updatedCount, failedCount)
			return nil
		default:
		}

		repoName := git.ExtractRepoName(url)
		fmt.Printf("[%d/%d] Caching mirror: %s...\n", i+1, total, repoName)

		created, err := gitMgr.EnsureMirror(url, cfg.CacheDir)
		if err != nil {
			fmt.Printf("[%d/%d] ✗ %s: %v\n", i+1, total, repoName, err)
			failedCount++
			continue
		}

		if created {
			fmt.Printf("[%d/%d] ✓ %s (new)\n", i+1, total, repoName)
			newCount++
		} else {
			fmt.Printf("[%d/%d] ✓ %s (updated)\n", i+1, total, repoName)
			updatedCount++
		}
	}

	fmt.Printf("\nTotal: %d | New: %d | Updated: %d | Failed: %d\n",
		total, newCount, updatedCount, failedCount)

	return nil
}

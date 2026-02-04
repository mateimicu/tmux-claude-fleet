package repos

import (
	"context"

	"github.com/mateimicu/tmux-claude-matrix/pkg/types"
)

// Source is the interface for repository discovery
type Source interface {
	List(ctx context.Context) ([]*types.Repository, error)
	Name() string // "local", "github"
}

// Discoverer aggregates multiple sources
type Discoverer struct {
	sources []Source
}

// NewDiscoverer creates a new repository discoverer
func NewDiscoverer(sources ...Source) *Discoverer {
	return &Discoverer{sources: sources}
}

// ListAll fetches repos from all enabled sources and deduplicates
func (d *Discoverer) ListAll(ctx context.Context) ([]*types.Repository, error) {
	var allRepos []*types.Repository
	seen := make(map[string]bool)

	for _, source := range d.sources {
		repos, err := source.List(ctx)
		if err != nil {
			// Log warning but continue with other sources
			continue
		}

		for _, repo := range repos {
			if !seen[repo.URL] {
				allRepos = append(allRepos, repo)
				seen[repo.URL] = true
			}
		}
	}

	return allRepos, nil
}

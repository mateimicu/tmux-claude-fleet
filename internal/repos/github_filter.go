package repos

import (
	"strings"

	"github.com/mateimicu/tmux-claude-fleet/pkg/types"
)

// filterByOrgs filters repositories by organization (case-insensitive)
func (g *GitHubSource) filterByOrgs(repos []*types.Repository) []*types.Repository {
	// No filter, return all
	if len(g.orgs) == 0 {
		return repos
	}

	// Create org lookup map for O(1) lookups (case-insensitive)
	orgMap := make(map[string]bool)
	for _, org := range g.orgs {
		orgMap[strings.ToLower(org)] = true
	}

	// Filter repos by org
	filtered := make([]*types.Repository, 0)
	for _, repo := range repos {
		// Extract org from full name (org/repo)
		parts := strings.SplitN(repo.Name, "/", 2)
		if len(parts) > 0 {
			repoOrg := strings.ToLower(parts[0])
			if orgMap[repoOrg] {
				filtered = append(filtered, repo)
			}
		}
	}

	return filtered
}

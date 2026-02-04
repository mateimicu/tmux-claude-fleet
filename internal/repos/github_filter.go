package repos

import (
	"strings"

	"github.com/mateimicu/tmux-claude-fleet/pkg/types"
)

// filterByOrgs filters repositories by organization
func (g *GitHubSource) filterByOrgs(repos []*types.Repository) []*types.Repository {
	// No filter, return all
	if len(g.orgs) == 0 {
		return repos
	}

	filtered := make([]*types.Repository, 0)
	for _, repo := range repos {
		// Extract org from full name (org/repo)
		parts := strings.Split(repo.Name, "/")
		if len(parts) >= 2 {
			repoOrg := parts[0]

			// Check if this org is in the filter list
			for _, allowedOrg := range g.orgs {
				if repoOrg == allowedOrg {
					filtered = append(filtered, repo)
					break
				}
			}
		}
	}

	return filtered
}

package repos

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mateimicu/tmux-claude-fleet/pkg/types"
)

type GitHubSource struct {
	token    string
	cacheDir string
	cacheTTL time.Duration
	client   *http.Client
	orgs     []string // Filter by these organizations (empty = all)
}

func NewGitHubSource(token, cacheDir string, cacheTTL time.Duration, orgs []string) *GitHubSource {
	return &GitHubSource{
		token:    token,
		cacheDir: cacheDir,
		cacheTTL: cacheTTL,
		client:   &http.Client{Timeout: 30 * time.Second},
		orgs:     orgs,
	}
}

func (g *GitHubSource) Name() string {
	return "github"
}

type ghRepo struct {
	FullName    string `json:"full_name"`
	Description string `json:"description"`
	CloneURL    string `json:"clone_url"`
}

type cacheData struct {
	Timestamp time.Time           `json:"timestamp"`
	Repos     []*types.Repository `json:"repos"`
}

func (g *GitHubSource) List(ctx context.Context) ([]*types.Repository, error) {
	// Check cache
	if repos, ok := g.checkCache(); ok {
		return repos, nil
	}

	// Fetch from API
	repos, err := g.fetchFromAPI(ctx)
	if err != nil {
		return nil, err
	}

	// Update cache
	g.saveCache(repos)

	return repos, nil
}

func (g *GitHubSource) fetchFromAPI(ctx context.Context) ([]*types.Repository, error) {
	var allRepos []*types.Repository
	page := 1
	perPage := 100

	for {
		url := fmt.Sprintf("https://api.github.com/user/repos?per_page=%d&page=%d", perPage, page)
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, err
		}

		if g.token != "" {
			req.Header.Set("Authorization", "token "+g.token)
		}
		req.Header.Set("Accept", "application/vnd.github.v3+json")

		resp, err := g.client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
		}

		var ghRepos []ghRepo
		if err := json.NewDecoder(resp.Body).Decode(&ghRepos); err != nil {
			return nil, err
		}

		if len(ghRepos) == 0 {
			break
		}

		for _, gr := range ghRepos {
			// Filter by organization if specified
			if len(g.orgs) > 0 {
				repo := &types.Repository{
					Source:      "github",
					URL:         gr.CloneURL,
					Name:        gr.FullName,
					Description: gr.Description,
				}

				// Extract org from full name (org/repo)
				parts := strings.Split(gr.FullName, "/")
				if len(parts) >= 2 {
					repoOrg := parts[0]

					// Check if this org is in the filter list
					for _, allowedOrg := range g.orgs {
						if repoOrg == allowedOrg {
							allRepos = append(allRepos, repo)
							break
						}
					}
				}
			} else {
				// No filter, include all repos
				allRepos = append(allRepos, &types.Repository{
					Source:      "github",
					URL:         gr.CloneURL,
					Name:        gr.FullName,
					Description: gr.Description,
				})
			}
		}

		// Check if there are more pages
		if len(ghRepos) < perPage {
			break
		}

		page++
	}

	return allRepos, nil
}

func (g *GitHubSource) checkCache() ([]*types.Repository, bool) {
	cachePath := filepath.Join(g.cacheDir, "github-repos.json")

	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, false
	}

	var cache cacheData
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, false
	}

	// Check if cache is still valid
	if time.Since(cache.Timestamp) > g.cacheTTL {
		return nil, false
	}

	return cache.Repos, true
}

func (g *GitHubSource) saveCache(repos []*types.Repository) {
	if err := os.MkdirAll(g.cacheDir, 0755); err != nil {
		return
	}

	cache := cacheData{
		Timestamp: time.Now(),
		Repos:     repos,
	}

	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return
	}

	cachePath := filepath.Join(g.cacheDir, "github-repos.json")
	os.WriteFile(cachePath, data, 0644)
}

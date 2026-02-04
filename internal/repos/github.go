package repos

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/mateimicu/tmux-claude-fleet/pkg/types"
)

// GitHubSource discovers repositories from GitHub
type GitHubSource struct {
	client   *http.Client
	token    string
	cacheDir string
	orgs     []string
	cacheTTL time.Duration
}

// NewGitHubSource creates a new GitHub repository source
func NewGitHubSource(token, cacheDir string, cacheTTL time.Duration, orgs []string) *GitHubSource {
	return &GitHubSource{
		token:    token,
		cacheDir: cacheDir,
		cacheTTL: cacheTTL,
		client:   &http.Client{Timeout: 30 * time.Second},
		orgs:     orgs,
	}
}

// Name returns the source name
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

// List returns all repositories from GitHub
func (g *GitHubSource) List(ctx context.Context) ([]*types.Repository, error) {
	// Check cache
	if repos, ok := g.checkCache(); ok {
		// Apply organization filter to cached repos
		return g.filterByOrgs(repos), nil
	}

	// Fetch from API (gets all repos)
	repos, err := g.fetchFromAPI(ctx)
	if err != nil {
		return nil, err
	}

	// Update cache (with all repos for flexibility)
	g.saveCache(repos)

	// Apply organization filter before returning
	return g.filterByOrgs(repos), nil
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
		defer func() {
			if cerr := resp.Body.Close(); cerr != nil && err == nil {
				err = cerr
			}
		}()

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
			// Add all repos (filtering will be done by filterByOrgs)
			allRepos = append(allRepos, &types.Repository{
				Source:      "github",
				URL:         gr.CloneURL,
				Name:        gr.FullName,
				Description: gr.Description,
			})
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
	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		// Silently ignore cache write errors
		return
	}
}

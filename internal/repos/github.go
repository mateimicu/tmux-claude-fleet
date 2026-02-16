package repos

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/mateimicu/tmux-claude-matrix/pkg/types"
)

// GitHubSource discovers repositories from GitHub
type GitHubSource struct {
	client       *http.Client
	token        string
	cacheDir     string
	orgs         []string
	cacheTTL     time.Duration
	logger       io.Writer // Output for logging
	forceRefresh bool
}

// NewGitHubSource creates a new GitHub repository source
func NewGitHubSource(token, cacheDir string, cacheTTL time.Duration, orgs []string) *GitHubSource {
	return &GitHubSource{
		token:    token,
		cacheDir: cacheDir,
		cacheTTL: cacheTTL,
		client:   &http.Client{Timeout: 30 * time.Second},
		orgs:     orgs,
		logger:   os.Stdout,
	}
}

// SetLogger sets the logger for this source
func (g *GitHubSource) SetLogger(w io.Writer) {
	g.logger = w
}

// SetForceRefresh enables force refresh mode.
// When enabled, List() bypasses TTL and always attempts API fetch.
// On API failure, it falls back to stale cached data.
func (g *GitHubSource) SetForceRefresh(force bool) {
	g.forceRefresh = force
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
	cached, cacheAge, cacheValid := g.checkCache()

	// If not force-refreshing and cache is valid, use it
	if !g.forceRefresh && cacheValid {
		if g.logger != nil {
			fmt.Fprintf(g.logger, "  ✓ Using cached GitHub repos (age: %s)\n", formatDuration(cacheAge)) //nolint:errcheck // Logging output is non-critical
		}
		return g.filterByOrgs(cached), nil
	}

	// Fetch from API
	if g.logger != nil {
		fmt.Fprintf(g.logger, "  ⟳ Fetching GitHub repos from API...\n") //nolint:errcheck // Logging output is non-critical
	}
	repos, err := g.fetchFromAPI(ctx)
	if err != nil {
		// On force-refresh failure, fall back to stale cache
		if g.forceRefresh && cached != nil {
			if g.logger != nil {
				fmt.Fprintf(g.logger, "  ⚠️ API fetch failed, using stale cache\n") //nolint:errcheck // Logging output is non-critical
			}
			return g.filterByOrgs(cached), nil
		}
		return nil, err
	}

	// Update cache (with all repos for flexibility)
	g.saveCache(repos)
	if g.logger != nil {
		fmt.Fprintf(g.logger, "  ✓ Cached %d repos for future use\n", len(repos)) //nolint:errcheck // Logging output is non-critical
	}

	return g.filterByOrgs(repos), nil
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	}
	return fmt.Sprintf("%.1fh", d.Hours())
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

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close() //nolint:errcheck // Already returning an error
			return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
		}

		var ghRepos []ghRepo
		decodeErr := json.NewDecoder(resp.Body).Decode(&ghRepos)
		resp.Body.Close() //nolint:errcheck // Close error is non-critical after reading
		if decodeErr != nil {
			return nil, decodeErr
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

		// Show progress for multiple pages
		if page > 1 && g.logger != nil {
			fmt.Fprintf(g.logger, "  ⟳ Fetched %d repos (page %d)...\n", len(allRepos), page) //nolint:errcheck // Logging output is non-critical
		}

		// Check if there are more pages
		if len(ghRepos) < perPage {
			break
		}

		page++
	}

	return allRepos, nil
}

func (g *GitHubSource) checkCache() (repos []*types.Repository, age time.Duration, valid bool) {
	cachePath := filepath.Join(g.cacheDir, "github-repos.json")

	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, 0, false
	}

	var cache cacheData
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, 0, false
	}

	age = time.Since(cache.Timestamp)
	return cache.Repos, age, age <= g.cacheTTL
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

// ClearCache removes the cache file
func (g *GitHubSource) ClearCache() error {
	cachePath := filepath.Join(g.cacheDir, "github-repos.json")
	if err := os.Remove(cachePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to clear cache: %w", err)
	}
	return nil
}

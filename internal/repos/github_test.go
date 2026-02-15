package repos

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mateimicu/tmux-claude-matrix/pkg/types"
)

// redirectTransport rewrites all requests to target a local test server.
type redirectTransport struct {
	targetHost string
}

func (t *redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.URL.Scheme = "http"
	req.URL.Host = t.targetHost
	return http.DefaultTransport.RoundTrip(req)
}

func TestGitHubSource_Cache(t *testing.T) {
	// Create temporary cache directory
	tmpDir, err := os.MkdirTemp("", "github-cache-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test repos
	testRepos := []*types.Repository{
		{Source: "github", URL: "https://github.com/test/repo1", Name: "test/repo1", Description: "Test repo 1"},
		{Source: "github", URL: "https://github.com/test/repo2", Name: "test/repo2", Description: "Test repo 2"},
	}

	// Create GitHub source with short TTL
	source := NewGitHubSource("", tmpDir, 10*time.Second, []string{})
	var logBuf bytes.Buffer
	source.SetLogger(&logBuf)

	t.Run("SaveAndLoadCache", func(t *testing.T) {
		// Save cache
		source.saveCache(testRepos)

		// Load cache
		repos, age, ok := source.checkCache()
		if !ok {
			t.Fatal("Failed to load cache")
		}

		if len(repos) != len(testRepos) {
			t.Errorf("Expected %d repos, got %d", len(testRepos), len(repos))
		}

		if age > 1*time.Second {
			t.Errorf("Cache age should be < 1s, got %v", age)
		}

		// Verify repos match
		for i, repo := range repos {
			if repo.URL != testRepos[i].URL {
				t.Errorf("Repo %d URL mismatch: expected %s, got %s", i, testRepos[i].URL, repo.URL)
			}
		}
	})

	t.Run("CacheExpiration", func(t *testing.T) {
		// Create source with very short TTL
		shortTTLSource := NewGitHubSource("", tmpDir, 100*time.Millisecond, []string{})

		// Save cache
		shortTTLSource.saveCache(testRepos)

		// Wait for cache to expire
		time.Sleep(150 * time.Millisecond)

		// Check cache - should be expired but still return data
		repos, _, valid := shortTTLSource.checkCache()
		if valid {
			t.Error("Cache should have expired")
		}
		if len(repos) != len(testRepos) {
			t.Errorf("Expected %d stale repos, got %d", len(testRepos), len(repos))
		}
	})

	t.Run("ClearCache", func(t *testing.T) {
		// Save cache
		source.saveCache(testRepos)

		// Verify cache exists
		_, _, ok := source.checkCache()
		if !ok {
			t.Fatal("Cache should exist")
		}

		// Clear cache
		if err := source.ClearCache(); err != nil {
			t.Fatalf("Failed to clear cache: %v", err)
		}

		// Verify cache is cleared
		_, _, ok = source.checkCache()
		if ok {
			t.Error("Cache should be cleared")
		}
	})

	t.Run("CacheLogging", func(t *testing.T) {
		// Save cache
		source.saveCache(testRepos)

		// Reset log buffer
		logBuf.Reset()

		// Load cache (should log cache hit)
		repos, _, ok := source.checkCache()
		if !ok {
			t.Fatal("Failed to load cache")
		}

		if len(repos) != len(testRepos) {
			t.Errorf("Expected %d repos, got %d", len(testRepos), len(repos))
		}
	})
}

func TestGitHubSource_FilterByOrgs(t *testing.T) {
	testRepos := []*types.Repository{
		{Name: "org1/repo1"},
		{Name: "org2/repo2"},
		{Name: "org1/repo3"},
		{Name: "org3/repo4"},
	}

	t.Run("NoFilter", func(t *testing.T) {
		source := NewGitHubSource("", "", 5*time.Minute, []string{})
		filtered := source.filterByOrgs(testRepos)

		if len(filtered) != len(testRepos) {
			t.Errorf("Expected %d repos, got %d", len(testRepos), len(filtered))
		}
	})

	t.Run("SingleOrgFilter", func(t *testing.T) {
		source := NewGitHubSource("", "", 5*time.Minute, []string{"org1"})
		filtered := source.filterByOrgs(testRepos)

		if len(filtered) != 2 {
			t.Errorf("Expected 2 repos, got %d", len(filtered))
		}

		for _, repo := range filtered {
			if !strings.HasPrefix(repo.Name, "org1/") {
				t.Errorf("Unexpected repo: %s", repo.Name)
			}
		}
	})

	t.Run("MultipleOrgFilter", func(t *testing.T) {
		source := NewGitHubSource("", "", 5*time.Minute, []string{"org1", "org2"})
		filtered := source.filterByOrgs(testRepos)

		if len(filtered) != 3 {
			t.Errorf("Expected 3 repos, got %d", len(filtered))
		}
	})

	t.Run("CaseInsensitiveFilter", func(t *testing.T) {
		source := NewGitHubSource("", "", 5*time.Minute, []string{"ORG1"})
		filtered := source.filterByOrgs(testRepos)

		if len(filtered) != 2 {
			t.Errorf("Expected 2 repos (case-insensitive), got %d", len(filtered))
		}
	})
}

func TestCacheData_Serialization(t *testing.T) {
	testRepos := []*types.Repository{
		{Source: "github", URL: "https://github.com/test/repo1", Name: "test/repo1", Description: "Test repo 1"},
	}

	cache := cacheData{
		Timestamp: time.Now(),
		Repos:     testRepos,
	}

	// Serialize
	data, err := json.Marshal(cache)
	if err != nil {
		t.Fatalf("Failed to marshal cache: %v", err)
	}

	// Deserialize
	var decoded cacheData
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal cache: %v", err)
	}

	// Verify
	if len(decoded.Repos) != len(testRepos) {
		t.Errorf("Expected %d repos, got %d", len(testRepos), len(decoded.Repos))
	}

	if decoded.Repos[0].URL != testRepos[0].URL {
		t.Errorf("URL mismatch: expected %s, got %s", testRepos[0].URL, decoded.Repos[0].URL)
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{30 * time.Second, "30s"},
		{1 * time.Minute, "1.0m"},
		{5*time.Minute + 30*time.Second, "5.5m"},
		{1 * time.Hour, "1.0h"},
		{2*time.Hour + 30*time.Minute, "2.5h"},
	}

	for _, test := range tests {
		result := formatDuration(test.duration)
		if result != test.expected {
			t.Errorf("formatDuration(%v) = %s, expected %s", test.duration, result, test.expected)
		}
	}
}

func TestGitHubSource_CachePath(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "github-cache-path-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	source := NewGitHubSource("", tmpDir, 5*time.Minute, []string{})

	// Save cache
	testRepos := []*types.Repository{
		{Source: "github", URL: "https://github.com/test/repo1", Name: "test/repo1"},
	}
	source.saveCache(testRepos)

	// Verify cache file exists at expected location
	expectedPath := filepath.Join(tmpDir, "github-repos.json")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("Cache file not found at expected path: %s", expectedPath)
	}
}

func TestGitHubSource_CacheCorruption(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "github-cache-corrupt-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	source := NewGitHubSource("", tmpDir, 5*time.Minute, []string{})

	// Write corrupted cache data
	cachePath := filepath.Join(tmpDir, "github-repos.json")
	if err := os.WriteFile(cachePath, []byte("invalid json"), 0644); err != nil {
		t.Fatalf("Failed to write corrupted cache: %v", err)
	}

	// Try to load cache - should handle corruption gracefully
	_, _, ok := source.checkCache()
	if ok {
		t.Error("Should not load corrupted cache")
	}
}

func TestGitHubSource_SetLogger(t *testing.T) {
	source := NewGitHubSource("", "", 5*time.Minute, []string{})

	// Default logger should be os.Stdout
	if source.logger != os.Stdout {
		t.Error("Default logger should be os.Stdout")
	}

	// Set custom logger
	var buf bytes.Buffer
	source.SetLogger(&buf)

	if source.logger != &buf {
		t.Error("Logger not set correctly")
	}
}

func TestGitHubSource_Name(t *testing.T) {
	source := NewGitHubSource("", "", 5*time.Minute, []string{})

	if source.Name() != "github" {
		t.Errorf("Expected name 'github', got '%s'", source.Name())
	}
}

func TestGitHubSource_ForceRefresh(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "github-force-refresh-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	t.Run("ForceRefreshFlag", func(t *testing.T) {
		source := NewGitHubSource("", tmpDir, 24*time.Hour, []string{})
		source.SetForceRefresh(true)

		if !source.forceRefresh {
			t.Error("forceRefresh should be true")
		}
	})
}

func TestGitHubSource_StaleCacheViaCheckCache(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "github-stale-cache-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testRepos := []*types.Repository{
		{Source: "github", URL: "https://github.com/test/repo1", Name: "test/repo1"},
	}

	source := NewGitHubSource("", tmpDir, 1*time.Millisecond, []string{})
	source.saveCache(testRepos)
	time.Sleep(5 * time.Millisecond) // Let cache expire

	// checkCache should return stale data with valid=false
	repos, _, valid := source.checkCache()
	if valid {
		t.Error("expected valid=false for expired cache")
	}
	if len(repos) != 1 {
		t.Fatalf("expected 1 stale repo, got %d", len(repos))
	}
	if repos[0].URL != testRepos[0].URL {
		t.Errorf("expected URL %q, got %q", testRepos[0].URL, repos[0].URL)
	}
}

func TestGitHubSource_List(t *testing.T) {
	apiRepos := []ghRepo{
		{FullName: "org/repo1", Description: "Repo one", CloneURL: "https://github.com/org/repo1.git"},
		{FullName: "org/repo2", Description: "Repo two", CloneURL: "https://github.com/org/repo2.git"},
	}

	t.Run("FreshFetchFromAPI", func(t *testing.T) {
		var apiCalls atomic.Int32
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiCalls.Add(1)
			json.NewEncoder(w).Encode(apiRepos) //nolint:errcheck
		}))
		defer ts.Close()

		tmpDir := t.TempDir()
		source := NewGitHubSource("test-token", tmpDir, 24*time.Hour, []string{})
		source.client = &http.Client{Transport: &redirectTransport{targetHost: ts.Listener.Addr().String()}}
		source.SetLogger(&bytes.Buffer{})

		repos, err := source.List(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(repos) != 2 {
			t.Fatalf("expected 2 repos, got %d", len(repos))
		}
		if apiCalls.Load() != 1 {
			t.Errorf("expected 1 API call, got %d", apiCalls.Load())
		}
		// Verify cache was saved
		_, _, valid := source.checkCache()
		if !valid {
			t.Error("cache should be valid after fresh fetch")
		}
	})

	t.Run("UsesValidCache", func(t *testing.T) {
		var apiCalls atomic.Int32
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiCalls.Add(1)
			json.NewEncoder(w).Encode(apiRepos) //nolint:errcheck
		}))
		defer ts.Close()

		tmpDir := t.TempDir()
		source := NewGitHubSource("test-token", tmpDir, 24*time.Hour, []string{})
		source.client = &http.Client{Transport: &redirectTransport{targetHost: ts.Listener.Addr().String()}}
		source.SetLogger(&bytes.Buffer{})

		// Pre-populate valid cache
		cachedRepos := []*types.Repository{
			{Source: "github", URL: "https://github.com/cached/repo1.git", Name: "cached/repo1"},
		}
		source.saveCache(cachedRepos)

		repos, err := source.List(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if apiCalls.Load() != 0 {
			t.Errorf("expected 0 API calls (cache hit), got %d", apiCalls.Load())
		}
		if len(repos) != 1 || repos[0].Name != "cached/repo1" {
			t.Errorf("expected cached repo, got %v", repos)
		}
	})

	t.Run("ForceRefreshBypassesValidCache", func(t *testing.T) {
		var apiCalls atomic.Int32
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiCalls.Add(1)
			json.NewEncoder(w).Encode(apiRepos) //nolint:errcheck
		}))
		defer ts.Close()

		tmpDir := t.TempDir()
		source := NewGitHubSource("test-token", tmpDir, 24*time.Hour, []string{})
		source.client = &http.Client{Transport: &redirectTransport{targetHost: ts.Listener.Addr().String()}}
		source.SetLogger(&bytes.Buffer{})
		source.SetForceRefresh(true)

		// Pre-populate valid cache with different data
		cachedRepos := []*types.Repository{
			{Source: "github", URL: "https://github.com/stale/old.git", Name: "stale/old"},
		}
		source.saveCache(cachedRepos)

		repos, err := source.List(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if apiCalls.Load() != 1 {
			t.Errorf("expected 1 API call (force refresh), got %d", apiCalls.Load())
		}
		if len(repos) != 2 {
			t.Fatalf("expected 2 fresh repos, got %d", len(repos))
		}
		if repos[0].Name != "org/repo1" {
			t.Errorf("expected fresh repo 'org/repo1', got %q", repos[0].Name)
		}
	})

	t.Run("ForceRefreshFallsBackToStaleCache", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer ts.Close()

		tmpDir := t.TempDir()
		source := NewGitHubSource("test-token", tmpDir, 1*time.Millisecond, []string{})
		source.client = &http.Client{Transport: &redirectTransport{targetHost: ts.Listener.Addr().String()}}
		var logBuf bytes.Buffer
		source.SetLogger(&logBuf)
		source.SetForceRefresh(true)

		// Pre-populate stale cache
		staleRepos := []*types.Repository{
			{Source: "github", URL: "https://github.com/stale/repo1.git", Name: "stale/repo1"},
		}
		source.saveCache(staleRepos)
		time.Sleep(5 * time.Millisecond) // Ensure cache is expired

		repos, err := source.List(context.Background())
		if err != nil {
			t.Fatalf("expected stale cache fallback, got error: %v", err)
		}
		if len(repos) != 1 || repos[0].Name != "stale/repo1" {
			t.Errorf("expected stale cache repo, got %v", repos)
		}
		if !strings.Contains(logBuf.String(), "stale cache") {
			t.Errorf("expected stale cache log message, got %q", logBuf.String())
		}
	})

	t.Run("NoCacheAPIFailureReturnsError", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer ts.Close()

		tmpDir := t.TempDir()
		source := NewGitHubSource("test-token", tmpDir, 24*time.Hour, []string{})
		source.client = &http.Client{Transport: &redirectTransport{targetHost: ts.Listener.Addr().String()}}
		source.SetLogger(&bytes.Buffer{})

		// No cache at all — API failure should be a real error
		_, err := source.List(context.Background())
		if err == nil {
			t.Fatal("expected error when API fails with no cache")
		}
	})

	t.Run("OrgFilterAppliedToAPIResults", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mixed := []ghRepo{
				{FullName: "org1/repo1", CloneURL: "https://github.com/org1/repo1.git"},
				{FullName: "org2/repo2", CloneURL: "https://github.com/org2/repo2.git"},
			}
			json.NewEncoder(w).Encode(mixed) //nolint:errcheck
		}))
		defer ts.Close()

		tmpDir := t.TempDir()
		source := NewGitHubSource("test-token", tmpDir, 24*time.Hour, []string{"org1"})
		source.client = &http.Client{Transport: &redirectTransport{targetHost: ts.Listener.Addr().String()}}
		source.SetLogger(&bytes.Buffer{})

		repos, err := source.List(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(repos) != 1 || repos[0].Name != "org1/repo1" {
			t.Errorf("expected only org1 repos, got %v", repos)
		}
	})
}

// Benchmark tests
func BenchmarkCheckCache(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "github-cache-bench")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	source := NewGitHubSource("", tmpDir, 5*time.Minute, []string{})

	// Create test data with many repos
	testRepos := make([]*types.Repository, 100)
	for i := 0; i < 100; i++ {
		testRepos[i] = &types.Repository{
			Source:      "github",
			URL:         "https://github.com/test/repo" + string(rune(i)),
			Name:        "test/repo" + string(rune(i)),
			Description: "Test repo",
		}
	}
	source.saveCache(testRepos)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		source.checkCache()
	}
}

func BenchmarkFilterByOrgs(b *testing.B) {
	// Create test data with many repos
	testRepos := make([]*types.Repository, 1000)
	for i := 0; i < 1000; i++ {
		orgNum := i % 10
		testRepos[i] = &types.Repository{
			Name: "org" + string(rune(orgNum)) + "/repo" + string(rune(i)),
		}
	}

	source := NewGitHubSource("", "", 5*time.Minute, []string{"org1", "org2", "org3"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		source.filterByOrgs(testRepos)
	}
}

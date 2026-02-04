package repos

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mateimicu/tmux-claude-fleet/pkg/types"
)

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

		// Check cache - should be expired
		_, _, ok := shortTTLSource.checkCache()
		if ok {
			t.Error("Cache should have expired")
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

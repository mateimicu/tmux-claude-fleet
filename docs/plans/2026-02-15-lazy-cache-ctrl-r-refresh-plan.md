# Lazy Cache Load + Ctrl+R Refresh Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Allow users to refresh the repository list via Ctrl+R in the FZF picker, with a 24h cache TTL so repos don't auto-refresh on every create.

**Architecture:** New `list-repos` CLI subcommand outputs FZF-formatted lines. FZF's `--bind ctrl-r:reload(...)` calls this subcommand with `--force-refresh` to clear cache and re-fetch. Shared source-building logic is extracted to avoid duplication across commands.

**Tech Stack:** Go 1.23, Cobra CLI, FZF `--bind reload`

---

### Task 1: Extract shared source builder

**Files:**
- Create: `cmd/claude-matrix/sources.go`
- Create: `cmd/claude-matrix/sources_test.go`
- Modify: `cmd/claude-matrix/create.go:40-68`
- Modify: `cmd/claude-matrix/refresh.go:43-69`

**Step 1: Write the test for buildSources**

Create `cmd/claude-matrix/sources_test.go`:

```go
package main

import (
	"io"
	"testing"
	"time"

	"github.com/mateimicu/tmux-claude-matrix/pkg/types"
)

func TestBuildSources_LocalOnly(t *testing.T) {
	cfg := &types.Config{
		LocalConfigEnabled: true,
		LocalReposFile:     "/tmp/nonexistent-repos.txt",
		GitHubEnabled:      false,
		WorkspacesEnabled:  false,
	}

	sources, err := buildSources(t.Context(), cfg, io.Discard)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(sources))
	}

	if sources[0].Name() != "local" {
		t.Errorf("expected source name 'local', got '%s'", sources[0].Name())
	}
}

func TestBuildSources_NoSourcesConfigured(t *testing.T) {
	cfg := &types.Config{
		LocalConfigEnabled: false,
		GitHubEnabled:      false,
		WorkspacesEnabled:  false,
	}

	_, err := buildSources(t.Context(), cfg, io.Discard)
	if err == nil {
		t.Fatal("expected error when no sources configured")
	}
}

func TestBuildSources_WorkspacesEnabled(t *testing.T) {
	cfg := &types.Config{
		WorkspacesEnabled: true,
		WorkspacesFile:    "/tmp/nonexistent-workspaces.yaml",
		LocalConfigEnabled: false,
		GitHubEnabled:      false,
	}

	sources, err := buildSources(t.Context(), cfg, io.Discard)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(sources))
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v -run TestBuildSources ./cmd/claude-matrix/`
Expected: FAIL with "undefined: buildSources"

**Step 3: Write buildSources implementation**

Create `cmd/claude-matrix/sources.go`:

```go
package main

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/mateimicu/tmux-claude-matrix/internal/repos"
	"github.com/mateimicu/tmux-claude-matrix/pkg/types"
)

// buildSources creates the list of repository sources based on config.
// The logger parameter controls where status messages are written (use io.Discard to suppress).
func buildSources(ctx context.Context, cfg *types.Config, logger io.Writer) ([]repos.Source, error) {
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
			fmt.Fprintln(logger, "⚠️  GitHub authentication not found, skipping GitHub repositories")
		} else {
			fmt.Fprintf(logger, "✓ GitHub integration enabled (using %s)\n", source)
			if len(cfg.GitHubOrgs) > 0 {
				fmt.Fprintf(logger, "  Filtering by organizations: %s\n", strings.Join(cfg.GitHubOrgs, ", "))
			}
			ghSource := repos.NewGitHubSource(token, cfg.CacheDir, cfg.CacheTTL, cfg.GitHubOrgs)
			ghSource.SetLogger(logger)
			sources = append(sources, ghSource)
		}
	}

	if len(sources) == 0 {
		return nil, fmt.Errorf("no repository sources configured")
	}

	return sources, nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test -v -run TestBuildSources ./cmd/claude-matrix/`
Expected: PASS

**Step 5: Refactor create.go to use buildSources**

In `cmd/claude-matrix/create.go`, replace lines 40-68 (the source-building block) with:

```go
	sources, err := buildSources(ctx, cfg, os.Stdout)
	if err != nil {
		return err
	}
```

Remove these imports from create.go if now unused: `"strings"`, and the `repos` import if only used for source building. Keep `repos` if still needed for other calls.

**Step 6: Refactor refresh.go to use buildSources**

In `cmd/claude-matrix/refresh.go`, replace lines 43-69 with:

```go
	sources, err := buildSources(ctx, cfg, os.Stdout)
	if err != nil {
		return err
	}
```

Remove unused imports.

**Step 7: Run all tests**

Run: `go test -v -race ./...`
Expected: All PASS

**Step 8: Run linter**

Run: `make lint`
Expected: No errors

**Step 9: Commit**

```bash
git add cmd/claude-matrix/sources.go cmd/claude-matrix/sources_test.go cmd/claude-matrix/create.go cmd/claude-matrix/refresh.go
git commit -m "refactor: extract shared buildSources helper from create and refresh"
```

---

### Task 2: Change default cache TTL to 24 hours

**Files:**
- Modify: `internal/config/config.go:56`

**Step 1: Change the default TTL**

In `internal/config/config.go`, line 56, change:

```go
CacheTTL: 30 * time.Minute, // Increased from 5m to 30m for better performance
```

to:

```go
CacheTTL: 24 * time.Hour,
```

**Step 2: Run tests**

Run: `go test -v -race ./...`
Expected: All PASS (no tests depend on the specific default value)

**Step 3: Run linter**

Run: `make lint`
Expected: No errors

**Step 4: Commit**

```bash
git add internal/config/config.go
git commit -m "feat: change default cache TTL to 24 hours"
```

---

### Task 3: Add list-repos subcommand

**Files:**
- Create: `cmd/claude-matrix/list_repos.go`
- Modify: `cmd/claude-matrix/main.go:24-33`

**Step 1: Create the list-repos command**

Create `cmd/claude-matrix/list_repos.go`:

```go
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/mateimicu/tmux-claude-matrix/internal/config"
	"github.com/mateimicu/tmux-claude-matrix/internal/fzf"
	"github.com/mateimicu/tmux-claude-matrix/internal/repos"
)

func listReposCmd() *cobra.Command {
	var forceRefresh bool

	cmd := &cobra.Command{
		Use:   "list-repos",
		Short: "List repositories in FZF-compatible format",
		Long:  `Output discovered repositories as formatted lines for FZF consumption. Used internally by the FZF reload binding.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runListRepos(cmd.Context(), forceRefresh)
		},
	}

	cmd.Flags().BoolVar(&forceRefresh, "force-refresh", false, "Clear cache before fetching")

	return cmd
}

func runListRepos(ctx context.Context, forceRefresh bool) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// If force refresh, clear the GitHub cache first
	if forceRefresh {
		cachePath := filepath.Join(cfg.CacheDir, "github-repos.json")
		if err := os.Remove(cachePath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to clear cache: %w", err)
		}
	}

	// Build sources with logging suppressed
	sources, err := buildSources(ctx, cfg, io.Discard)
	if err != nil {
		return err
	}

	discoverer := repos.NewDiscoverer(sources...)

	discoveryCtx, discoveryCancel := context.WithTimeout(ctx, 30*time.Second)
	defer discoveryCancel()

	repoList, err := discoverer.ListAll(discoveryCtx)
	if err != nil {
		return fmt.Errorf("failed to discover repositories: %w", err)
	}

	// Output formatted lines for FZF
	for _, repo := range repoList {
		fmt.Println(fzf.FormatRepoLine(repo))
	}

	return nil
}
```

**Step 2: Export FormatRepoLine in fzf package**

In `internal/fzf/fzf.go`, rename `formatRepoLine` to `FormatRepoLine` (capitalize). Update its usage in `SelectRepository` accordingly (line 23).

**Step 3: Register the command in main.go**

In `cmd/claude-matrix/main.go`, add `listReposCmd()` to the `rootCmd.AddCommand(...)` call:

```go
	rootCmd.AddCommand(
		createCmd(),
		listCmd(),
		listReposCmd(),
		renameCmd(),
		diagnoseCmd(),
		refreshCmd(),
		hookHandlerCmd(),
		setupHooksCmd(),
		removeHooksCmd(),
	)
```

**Step 4: Run tests**

Run: `go test -v -race ./...`
Expected: All PASS

**Step 5: Run linter**

Run: `make lint`
Expected: No errors

**Step 6: Commit**

```bash
git add cmd/claude-matrix/list_repos.go cmd/claude-matrix/main.go internal/fzf/fzf.go
git commit -m "feat: add list-repos subcommand for FZF reload support"
```

---

### Task 4: Add Ctrl+R reload binding to FZF repo selector

**Files:**
- Modify: `internal/fzf/fzf.go` (SelectRepository function)
- Modify: `internal/fzf/fzf_test.go`
- Modify: `cmd/claude-matrix/create.go` (pass binary path)

**Step 1: Write test for FZF args with reload binding**

Add to `internal/fzf/fzf_test.go`:

```go
func TestBuildRepoFZFArgs(t *testing.T) {
	args := buildRepoFZFArgs("/usr/local/bin/claude-matrix")

	// Should contain the reload binding
	hasReload := false
	for _, arg := range args {
		if strings.Contains(arg, "ctrl-r:reload") {
			hasReload = true
			if !strings.Contains(arg, "/usr/local/bin/claude-matrix") {
				t.Errorf("reload binding should contain binary path, got %q", arg)
			}
			if !strings.Contains(arg, "--force-refresh") {
				t.Errorf("reload binding should contain --force-refresh, got %q", arg)
			}
		}
	}
	if !hasReload {
		t.Error("FZF args should contain ctrl-r reload binding")
	}

	// Should contain ctrl-r in header
	hasHeader := false
	for _, arg := range args {
		if strings.Contains(arg, "ctrl-r") && strings.Contains(arg, "refresh") && strings.HasPrefix(arg, "--header=") {
			hasHeader = true
		}
	}
	if !hasHeader {
		t.Error("FZF header should mention ctrl-r refresh")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v -run TestBuildRepoFZFArgs ./internal/fzf/`
Expected: FAIL with "undefined: buildRepoFZFArgs"

**Step 3: Implement the change**

In `internal/fzf/fzf.go`, add a helper function and modify `SelectRepository`:

```go
// buildRepoFZFArgs returns the FZF arguments for repository selection.
// The binaryPath is used to construct the Ctrl+R reload command.
func buildRepoFZFArgs(binaryPath string) []string {
	reloadCmd := fmt.Sprintf("%s list-repos --force-refresh", binaryPath)
	return []string{
		"--prompt=📁 Select repository > ",
		"--reverse",
		"--border=rounded",
		"--header=↑↓ navigate | enter: select | ctrl-r: refresh | ctrl-c: cancel",
		"--height=80%",
		fmt.Sprintf("--bind=ctrl-r:reload(%s)+change-header(Refreshing repositories...)", reloadCmd),
	}
}

// SelectRepository shows FZF interface for repo selection.
// binaryPath is the path to the claude-matrix binary, used for Ctrl+R reload.
func SelectRepository(repos []*types.Repository, binaryPath string) (*types.Repository, error) {
	if len(repos) == 0 {
		return nil, fmt.Errorf("no repositories found")
	}

	var lines []string
	for _, repo := range repos {
		line := FormatRepoLine(repo)
		lines = append(lines, line)
	}

	args := buildRepoFZFArgs(binaryPath)
	selected, err := runFZF(strings.Join(lines, "\n"), args...)
	if err != nil {
		return nil, err
	}

	identifier := extractURL(selected)

	if name, ok := strings.CutPrefix(identifier, "workspace:"); ok {
		for _, repo := range repos {
			if repo.IsWorkspace && repo.Name == name {
				return repo, nil
			}
		}
		return nil, fmt.Errorf("selected workspace not found: %s", name)
	}

	for _, repo := range repos {
		if repo.URL == identifier {
			return repo, nil
		}
	}

	return nil, fmt.Errorf("selected repo not found")
}
```

**Step 4: Update create.go to pass binary path**

In `cmd/claude-matrix/create.go`, update the call to `fzf.SelectRepository`:

```go
	// Get binary path for FZF reload
	binaryPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get binary path: %w", err)
	}

	selected, err := fzf.SelectRepository(repoList, binaryPath)
```

**Step 5: Run tests**

Run: `go test -v -race ./...`
Expected: All PASS

**Step 6: Run linter**

Run: `make lint`
Expected: No errors

**Step 7: Commit**

```bash
git add internal/fzf/fzf.go internal/fzf/fzf_test.go cmd/claude-matrix/create.go
git commit -m "feat: add ctrl-r reload binding to FZF repo picker"
```

---

### Task 5: Integration verification

**Step 1: Build the binary**

Run: `make build`
Expected: Build succeeds

**Step 2: Run full test suite**

Run: `make ci`
Expected: fmt + lint + test + build all pass

**Step 3: Manual verification (if tmux available)**

1. Delete cache: `rm -f ~/.tmux-claude-matrix/.cache/github-repos.json`
2. Run: `./bin/claude-matrix create`
3. Verify repos are fetched from API (first load)
4. Press Ctrl+R in the FZF picker
5. Verify "Refreshing repositories..." appears briefly, then list reloads
6. Press Ctrl+C to cancel
7. Run again - should use cached repos (no API fetch)

**Step 4: Commit (if any fixes needed)**

```bash
git add -A
git commit -m "fix: address integration issues from manual testing"
```

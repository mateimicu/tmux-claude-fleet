package repos

import (
	"context"
	"os"
	"sort"
	"testing"
)

func TestWorkspaceSource(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "workspaces-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	content := `workspaces:
  my-project:
    repos:
      - https://github.com/org/frontend
      - https://github.com/org/backend
      - git@github.com:org/shared-libs.git
    description: "Frontend + backend + shared libraries"
  infra:
    repos:
      - https://github.com/org/terraform
      - https://github.com/org/k8s-configs
`
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	source := NewWorkspaceSource(tmpFile.Name())

	if source.Name() != "workspace" {
		t.Errorf("Expected source name 'workspace', got %q", source.Name())
	}

	repos, err := source.List(context.Background())
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(repos) != 2 {
		t.Fatalf("Expected 2 workspaces, got %d", len(repos))
	}

	// Sort for deterministic testing since map iteration order is random
	sort.Slice(repos, func(i, j int) bool {
		return repos[i].Name < repos[j].Name
	})

	// Check infra workspace
	infra := repos[0]
	if infra.Name != "infra" {
		t.Errorf("Expected name 'infra', got %q", infra.Name)
	}
	if !infra.IsWorkspace {
		t.Error("Expected IsWorkspace to be true")
	}
	if len(infra.WorkspaceRepos) != 2 {
		t.Errorf("Expected 2 repos in infra workspace, got %d", len(infra.WorkspaceRepos))
	}
	if infra.Description != "2 repos" {
		t.Errorf("Expected auto-generated description '2 repos', got %q", infra.Description)
	}

	// Check my-project workspace
	project := repos[1]
	if project.Name != "my-project" {
		t.Errorf("Expected name 'my-project', got %q", project.Name)
	}
	if project.Description != "Frontend + backend + shared libraries" {
		t.Errorf("Expected description 'Frontend + backend + shared libraries', got %q", project.Description)
	}
	if len(project.WorkspaceRepos) != 3 {
		t.Errorf("Expected 3 repos in my-project workspace, got %d", len(project.WorkspaceRepos))
	}
	if project.Source != "workspace" {
		t.Errorf("Expected source 'workspace', got %q", project.Source)
	}
}

func TestWorkspaceSourceEmptyRepos(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "workspaces-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	content := `workspaces:
  empty:
    repos: []
    description: "This has no repos"
  valid:
    repos:
      - https://github.com/org/repo
`
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	source := NewWorkspaceSource(tmpFile.Name())
	repos, err := source.List(context.Background())
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	// Empty workspace should be skipped
	if len(repos) != 1 {
		t.Fatalf("Expected 1 workspace (empty skipped), got %d", len(repos))
	}

	if repos[0].Name != "valid" {
		t.Errorf("Expected name 'valid', got %q", repos[0].Name)
	}
}

func TestWorkspaceSourceMissingFile(t *testing.T) {
	source := NewWorkspaceSource("/nonexistent/workspaces.yaml")
	_, err := source.List(context.Background())
	if err == nil {
		t.Error("Expected error for missing file")
	}
}

func TestWorkspaceSourceInvalidYAML(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "workspaces-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString("invalid: [yaml: {broken"); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	source := NewWorkspaceSource(tmpFile.Name())
	_, err = source.List(context.Background())
	if err == nil {
		t.Error("Expected error for invalid YAML")
	}
}

func TestParseWorkspacesFile(t *testing.T) {
	data := []byte(`workspaces:
  project-a:
    repos:
      - https://github.com/org/repo1
      - https://github.com/org/repo2
  project-b:
    repos:
      - https://github.com/org/repo3
`)

	result, err := ParseWorkspacesFile(data)
	if err != nil {
		t.Fatalf("ParseWorkspacesFile failed: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("Expected 2 workspaces, got %d", len(result))
	}

	if len(result["project-a"]) != 2 {
		t.Errorf("Expected 2 repos in project-a, got %d", len(result["project-a"]))
	}

	if len(result["project-b"]) != 1 {
		t.Errorf("Expected 1 repo in project-b, got %d", len(result["project-b"]))
	}
}

func TestFormatWorkspaceRepoList(t *testing.T) {
	urls := []string{
		"https://github.com/org/frontend",
		"https://github.com/org/backend",
		"git@github.com:org/shared-libs.git",
	}

	result := FormatWorkspaceRepoList(urls)
	expected := "frontend, backend, shared-libs"
	if result != expected {
		t.Errorf("FormatWorkspaceRepoList() = %q, want %q", result, expected)
	}
}

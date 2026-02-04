package repos

import (
	"context"
	"os"
	"testing"
)

func TestLocalSource(t *testing.T) {
	// Create temporary repos file
	tmpFile, err := os.CreateTemp("", "repos-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	// Write test data
	content := `# Test repositories
https://github.com/test/repo1:Test repo 1
https://github.com/test/repo2
git@github.com:test/repo3.git:Private repo

# Comment line
https://github.com/test/repo4
`
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	// Test loading
	source := NewLocalSource(tmpFile.Name())
	repos, err := source.List(context.Background())
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(repos) != 4 {
		t.Errorf("Expected 4 repos, got %d", len(repos))
	}

	// Check first repo
	if repos[0].Name != "test/repo1" {
		t.Errorf("Expected name 'test/repo1', got %q", repos[0].Name)
	}
	if repos[0].Description != "Test repo 1" {
		t.Errorf("Expected description 'Test repo 1', got %q", repos[0].Description)
	}

	// Check source type
	if repos[0].Source != "local" {
		t.Errorf("Expected source 'local', got %q", repos[0].Source)
	}
}

func TestParseLine(t *testing.T) {
	tests := []struct {
		name         string
		line         string
		expectedURL  string
		expectedDesc string
	}{
		{
			name:         "HTTPS URL only",
			line:         "https://github.com/test/repo",
			expectedURL:  "https://github.com/test/repo",
			expectedDesc: "",
		},
		{
			name:         "HTTPS URL with description",
			line:         "https://github.com/test/repo:My awesome repo",
			expectedURL:  "https://github.com/test/repo",
			expectedDesc: "My awesome repo",
		},
		{
			name:         "SSH URL only",
			line:         "git@github.com:test/repo.git",
			expectedURL:  "git@github.com:test/repo.git",
			expectedDesc: "",
		},
		{
			name:         "SSH URL with description",
			line:         "git@github.com:test/repo:Private repo",
			expectedURL:  "git@github.com:test/repo",
			expectedDesc: "Private repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, desc := parseLine(tt.line)
			if url != tt.expectedURL {
				t.Errorf("URL: got %q, want %q", url, tt.expectedURL)
			}
			if desc != tt.expectedDesc {
				t.Errorf("Description: got %q, want %q", desc, tt.expectedDesc)
			}
		})
	}
}

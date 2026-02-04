package repos

import (
	"bufio"
	"context"
	"os"
	"strings"

	"github.com/mateimicu/tmux-claude-matrix/internal/git"
	"github.com/mateimicu/tmux-claude-matrix/pkg/types"
)

// LocalSource discovers repositories from a local file
type LocalSource struct {
	filePath string
}

// NewLocalSource creates a new local repository source
func NewLocalSource(filePath string) *LocalSource {
	return &LocalSource{filePath: filePath}
}

// Name returns the source name
func (l *LocalSource) Name() string {
	return "local"
}

// List returns all repositories from the local file
func (l *LocalSource) List(ctx context.Context) ([]*types.Repository, error) {
	file, err := os.Open(l.filePath)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := file.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	var repos []*types.Repository
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		url, desc := parseLine(line)
		repos = append(repos, &types.Repository{
			Source:      "local",
			URL:         url,
			Name:        git.ExtractRepoName(url),
			Description: desc,
		})
	}

	return repos, scanner.Err()
}

// parseLine extracts URL and optional description from a line
func parseLine(line string) (url, description string) {
	// Format: URL or URL:description

	// Check if it's an SSH URL (git@...)
	switch {
	case strings.Contains(line, "@") && !strings.HasPrefix(line, "http"):
		// SSH format: git@github.com:org/repo or git@github.com:org/repo:description
		// Find the second colon (after the SSH part)
		parts := strings.SplitN(line, ":", 3)
		if len(parts) >= 3 {
			// Has description
			url = parts[0] + ":" + parts[1]
			description = parts[2]
		} else {
			// No description
			url = line
			description = ""
		}
	case strings.HasPrefix(line, "http://") || strings.HasPrefix(line, "https://"):
		// HTTPS format: https://github.com/org/repo or https://github.com/org/repo:description
		// Split only after the protocol part
		protocolEnd := strings.Index(line, "://")
		if protocolEnd > 0 {
			afterProtocol := line[protocolEnd+3:]
			colonIndex := strings.Index(afterProtocol, ":")
			if colonIndex > 0 {
				// Has description
				url = line[:protocolEnd+3+colonIndex]
				description = afterProtocol[colonIndex+1:]
			} else {
				// No description
				url = line
				description = ""
			}
		} else {
			url = line
			description = ""
		}
	default:
		// Plain path or other format
		url = line
		description = ""
	}

	return strings.TrimSpace(url), strings.TrimSpace(description)
}

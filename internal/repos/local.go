package repos

import (
	"bufio"
	"context"
	"os"
	"strings"

	"github.com/mateimicu/tmux-claude-fleet/internal/git"
	"github.com/mateimicu/tmux-claude-fleet/pkg/types"
)

type LocalSource struct {
	filePath string
}

func NewLocalSource(filePath string) *LocalSource {
	return &LocalSource{filePath: filePath}
}

func (l *LocalSource) Name() string {
	return "local"
}

func (l *LocalSource) List(ctx context.Context) ([]*types.Repository, error) {
	file, err := os.Open(l.filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

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
	if strings.Contains(line, "@") && !strings.HasPrefix(line, "http") {
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
	} else if strings.HasPrefix(line, "http://") || strings.HasPrefix(line, "https://") {
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
	} else {
		// Plain path or other format
		url = line
		description = ""
	}

	return strings.TrimSpace(url), strings.TrimSpace(description)
}

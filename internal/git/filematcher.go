package git

import (
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
)

// ListIgnoredFiles returns files ignored by .gitignore
func (s *Service) ListIgnoredFiles(repoPath string) ([]string, error) {
	cmd := exec.Command("git", "ls-files", "--others", "--ignored", "--exclude-standard")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return parseFileList(string(output)), nil
}

// ListUntrackedFiles returns untracked files (not ignored)
func (s *Service) ListUntrackedFiles(repoPath string) ([]string, error) {
	cmd := exec.Command("git", "ls-files", "--others", "--exclude-standard")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return parseFileList(string(output)), nil
}

// MatchesPattern checks if a path matches any of the given patterns (gitignore-style)
func MatchesPattern(path string, patterns []string) bool {
	if len(patterns) == 0 {
		return false
	}

	var parsedPatterns []gitignore.Pattern
	for _, p := range patterns {
		parsedPatterns = append(parsedPatterns, gitignore.ParsePattern(p, nil))
	}

	matcher := gitignore.NewMatcher(parsedPatterns)
	pathComponents := strings.Split(filepath.ToSlash(path), "/")
	return matcher.Match(pathComponents, false)
}

// parseFileList parses the output from git ls-files
func parseFileList(out string) []string {
	var files []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Skip .git directory
		if strings.HasPrefix(line, ".git/") {
			continue
		}
		files = append(files, line)
	}
	return files
}

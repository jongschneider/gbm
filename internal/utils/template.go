// Package utils provides shared utility functions for the gbm CLI.
package utils

import (
	"os"
	"path/filepath"
	"strings"
)

// ExpandTemplate replaces template variables in a string.
// Supported variables:
//   - {gitroot}: repository root directory name
//   - {branch}: current/target branch name (must be provided in vars)
//   - {issue}: JIRA issue key (must be provided in vars)
//
// Example:
//
//	ExpandTemplate("../{gitroot}-worktrees", map[string]string{"gitroot": "gbm"})
//	returns "../gbm-worktrees"
func ExpandTemplate(path string, vars map[string]string) string {
	result := path

	// Replace each variable in the template
	for varName, varValue := range vars {
		placeholder := "{" + varName + "}"
		result = strings.ReplaceAll(result, placeholder, varValue)
	}

	return result
}

// GetTemplateVars returns available template variables based on the repository.
// Currently supports {gitroot}. Other variables like {branch} and {issue}
// are context-specific and should be provided separately.
//
// Example:
//
//	vars := GetTemplateVars("/path/to/gbm")
//	// Returns: {"gitroot": "gbm"}
func GetTemplateVars(repoRoot string) map[string]string {
	return map[string]string{
		"gitroot": filepath.Base(repoRoot),
	}
}

// ExpandPath expands ~ to home directory and resolves relative paths.
// Relative paths are resolved from the main repository root, not the current worktree.
//
// Examples:
//
//	ExpandPath("~/dev/branches", "/path/to/repo") → "/home/user/dev/branches"
//	ExpandPath("../worktrees", "/path/to/repo") → "/path/to/worktrees"
//	ExpandPath("/absolute/path", "/path/to/repo") → "/absolute/path"
func ExpandPath(path, repoRoot string) string {
	// Expand ~
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path // Return unchanged if can't expand
		}
		path = filepath.Join(home, path[2:])
	} else if path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return path // Return unchanged if can't expand
		}
		return home
	}

	// If already absolute, return as is
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}

	// Relative path - resolve from repo root
	return filepath.Clean(filepath.Join(repoRoot, path))
}

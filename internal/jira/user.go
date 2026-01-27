package jira

import (
	"fmt"
	"os/exec"
	"strings"
)

// IsJiraCliAvailable checks if the JIRA CLI is installed and available
// Commands can use this to gracefully handle missing JIRA CLI.
func (s *Service) IsJiraCliAvailable() bool {
	_, err := exec.LookPath("jira")
	return err == nil
}

// GetJiraUser gets the current JIRA user, using cached value if available
// If cachedUser is not empty, returns it immediately
// Otherwise, fetches from jira CLI and returns (user, true, nil) to indicate cache miss
// Returns (user, false, nil) if using cached value
// Returns ErrJiraCliNotFound if jira CLI is not available
// The caller is responsible for updating the config cache.
func (s *Service) GetJiraUser(cachedUser string, dryRun bool) (string, bool, error) {
	// If we have a cached value, use it
	if cachedUser != "" {
		return cachedUser, false, nil
	}

	// Check if JIRA CLI is available
	if !s.IsJiraCliAvailable() {
		return "", false, ErrJiraCliNotFound
	}

	// Otherwise, fetch it from jira CLI
	cmd := exec.Command("jira", "me")

	if dryRun {
		printDryRun(cmd)
		return "testuser", true, nil // true indicates cache miss, caller should save
	}

	userOutput, err := cmd.Output()
	if err != nil {
		return "", false, fmt.Errorf("failed to get current JIRA user: %w", err)
	}

	user := strings.TrimSpace(string(userOutput))
	return user, true, nil // true indicates cache miss, caller should save
}

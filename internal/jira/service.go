package jira

import (
	"errors"
	"fmt"
	"log"
	"os/exec"

	"gbm/internal/utils"
)

// ErrJiraCliNotFound is returned when the JIRA CLI is not available
var ErrJiraCliNotFound = errors.New("jira CLI not found")

// CacheStore defines an interface for loading and saving the issues cache
// This allows different storage backends (file, memory, database, etc.)
type CacheStore interface {
	// Load returns the cached issues and cached user
	Load() (*IssuesCache, string, error)
	// Save persists the cache and user
	Save(cache *IssuesCache, user string) error
}

// Service provides JIRA CLI integration
type Service struct {
	debug bool
	store CacheStore
}

// NewService creates a new JIRA service instance
// Unlike git service, JIRA is optional - logs warning if not found but doesn't fail
func NewService(debug bool, store CacheStore) *Service {
	// Check for jira CLI availability like git service does
	// But unlike git, jira is optional - just log warning if not found
	if _, err := exec.LookPath("jira"); err != nil {
		// Log warning if debug logging is enabled
		if debug {
			log.Printf("Warning: jira command not found in PATH - JIRA features will be unavailable")
		}
		// Don't fail - JIRA is optional
	}

	return &Service{
		debug: debug,
		store: store,
	}
}

// runCommand executes a jira CLI command with dry-run support
// Follows git.Service.runCommand pattern
func (s *Service) runCommand(cmd *exec.Cmd, dryRun bool) ([]byte, error) {
	cmdStr := utils.FormatCommand(cmd)
	if dryRun {
		fmt.Printf("[DRY RUN] %s\n", cmdStr)
		// Still execute reads - they're safe
	}
	return cmd.CombinedOutput()
}

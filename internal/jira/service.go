package jira

import (
	"errors"
	"fmt"
	"gbm/internal/utils"
	"log"
	"os"
	"os/exec"
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

// printDryRun prints a dry-run message to stderr for visibility
func printDryRun(cmd *exec.Cmd) {
	cmdStr := utils.FormatCommand(cmd)
	fmt.Fprintf(os.Stderr, "[DRY RUN] %s\n", cmdStr)
}

package jira

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

// ErrJiraCliNotFound is returned when the JIRA CLI is not available.
var ErrJiraCliNotFound = errors.New("jira CLI not found")

// CacheStore defines an interface for loading and saving the issues cache
// This allows different storage backends (file, memory, database, etc.)
type CacheStore interface {
	// Load returns the cached issues and cached user
	Load() (*IssuesCache, string, error)
	// Save persists the cache and user
	Save(cache *IssuesCache, user string) error
}

// Service provides JIRA CLI integration.
type Service struct {
	store CacheStore
	debug bool
}

// NewService creates a new JIRA service instance
// Unlike git service, JIRA is optional - logs warning if not found but doesn't fail.
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

// printDryRun prints a dry-run message to stderr for visibility.
func printDryRun(cmd *exec.Cmd) {
	fmt.Fprintf(os.Stderr, "[DRY RUN] %s\n", formatCommand(cmd))
}

// formatCommand renders an *exec.Cmd as a shell-like string for logging.
func formatCommand(cmd *exec.Cmd) string {
	parts := []string{cmd.Path}
	parts = append(parts, cmd.Args[1:]...)

	if cmd.Dir != "" {
		return fmt.Sprintf("(cd %s && %s)", cmd.Dir, strings.Join(parts, " "))
	}

	for _, env := range cmd.Env {
		if after, ok := strings.CutPrefix(env, "GIT_DIR="); ok {
			return fmt.Sprintf("GIT_DIR=%s %s", after, strings.Join(parts, " "))
		}
	}

	return strings.Join(parts, " ")
}

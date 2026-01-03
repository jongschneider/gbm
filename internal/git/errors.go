package git

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// Sentinel errors for git operations
var (
	// Parameter validation errors
	ErrWorktreesDirectoryEmpty = errors.New("worktrees directory cannot be empty")
	ErrWorktreeNameEmpty       = errors.New("worktree name cannot be empty")
	ErrBranchNameEmpty         = errors.New("branch name cannot be empty")
	ErrWorktreePathEmpty       = errors.New("worktree path cannot be empty")
	ErrOldWorktreeNameEmpty    = errors.New("old worktree name cannot be empty")
	ErrNewWorktreeNameEmpty    = errors.New("new worktree name cannot be empty")

	// State errors
	ErrNotInWorktree                  = errors.New("not in a worktree")
	ErrCouldNotDetermineDefaultBranch = errors.New("could not determine default branch from remote")

	// Git operation errors (typed)
	ErrBranchNotFound         = errors.New("branch not found")
	ErrWorktreeNotFound       = errors.New("worktree not found")
	ErrBranchExists           = errors.New("branch already exists")
	ErrWorktreeExists         = errors.New("worktree already exists")
	ErrNotMerged              = errors.New("branch not merged")
	ErrDirtyWorktree          = errors.New("worktree has uncommitted changes")
	ErrRemoteNotFound         = errors.New("remote not found")
	ErrNoUpstream             = errors.New("branch has no upstream tracking branch")
	ErrNotAGitRepository      = errors.New("not a git repository")
	ErrNoRemoteTrackingBranch = errors.New("no remote tracking branch")
)

// GitError represents a git operation failure with context and exit code
type GitError struct {
	Op       string // Operation that failed (e.g., "worktree add", "branch delete")
	ExitCode int    // Exit code from git command
	Stderr   string // Standard error output
	Err      error  // Underlying error
}

// Error implements the error interface for GitError
func (e *GitError) Error() string {
	msg := fmt.Sprintf("git %s: %v", e.Op, e.Err)
	if e.Stderr != "" {
		msg += fmt.Sprintf(" (%s)", e.Stderr)
	}
	return msg
}

// Unwrap returns the underlying error for error wrapping chains
func (e *GitError) Unwrap() error {
	return e.Err
}

// IsExitCode checks if this error has a specific exit code
func (e *GitError) IsExitCode(code int) bool {
	return e.ExitCode == code
}

// NewGitError creates a new GitError with operation context
func NewGitError(op string, err error, exitCode int, stderr string) error {
	if err == nil {
		return nil
	}

	// Trim stderr for cleaner output
	stderr = strings.TrimSpace(stderr)

	return &GitError{
		Op:       op,
		ExitCode: exitCode,
		Stderr:   stderr,
		Err:      err,
	}
}

// ClassifyError analyzes a git error and returns a typed error if recognized.
// This enables callers to handle specific git failures consistently.
//
// Common patterns:
//   - "not found" in stderr → ErrBranchNotFound, ErrWorktreeNotFound, ErrRemoteNotFound
//   - "already exists" in stderr → ErrBranchExists, ErrWorktreeExists
//   - "no changes added" or "nothing to commit" → ErrNotMerged
//   - "dirty" in stderr → ErrDirtyWorktree
//   - "no remote" or "no tracking" → ErrNoUpstream
//   - "fatal: not a git repository" → ErrNotAGitRepository
//
// Example:
//
//	cmd := exec.Command("git", "branch", "-d", "branch-name")
//	output, err := cmd.CombinedOutput()
//	if err != nil {
//	    typedErr := ClassifyError("branch delete", err, string(output))
//	    if errors.Is(typedErr, ErrBranchNotFound) {
//	        // Handle branch not found
//	    }
//	}
func ClassifyError(op string, err error, output string) error {
	if err == nil {
		return nil
	}

	// Extract exit code if available
	exitCode := -1
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		exitCode = exitErr.ExitCode()
	}

	stderr := output
	if exitErr != nil {
		stderr = string(exitErr.Stderr)
	}
	stderr = strings.ToLower(strings.TrimSpace(stderr))

	// Try to classify based on stderr content and operation
	typedErr := classifyByContent(op, stderr)
	if typedErr != nil {
		return NewGitError(op, typedErr, exitCode, stderr)
	}

	// Fallback: wrap with context
	return NewGitError(op, err, exitCode, stderr)
}

// classifyByContent analyzes output content and returns typed errors
func classifyByContent(op string, stderr string) error {
	// Branch not found patterns
	if strings.Contains(stderr, "not found") && (strings.Contains(op, "branch") || strings.Contains(op, "delete")) {
		return ErrBranchNotFound
	}
	if strings.Contains(stderr, "no such branch") {
		return ErrBranchNotFound
	}

	// Worktree not found patterns
	if strings.Contains(stderr, "no such worktree") {
		return ErrWorktreeNotFound
	}
	if strings.Contains(stderr, "is not a working tree") {
		return ErrWorktreeNotFound
	}
	if strings.Contains(stderr, "not found") && strings.Contains(op, "worktree") {
		return ErrWorktreeNotFound
	}

	// Already exists patterns
	if strings.Contains(stderr, "already exists") && strings.Contains(op, "branch") {
		return ErrBranchExists
	}
	if strings.Contains(stderr, "already exists") && strings.Contains(op, "worktree") {
		return ErrWorktreeExists
	}

	// Not merged patterns
	if strings.Contains(stderr, "not fully merged") || strings.Contains(stderr, "not merged") {
		return ErrNotMerged
	}

	// Dirty worktree patterns
	if strings.Contains(stderr, "dirty") || strings.Contains(stderr, "uncommitted") {
		return ErrDirtyWorktree
	}

	// No upstream patterns
	if strings.Contains(stderr, "no upstream") || strings.Contains(stderr, "no tracking") {
		return ErrNoUpstream
	}
	if strings.Contains(stderr, "no remote tracking branch") {
		return ErrNoRemoteTrackingBranch
	}

	// Remote not found
	if strings.Contains(stderr, "no such remote") || strings.Contains(stderr, "unknown remote") {
		return ErrRemoteNotFound
	}

	// Not a repository
	if strings.Contains(stderr, "not a git repository") || strings.Contains(stderr, "fatal:") && strings.Contains(stderr, "not git repository") {
		return ErrNotAGitRepository
	}

	return nil
}

// Wrap wraps an error with git operation context, using ClassifyError for typing
// Use this when you have an existing error and want to add operation context
// Example:
//
//	worktrees, err := s.ListWorktrees(false)
//	if err != nil {
//	    return Wrap("list worktrees", err)
//	}
func Wrap(op string, err error) error {
	if err == nil {
		return nil
	}
	return NewGitError(op, err, -1, "")
}

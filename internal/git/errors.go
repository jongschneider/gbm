package git

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// Sentinel errors for git operations.
// These errors help distinguish different failure modes and allow callers
// to handle specific cases (e.g., "branch not found" vs "permission denied").
//
// Use errors.Is() to check for specific error types:
//
//	if errors.Is(err, git.ErrBranchNotFound) {
//	    // Handle branch not found
//	}
var (
	// Parameter validation errors - used when required parameters are empty.
	ErrWorktreesDirectoryEmpty = errors.New("worktrees directory cannot be empty")
	ErrWorktreeNameEmpty       = errors.New("worktree name cannot be empty")
	ErrBranchNameEmpty         = errors.New("branch name cannot be empty")
	ErrWorktreePathEmpty       = errors.New("worktree path cannot be empty")
	ErrOldWorktreeNameEmpty    = errors.New("old worktree name cannot be empty")
	ErrNewWorktreeNameEmpty    = errors.New("new worktree name cannot be empty")

	// State errors - used when the repository or worktree is in an unexpected state.
	ErrNotInWorktree                  = errors.New("not in a worktree")
	ErrCouldNotDetermineDefaultBranch = errors.New("could not determine default branch from remote")

	// Git operation errors - classified from git command output.
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

// GitError represents a git operation failure with context and exit code.
//
// This provides richer error information for git failures, including the
// operation that failed, the exit code, and the stderr output. This allows
// callers to diagnose and handle different types of failures.
//
// Fields:
//   - Op: The git operation that failed (e.g., "worktree add", "branch delete")
//   - ExitCode: Exit code returned by the git command
//   - Stderr: Standard error output from git (trimmed)
//   - Err: Underlying error from the exec package
type GitError struct {
	Err      error
	Op       string
	Stderr   string
	ExitCode int
}

// Error implements the error interface for GitError.
// Returns a formatted error message including the operation, underlying error, and stderr output.
func (e *GitError) Error() string {
	msg := fmt.Sprintf("git %s: %v", e.Op, e.Err)
	if e.Stderr != "" {
		msg += fmt.Sprintf(" (%s)", e.Stderr)
	}
	return msg
}

// Unwrap returns the underlying error for error wrapping chains.
// This enables Go 1.13+ error wrapping to work correctly.
func (e *GitError) Unwrap() error {
	return e.Err
}

// IsExitCode checks if this error has a specific exit code.
//
// Example:
//
//	if gitErr, ok := err.(*git.GitError); ok && gitErr.IsExitCode(1) {
//	    // Handle exit code 1
//	}
func (e *GitError) IsExitCode(code int) bool {
	return e.ExitCode == code
}

// NewGitError creates a new GitError with operation context.
// Returns nil if the error is nil (for convenient chaining).
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

	// Prefer passed output (from CombinedOutput), fall back to exitErr.Stderr
	stderr := strings.TrimSpace(output)
	if stderr == "" && exitErr != nil {
		stderr = strings.TrimSpace(string(exitErr.Stderr))
	}
	stderrLower := strings.ToLower(stderr)

	// Try to classify based on stderr content and operation
	typedErr := classifyByContent(op, stderrLower)
	if typedErr != nil {
		return NewGitError(op, typedErr, exitCode, stderr)
	}

	// Fallback: wrap with context
	return NewGitError(op, err, exitCode, stderr)
}

// errorPattern defines a pattern to match in stderr and the corresponding error.
type errorPattern struct {
	err      error
	opMatch  string
	patterns []string
}

// errorPatterns defines the patterns used to classify git errors.
// Patterns are checked in order; the first match wins.
var errorPatterns = []errorPattern{
	// Branch not found patterns
	{patterns: []string{"no such branch"}, err: ErrBranchNotFound},
	// Worktree not found patterns
	{patterns: []string{"no such worktree"}, err: ErrWorktreeNotFound},
	{patterns: []string{"is not a working tree"}, err: ErrWorktreeNotFound},
	{patterns: []string{"not found"}, opMatch: "worktree", err: ErrWorktreeNotFound},
	// Branch not found (must check after worktree patterns)
	{patterns: []string{"not found"}, opMatch: "branch", err: ErrBranchNotFound},
	{patterns: []string{"not found"}, opMatch: "delete", err: ErrBranchNotFound},
	// Already exists patterns
	{patterns: []string{"already exists"}, opMatch: "branch", err: ErrBranchExists},
	{patterns: []string{"already exists"}, opMatch: "worktree", err: ErrWorktreeExists},
	// Not merged patterns
	{patterns: []string{"not fully merged"}, err: ErrNotMerged},
	{patterns: []string{"not merged"}, err: ErrNotMerged},
	// Dirty worktree patterns
	{patterns: []string{"dirty"}, err: ErrDirtyWorktree},
	{patterns: []string{"uncommitted"}, err: ErrDirtyWorktree},
	// No upstream patterns
	{patterns: []string{"no remote tracking branch"}, err: ErrNoRemoteTrackingBranch},
	{patterns: []string{"no upstream"}, err: ErrNoUpstream},
	{patterns: []string{"no tracking"}, err: ErrNoUpstream},
	// Remote not found
	{patterns: []string{"no such remote"}, err: ErrRemoteNotFound},
	{patterns: []string{"unknown remote"}, err: ErrRemoteNotFound},
	// Not a repository
	{patterns: []string{"not a git repository"}, err: ErrNotAGitRepository},
	{patterns: []string{"fatal:", "not git repository"}, err: ErrNotAGitRepository},
}

// classifyByContent analyzes output content and returns typed errors.
func classifyByContent(op, stderr string) error {
	for _, pattern := range errorPatterns {
		if matchesPattern(pattern, op, stderr) {
			return pattern.err
		}
	}
	return nil
}

// matchesPattern checks if the given stderr and operation match the pattern.
func matchesPattern(p errorPattern, op, stderr string) bool {
	// Check operation match if specified
	if p.opMatch != "" && !strings.Contains(op, p.opMatch) {
		return false
	}
	// All patterns must match
	for _, pat := range p.patterns {
		if !strings.Contains(stderr, pat) {
			return false
		}
	}
	return true
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

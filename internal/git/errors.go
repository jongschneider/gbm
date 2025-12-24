package git

import "errors"

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
)

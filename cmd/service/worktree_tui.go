package service

import (
	"errors"
)

// Sentinel errors for service operations.
var (
	// State errors.
	ErrNotInGitRepository = errors.New("not in a git repository")
	ErrNoPreviousWorktree = errors.New("no previous worktree to switch to")

	// Validation errors.
	ErrBranchCreationCancelled = errors.New("branch creation cancelled")
)

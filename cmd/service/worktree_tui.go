package service

import (
	"context"
	"errors"
	"time"

	"github.com/spf13/cobra"
)

// Sentinel errors for service operations
var (
	// User interaction errors
	ErrCancelled = errors.New("cancelled")
	ErrGoBack    = errors.New("go back")

	// State errors
	ErrNotInGitRepository = errors.New("not in a git repository")
	ErrNoPreviousWorktree = errors.New("no previous worktree to switch to")

	// Validation errors
	ErrWorktreeNameEmpty       = errors.New("worktree name cannot be empty")
	ErrBranchNameEmpty         = errors.New("branch name cannot be empty")
	ErrInvalidCharacters       = errors.New("invalid characters in worktree name")
	ErrInvalidBranchNameFormat = errors.New("invalid branch name format")
	ErrBranchCreationCancelled = errors.New("branch creation cancelled")
	ErrUnexpectedModelType     = errors.New("unexpected model type")
)

// runWorktreeAddTUI launches the interactive TUI workflow for creating worktrees
func runWorktreeAddTUI(cmd *cobra.Command, svc *Service) error {
	// Create and run the unified FSM
	// The FSM handles:
	// - Type selection
	// - All workflow paths (feature/bug/hotfix/mergeback)
	// - Navigation (back, cancel)
	// - Terminal states (success, cancelled, error)

	// Create context with 30-minute timeout for the entire workflow
	// This prevents hanging while allowing plenty of time for user interaction
	// Use cmd.Context() as base to inherit any cancellation from the command framework
	ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Minute)
	defer cancel()

	fsm := NewWorktreeAddFSM(svc)
	return fsm.Run(ctx)
}

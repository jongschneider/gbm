package service

import (
	"errors"

	"github.com/spf13/cobra"
)

// ErrCancelled is a special error that signals the user cancelled the operation
var ErrCancelled = errors.New("cancelled")

// ErrGoBack is a special error that signals the user wants to go back
var ErrGoBack = errors.New("go back")

// runWorktreeAddTUI launches the interactive TUI workflow for creating worktrees
func runWorktreeAddTUI(_ *cobra.Command, svc *Service) error {
	// Create and run the unified FSM
	// The FSM handles:
	// - Type selection
	// - All workflow paths (feature/bug/hotfix/mergeback)
	// - Navigation (back, cancel)
	// - Terminal states (success, cancelled, error)
	fsm := NewWorktreeAddFSM(svc)
	return fsm.Run()
}

package service

import (
	"errors"
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

// ErrGoBack is a special error that signals to go back to the previous screen
var ErrGoBack = errors.New("go back to previous screen")

// ErrCancelled is a special error that signals the user cancelled the operation
var ErrCancelled = errors.New("cancelled")

// runWorktreeAddTUI launches the interactive TUI workflow for creating worktrees
func runWorktreeAddTUI(_ *cobra.Command, svc *Service) error {
	// Loop to allow going back from worktree type-specific flows to type selection
	for {
		var worktreeType string

		// Step 1: Ask what type of worktree to create
		typeForm := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("What type of worktree do you want to create?").
					Options(
						huh.NewOption("Feature", "feature"),
						huh.NewOption("Bug Fix", "bug"),
						huh.NewOption("Hotfix", "hotfix"),
						huh.NewOption("Mergeback", "mergeback"),
					).
					Value(&worktreeType),
			),
		)

		// Run the type selection wizard
		wizard := NewWizard([]WizardStep{{form: typeForm}})
		completed, cancelled, err := wizard.Run()
		if err != nil {
			return err
		}
		if cancelled {
			// User pressed Ctrl+C - cancel entirely
			return ErrCancelled
		}
		if !completed {
			// User pressed ESC on first screen - cancel entirely
			return ErrCancelled
		}

		// Step 2: Run type-specific flow
		var flowErr error
		switch worktreeType {
		case "feature":
			flowErr = NewFeatureWorkflow(svc, "feature").Run()
		case "bug":
			flowErr = NewFeatureWorkflow(svc, "bug").Run() // Same flow, different prefix
		case "hotfix":
			flowErr = NewHotfixWorkflow(svc).Run()
		case "mergeback":
			flowErr = NewMergebackWorkflow(svc).Run()
		default:
			return fmt.Errorf("unknown worktree type: %s", worktreeType)
		}

		// Check if user wants to go back to type selection
		if flowErr == ErrGoBack {
			continue // Loop back to step 1
		}

		// Any other error or success - return it
		return flowErr
	}
}

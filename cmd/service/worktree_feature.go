package service

import (
	"fmt"

	"github.com/charmbracelet/huh"
)

// FeatureWorkflow handles the feature and bug fix worktree creation workflow
type FeatureWorkflow struct {
	state  WorkflowState
	prefix string
}

// NewFeatureWorkflow creates a new feature workflow
func NewFeatureWorkflow(svc *Service, prefix string) *FeatureWorkflow {
	return &FeatureWorkflow{
		state: WorkflowState{
			Service: svc,
		},
		prefix: prefix,
	}
}

// Run executes the feature workflow
func (w *FeatureWorkflow) Run() error {
	// Step 1: Collect worktree name
	if err := w.collectWorktreeName(); err != nil {
		return err
	}

	// Step 2: Collect branch name
	if err := w.collectBranchName(); err != nil {
		return err
	}

	// Step 3: Try to create with existing branch
	if err := w.tryCreateWithExistingBranch(); err == nil {
		return nil // Success
	}

	// Step 4: Create with new branch
	return w.createWithNewBranch()
}

func (w *FeatureWorkflow) collectWorktreeName() error {
	items := fetchJiraItems(w.state.Service)

	wizard := NewWizard([]WizardStep{
		{customModel: NewFilterableSelect(
			"Worktree name",
			"Select JIRA ticket or enter custom name",
			items,
		), isCustom: true},
	})

	completed, cancelled, err := wizard.Run()
	if err != nil {
		return err
	}
	if cancelled {
		return ErrCancelled
	}
	if !completed {
		return ErrGoBack
	}

	// Extract and validate
	model := wizard.Steps[0].customModel.(FilterableSelectModel)
	w.state.WorktreeName = model.GetSelected()

	// Validate
	if err := createWorktreeNameValidator(w.state.Service)(w.state.WorktreeName); err != nil {
		// Show error and return to allow retry
		errorForm := huh.NewForm(
			huh.NewGroup(
				huh.NewNote().
					Title("Validation Error").
					Description(err.Error()),
			),
		)
		errorWizard := NewWizard([]WizardStep{{form: errorForm}})
		_, _, _ = errorWizard.Run()
		return ErrGoBack
	}

	return nil
}

func (w *FeatureWorkflow) collectBranchName() error {
	// Set default based on JIRA issue or custom input
	w.state.BranchName = w.generateDefaultBranchName()

	wizard := NewWizard([]WizardStep{
		{form: huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Branch name").
					Value(&w.state.BranchName).
					Validate(createBranchNameValidator(w.state.Service)).
					Description("Edit if needed"),
			),
		)},
	})

	completed, cancelled, err := wizard.Run()
	if err != nil {
		return err
	}
	if cancelled {
		return ErrCancelled
	}
	if !completed {
		return ErrGoBack
	}

	return nil
}

func (w *FeatureWorkflow) tryCreateWithExistingBranch() error {
	worktreesDir, err := w.state.Service.GetWorktreesPath()
	if err != nil {
		return fmt.Errorf("failed to get worktrees directory: %w", err)
	}

	wt, err := w.state.Service.Git.AddWorktree(
		worktreesDir,
		w.state.WorktreeName,
		w.state.BranchName,
		false, // don't create branch
		"",
		false,
	)
	if err != nil {
		return err // Branch doesn't exist
	}

	// Success - branch existed
	if copyErr := w.state.Service.CopyFilesToWorktree(w.state.WorktreeName); copyErr != nil {
		fmt.Printf("Warning: failed to copy files to worktree: %v\n", copyErr)
	}

	printWorktreeSuccess(wt)
	return nil
}

func (w *FeatureWorkflow) createWithNewBranch() error {
	// Collect base branch
	if err := w.collectBaseBranch(); err != nil {
		return err
	}

	// Confirm creation
	if err := w.confirmCreation(); err != nil {
		return err
	}

	// Execute creation
	return w.executeCreation()
}

func (w *FeatureWorkflow) collectBaseBranch() error {
	items, err := createSortedBranchItems(w.state.Service)
	if err != nil {
		return fmt.Errorf("failed to get branches: %w", err)
	}

	wizard := NewWizard([]WizardStep{
		{customModel: NewFilterableSelect(
			"Base branch",
			fmt.Sprintf("Branch '%s' doesn't exist. Select base:", w.state.BranchName),
			items,
		), isCustom: true},
	})

	completed, cancelled, err := wizard.Run()
	if err != nil {
		return err
	}
	if cancelled {
		return ErrCancelled
	}
	if !completed {
		return ErrGoBack
	}

	model := wizard.Steps[0].customModel.(FilterableSelectModel)
	w.state.BaseBranch = model.GetSelected()
	return nil
}

func (w *FeatureWorkflow) confirmCreation() error {
	var confirm bool
	wizard := NewWizard([]WizardStep{
		{form: huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title(fmt.Sprintf("Create branch '%s'?", w.state.BranchName)).
					Description(fmt.Sprintf("From base: %s", w.state.BaseBranch)).
					Value(&confirm),
			),
		)},
	})

	completed, cancelled, _ := wizard.Run()
	if cancelled || !completed || !confirm {
		return ErrCancelled
	}

	return nil
}

func (w *FeatureWorkflow) executeCreation() error {
	worktreesDir, err := w.state.Service.GetWorktreesPath()
	if err != nil {
		return err
	}

	wt, err := w.state.Service.Git.AddWorktree(
		worktreesDir,
		w.state.WorktreeName,
		w.state.BranchName,
		true, // create branch
		w.state.BaseBranch,
		w.state.DryRun,
	)
	if err != nil {
		return err
	}

	if copyErr := w.state.Service.CopyFilesToWorktree(w.state.WorktreeName); copyErr != nil {
		fmt.Printf("Warning: failed to copy files to worktree: %v\n", copyErr)
	}

	printWorktreeSuccess(wt)
	return nil
}

func (w *FeatureWorkflow) generateDefaultBranchName() string {
	// Check if worktree name is a JIRA key
	issues, err := w.state.Service.Jira.GetJiraIssues(false)
	if err == nil {
		for _, issue := range issues {
			if issue.Key == w.state.WorktreeName {
				sanitized := sanitizeSummaryForBranch(issue.Summary)
				return fmt.Sprintf("%s/%s-%s", w.prefix, issue.Key, sanitized)
			}
		}
	}
	return w.prefix + "/"
}

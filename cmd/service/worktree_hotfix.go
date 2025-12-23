package service

import (
	"fmt"

	"github.com/charmbracelet/huh"
)

// HotfixWorkflow handles the hotfix worktree creation workflow
type HotfixWorkflow struct {
	state WorkflowState
}

// NewHotfixWorkflow creates a new hotfix workflow
func NewHotfixWorkflow(svc *Service) *HotfixWorkflow {
	return &HotfixWorkflow{
		state: WorkflowState{
			Service: svc,
		},
	}
}

// Run executes the hotfix workflow
func (w *HotfixWorkflow) Run() error {
	// Step 1: Collect worktree name (with HOTFIX_ prefix)
	if err := w.collectWorktreeName(); err != nil {
		return err
	}

	// Step 2: Collect base branch
	if err := w.collectBaseBranch(); err != nil {
		return err
	}

	// Step 3: Collect branch name
	if err := w.collectBranchName(); err != nil {
		return err
	}

	// Step 4: Execute creation
	return w.executeCreation()
}

func (w *HotfixWorkflow) collectWorktreeName() error {
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

	model := wizard.Steps[0].customModel.(FilterableSelectModel)
	selectedName := model.GetSelected()

	// Add HOTFIX_ prefix
	w.state.WorktreeName = "HOTFIX_" + selectedName

	return createWorktreeNameValidator(w.state.Service)(w.state.WorktreeName)
}

func (w *HotfixWorkflow) collectBaseBranch() error {
	items, err := createSortedBranchItems(w.state.Service)
	if err != nil {
		return err
	}

	wizard := NewWizard([]WizardStep{
		{customModel: NewFilterableSelect(
			"Base branch for hotfix",
			"Select branch (typically production or release)",
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

func (w *HotfixWorkflow) collectBranchName() error {
	// Generate default from JIRA issue if applicable
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

func (w *HotfixWorkflow) executeCreation() error {
	worktreesDir, err := w.state.Service.GetWorktreesPath()
	if err != nil {
		return err
	}

	wt, err := w.state.Service.Git.AddWorktree(
		worktreesDir,
		w.state.WorktreeName,
		w.state.BranchName,
		true, // Always create new branch for hotfix
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

func (w *HotfixWorkflow) generateDefaultBranchName() string {
	// Extract the base name (remove HOTFIX_ prefix)
	baseName := w.state.WorktreeName
	if len(baseName) > 7 && baseName[:7] == "HOTFIX_" {
		baseName = baseName[7:]
	}

	// Check if it's a JIRA key
	issues, err := w.state.Service.Jira.GetJiraIssues(false)
	if err == nil {
		for _, issue := range issues {
			if issue.Key == baseName {
				sanitized := sanitizeSummaryForBranch(issue.Summary)
				return fmt.Sprintf("hotfix/%s-%s", issue.Key, sanitized)
			}
		}
	}
	return "hotfix/"
}

package service

import (
	"fmt"

	"github.com/charmbracelet/huh"
)

// MergebackWorkflow handles the mergeback worktree creation workflow
type MergebackWorkflow struct {
	state        WorkflowState
	sourceBranch string
	targetBranch string
}

// NewMergebackWorkflow creates a new mergeback workflow
func NewMergebackWorkflow(svc *Service) *MergebackWorkflow {
	return &MergebackWorkflow{
		state: WorkflowState{
			Service: svc,
		},
	}
}

// Run executes the mergeback workflow
func (w *MergebackWorkflow) Run() error {
	// Step 1: Select source branch
	if err := w.selectSourceBranch(); err != nil {
		return err
	}

	// Step 2: Select target branch
	if err := w.selectTargetBranch(); err != nil {
		return err
	}

	// Step 3: Generate worktree name
	if err := w.collectWorktreeName(); err != nil {
		return err
	}

	// Step 4: Generate branch name
	if err := w.collectBranchName(); err != nil {
		return err
	}

	// Step 5: Confirm and execute
	return w.confirmAndExecute()
}

func (w *MergebackWorkflow) selectSourceBranch() error {
	items, err := createSortedBranchItems(w.state.Service)
	if err != nil {
		return err
	}

	// Convert to huh options
	options := make([]huh.Option[string], len(items))
	for i, item := range items {
		options[i] = huh.NewOption(item.Label, item.Value)
	}

	wizard := NewWizard([]WizardStep{
		{form: huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Source branch (merge FROM)").
					Description("The branch containing changes to merge").
					Options(options...).
					Value(&w.sourceBranch).
					Height(10),
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

func (w *MergebackWorkflow) selectTargetBranch() error {
	items, err := createSortedBranchItems(w.state.Service)
	if err != nil {
		return err
	}

	// Check for suggested target from config
	suggestedTarget := w.getSuggestedTargetBranch()

	options := w.buildTargetBranchOptions(items, suggestedTarget)

	wizard := NewWizard([]WizardStep{
		{form: huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Target branch (merge INTO)").
					Description("The branch that will receive the changes").
					Options(options...).
					Value(&w.targetBranch).
					Height(10),
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

func (w *MergebackWorkflow) collectWorktreeName() error {
	// Auto-generate: MERGE_<source>-to-<target>
	w.state.WorktreeName = fmt.Sprintf(
		"MERGE_%s-to-%s",
		sanitizeBranchName(w.sourceBranch),
		sanitizeBranchName(w.targetBranch),
	)

	wizard := NewWizard([]WizardStep{
		{form: huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Worktree name").
					Value(&w.state.WorktreeName).
					Validate(createWorktreeNameValidator(w.state.Service)).
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

func (w *MergebackWorkflow) collectBranchName() error {
	// Auto-generate: merge/<source>-to-<target>
	w.state.BranchName = fmt.Sprintf(
		"merge/%s-to-%s",
		sanitizeBranchName(w.sourceBranch),
		sanitizeBranchName(w.targetBranch),
	)

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

func (w *MergebackWorkflow) confirmAndExecute() error {
	var confirm bool

	wizard := NewWizard([]WizardStep{
		{form: huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title("Create mergeback worktree?").
					Description(fmt.Sprintf(
						"Source: %s → Target: %s\nWorktree: %s\nBranch: %s",
						w.sourceBranch,
						w.targetBranch,
						w.state.WorktreeName,
						w.state.BranchName,
					)).
					Value(&confirm),
			),
		)},
	})

	completed, cancelled, err := wizard.Run()
	if err != nil {
		return err
	}
	if cancelled || !completed || !confirm {
		return ErrCancelled
	}

	return w.executeCreation()
}

func (w *MergebackWorkflow) executeCreation() error {
	worktreesDir, err := w.state.Service.GetWorktreesPath()
	if err != nil {
		return err
	}

	// Create worktree
	wt, err := w.state.Service.Git.AddWorktree(
		worktreesDir,
		w.state.WorktreeName,
		w.state.BranchName,
		true,
		w.targetBranch,
		w.state.DryRun,
	)
	if err != nil {
		return err
	}

	printWorktreeSuccess(wt)

	fmt.Printf("\nInitiating merge from %s...\n", w.sourceBranch)

	// Execute merge
	commitMsg := generateMergeCommitMessage(
		w.state.Service,
		w.sourceBranch,
		w.targetBranch,
	)

	err = w.state.Service.Git.MergeBranchWithCommit(
		wt.Path,
		w.sourceBranch,
		commitMsg,
		w.state.DryRun,
	)

	if err != nil {
		fmt.Printf("\n⚠ Worktree created, but merge failed: %v\n", err)
		fmt.Printf("You can manually run the merge in the worktree:\n")
		fmt.Printf("  cd %s\n", wt.Path)
		fmt.Printf("  git merge %s\n", w.sourceBranch)
	} else {
		fmt.Printf("✓ Merge completed successfully!\n")
		fmt.Printf("  Commit message: %s\n", commitMsg)
		fmt.Printf("\nWorktree ready at:\n")
		fmt.Printf("  cd %s\n", wt.Path)
	}

	return nil
}

func (w *MergebackWorkflow) getSuggestedTargetBranch() string {
	config := w.state.Service.GetConfig()
	for _, wtConfig := range config.Worktrees {
		if wtConfig.Branch == w.sourceBranch {
			return wtConfig.MergeInto
		}
	}
	return ""
}

func (w *MergebackWorkflow) buildTargetBranchOptions(
	items []FilterableItem,
	suggestedTarget string,
) []huh.Option[string] {
	options := []huh.Option[string]{}

	// Add suggested first
	if suggestedTarget != "" {
		label := suggestedTarget
		for _, item := range items {
			if item.Value == suggestedTarget {
				label = item.Label
				break
			}
		}
		options = append(options, huh.NewOption(
			fmt.Sprintf("%s (suggested from config)", label),
			suggestedTarget,
		))
	}

	// Add others (excluding source and suggestion)
	for _, item := range items {
		if item.Value != w.sourceBranch && item.Value != suggestedTarget {
			options = append(options, huh.NewOption(item.Label, item.Value))
		}
	}

	return options
}

package service

import (
	"fmt"

	"github.com/charmbracelet/huh"
)

// buildUIForState creates a WizardModel for the given FSM state
// This replaces the individual runXXX state handlers
func (w *WorktreeAddFSM) buildUIForState(state string) *WizardModel {
	switch state {
	case StateSelectType:
		return w.buildSelectTypeUI()
	case StateFeatureWorktreeName:
		return w.buildFeatureWorktreeNameUI()
	case StateFeatureBranchName:
		return w.buildFeatureBranchNameUI()
	case StateFeatureBaseBranch:
		return w.buildFeatureBaseBranchUI()
	case StateFeatureConfirmCreate:
		return w.buildFeatureConfirmCreateUI()
	case StateHotfixWorktreeName:
		return w.buildHotfixWorktreeNameUI()
	case StateHotfixBaseBranch:
		return w.buildHotfixBaseBranchUI()
	case StateHotfixBranchName:
		return w.buildHotfixBranchNameUI()
	case StateMergebackSourceBranch:
		return w.buildMergebackSourceBranchUI()
	case StateMergebackTargetBranch:
		return w.buildMergebackTargetBranchUI()
	case StateMergebackWorktreeName:
		return w.buildMergebackWorktreeNameUI()
	case StateMergebackBranchName:
		return w.buildMergebackBranchNameUI()
	case StateMergebackConfirmMerge:
		return w.buildMergebackConfirmMergeUI()
	}
	return nil
}

// buildSelectTypeUI builds the workflow type selection UI
func (w *WorktreeAddFSM) buildSelectTypeUI() *WizardModel {
	// Use a temp variable but store reference for capture
	var worktreeType string
	
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

	wizard := NewWizard([]WizardStep{{form: typeForm}})
	// Store reference to capture it after completion
	wizard.userData = map[string]interface{}{"selectedType": &worktreeType}
	return &wizard
}

// buildFeatureWorktreeNameUI builds the feature workflow name selection UI
func (w *WorktreeAddFSM) buildFeatureWorktreeNameUI() *WizardModel {
	items := fetchJiraItems(w.state.Service)

	wizard := NewWizard([]WizardStep{
		{customModel: NewFilterableSelect(
			"Worktree name",
			"Select JIRA ticket or enter custom name",
			items,
		)},
	})

	return &wizard
}

// buildFeatureBranchNameUI builds the feature branch name input UI
func (w *WorktreeAddFSM) buildFeatureBranchNameUI() *WizardModel {
	w.state.BranchName = w.generateFeatureDefaultBranchName()

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

	return &wizard
}

// buildFeatureBaseBranchUI builds the base branch selection UI
func (w *WorktreeAddFSM) buildFeatureBaseBranchUI() *WizardModel {
	items, _ := createSortedBranchItems(w.state.Service)

	wizard := NewWizard([]WizardStep{
		{customModel: NewFilterableSelect(
			"Base branch",
			fmt.Sprintf("Branch '%s' doesn't exist. Select base:", w.state.BranchName),
			items,
		)},
	})

	return &wizard
}

// buildFeatureConfirmCreateUI builds the confirmation UI
func (w *WorktreeAddFSM) buildFeatureConfirmCreateUI() *WizardModel {
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

	// Store reference to confirm for later retrieval
	wizard.userData = map[string]interface{}{"confirm": &confirm}
	return &wizard
}

// === Hotfix UI Builders ===

func (w *WorktreeAddFSM) buildHotfixWorktreeNameUI() *WizardModel {
	items := fetchJiraItems(w.state.Service)

	wizard := NewWizard([]WizardStep{
		{customModel: NewFilterableSelect(
			"Worktree name",
			"Select JIRA ticket or enter custom name",
			items,
		)},
	})

	return &wizard
}

func (w *WorktreeAddFSM) buildHotfixBaseBranchUI() *WizardModel {
	items, _ := createSortedBranchItems(w.state.Service)

	wizard := NewWizard([]WizardStep{
		{customModel: NewFilterableSelect(
			"Base branch for hotfix",
			"Select branch (typically production or release)",
			items,
		)},
	})

	return &wizard
}

func (w *WorktreeAddFSM) buildHotfixBranchNameUI() *WizardModel {
	w.state.BranchName = w.generateHotfixDefaultBranchName()

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

	return &wizard
}

// === Mergeback UI Builders ===

func (w *WorktreeAddFSM) buildMergebackSourceBranchUI() *WizardModel {
	items, _ := createSortedBranchItems(w.state.Service)

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
					Value(&w.state.SourceBranch).
					Height(10),
			),
		)},
	})

	return &wizard
}

func (w *WorktreeAddFSM) buildMergebackTargetBranchUI() *WizardModel {
	items, _ := createSortedBranchItems(w.state.Service)
	suggestedTarget := w.getSuggestedTargetBranch()
	options := w.buildTargetBranchOptions(items, suggestedTarget)

	wizard := NewWizard([]WizardStep{
		{form: huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Target branch (merge INTO)").
					Description("The branch that will receive the changes").
					Options(options...).
					Value(&w.state.TargetBranch).
					Height(10),
			),
		)},
	})

	return &wizard
}

func (w *WorktreeAddFSM) buildMergebackWorktreeNameUI() *WizardModel {
	// Auto-generate default name
	w.state.WorktreeName = fmt.Sprintf(
		"%s%s-to-%s",
		MergebackPrefix,
		sanitizeBranchName(w.state.SourceBranch),
		sanitizeBranchName(w.state.TargetBranch),
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

	return &wizard
}

func (w *WorktreeAddFSM) buildMergebackBranchNameUI() *WizardModel {
	// Auto-generate default name
	w.state.BranchName = fmt.Sprintf(
		"%s%s-to-%s",
		MergebackBranchPrefix,
		sanitizeBranchName(w.state.SourceBranch),
		sanitizeBranchName(w.state.TargetBranch),
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

	return &wizard
}

func (w *WorktreeAddFSM) buildMergebackConfirmMergeUI() *WizardModel {
	var confirm bool

	wizard := NewWizard([]WizardStep{
		{form: huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title("Create mergeback worktree?").
					Description(fmt.Sprintf(
						"Source: %s → Target: %s\nWorktree: %s\nBranch: %s",
						w.state.SourceBranch,
						w.state.TargetBranch,
						w.state.WorktreeName,
						w.state.BranchName,
					)).
					Value(&confirm),
			),
		)},
	})

	wizard.userData = map[string]interface{}{"confirm": &confirm}
	return &wizard
}

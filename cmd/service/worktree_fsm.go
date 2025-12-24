package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/looplab/fsm"
)

// WorktreeAddFSM manages the unified state machine for all worktree creation workflows
type WorktreeAddFSM struct {
	fsm          *fsm.FSM
	state        *WorkflowState
	svc          *Service
	selectedType string // "feature", "bug", "hotfix", "mergeback"
}

// WorkflowState stores data collected across workflow steps
type WorkflowState struct {
	// Shared workflow data
	Service      *Service
	WorktreeName string
	BranchName   string
	BaseBranch   string
	DryRun       bool

	// Mergeback-specific fields
	SourceBranch string
	TargetBranch string

	// Error handling
	LastError error

	// Debugging/logging
	StateHistory []string
	EventHistory []string
}

// NewWorktreeAddFSM creates a new unified FSM for worktree creation workflows
func NewWorktreeAddFSM(svc *Service) *WorktreeAddFSM {
	w := &WorktreeAddFSM{
		svc:   svc,
		state: &WorkflowState{Service: svc},
	}

	// Define ALL state transitions for all workflows in one FSM
	w.fsm = fsm.NewFSM(
		StateSelectType, // Initial state
		fsm.Events{
			// Type selection transitions
			{Name: EventSelectFeature, Src: []string{StateSelectType}, Dst: StateFeatureWorktreeName},
			{Name: EventSelectBug, Src: []string{StateSelectType}, Dst: StateFeatureWorktreeName},
			{Name: EventSelectHotfix, Src: []string{StateSelectType}, Dst: StateHotfixWorktreeName},
			{Name: EventSelectMergeback, Src: []string{StateSelectType}, Dst: StateMergebackSourceBranch},
			{Name: EventCancel, Src: []string{StateSelectType}, Dst: StateCancelled},

			// Feature workflow transitions
			{Name: EventComplete, Src: []string{StateFeatureWorktreeName}, Dst: StateFeatureBranchName},
			{Name: EventGoBack, Src: []string{StateFeatureWorktreeName}, Dst: StateSelectType},
			{Name: EventCancel, Src: []string{StateFeatureWorktreeName}, Dst: StateCancelled},

			{Name: EventComplete, Src: []string{StateFeatureBranchName}, Dst: StateFeatureCheckBranch},
			{Name: EventGoBack, Src: []string{StateFeatureBranchName}, Dst: StateFeatureWorktreeName},
			{Name: EventCancel, Src: []string{StateFeatureBranchName}, Dst: StateCancelled},

			{Name: EventBranchExists, Src: []string{StateFeatureCheckBranch}, Dst: StateFeatureExecuteCreate},
			{Name: EventBranchMissing, Src: []string{StateFeatureCheckBranch}, Dst: StateFeatureBaseBranch},
			{Name: EventError, Src: []string{StateFeatureCheckBranch}, Dst: StateError},

			{Name: EventComplete, Src: []string{StateFeatureBaseBranch}, Dst: StateFeatureConfirmCreate},
			{Name: EventGoBack, Src: []string{StateFeatureBaseBranch}, Dst: StateFeatureBranchName},
			{Name: EventCancel, Src: []string{StateFeatureBaseBranch}, Dst: StateCancelled},

			{Name: EventConfirmYes, Src: []string{StateFeatureConfirmCreate}, Dst: StateFeatureExecuteCreate},
			{Name: EventConfirmNo, Src: []string{StateFeatureConfirmCreate}, Dst: StateCancelled},
			{Name: EventGoBack, Src: []string{StateFeatureConfirmCreate}, Dst: StateFeatureBaseBranch},
			{Name: EventCancel, Src: []string{StateFeatureConfirmCreate}, Dst: StateCancelled},

			{Name: EventComplete, Src: []string{StateFeatureExecuteCreate}, Dst: StateSuccess},
			{Name: EventError, Src: []string{StateFeatureExecuteCreate}, Dst: StateError},

			// Hotfix workflow transitions
			{Name: EventComplete, Src: []string{StateHotfixWorktreeName}, Dst: StateHotfixBaseBranch},
			{Name: EventGoBack, Src: []string{StateHotfixWorktreeName}, Dst: StateSelectType},
			{Name: EventCancel, Src: []string{StateHotfixWorktreeName}, Dst: StateCancelled},

			{Name: EventComplete, Src: []string{StateHotfixBaseBranch}, Dst: StateHotfixBranchName},
			{Name: EventGoBack, Src: []string{StateHotfixBaseBranch}, Dst: StateHotfixWorktreeName},
			{Name: EventCancel, Src: []string{StateHotfixBaseBranch}, Dst: StateCancelled},

			{Name: EventComplete, Src: []string{StateHotfixBranchName}, Dst: StateHotfixExecuteCreate},
			{Name: EventGoBack, Src: []string{StateHotfixBranchName}, Dst: StateHotfixBaseBranch},
			{Name: EventCancel, Src: []string{StateHotfixBranchName}, Dst: StateCancelled},

			{Name: EventComplete, Src: []string{StateHotfixExecuteCreate}, Dst: StateSuccess},
			{Name: EventError, Src: []string{StateHotfixExecuteCreate}, Dst: StateError},

			// Mergeback workflow transitions
			{Name: EventComplete, Src: []string{StateMergebackSourceBranch}, Dst: StateMergebackTargetBranch},
			{Name: EventGoBack, Src: []string{StateMergebackSourceBranch}, Dst: StateSelectType},
			{Name: EventCancel, Src: []string{StateMergebackSourceBranch}, Dst: StateCancelled},

			{Name: EventComplete, Src: []string{StateMergebackTargetBranch}, Dst: StateMergebackWorktreeName},
			{Name: EventGoBack, Src: []string{StateMergebackTargetBranch}, Dst: StateMergebackSourceBranch},
			{Name: EventCancel, Src: []string{StateMergebackTargetBranch}, Dst: StateCancelled},

			{Name: EventComplete, Src: []string{StateMergebackWorktreeName}, Dst: StateMergebackBranchName},
			{Name: EventGoBack, Src: []string{StateMergebackWorktreeName}, Dst: StateMergebackTargetBranch},
			{Name: EventCancel, Src: []string{StateMergebackWorktreeName}, Dst: StateCancelled},

			{Name: EventComplete, Src: []string{StateMergebackBranchName}, Dst: StateMergebackConfirmMerge},
			{Name: EventGoBack, Src: []string{StateMergebackBranchName}, Dst: StateMergebackWorktreeName},
			{Name: EventCancel, Src: []string{StateMergebackBranchName}, Dst: StateCancelled},

			{Name: EventConfirmYes, Src: []string{StateMergebackConfirmMerge}, Dst: StateMergebackExecuteCreate},
			{Name: EventConfirmNo, Src: []string{StateMergebackConfirmMerge}, Dst: StateCancelled},
			{Name: EventGoBack, Src: []string{StateMergebackConfirmMerge}, Dst: StateMergebackBranchName},
			{Name: EventCancel, Src: []string{StateMergebackConfirmMerge}, Dst: StateCancelled},

			{Name: EventComplete, Src: []string{StateMergebackExecuteCreate}, Dst: StateMergebackExecuteMerge},
			{Name: EventError, Src: []string{StateMergebackExecuteCreate}, Dst: StateError},

			{Name: EventComplete, Src: []string{StateMergebackExecuteMerge}, Dst: StateSuccess},
			{Name: EventError, Src: []string{StateMergebackExecuteMerge}, Dst: StateError},

			// Terminal state transitions (for looping)
			{Name: EventRetry, Src: []string{StateSuccess}, Dst: StateSelectType},
			{Name: EventRetry, Src: []string{StateCancelled}, Dst: StateSelectType},
		},
		fsm.Callbacks{
			"enter_state": func(ctx context.Context, e *fsm.Event) {
				w.state.StateHistory = append(w.state.StateHistory, e.Dst)
				w.state.EventHistory = append(w.state.EventHistory, e.Event)
			},
		},
	)

	return w
}

// Run executes the unified FSM workflow loop
func (w *WorktreeAddFSM) Run() error {
	ctx := context.Background()

	for {
		current := w.fsm.Current()

		// Check terminal states
		switch current {
		case StateSuccess:
			// For now, don't loop - just return success
			// TODO: Optionally prompt user if they want to create another worktree
			return nil

		case StateCancelled:
			return ErrCancelled

		case StateError:
			return w.state.LastError
		}

		// Run current state's UI and get next event
		event, err := w.runCurrentState(current)
		if err != nil {
			return err
		}

		// Trigger FSM transition
		if err := w.fsm.Event(ctx, event); err != nil {
			return fmt.Errorf("FSM transition error from %s via %s: %w",
				current, event, err)
		}
	}
}

// runCurrentState dispatches to the appropriate state handler
func (w *WorktreeAddFSM) runCurrentState(state string) (string, error) {
	switch state {
	// Entry point
	case StateSelectType:
		return w.runSelectType()

	// Feature workflow states
	case StateFeatureWorktreeName:
		return w.runFeatureWorktreeName()
	case StateFeatureBranchName:
		return w.runFeatureBranchName()
	case StateFeatureCheckBranch:
		return w.runFeatureCheckBranch()
	case StateFeatureBaseBranch:
		return w.runFeatureBaseBranch()
	case StateFeatureConfirmCreate:
		return w.runFeatureConfirmCreate()
	case StateFeatureExecuteCreate:
		return w.runFeatureExecuteCreate()

	// Hotfix workflow states
	case StateHotfixWorktreeName:
		return w.runHotfixWorktreeName()
	case StateHotfixBaseBranch:
		return w.runHotfixBaseBranch()
	case StateHotfixBranchName:
		return w.runHotfixBranchName()
	case StateHotfixExecuteCreate:
		return w.runHotfixExecuteCreate()

	// Mergeback workflow states
	case StateMergebackSourceBranch:
		return w.runMergebackSourceBranch()
	case StateMergebackTargetBranch:
		return w.runMergebackTargetBranch()
	case StateMergebackWorktreeName:
		return w.runMergebackWorktreeName()
	case StateMergebackBranchName:
		return w.runMergebackBranchName()
	case StateMergebackConfirmMerge:
		return w.runMergebackConfirmMerge()
	case StateMergebackExecuteCreate:
		return w.runMergebackExecuteCreate()
	case StateMergebackExecuteMerge:
		return w.runMergebackExecuteMerge()

	default:
		return "", fmt.Errorf("unknown state: %s", state)
	}
}

// === State Handler Implementations ===

// runSelectType - Initial state: select workflow type
func (w *WorktreeAddFSM) runSelectType() (string, error) {
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
	_, err := wizard.Run()
	if err != nil {
		if errors.Is(err, ErrCancelled) {
			return EventCancel, nil
		}
		if errors.Is(err, ErrGoBack) {
			// ESC on first screen - no previous screen, treat as cancel
			return EventCancel, nil
		}
		return "", err
	}

	// Store the selected type
	w.selectedType = worktreeType

	// Return the appropriate event to transition to the workflow
	switch worktreeType {
	case "feature":
		return EventSelectFeature, nil
	case "bug":
		return EventSelectBug, nil
	case "hotfix":
		return EventSelectHotfix, nil
	case "mergeback":
		return EventSelectMergeback, nil
	default:
		w.state.LastError = fmt.Errorf("unknown worktree type: %s", worktreeType)
		return EventError, nil
	}
}

// === Feature Workflow Handlers ===

func (w *WorktreeAddFSM) runFeatureWorktreeName() (string, error) {
	items := fetchJiraItems(w.state.Service)

	wizard := NewWizard([]WizardStep{
		{customModel: NewFilterableSelect(
			"Worktree name",
			"Select JIRA ticket or enter custom name",
			items,
		), isCustom: true},
	})

	finalWizard, err := wizard.Run()
	if err != nil {
		if errors.Is(err, ErrCancelled) {
			return EventCancel, nil
		}
		if errors.Is(err, ErrGoBack) {
			return EventGoBack, nil
		}
		return "", err
	}

	// Extract and validate
	model := finalWizard.Steps[0].customModel.(FilterableSelectModel)
	w.state.WorktreeName = model.GetSelected()

	// Validate
	if err := createWorktreeNameValidator(w.state.Service)(w.state.WorktreeName); err != nil {
		// Show error and retry in the same state
		errorForm := huh.NewForm(
			huh.NewGroup(
				huh.NewNote().
					Title("Validation Error").
					Description(err.Error()),
			),
		)
		errorWizard := NewWizard([]WizardStep{{form: errorForm}})
		_, _ = errorWizard.Run() // Ignore error - just showing message
		// Stay in same state - user will retry
		return w.runFeatureWorktreeName()
	}

	return EventComplete, nil
}

func (w *WorktreeAddFSM) runFeatureBranchName() (string, error) {
	// Set default based on JIRA issue or custom input
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

	_, err := wizard.Run()
	if err != nil {
		if errors.Is(err, ErrCancelled) {
			return EventCancel, nil
		}
		if errors.Is(err, ErrGoBack) {
			return EventGoBack, nil
		}
		return "", err
	}

	return EventComplete, nil
}

func (w *WorktreeAddFSM) runFeatureCheckBranch() (string, error) {
	// Check if branch exists
	exists, err := w.state.Service.Git.BranchExists(w.state.BranchName)
	if err != nil {
		w.state.LastError = fmt.Errorf("failed to check if branch exists: %w", err)
		return EventError, nil
	}

	if exists {
		// Branch exists - can use existing
		return EventBranchExists, nil
	}

	// Branch doesn't exist - need to create new branch
	return EventBranchMissing, nil
}

func (w *WorktreeAddFSM) runFeatureBaseBranch() (string, error) {
	items, err := createSortedBranchItems(w.state.Service)
	if err != nil {
		w.state.LastError = fmt.Errorf("failed to get branches: %w", err)
		return EventError, nil
	}

	wizard := NewWizard([]WizardStep{
		{customModel: NewFilterableSelect(
			"Base branch",
			fmt.Sprintf("Branch '%s' doesn't exist. Select base:", w.state.BranchName),
			items,
		), isCustom: true},
	})

	finalWizard, err := wizard.Run()
	if err != nil {
		if errors.Is(err, ErrCancelled) {
			return EventCancel, nil
		}
		if errors.Is(err, ErrGoBack) {
			return EventGoBack, nil
		}
		return "", err
	}

	model := finalWizard.Steps[0].customModel.(FilterableSelectModel)
	w.state.BaseBranch = model.GetSelected()
	return EventComplete, nil
}

func (w *WorktreeAddFSM) runFeatureConfirmCreate() (string, error) {
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

	_, err := wizard.Run()
	if err != nil {
		if errors.Is(err, ErrCancelled) {
			return EventCancel, nil
		}
		if errors.Is(err, ErrGoBack) {
			return EventGoBack, nil
		}
		return "", err
	}
	if !confirm {
		return EventConfirmNo, nil
	}

	return EventConfirmYes, nil
}

func (w *WorktreeAddFSM) runFeatureExecuteCreate() (string, error) {
	worktreesDir, err := w.state.Service.GetWorktreesPath()
	if err != nil {
		w.state.LastError = err
		return EventError, nil
	}

	// Determine if we need to create a branch
	createBranch := (w.state.BaseBranch != "")

	wt, err := w.state.Service.Git.AddWorktree(
		worktreesDir,
		w.state.WorktreeName,
		w.state.BranchName,
		createBranch,
		w.state.BaseBranch,
		w.state.DryRun,
	)
	if err != nil {
		w.state.LastError = fmt.Errorf("failed to create worktree: %w", err)
		return EventError, nil
	}

	// Copy files if configured
	if copyErr := w.state.Service.CopyFilesToWorktree(w.state.WorktreeName); copyErr != nil {
		fmt.Printf("Warning: failed to copy files to worktree: %v\n", copyErr)
	}

	printWorktreeSuccess(wt)
	return EventComplete, nil
}

// generateFeatureDefaultBranchName creates a default branch name for feature workflow
func (w *WorktreeAddFSM) generateFeatureDefaultBranchName() string {
	// Determine prefix based on selected type
	prefix := w.selectedType
	if prefix == "" {
		prefix = "feature"
	}

	// Check if worktree name is a JIRA key
	issues, err := w.state.Service.Jira.GetJiraIssues(false)
	if err == nil {
		for _, issue := range issues {
			if issue.Key == w.state.WorktreeName {
				sanitized := sanitizeSummaryForBranch(issue.Summary)
				return fmt.Sprintf("%s/%s-%s", prefix, issue.Key, sanitized)
			}
		}
	}
	return prefix + "/"
}

// === Hotfix Workflow Handlers ===

func (w *WorktreeAddFSM) runHotfixWorktreeName() (string, error) {
	items := fetchJiraItems(w.state.Service)

	wizard := NewWizard([]WizardStep{
		{customModel: NewFilterableSelect(
			"Worktree name",
			"Select JIRA ticket or enter custom name",
			items,
		), isCustom: true},
	})

	finalWizard, err := wizard.Run()
	if err != nil {
		if errors.Is(err, ErrCancelled) {
			return EventCancel, nil
		}
		if errors.Is(err, ErrGoBack) {
			return EventGoBack, nil
		}
		return "", err
	}

	model := finalWizard.Steps[0].customModel.(FilterableSelectModel)
	selectedName := model.GetSelected()

	// Add HOTFIX_ prefix
	w.state.WorktreeName = "HOTFIX_" + selectedName

	// Validate
	if err := createWorktreeNameValidator(w.state.Service)(w.state.WorktreeName); err != nil {
		// Show error and retry
		errorForm := huh.NewForm(
			huh.NewGroup(
				huh.NewNote().
					Title("Validation Error").
					Description(err.Error()),
			),
		)
		errorWizard := NewWizard([]WizardStep{{form: errorForm}})
		_, _ = errorWizard.Run() // Ignore error - just showing message
		return w.runHotfixWorktreeName()
	}

	return EventComplete, nil
}

func (w *WorktreeAddFSM) runHotfixBaseBranch() (string, error) {
	items, err := createSortedBranchItems(w.state.Service)
	if err != nil {
		w.state.LastError = err
		return EventError, nil
	}

	wizard := NewWizard([]WizardStep{
		{customModel: NewFilterableSelect(
			"Base branch for hotfix",
			"Select branch (typically production or release)",
			items,
		), isCustom: true},
	})

	finalWizard, err := wizard.Run()
	if err != nil {
		if errors.Is(err, ErrCancelled) {
			return EventCancel, nil
		}
		if errors.Is(err, ErrGoBack) {
			return EventGoBack, nil
		}
		return "", err
	}

	model := finalWizard.Steps[0].customModel.(FilterableSelectModel)
	w.state.BaseBranch = model.GetSelected()
	return EventComplete, nil
}

func (w *WorktreeAddFSM) runHotfixBranchName() (string, error) {
	// Generate default from JIRA issue if applicable
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

	_, err := wizard.Run()
	if err != nil {
		if errors.Is(err, ErrCancelled) {
			return EventCancel, nil
		}
		if errors.Is(err, ErrGoBack) {
			return EventGoBack, nil
		}
		return "", err
	}

	return EventComplete, nil
}

func (w *WorktreeAddFSM) runHotfixExecuteCreate() (string, error) {
	worktreesDir, err := w.state.Service.GetWorktreesPath()
	if err != nil {
		w.state.LastError = err
		return EventError, nil
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
		w.state.LastError = fmt.Errorf("failed to create worktree: %w", err)
		return EventError, nil
	}

	if copyErr := w.state.Service.CopyFilesToWorktree(w.state.WorktreeName); copyErr != nil {
		fmt.Printf("Warning: failed to copy files to worktree: %v\n", copyErr)
	}

	printWorktreeSuccess(wt)
	return EventComplete, nil
}

// generateHotfixDefaultBranchName creates a default branch name for hotfix workflow
func (w *WorktreeAddFSM) generateHotfixDefaultBranchName() string {
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

// === Mergeback Workflow Handlers ===

func (w *WorktreeAddFSM) runMergebackSourceBranch() (string, error) {
	items, err := createSortedBranchItems(w.state.Service)
	if err != nil {
		w.state.LastError = err
		return EventError, nil
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
					Value(&w.state.SourceBranch).
					Height(10),
			),
		)},
	})

	_, err = wizard.Run()
	if err != nil {
		if errors.Is(err, ErrCancelled) {
			return EventCancel, nil
		}
		if errors.Is(err, ErrGoBack) {
			return EventGoBack, nil
		}
		return "", err
	}

	return EventComplete, nil
}

func (w *WorktreeAddFSM) runMergebackTargetBranch() (string, error) {
	items, err := createSortedBranchItems(w.state.Service)
	if err != nil {
		w.state.LastError = err
		return EventError, nil
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
					Value(&w.state.TargetBranch).
					Height(10),
			),
		)},
	})

	_, err = wizard.Run()
	if err != nil {
		if errors.Is(err, ErrCancelled) {
			return EventCancel, nil
		}
		if errors.Is(err, ErrGoBack) {
			return EventGoBack, nil
		}
		return "", err
	}

	return EventComplete, nil
}

func (w *WorktreeAddFSM) runMergebackWorktreeName() (string, error) {
	// Auto-generate: MERGE_<source>-to-<target>
	w.state.WorktreeName = fmt.Sprintf(
		"MERGE_%s-to-%s",
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

	_, err := wizard.Run()
	if err != nil {
		if errors.Is(err, ErrCancelled) {
			return EventCancel, nil
		}
		if errors.Is(err, ErrGoBack) {
			return EventGoBack, nil
		}
		return "", err
	}

	return EventComplete, nil
}

func (w *WorktreeAddFSM) runMergebackBranchName() (string, error) {
	// Auto-generate: merge/<source>-to-<target>
	w.state.BranchName = fmt.Sprintf(
		"merge/%s-to-%s",
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

	_, err := wizard.Run()
	if err != nil {
		if errors.Is(err, ErrCancelled) {
			return EventCancel, nil
		}
		if errors.Is(err, ErrGoBack) {
			return EventGoBack, nil
		}
		return "", err
	}

	return EventComplete, nil
}

func (w *WorktreeAddFSM) runMergebackConfirmMerge() (string, error) {
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

	_, err := wizard.Run()
	if err != nil {
		if errors.Is(err, ErrCancelled) {
			return EventCancel, nil
		}
		if errors.Is(err, ErrGoBack) {
			return EventGoBack, nil
		}
		return "", err
	}
	if !confirm {
		return EventConfirmNo, nil
	}

	return EventConfirmYes, nil
}

func (w *WorktreeAddFSM) runMergebackExecuteCreate() (string, error) {
	worktreesDir, err := w.state.Service.GetWorktreesPath()
	if err != nil {
		w.state.LastError = err
		return EventError, nil
	}

	// Create worktree
	wt, err := w.state.Service.Git.AddWorktree(
		worktreesDir,
		w.state.WorktreeName,
		w.state.BranchName,
		true,
		w.state.TargetBranch,
		w.state.DryRun,
	)
	if err != nil {
		w.state.LastError = fmt.Errorf("failed to create worktree: %w", err)
		return EventError, nil
	}

	printWorktreeSuccess(wt)
	return EventComplete, nil
}

func (w *WorktreeAddFSM) runMergebackExecuteMerge() (string, error) {
	worktreesDir, err := w.state.Service.GetWorktreesPath()
	if err != nil {
		w.state.LastError = err
		return EventError, nil
	}

	// Get worktree path
	wtPath := fmt.Sprintf("%s/%s", worktreesDir, w.state.WorktreeName)

	fmt.Printf("\nInitiating merge from %s...\n", w.state.SourceBranch)

	// Execute merge
	commitMsg := generateMergeCommitMessage(
		w.state.Service,
		w.state.SourceBranch,
		w.state.TargetBranch,
	)

	err = w.state.Service.Git.MergeBranchWithCommit(
		wtPath,
		w.state.SourceBranch,
		commitMsg,
		w.state.DryRun,
	)

	if err != nil {
		fmt.Printf("\n⚠ Worktree created, but merge failed: %v\n", err)
		fmt.Printf("You can manually run the merge in the worktree:\n")
		fmt.Printf("  cd %s\n", wtPath)
		fmt.Printf("  git merge %s\n", w.state.SourceBranch)
		// Don't return error - workflow succeeded even if merge failed
	} else {
		fmt.Printf("✓ Merge completed successfully!\n")
		fmt.Printf("  Commit message: %s\n", commitMsg)
		fmt.Printf("\nWorktree ready at:\n")
		fmt.Printf("  cd %s\n", wtPath)
	}

	return EventComplete, nil
}

// getSuggestedTargetBranch looks up suggested merge target from config
func (w *WorktreeAddFSM) getSuggestedTargetBranch() string {
	config := w.state.Service.GetConfig()
	for _, wtConfig := range config.Worktrees {
		if wtConfig.Branch == w.state.SourceBranch {
			return wtConfig.MergeInto
		}
	}
	return ""
}

// buildTargetBranchOptions creates branch options with suggested target first
func (w *WorktreeAddFSM) buildTargetBranchOptions(
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
		if item.Value != w.state.SourceBranch && item.Value != suggestedTarget {
			options = append(options, huh.NewOption(item.Label, item.Value))
		}
	}

	return options
}

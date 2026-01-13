package service

import (
	"fmt"
	"os"

	"gbm/internal/jira"
	"gbm/pkg/tui"
	"gbm/pkg/tui/workflows"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

func newWorktreeTestaddCommand(svc *Service) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "testadd",
		Short: "Test the wizard UI with mock data",
		Long: `Launch the interactive TUI workflow with mock JIRA and Git services.

This command is useful for testing and developing the wizard UI without
affecting real repositories or making actual API calls.

The wizard follows a two-step process:
1. Select a workflow type: Feature, Bug, Hotfix, or Merge
2. Follow the workflow-specific steps (e.g., select JIRA issue, branch name, base branch)

Each workflow type has different configurations:
- Feature: Creates feature branches from JIRA issues with optional base branch selection
- Bug: Like feature, but for bug fixes with bug/ prefix
- Hotfix: Requires mandatory base branch selection (for production hotfixes)
- Merge: Merge branches without JIRA issues with optional merge_into suggestion from config

With real services:
- The wizard uses actual JIRA issues from the configured board
- Git operations work with the real repository
- This allows testing the full workflow with realistic data

No actual worktrees or branches are created (dry-run mode).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWorktreeTestaddCommand(svc)
		},
	}

	return cmd
}

// testaddNavigatorAdapter wraps testadd workflow to work with Navigator.
// It handles type selection and workflow progression via the Navigator stack.
type testaddNavigatorAdapter struct {
	nav      *tui.Navigator
	stepsMap map[string][]tui.Step
	ctx      *tui.Context
}

// newTestaddNavigatorAdapter creates a new adapter with Navigator initialized with type selector.
func newTestaddNavigatorAdapter(ctx *tui.Context, stepsMap map[string][]tui.Step) *testaddNavigatorAdapter {
	typeSelector := workflows.SelectWorkflowType()
	typeSelectorStep := tui.Step{
		Name:  "workflow_type_selector",
		Field: typeSelector,
	}
	typeWizard := tui.NewWizard([]tui.Step{typeSelectorStep}, ctx)

	return &testaddNavigatorAdapter{
		nav:      tui.NewNavigator(typeWizard),
		stepsMap: stepsMap,
		ctx:      ctx,
	}
}

// Init delegates to Navigator.
func (a *testaddNavigatorAdapter) Init() tea.Cmd {
	return a.nav.Init()
}

// Update handles type selection completion and workflow transitions.
func (a *testaddNavigatorAdapter) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle Ctrl+C to quit
	if km, ok := msg.(tea.KeyMsg); ok && km.Type == tea.KeyCtrlC {
		return a, tea.Quit
	}

	// Handle back navigation from workflow
	if _, ok := msg.(tui.BackBoundaryMsg); ok && a.nav.Depth() > 1 {
		a.nav.Pop()
		newTypeWizard := a.createTypeWizard()
		a.nav.Pop()
		a.nav.Push(newTypeWizard)
		return a, newTypeWizard.Init()
	}

	// Update through Navigator
	_, cmd := a.nav.Update(msg)
	currentModel := a.nav.Current()

	// Handle type selector completion
	if a.nav.Depth() == 1 {
		if wiz, ok := currentModel.(*tui.Wizard); ok && wiz.IsComplete() {
			cmd = a.transitionToWorkflow(wiz)
			if cmd != nil {
				return a, cmd
			}
		}
	}

	// Handle workflow completion or workflow complete message
	if a.nav.Depth() > 1 {
		if wiz, ok := currentModel.(*tui.Wizard); ok && wiz.IsComplete() {
			return a, tea.Quit
		}
		if _, ok := msg.(tui.WorkflowCompleteMsg); ok {
			return a, tea.Quit
		}
	}

	return a, cmd
}

// transitionToWorkflow creates and pushes workflow wizard for selected type.
func (a *testaddNavigatorAdapter) transitionToWorkflow(typeWiz *tui.Wizard) tea.Cmd {
	selectedType := typeWiz.State().WorkflowType
	if selectedType == "" || selectedType == "unknown" {
		return nil
	}

	steps, ok := a.stepsMap[selectedType]
	if !ok {
		return nil
	}

	workflowWizard := tui.NewWizard(steps, a.ctx)
	workflowWizard.State().WorkflowType = selectedType
	a.nav.Push(workflowWizard)
	return workflowWizard.Init()
}

// createTypeWizard creates a fresh type selector wizard.
func (a *testaddNavigatorAdapter) createTypeWizard() *tui.Wizard {
	typeSelector := workflows.SelectWorkflowType()
	typeSelectorStep := tui.Step{
		Name:  "workflow_type_selector",
		Field: typeSelector,
	}
	return tui.NewWizard([]tui.Step{typeSelectorStep}, a.ctx)
}

// View delegates to Navigator.
func (a *testaddNavigatorAdapter) View() string {
	return a.nav.View()
}

// jiraServiceAdapter adapts *jira.Service to tui.JiraService interface.
type jiraServiceAdapter struct {
	jiraService jiraService
}

type jiraService interface {
	GetJiraIssues(filters jira.JiraFilters, dryRun bool) ([]jira.JiraIssue, error)
}

func newJiraServiceAdapter(jiraSvc jiraService) *jiraServiceAdapter {
	return &jiraServiceAdapter{jiraService: jiraSvc}
}

func (a *jiraServiceAdapter) FetchIssues() ([]tui.JiraIssue, error) {
	// Fetch issues using an empty filter and no dry-run
	issues, err := a.jiraService.GetJiraIssues(jira.JiraFilters{}, false)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JIRA issues: %w", err)
	}

	// Convert internal JiraIssue to tui.JiraIssue
	result := make([]tui.JiraIssue, len(issues))
	for i, issue := range issues {
		result[i] = tui.JiraIssue{
			Key:     issue.Key,
			Summary: issue.Summary,
		}
	}
	return result, nil
}

// runWorktreeTestaddCommand runs the testadd wizard using real service dependencies.
// Uses Navigator to manage seamless transitions between:
// 1. Type selector screen - user chooses Feature/Bug/Hotfix/Merge
// 2. Workflow screen - workflow-specific steps for the selected type
func runWorktreeTestaddCommand(svc *Service) error {
	// Build context with services
	ctx := tui.NewContext().
		WithDimensions(100, 30).
		WithTheme(tui.DefaultTheme()).
		WithGitService(svc.Git).
		WithJiraService(newJiraServiceAdapter(svc.Jira)).
		WithConfig(svc.GetConfig())

	// Build stepsMap for all workflow types
	stepsMap, err := buildStepsMap(ctx)
	if err != nil {
		return err
	}

	// Open input and run program
	input, err := os.Open("/dev/tty")
	if err != nil {
		return fmt.Errorf("failed to open /dev/tty: %w", err)
	}
	defer func() {
		_ = input.Close()
	}()

	// Run the adapter program
	adapter := newTestaddNavigatorAdapter(ctx, stepsMap)
	p := tea.NewProgram(adapter, tea.WithInput(input), tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("testadd error: %w", err)
	}

	// Handle final state
	return handleFinalState(finalModel)
}

// buildStepsMap creates workflow steps for each workflow type.
func buildStepsMap(ctx *tui.Context) (map[string][]tui.Step, error) {
	stepsMap := make(map[string][]tui.Step)
	for _, workflowType := range []string{"feature", "bug", "hotfix", "merge"} {
		steps, err := workflows.GetWorkflowSteps(workflowType, ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get steps for workflow %s: %w", workflowType, err)
		}
		stepsMap[workflowType] = steps
	}
	return stepsMap, nil
}

// handleFinalState processes the final wizard state and prints results.
func handleFinalState(finalModel tea.Model) error {
	adapter, ok := finalModel.(*testaddNavigatorAdapter)
	if !ok {
		return fmt.Errorf("unexpected model type: %T", finalModel)
	}

	currentModel := adapter.nav.Current()
	if currentModel == nil {
		return fmt.Errorf("no wizard found")
	}

	w, ok := currentModel.(*tui.Wizard)
	if !ok {
		return fmt.Errorf("unexpected wizard type: %T", currentModel)
	}

	// Handle completion
	if w.IsComplete() {
		state := w.State()
		fmt.Fprintf(os.Stderr, "Would create worktree: %s\n", state.WorktreeName)
		fmt.Fprintf(os.Stderr, "Would create branch: %s\n", state.BranchName)
		if state.BaseBranch != "" {
			fmt.Fprintf(os.Stderr, "Based on: %s\n", state.BaseBranch)
		}
		return nil
	}

	// Handle cancellation (including Ctrl+C)
	fmt.Fprintf(os.Stderr, "Cancelled\n")
	return nil
}

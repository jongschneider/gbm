package service

import (
	"fmt"
	"os"
	"time"

	"gbm/internal/testing"
	"gbm/pkg/tui"
	"gbm/pkg/tui/workflows"
	"gbm/testutil"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

func newWorktreeTestaddCommand(svc *Service) *cobra.Command {
	var delayMs int
	var withConfig bool

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

With --config flag enabled:
- Merge workflow will include a suggested target branch (merge_into from config)
- This allows testing merge suggestion logic with realistic worktree configurations

No actual worktrees or branches are created (dry-run mode).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWorktreeTestaddCommand(cmd, delayMs, withConfig)
		},
	}

	cmd.Flags().IntVar(&delayMs, "delay", 0, "simulate network latency in milliseconds (0-5000)")
	cmd.Flags().BoolVar(&withConfig, "config", false, "include mock repository configuration for testing merge suggestions")

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

	// Get current model before update
	currentModel := a.nav.Current()
	var cmd tea.Cmd

	// Update through Navigator
	_, cmd = a.nav.Update(msg)

	// Check if type selector just completed
	if typeWiz, ok := currentModel.(*tui.Wizard); ok && typeWiz.IsComplete() {
		selectedType := typeWiz.State().WorkflowType
		if selectedType != "" && selectedType != "unknown" {
			// Look up workflow steps for selected type
			if steps, ok := a.stepsMap[selectedType]; ok {
				// Create new workflow wizard preserving context and selected type
				workflowWizard := tui.NewWizard(steps, a.ctx)
				workflowWizard.State().WorkflowType = selectedType

				// Push workflow wizard onto Navigator stack
				a.nav.Push(workflowWizard)
				// Return init command for the new wizard
				return a, workflowWizard.Init()
			}
		}
	}

	// Check for back navigation from workflow (ESC at first step)
	if _, ok := msg.(tui.BackBoundaryMsg); ok && a.nav.Depth() > 1 {
		// Pop workflow wizard, return to type selector
		a.nav.Pop()
		// Reset type selector
		typeSelector := workflows.SelectWorkflowType()
		typeSelectorStep := tui.Step{
			Name:  "workflow_type_selector",
			Field: typeSelector,
		}
		newTypeWizard := tui.NewWizard([]tui.Step{typeSelectorStep}, a.ctx)
		a.nav.Pop() // Remove old type selector
		a.nav.Push(newTypeWizard)
		return a, newTypeWizard.Init()
	}

	return a, cmd
}

// View delegates to Navigator.
func (a *testaddNavigatorAdapter) View() string {
	return a.nav.View()
}

// runWorktreeTestaddCommand runs the test wizard with mock services using Navigator.
// Uses Navigator to manage seamless transitions between:
// 1. Type selector screen - user chooses Feature/Bug/Hotfix/Merge
// 2. Workflow screen - workflow-specific steps for the selected type
//
// If withConfig is true, creates a MockRepoConfig with sample worktrees to test merge suggestions.
func runWorktreeTestaddCommand(cmd *cobra.Command, delayMs int, withConfig bool) error {
	// Validate delay flag
	if delayMs < 0 || delayMs > 5000 {
		return fmt.Errorf("delay must be between 0 and 5000 milliseconds")
	}

	// Create mock services with delay if specified
	mockGit := testing.NewMockGitService().
		WithBranches([]string{
			"main",
			"master",
			"develop",
			"staging",
			"production",
			"release/v1.0",
			"release/v2.0",
		})
	if delayMs > 0 {
		mockGit = mockGit.WithDelay(time.Duration(delayMs) * time.Millisecond)
	}

	mockJira := testing.NewMockJiraService()
	if delayMs > 0 {
		mockJira = mockJira.WithDelay(time.Duration(delayMs) * time.Millisecond)
	}

	// Create context with mock services and optional config
	ctx := tui.NewContext().
		WithDimensions(100, 30).
		WithTheme(tui.DefaultTheme()).
		WithGitService(mockGit).
		WithJiraService(mockJira)

	// If --config flag is set, add a mock repository configuration for testing merge suggestions
	if withConfig {
		mockConfig := testutil.NewMockRepoConfig().
			WithWorktree("feature_auth", "feature/auth", "main").
			WithWorktree("bugfix_performance", "bugfix/performance", "main").
			WithWorktree("release_v1", "release/v1.0", "production")
		ctx = ctx.WithConfig(mockConfig)
	}

	// Build stepsMap: maps workflow type to its step configuration
	stepsMap := make(map[string][]tui.Step)
	for _, workflowType := range []string{"feature", "bug", "hotfix", "merge"} {
		steps, err := workflows.GetWorkflowSteps(workflowType, ctx)
		if err != nil {
			return fmt.Errorf("failed to get steps for workflow %s: %w", workflowType, err)
		}
		stepsMap[workflowType] = steps
	}

	// Open input for the program
	input, err := os.Open("/dev/tty")
	if err != nil {
		return fmt.Errorf("failed to open /dev/tty: %w", err)
	}
	defer func() {
		_ = input.Close()
	}()

	// Create Navigator adapter and run the program
	adapter := newTestaddNavigatorAdapter(ctx, stepsMap)
	p := tea.NewProgram(adapter, tea.WithInput(input), tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("testadd error: %w", err)
	}

	// Check the final state - extract wizard from adapter
	if adapter, ok := finalModel.(*testaddNavigatorAdapter); ok {
		if currentModel := adapter.nav.Current(); currentModel != nil {
			if w, ok := currentModel.(*tui.Wizard); ok {
				if w.IsCancelled() {
					fmt.Fprintf(os.Stderr, "Cancelled\n")
					return nil
				}

				if w.IsComplete() {
					// Print dry-run summary
					state := w.State()
					fmt.Fprintf(os.Stderr, "Would create worktree: %s\n", state.WorktreeName)
					fmt.Fprintf(os.Stderr, "Would create branch: %s\n", state.BranchName)
					if state.BaseBranch != "" {
						fmt.Fprintf(os.Stderr, "Based on: %s\n", state.BaseBranch)
					}
					return nil
				}

				return fmt.Errorf("wizard did not complete or cancel")
			}
		}
	}

	return fmt.Errorf("unexpected model type: %T", finalModel)
}

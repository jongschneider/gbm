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

// testaddWrapperModel manages the testadd flow with multiple wizards.
// It seamlessly transitions between:
// 1. Type selector stage - user chooses Feature/Bug/Hotfix/Merge
// 2. Workflow stage - workflow-specific steps for the selected type
//
// The model handles back navigation from workflow→type_selector and
// forward navigation from type_selector→workflow without program restart.
type testaddWrapperModel struct {
	stage         string // StageTypeSelection or StageWorkflow
	currentWizard *tui.Wizard
	typeWizard    *tui.Wizard
	stepsMap      map[string][]tui.Step
	selectedType  string
	ctx           *tui.Context
}

// newTestaddWrapperModel creates a new wrapper model with initialized fields.
func newTestaddWrapperModel(ctx *tui.Context, stepsMap map[string][]tui.Step, typeWizard *tui.Wizard) *testaddWrapperModel {
	return &testaddWrapperModel{
		stage:         StageTypeSelection,
		currentWizard: typeWizard,
		typeWizard:    typeWizard,
		stepsMap:      stepsMap,
		ctx:           ctx,
	}
}

// Init initializes the type selector wizard.
func (m *testaddWrapperModel) Init() tea.Cmd {
	return m.currentWizard.Init()
}

// Update processes messages and delegates to currentWizard.
// It handles transitions between stages based on wizard completion/cancellation.
func (m *testaddWrapperModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle program exit on Ctrl+C at the root level
	if km, ok := msg.(tea.KeyMsg); ok && km.Type == tea.KeyCtrlC {
		return m, tea.Quit
	}

	// Update current wizard
	updatedModel, cmd := m.currentWizard.Update(msg)
	if w, ok := updatedModel.(*tui.Wizard); ok {
		m.currentWizard = w
	}

	// Handle transitions based on stage and wizard state
	switch m.stage {
	case StageTypeSelection:
		// Check if type selector completed
		if m.currentWizard.IsComplete() {
			selectedType := m.currentWizard.State().WorkflowType
			if selectedType != "" {
				m.selectedType = selectedType

				// Look up workflow steps for selected type
				if steps, ok := m.stepsMap[selectedType]; ok {
					// Create new wizard with workflow steps, preserving context
					workflowWizard := tui.NewWizard(steps, m.ctx)
					workflowWizard.State().WorkflowType = selectedType
					m.currentWizard = workflowWizard
					m.stage = StageWorkflow

					// Initialize the workflow wizard
					return m, m.currentWizard.Init()
				}
			}
		}
	case StageWorkflow:
		// Check if we received BackBoundaryMsg (ESC at first step)
		if _, ok := msg.(tui.BackBoundaryMsg); ok {
			// Return to type selector
			m.stage = StageTypeSelection
			m.currentWizard = m.typeWizard
			// Reset type wizard state
			m.typeWizard = tui.NewWizard([]tui.Step{{Name: "workflow_type_selector", Field: workflows.SelectWorkflowType()}}, m.ctx)
			m.currentWizard = m.typeWizard

			// Initialize the type wizard
			return m, m.currentWizard.Init()
		}
	}

	// Delegate all other messages
	return m, cmd
}

// View delegates to currentWizard.View().
func (m *testaddWrapperModel) View() string {
	return m.currentWizard.View()
}

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

// runWorktreeTestaddCommand runs the test wizard with mock services.
// Uses a single testaddWrapperModel to manage seamless transitions between:
// 1. Type selector stage - user chooses Feature/Bug/Hotfix/Merge
// 2. Workflow stage - workflow-specific steps for the selected type
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

	// Open input for both wizards
	input, err := os.Open("/dev/tty")
	if err != nil {
		return fmt.Errorf("failed to open /dev/tty: %w", err)
	}
	defer func() {
		_ = input.Close()
	}()

	// Create type selector wizard
	typeSelector := workflows.SelectWorkflowType()
	typeSelectorStep := tui.Step{
		Name:  "workflow_type_selector",
		Field: typeSelector,
	}
	typeWizard := tui.NewWizard([]tui.Step{typeSelectorStep}, ctx)

	// Create and run the wrapper model - single program for entire flow
	wrapper := newTestaddWrapperModel(ctx, stepsMap, typeWizard)
	p := tea.NewProgram(wrapper, tea.WithInput(input), tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("testadd error: %w", err)
	}

	// Check the final state
	if w, ok := finalModel.(*testaddWrapperModel); ok {
		if w.currentWizard.IsCancelled() {
			fmt.Fprintf(os.Stderr, "Cancelled\n")
			return nil
		}

		if w.currentWizard.IsComplete() {
			// Print dry-run summary
			state := w.currentWizard.State()
			fmt.Fprintf(os.Stderr, "Would create worktree: %s\n", state.WorktreeName)
			fmt.Fprintf(os.Stderr, "Would create branch: %s\n", state.BranchName)
			if state.BaseBranch != "" {
				fmt.Fprintf(os.Stderr, "Based on: %s\n", state.BaseBranch)
			}
			return nil
		}

		return fmt.Errorf("wizard did not complete or cancel")
	}

	return fmt.Errorf("unexpected model type: %T", finalModel)
}

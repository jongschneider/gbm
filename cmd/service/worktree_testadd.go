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

// runWorktreeTestaddCommand runs the test wizard with mock services.
// This implements a two-wizard pattern:
// 1. First wizard: SelectWorkflowType() - user chooses Feature/Bug/Hotfix/Merge
// 2. Second wizard: GetWorkflowSteps(selectedType) - workflow-specific steps
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

	// Open input once - reuse for both wizards
	input, err := os.Open("/dev/tty")
	if err != nil {
		return fmt.Errorf("failed to open /dev/tty: %w", err)
	}
	defer func() {
		_ = input.Close()
	}()

	// Step 1: Run the workflow type selector
	typeSelector := workflows.SelectWorkflowType()
	typeSelectorStep := tui.Step{
		Name:  "workflow_type_selector",
		Field: typeSelector,
	}
	typeWizard := tui.NewWizard([]tui.Step{typeSelectorStep}, ctx)

	p := tea.NewProgram(typeWizard, tea.WithInput(input), tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("type selector error: %w", err)
	}

	// Check if type selector was completed
	if w, ok := finalModel.(*tui.Wizard); ok {
		if w.IsCancelled() {
			fmt.Fprintf(os.Stderr, "Cancelled\n")
			return nil
		}

		if !w.IsComplete() {
			return fmt.Errorf("type selector did not complete")
		}

		// Get the selected workflow type from the completed wizard state
		selectedType := w.State().WorkflowType
		if selectedType == "" {
			return fmt.Errorf("no workflow type was selected")
		}

		// Step 2: Get the appropriate workflow steps and run the workflow
		workflowSteps, err := workflows.GetWorkflowSteps(selectedType, ctx)
		if err != nil {
			return fmt.Errorf("failed to get workflow steps for %s: %w", selectedType, err)
		}

		// Create a new wizard with the workflow-specific steps and the same context/state
		// This reuses the state from the type selector, preserving any context
		workflowWizard := tui.NewWizard(workflowSteps, ctx)
		workflowWizard.State().WorkflowType = selectedType

		// Run the workflow wizard
		p2 := tea.NewProgram(workflowWizard, tea.WithInput(input), tea.WithAltScreen())
		finalModel2, err := p2.Run()
		if err != nil {
			return fmt.Errorf("workflow error: %w", err)
		}

		// Check the result
		if w2, ok := finalModel2.(*tui.Wizard); ok {
			if w2.IsCancelled() {
				fmt.Fprintf(os.Stderr, "Cancelled\n")
				return nil
			}

			if w2.IsComplete() {
				// Print dry-run summary
				state := w2.State()
				fmt.Fprintf(os.Stderr, "Would create worktree: %s\n", state.WorktreeName)
				fmt.Fprintf(os.Stderr, "Would create branch: %s\n", state.BranchName)
				if state.BaseBranch != "" {
					fmt.Fprintf(os.Stderr, "Based on: %s\n", state.BaseBranch)
				}
				return nil
			}

			// Not complete and not cancelled (shouldn't happen)
			return fmt.Errorf("wizard did not complete or cancel")
		}

		return fmt.Errorf("unexpected model type: %T", finalModel2)
	}

	return fmt.Errorf("unexpected model type: %T", finalModel)
}

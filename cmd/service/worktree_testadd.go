package service

import (
	"fmt"
	"os"

	"gbm/internal/testing"
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

The wizard will display:
- Mock JIRA issues for selection
- Mock Git branches for selection
- Interactive steps for creating a feature worktree

No actual worktrees or branches are created.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWorktreeTestaddCommand(cmd)
		},
	}

	return cmd
}

// runWorktreeTestaddCommand runs the test wizard with mock services.
func runWorktreeTestaddCommand(cmd *cobra.Command) error {
	// Create mock services
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

	mockJira := testing.NewMockJiraService()

	// Create context with mock services
	ctx := tui.NewContext().
		WithDimensions(100, 30).
		WithTheme(tui.DefaultTheme()).
		WithGitService(mockGit).
		WithJiraService(mockJira)

	// Create feature workflow
	wizard := workflows.FeatureWorkflow(ctx)

	// Run wizard with tea.NewProgram using /dev/tty for input
	input, err := os.Open("/dev/tty")
	if err != nil {
		return fmt.Errorf("failed to open /dev/tty: %w", err)
	}
	defer func() {
		_ = input.Close()
	}()

	p := tea.NewProgram(wizard, tea.WithInput(input), tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("wizard error: %w", err)
	}

	// Check the result
	if w, ok := finalModel.(*tui.Wizard); ok {
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

		// Not complete and not cancelled (shouldn't happen)
		return fmt.Errorf("wizard did not complete or cancel")
	}

	return fmt.Errorf("unexpected model type: %T", finalModel)
}

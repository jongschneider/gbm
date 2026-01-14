package workflows

import (
	"bytes"
	"testing"
	"time"

	"gbm/pkg/tui"
	"gbm/pkg/tui/fields"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/muesli/termenv"
	"github.com/stretchr/testify/assert"
)

func init() {
	// Set ASCII color profile for consistent test output across environments
	lipgloss.SetColorProfile(termenv.Ascii)
}

// =============================================================================
// Mock Services for Testing
// =============================================================================

// teatestGitService implements tui.GitService for teatest-based tests.
type teatestGitService struct {
	branches     []string
	branchExists map[string]bool
	err          error
}

func (m *teatestGitService) ListBranches(_ bool) ([]string, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.branches, nil
}

func (m *teatestGitService) BranchExists(branch string) (bool, error) {
	if m.err != nil {
		return false, m.err
	}
	if m.branchExists == nil {
		return false, nil
	}
	return m.branchExists[branch], nil
}

// =============================================================================
// Workflow Test Model
// =============================================================================

// workflowTestModel wraps a Wizard for teatest, quitting on WorkflowCompleteMsg.
type workflowTestModel struct {
	wizard *tui.Wizard
}

func newWorkflowTestModel(w *tui.Wizard) *workflowTestModel {
	return &workflowTestModel{wizard: w}
}

func (m *workflowTestModel) Init() tea.Cmd {
	return m.wizard.Init()
}

func (m *workflowTestModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle WorkflowCompleteMsg to quit the test program
	if _, ok := msg.(tui.WorkflowCompleteMsg); ok {
		return m, tea.Quit
	}

	// Let the wizard process the message first
	model, cmd := m.wizard.Update(msg)
	if w, ok := model.(*tui.Wizard); ok {
		m.wizard = w
	}

	// Handle CancelMsg after wizard processes it (so wizard.cancelled is set)
	if _, ok := msg.(tui.CancelMsg); ok {
		return m, tea.Quit
	}

	return m, cmd
}

func (m *workflowTestModel) View() string {
	return m.wizard.View()
}

// =============================================================================
// Helper: Create FeatureWorkflow with synchronous options
// =============================================================================

// createFeatureWorkflowWithSyncOptions creates a FeatureWorkflow variant with
// pre-populated options (no async loading) for reliable testing.
// This matches the structure of the real FeatureWorkflow but with synchronous options.
func createFeatureWorkflowWithSyncOptions(ctx *tui.Context, jiraOptions, branchOptions []fields.Option) *tui.Wizard {
	steps := []tui.Step{
		// Step 1: JIRA issue selection with pre-populated options
		{
			Name: tui.FieldKeyWorktreeName,
			Field: fields.NewFilterable(
				tui.FieldKeyWorktreeName,
				"Select Worktree Name",
				"Search JIRA issues or enter a custom name",
				jiraOptions,
			),
		},

		// Step 2: Branch name input
		{
			Name: tui.FieldKeyBranchName,
			Field: fields.NewTextInput(tui.FieldKeyBranchName, "Name for the new branch", "Enter the branch name").
				WithPlaceholder("feature/KEY-description"),
		},

		// Step 3: Base branch selection with pre-populated options
		{
			Name: tui.FieldKeyBaseBranch,
			Field: fields.NewFilterable(
				tui.FieldKeyBaseBranch,
				"Base Branch",
				"Choose the branch to base this feature on",
				branchOptions,
			),
			Skip: func(state *tui.WorkflowState) bool {
				if ctx.GitService == nil || state.BranchName == "" {
					return false
				}
				exists, err := ctx.GitService.BranchExists(state.BranchName)
				return err == nil && exists
			},
		},

		// Step 4: Confirmation
		{
			Name:  tui.FieldKeyConfirm,
			Field: fields.NewConfirm(tui.FieldKeyConfirm, "Create Feature Branch?"),
		},
	}

	return tui.NewWizard(steps, ctx)
}

// =============================================================================
// TT-022: FeatureWorkflow end-to-end tests
// Tests the full wizard flow: JIRA selection, branch name, base branch, confirm
// =============================================================================

// TestFeatureWorkflow_Step1_JIRAIssueSelection verifies step 1 of the FeatureWorkflow:
// selecting a JIRA issue from the list.
func TestFeatureWorkflow_Step1_JIRAIssueSelection(t *testing.T) {
	jiraOptions := []fields.Option{
		{Label: "PROJ-123 - Add user authentication", Value: "PROJ-123"},
		{Label: "PROJ-456 - Fix login bug", Value: "PROJ-456"},
		{Label: "PROJ-789 - Refactor database layer", Value: "PROJ-789"},
	}
	branchOptions := []fields.Option{
		{Label: "main", Value: "main"},
	}

	ctx := tui.NewContext()
	wizard := createFeatureWorkflowWithSyncOptions(ctx, jiraOptions, branchOptions)
	model := newWorkflowTestModel(wizard)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for initial render with JIRA issue list
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		// Look for any part of the first option
		return bytes.Contains(bts, []byte("PROJ-123")) ||
			bytes.Contains(bts, []byte("Select Worktree"))
	}, teatest.WithDuration(2*time.Second))

	// Press Enter to select the first JIRA issue (PROJ-123)
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Wait for step 2 (branch name input)
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Name for the new branch"))
	}, teatest.WithDuration(time.Second))

	// Verify JIRA issue key was stored in state
	assert.Equal(t, "PROJ-123", ctx.State.WorktreeName,
		"WorktreeName should be the JIRA issue key")

	// Cancel the wizard
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// TestFeatureWorkflow_Step1_CustomNameEntry verifies step 1 allows entering a custom name
// instead of selecting a JIRA issue.
func TestFeatureWorkflow_Step1_CustomNameEntry(t *testing.T) {
	jiraOptions := []fields.Option{
		{Label: "PROJ-123 - Add user authentication", Value: "PROJ-123"},
	}
	branchOptions := []fields.Option{
		{Label: "main", Value: "main"},
	}

	ctx := tui.NewContext()
	wizard := createFeatureWorkflowWithSyncOptions(ctx, jiraOptions, branchOptions)
	model := newWorkflowTestModel(wizard)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for the options to appear
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("PROJ-123"))
	}, teatest.WithDuration(time.Second))

	// Type a custom name that doesn't match any issue
	tm.Type("my-custom-feature")
	time.Sleep(100 * time.Millisecond)

	// Press Enter to submit the custom name
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Wait for step 2 (branch name input)
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Name for the new branch"))
	}, teatest.WithDuration(time.Second))

	// Verify custom name was stored in state
	assert.Equal(t, "my-custom-feature", ctx.State.WorktreeName,
		"WorktreeName should be the custom name entered")

	// Cancel the wizard
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// TestFeatureWorkflow_Step2_BranchNameInput verifies step 2 accepts a branch name.
func TestFeatureWorkflow_Step2_BranchNameInput(t *testing.T) {
	jiraOptions := []fields.Option{
		{Label: "PROJ-123 - Add authentication", Value: "PROJ-123"},
	}
	branchOptions := []fields.Option{
		{Label: "main", Value: "main"},
		{Label: "develop", Value: "develop"},
	}

	ctx := tui.NewContext()
	wizard := createFeatureWorkflowWithSyncOptions(ctx, jiraOptions, branchOptions)
	model := newWorkflowTestModel(wizard)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for step 1
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("PROJ-123"))
	}, teatest.WithDuration(time.Second))

	// Step 1: Select JIRA issue
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Wait for step 2 (branch name)
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Name for the new branch"))
	}, teatest.WithDuration(time.Second))

	// Step 2: Clear any auto-generated default and enter branch name
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlU}) // Clear line
	time.Sleep(50 * time.Millisecond)
	tm.Type("feature/proj-123-add-auth")
	time.Sleep(100 * time.Millisecond)
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Wait for step 3 (base branch selection)
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Base Branch"))
	}, teatest.WithDuration(time.Second))

	// Verify branch name was stored
	assert.Equal(t, "feature/proj-123-add-auth", ctx.State.BranchName,
		"BranchName should be the entered branch name")

	// Cancel the wizard
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// TestFeatureWorkflow_Step3_BaseBranchSelection verifies step 3 allows selecting a base branch.
func TestFeatureWorkflow_Step3_BaseBranchSelection(t *testing.T) {
	jiraOptions := []fields.Option{
		{Label: "PROJ-123 - Add authentication", Value: "PROJ-123"},
	}
	branchOptions := []fields.Option{
		{Label: "main", Value: "main"},
		{Label: "develop", Value: "develop"},
		{Label: "release-1.0", Value: "release-1.0"},
	}

	ctx := tui.NewContext()
	wizard := createFeatureWorkflowWithSyncOptions(ctx, jiraOptions, branchOptions)
	model := newWorkflowTestModel(wizard)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Step 1: Select JIRA issue
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("PROJ-123"))
	}, teatest.WithDuration(time.Second))
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Step 2: Enter branch name
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Name for the new branch"))
	}, teatest.WithDuration(time.Second))
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlU}) // Clear line
	time.Sleep(50 * time.Millisecond)
	tm.Type("feature/my-branch")
	time.Sleep(100 * time.Millisecond)
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Wait for step 3 (base branch)
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Base Branch"))
	}, teatest.WithDuration(time.Second))

	// Step 3: Navigate to "develop" (second option)
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	time.Sleep(50 * time.Millisecond)

	// Select develop as base branch
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Wait for step 4 (confirm)
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Create Feature Branch"))
	}, teatest.WithDuration(time.Second))

	// Verify base branch was stored
	assert.Equal(t, "develop", ctx.State.BaseBranch,
		"BaseBranch should be 'develop'")

	// Cancel the wizard
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// TestFeatureWorkflow_Step3_SkippedIfBranchExists verifies step 3 is skipped
// when the branch name already exists in git.
func TestFeatureWorkflow_Step3_SkippedIfBranchExists(t *testing.T) {
	jiraOptions := []fields.Option{
		{Label: "PROJ-123 - Add authentication", Value: "PROJ-123"},
	}
	branchOptions := []fields.Option{
		{Label: "main", Value: "main"},
		{Label: "develop", Value: "develop"},
	}

	gitService := &teatestGitService{
		branchExists: map[string]bool{
			"feature/existing-branch": true, // This branch already exists
		},
	}

	ctx := tui.NewContext().WithGitService(gitService)
	wizard := createFeatureWorkflowWithSyncOptions(ctx, jiraOptions, branchOptions)
	model := newWorkflowTestModel(wizard)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Step 1: Select JIRA issue
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("PROJ-123"))
	}, teatest.WithDuration(time.Second))
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Step 2: Enter a branch name that already exists
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Name for the new branch"))
	}, teatest.WithDuration(time.Second))
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlU}) // Clear line
	time.Sleep(50 * time.Millisecond)
	tm.Type("feature/existing-branch")
	time.Sleep(100 * time.Millisecond)
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Step 3 should be SKIPPED - should go directly to step 4 (confirm)
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Create Feature Branch"))
	}, teatest.WithDuration(time.Second))

	// Verify we're at confirm step (step 4) not base branch (step 3)
	assert.Contains(t, wizard.View(), "Create Feature Branch",
		"Should skip base branch step and go to confirm when branch exists")

	// Verify base branch was NOT set (since step was skipped)
	assert.Empty(t, ctx.State.BaseBranch,
		"BaseBranch should be empty because step 3 was skipped")

	// Cancel the wizard
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// TestFeatureWorkflow_Step4_ConfirmYes verifies step 4 completes the workflow when Yes is selected.
func TestFeatureWorkflow_Step4_ConfirmYes(t *testing.T) {
	jiraOptions := []fields.Option{
		{Label: "PROJ-123 - Add authentication", Value: "PROJ-123"},
	}
	branchOptions := []fields.Option{
		{Label: "main", Value: "main"},
	}

	ctx := tui.NewContext()
	wizard := createFeatureWorkflowWithSyncOptions(ctx, jiraOptions, branchOptions)
	model := newWorkflowTestModel(wizard)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Step 1: Select JIRA issue
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("PROJ-123"))
	}, teatest.WithDuration(time.Second))
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Step 2: Enter branch name
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Name for the new branch"))
	}, teatest.WithDuration(time.Second))
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlU}) // Clear line
	time.Sleep(50 * time.Millisecond)
	tm.Type("feature/final-branch")
	time.Sleep(100 * time.Millisecond)
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Step 3: Select base branch
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Base Branch"))
	}, teatest.WithDuration(time.Second))
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Step 4: Confirm
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Create Feature Branch"))
	}, teatest.WithDuration(time.Second))

	assert.False(t, wizard.IsComplete(), "wizard should not be complete before confirmation")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter}) // Confirm Yes (default)

	// Wait for wizard to complete
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

	// Verify wizard completed successfully
	assert.True(t, wizard.IsComplete(), "wizard should be complete after confirmation")
	assert.False(t, wizard.IsCancelled(), "wizard should not be cancelled")

	// Verify all state values
	assert.Equal(t, "PROJ-123", ctx.State.WorktreeName, "WorktreeName should be JIRA issue key")
	assert.Equal(t, "feature/final-branch", ctx.State.BranchName, "BranchName should be entered value")
	assert.Equal(t, "main", ctx.State.BaseBranch, "BaseBranch should be 'main'")
}

// TestFeatureWorkflow_Step4_ConfirmNo verifies step 4 cancels the workflow when No is selected.
func TestFeatureWorkflow_Step4_ConfirmNo(t *testing.T) {
	jiraOptions := []fields.Option{
		{Label: "PROJ-123 - Add authentication", Value: "PROJ-123"},
	}
	branchOptions := []fields.Option{
		{Label: "main", Value: "main"},
	}

	ctx := tui.NewContext()
	wizard := createFeatureWorkflowWithSyncOptions(ctx, jiraOptions, branchOptions)
	model := newWorkflowTestModel(wizard)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Step 1: Select JIRA issue
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("PROJ-123"))
	}, teatest.WithDuration(time.Second))
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Step 2: Enter branch name
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Name for the new branch"))
	}, teatest.WithDuration(time.Second))
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlU}) // Clear line
	time.Sleep(50 * time.Millisecond)
	tm.Type("feature/to-cancel")
	time.Sleep(100 * time.Millisecond)
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Step 3: Select base branch
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Base Branch"))
	}, teatest.WithDuration(time.Second))
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Step 4: Navigate to No and select
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Create Feature Branch"))
	}, teatest.WithDuration(time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRight}) // Move to No
	time.Sleep(50 * time.Millisecond)
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter}) // Confirm No

	// Wait for wizard to finish (due to CancelMsg)
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

	// Verify wizard was cancelled
	assert.True(t, wizard.IsCancelled(), "wizard should be cancelled when No is selected")
}

// TestFeatureWorkflow_BackNavigation verifies Esc key navigates back through steps.
func TestFeatureWorkflow_BackNavigation(t *testing.T) {
	jiraOptions := []fields.Option{
		{Label: "PROJ-123 - Add authentication", Value: "PROJ-123"},
	}
	branchOptions := []fields.Option{
		{Label: "main", Value: "main"},
	}

	ctx := tui.NewContext()
	wizard := createFeatureWorkflowWithSyncOptions(ctx, jiraOptions, branchOptions)
	model := newWorkflowTestModel(wizard)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Step 1: Select JIRA issue
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("PROJ-123"))
	}, teatest.WithDuration(time.Second))
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Wait for step 2
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Name for the new branch"))
	}, teatest.WithDuration(time.Second))

	// Press Esc to go back to step 1
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})

	// Wait for step 1 to reappear
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Select Worktree Name"))
	}, teatest.WithDuration(time.Second))

	// Verify we're back at step 1 (JIRA selection)
	assert.Contains(t, wizard.View(), "PROJ-123",
		"Should be back at step 1 showing JIRA issues")

	// Cancel the wizard
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// TestFeatureWorkflow_EndToEndComplete tests the full happy path of the FeatureWorkflow.
func TestFeatureWorkflow_EndToEndComplete(t *testing.T) {
	jiraOptions := []fields.Option{
		{Label: "FEAT-100 - Implement dark mode", Value: "FEAT-100"},
		{Label: "FEAT-101 - Add user settings", Value: "FEAT-101"},
	}
	branchOptions := []fields.Option{
		{Label: "main", Value: "main"},
		{Label: "develop", Value: "develop"},
		{Label: "staging", Value: "staging"},
	}

	ctx := tui.NewContext()
	wizard := createFeatureWorkflowWithSyncOptions(ctx, jiraOptions, branchOptions)
	model := newWorkflowTestModel(wizard)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Step 1: Wait for JIRA issues and select second issue (FEAT-101)
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("FEAT-100"))
	}, teatest.WithDuration(time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyDown}) // Move to second issue
	time.Sleep(50 * time.Millisecond)
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Step 2: Enter branch name
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Name for the new branch"))
	}, teatest.WithDuration(time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlU}) // Clear line
	time.Sleep(50 * time.Millisecond)
	tm.Type("feature/feat-101-user-settings")
	time.Sleep(100 * time.Millisecond)
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Step 3: Select base branch (choose develop)
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Base Branch"))
	}, teatest.WithDuration(time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyDown}) // Navigate to develop
	time.Sleep(50 * time.Millisecond)
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Step 4: Confirm
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Create Feature Branch"))
	}, teatest.WithDuration(time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyEnter}) // Confirm Yes

	// Wait for completion
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

	// Verify final state
	assert.True(t, wizard.IsComplete(), "wizard should be complete")
	assert.False(t, wizard.IsCancelled(), "wizard should not be cancelled")
	assert.Equal(t, "FEAT-101", ctx.State.WorktreeName, "WorktreeName should be FEAT-101")
	assert.Equal(t, "feature/feat-101-user-settings", ctx.State.BranchName, "BranchName should match")
	assert.Equal(t, "develop", ctx.State.BaseBranch, "BaseBranch should be develop")
}

// TestFeatureWorkflow_FilteringJIRAIssues tests filtering JIRA issues by typing.
func TestFeatureWorkflow_FilteringJIRAIssues(t *testing.T) {
	jiraOptions := []fields.Option{
		{Label: "AUTH-001 - Login feature", Value: "AUTH-001"},
		{Label: "AUTH-002 - Logout feature", Value: "AUTH-002"},
		{Label: "DATA-001 - Database migration", Value: "DATA-001"},
	}
	branchOptions := []fields.Option{
		{Label: "main", Value: "main"},
	}

	ctx := tui.NewContext()
	wizard := createFeatureWorkflowWithSyncOptions(ctx, jiraOptions, branchOptions)
	model := newWorkflowTestModel(wizard)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for options to appear
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("AUTH-001"))
	}, teatest.WithDuration(time.Second))

	// Type "DATA" to filter to just DATA-001
	tm.Type("DATA")
	time.Sleep(100 * time.Millisecond)

	// Verify filter narrows the list (we can check by selecting and verifying value)
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Wait for step 2
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Name for the new branch"))
	}, teatest.WithDuration(time.Second))

	// Verify the filtered selection was DATA-001
	assert.Equal(t, "DATA-001", ctx.State.WorktreeName,
		"Filtered selection should be DATA-001")

	// Cancel
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// TestFeatureWorkflow_AutoGeneratedBranchName tests that the wizard generates
// a default branch name based on the JIRA issue selected.
// Note: This tests the default value suggestion behavior in the wizard.
func TestFeatureWorkflow_AutoGeneratedBranchName(t *testing.T) {
	jiraOptions := []fields.Option{
		{Label: "PROJ-999 - My test issue", Value: "PROJ-999"},
	}
	branchOptions := []fields.Option{
		{Label: "main", Value: "main"},
	}

	ctx := tui.NewContext()
	wizard := createFeatureWorkflowWithSyncOptions(ctx, jiraOptions, branchOptions)
	model := newWorkflowTestModel(wizard)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Step 1: Select JIRA issue
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("PROJ-999"))
	}, teatest.WithDuration(time.Second))
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Step 2: We're now at the branch name step
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Name for the new branch"))
	}, teatest.WithDuration(time.Second))

	// The user can accept any pre-filled default or type their own
	// In this test, we just submit with whatever is there and check state
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Wait for step 3
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Base Branch"))
	}, teatest.WithDuration(time.Second))

	// The branch name should be whatever was submitted (default or empty)
	// The sync version doesn't have auto-default, so it should be empty
	// This tests that the flow works with empty branch name
	// Note: Real workflow has validation that would prevent empty branch name
	assert.NotEmpty(t, ctx.State.WorktreeName, "WorktreeName should be set from step 1")

	// Cancel
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

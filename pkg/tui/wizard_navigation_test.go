package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

// mockField is a simple test field that can be configured for testing.
type mockField struct {
	value       any
	err         error
	theme       *Theme
	key         string
	title       string
	width       int
	height      int
	complete    bool
	cancelled   bool
	focused     bool
	nextOnEnter bool
}

func newMockField(key, title string) *mockField {
	return &mockField{
		key:         key,
		title:       title,
		nextOnEnter: true,
		width:       80,
		height:      24,
	}
}

func (m *mockField) Init() tea.Cmd { return nil }

func (m *mockField) Update(msg tea.Msg) (Field, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.Type == tea.KeyEnter && m.nextOnEnter {
			m.complete = true
			return m, func() tea.Msg { return NextStepMsg{} }
		}
	}
	return m, nil
}

func (m *mockField) View() string             { return m.title }
func (m *mockField) Focus() tea.Cmd           { m.focused = true; return nil }
func (m *mockField) Blur() tea.Cmd            { m.focused = false; return nil }
func (m *mockField) IsComplete() bool         { return m.complete }
func (m *mockField) IsCancelled() bool        { return m.cancelled }
func (m *mockField) Error() error             { return m.err }
func (m *mockField) Skip() bool               { return false }
func (m *mockField) WithTheme(t *Theme) Field { m.theme = t; return m }
func (m *mockField) WithWidth(w int) Field    { m.width = w; return m }
func (m *mockField) WithHeight(h int) Field   { m.height = h; return m }
func (m *mockField) GetKey() string           { return m.key }
func (m *mockField) GetValue() any            { return m.value }

func (m *mockField) SetValue(v any) { m.value = v }

// TestWizardNavigation_OneStepAtATime verifies that only the current step is rendered.
func TestWizardNavigation_OneStepAtATime(t *testing.T) {
	step1 := Step{
		Name:  "Step 1",
		Field: newMockField("test1", "Choose option"),
	}
	step2 := Step{
		Name:  "Step 2",
		Field: newMockField("test2", "Choose another"),
	}
	step3 := Step{
		Name:  "Step 3",
		Field: newMockField("test3", "Choose third"),
	}

	wizard := NewWizard([]Step{step1, step2, step3}, NewContext())
	wizard.Init()

	// Step 1 should be visible
	view := wizard.View()
	assert.NotEmpty(t, view)
	assert.Contains(t, view, "Choose option")

	// Navigate to step 2
	wizard.Update(NextStepMsg{})
	view = wizard.View()
	assert.Contains(t, view, "Choose another")
	assert.NotContains(t, view, "Choose option") // Previous step hidden

	// Navigate to step 3
	wizard.Update(NextStepMsg{})
	view = wizard.View()
	assert.Contains(t, view, "Choose third")
	assert.NotContains(t, view, "Choose another") // Previous step hidden
}

// TestWizardNavigation_AdvanceOnEnter verifies advancing to next step on field completion.
func TestWizardNavigation_AdvanceOnEnter(t *testing.T) {
	step1 := Step{
		Name:  "Step 1",
		Field: newMockField("test1", "Choose"),
	}
	step2 := Step{
		Name:  "Step 2",
		Field: newMockField("test2", "Next"),
	}

	wizard := NewWizard([]Step{step1, step2}, NewContext())
	wizard.Init()

	// Press Enter on step 1 and execute the command
	oldIdx := wizard.current
	_, cmd := wizard.Update(tea.KeyMsg{Type: tea.KeyEnter})
	msg := cmd()
	wizard.Update(msg)

	// Should have advanced to step 2
	assert.Greater(t, wizard.current, oldIdx)
	assert.Equal(t, 1, wizard.current)
}

// TestWizardNavigation_StateUpdatedOnAdvance verifies WorkflowState is updated when advancing.
func TestWizardNavigation_StateUpdatedOnAdvance(t *testing.T) {
	field1 := newMockField("worktree_name", "Choose")
	field1.value = "feature-1"

	step1 := Step{
		Name:  "Step 1",
		Field: field1,
	}
	step2 := Step{
		Name:  "Step 2",
		Field: newMockField("branch_name", "Branch"),
	}

	ctx := NewContext()
	wizard := NewWizard([]Step{step1, step2}, ctx)
	wizard.Init()

	// Complete step 1 by pressing enter (which generates NextStepMsg command)
	_, cmd := wizard.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Execute the command which returns NextStepMsg
	msg := cmd()

	// Send the NextStepMsg to wizard (this triggers storeFieldValue)
	wizard.Update(msg)

	// Value should be stored in state
	assert.Equal(t, "feature-1", ctx.State.WorktreeName)
}

// TestWizardNavigation_BackWithESC verifies ESC navigates back to previous step.
func TestWizardNavigation_BackWithESC(t *testing.T) {
	step1 := Step{
		Name:  "Step 1",
		Field: newMockField("test1", "Choose"),
	}
	step2 := Step{
		Name:  "Step 2",
		Field: newMockField("test2", "Next"),
	}

	wizard := NewWizard([]Step{step1, step2}, NewContext())
	wizard.Init()

	// Advance to step 2 and execute the command
	_, cmd := wizard.Update(tea.KeyMsg{Type: tea.KeyEnter})
	msg := cmd()
	wizard.Update(msg)
	assert.Equal(t, 1, wizard.current)

	// Press ESC to go back
	wizard.Update(tea.KeyMsg{Type: tea.KeyEsc})
	assert.Equal(t, 0, wizard.current)
}

// TestWizardNavigation_ESCOnFirstStep does nothing.
func TestWizardNavigation_ESCOnFirstStep(t *testing.T) {
	step1 := Step{
		Name:  "Step 1",
		Field: newMockField("test1", "Choose"),
	}

	wizard := NewWizard([]Step{step1}, NewContext())
	wizard.Init()

	// Press ESC on first step
	wizard.Update(tea.KeyMsg{Type: tea.KeyEsc})

	// Should still be on step 1
	assert.Equal(t, 0, wizard.current)
	assert.False(t, wizard.IsComplete())
	assert.False(t, wizard.IsCancelled())
}

// TestWizardNavigation_CancelWithCtrlC verifies Ctrl+C exits the wizard.
func TestWizardNavigation_CancelWithCtrlC(t *testing.T) {
	step1 := Step{
		Name:  "Step 1",
		Field: newMockField("test1", "Choose"),
	}

	wizard := NewWizard([]Step{step1}, NewContext())
	wizard.Init()

	assert.False(t, wizard.IsCancelled())

	wizard.Update(tea.KeyMsg{Type: tea.KeyCtrlC})

	assert.True(t, wizard.IsCancelled())
}

// TestWizardNavigation_SkipLogic verifies steps with Skip functions are skipped.
func TestWizardNavigation_SkipLogic(t *testing.T) {
	step1 := Step{
		Name:  "Step 1",
		Field: newMockField("test1", "Choose"),
	}
	step2 := Step{
		Name:  "Step 2 (skipped)",
		Field: newMockField("test2", "Skipped"),
		Skip: func(state *WorkflowState) bool {
			return true // Always skip
		},
	}
	step3 := Step{
		Name:  "Step 3",
		Field: newMockField("test3", "After skip"),
	}

	wizard := NewWizard([]Step{step1, step2, step3}, NewContext())
	wizard.Init()
	assert.Equal(t, 0, wizard.current)

	// Advance from step 1 and execute the command
	_, cmd := wizard.Update(tea.KeyMsg{Type: tea.KeyEnter})
	msg := cmd()
	wizard.Update(msg)

	// Should skip step 2 and land on step 3
	assert.Equal(t, 2, wizard.current)
}

// TestWizardNavigation_SkipFunctionCanAccessState verifies Skip function accesses WorkflowState.
func TestWizardNavigation_SkipFunctionCanAccessState(t *testing.T) {
	field1 := newMockField("worktree_name", "Choose")
	field1.value = "existing-branch"

	step1 := Step{
		Name:  "Step 1",
		Field: field1,
	}
	step2 := Step{
		Name:  "Step 2 (conditional skip)",
		Field: newMockField("base_branch", "Base"),
		Skip: func(state *WorkflowState) bool {
			// Skip if branch name matches a pattern
			return state.WorktreeName == "existing-branch"
		},
	}

	ctx := NewContext()
	wizard := NewWizard([]Step{step1, step2}, ctx)
	wizard.Init()

	// Select "existing-branch" and execute the command
	_, cmd := wizard.Update(tea.KeyMsg{Type: tea.KeyEnter})
	msg := cmd()
	wizard.Update(msg)

	// Step 2 should be skipped because of state value
	assert.True(t, wizard.IsComplete())
}

// TestWizardNavigation_SkipLogicAppliesBackward verifies Skip logic applies to backward navigation.
func TestWizardNavigation_SkipLogicAppliesBackward(t *testing.T) {
	step1 := Step{
		Name:  "Step 1",
		Field: newMockField("test1", "Choose"),
	}
	step2 := Step{
		Name:  "Step 2 (skipped)",
		Field: newMockField("test2", "Skipped"),
		Skip: func(state *WorkflowState) bool {
			return true // Always skip
		},
	}
	step3 := Step{
		Name:  "Step 3",
		Field: newMockField("test3", "After skip"),
	}

	wizard := NewWizard([]Step{step1, step2, step3}, NewContext())
	wizard.Init()

	// Advance to step 3 (step 2 is skipped) and execute the command
	_, cmd := wizard.Update(tea.KeyMsg{Type: tea.KeyEnter})
	msg := cmd()
	wizard.Update(msg)
	assert.Equal(t, 2, wizard.current)

	// Go back from step 3
	wizard.Update(tea.KeyMsg{Type: tea.KeyEsc})

	// Should skip step 2 and land on step 1
	assert.Equal(t, 0, wizard.current)
}

// TestWizardNavigation_CompleteOnLastStep verifies wizard completes when last step is finished.
func TestWizardNavigation_CompleteOnLastStep(t *testing.T) {
	step1 := Step{
		Name:  "Step 1",
		Field: newMockField("test1", "Choose"),
	}
	step2 := Step{
		Name:  "Step 2 (last)",
		Field: newMockField("test2", "Final"),
	}

	wizard := NewWizard([]Step{step1, step2}, NewContext())
	wizard.Init()

	// Advance to step 2 by pressing enter and executing the command
	_, cmd := wizard.Update(tea.KeyMsg{Type: tea.KeyEnter})
	msg := cmd()
	wizard.Update(msg)
	assert.False(t, wizard.IsComplete())

	// Complete step 2 by pressing enter and executing the command
	_, cmd = wizard.Update(tea.KeyMsg{Type: tea.KeyEnter})
	msg = cmd()
	wizard.Update(msg)
	assert.True(t, wizard.IsComplete())
}

// TestWizardNavigation_IsCompleteReturnsFalseUntilAllStepsDone verifies incremental completion.
func TestWizardNavigation_IsCompleteReturnsFalseUntilAllStepsDone(t *testing.T) {
	step1 := Step{
		Name:  "Step 1",
		Field: newMockField("test1", "Choose"),
	}
	step2 := Step{
		Name:  "Step 2",
		Field: newMockField("test2", "Choose"),
	}

	wizard := NewWizard([]Step{step1, step2}, NewContext())
	wizard.Init()

	assert.False(t, wizard.IsComplete())

	// Step 1: press enter and execute the command
	_, cmd := wizard.Update(tea.KeyMsg{Type: tea.KeyEnter})
	msg := cmd()
	wizard.Update(msg)
	assert.False(t, wizard.IsComplete())

	// Step 2: press enter and execute the command
	_, cmd = wizard.Update(tea.KeyMsg{Type: tea.KeyEnter})
	msg = cmd()
	wizard.Update(msg)
	assert.True(t, wizard.IsComplete())
}

// TestWizardNavigation_StatePopulatedWithAllValues verifies all values are collected.
func TestWizardNavigation_StatePopulatedWithAllValues(t *testing.T) {
	field1 := newMockField("worktree_name", "Choose")
	field1.value = "my-feature"

	field2 := newMockField("branch_name", "Choose")
	field2.value = "feature/my-feature"

	step1 := Step{
		Name:  "Step 1",
		Field: field1,
	}
	step2 := Step{
		Name:  "Step 2",
		Field: field2,
	}

	ctx := NewContext()
	wizard := NewWizard([]Step{step1, step2}, ctx)
	wizard.Init()

	// Step 1: press enter, get cmd, execute it to generate NextStepMsg
	_, cmd1 := wizard.Update(tea.KeyMsg{Type: tea.KeyEnter})
	msg1 := cmd1()
	wizard.Update(msg1)

	// Step 2: press enter, get cmd, execute it to generate NextStepMsg
	_, cmd2 := wizard.Update(tea.KeyMsg{Type: tea.KeyEnter})
	msg2 := cmd2()
	wizard.Update(msg2)

	assert.True(t, wizard.IsComplete())
	assert.Equal(t, "my-feature", ctx.State.WorktreeName)
	assert.Equal(t, "feature/my-feature", ctx.State.BranchName)
}

// TestWizardNavigation_AllStepsSkippedCompletesImmediately verifies all-skipped edge case.
func TestWizardNavigation_AllStepsSkippedCompletesImmediately(t *testing.T) {
	step1 := Step{
		Name:  "Step 1",
		Field: newMockField("test1", "Choose"),
		Skip: func(state *WorkflowState) bool {
			return true // Always skip
		},
	}

	wizard := NewWizard([]Step{step1}, NewContext())
	wizard.Init()

	// Should complete immediately if all steps are skipped
	assert.True(t, wizard.IsComplete())
}

// TestWizardNavigation_WindowResizeUpdatesContext verifies terminal resize updates dimensions.
func TestWizardNavigation_WindowResizeUpdatesContext(t *testing.T) {
	step1 := Step{
		Name:  "Step 1",
		Field: newMockField("test1", "Choose"),
	}

	ctx := NewContext()
	ctx.Width = 80
	ctx.Height = 24

	wizard := NewWizard([]Step{step1}, ctx)
	wizard.Init()

	// Simulate resize
	wizard.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	assert.Equal(t, 120, ctx.Width)
	assert.Equal(t, 40, ctx.Height)
}

// TestWizardNavigation_Integration tests a complex multi-step scenario.
func TestWizardNavigation_Integration(t *testing.T) {
	step1 := Step{
		Name:  "Workflow Type",
		Field: newMockField(FieldKeyWorkflowType, "Choose type"),
	}
	step2 := Step{
		Name:  "Branch Name",
		Field: newMockField(FieldKeyBranchName, "Branch"),
	}
	step3 := Step{
		Name:  "Base Branch (conditional)",
		Field: newMockField(FieldKeyBaseBranch, "Base"),
		Skip: func(state *WorkflowState) bool {
			// Skip if workflow type is feature
			return state.WorkflowType == WorkflowTypeFeature
		},
	}

	ctx := NewContext()
	wizard := NewWizard([]Step{step1, step2, step3}, ctx)
	wizard.Init()

	// Set workflow type to "feature"
	field1 := step1.Field.(*mockField)
	field1.value = WorkflowTypeFeature

	// Select "feature" and advance
	_, cmd := wizard.Update(tea.KeyMsg{Type: tea.KeyEnter})
	msg := cmd()
	wizard.Update(msg)
	assert.Equal(t, 1, wizard.current)
	assert.Equal(t, WorkflowTypeFeature, ctx.State.WorkflowType)

	// Select branch and advance
	_, cmd = wizard.Update(tea.KeyMsg{Type: tea.KeyEnter})
	msg = cmd()
	wizard.Update(msg)
	// Should skip step3 because it's a feature - so we're complete after just 2 steps
	assert.True(t, wizard.IsComplete())
}

// TestWizardBranchNamePreFill_WithJiraIssue tests that branch names are pre-filled from JIRA issues.
func TestWizardBranchNamePreFill_WithJiraIssue(t *testing.T) {
	// Create a context with mock JIRA service
	ctx := NewContext()
	ctx.JiraService = &mockJiraService{
		issues: []JiraIssue{
			{Key: "PROJ-123", Summary: "Fix bug in widget"},
			{Key: "PROJ-456", Summary: "Add new feature"},
		},
	}

	// Create a feature-like workflow: worktree_name -> branch_name -> confirm
	branchField := &mockBranchNameField{key: "branch_name", title: "Branch Name"}
	steps := []Step{
		{
			Name:  "worktree_name",
			Field: newMockField("worktree_name", "Select Issue"),
		},
		{
			Name:  "branch_name",
			Field: branchField,
		},
		{
			Name:  "confirm",
			Field: newMockField("confirm", "Confirm"),
		},
	}

	wizard := NewWizard(steps, ctx)
	wizard.Init()

	// Step 1: Select JIRA issue (PROJ-123)
	assert.Empty(t, branchField.defaultValue, "Initially no default")

	// Advance to branch_name step
	steps[0].Field.(*mockField).value = "PROJ-123"
	_, cmd := wizard.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		msg := cmd()
		wizard.Update(msg)
	}

	// Verify we're at the branch_name step
	assert.Equal(t, 1, wizard.current)

	// The branch name field should now have a default value
	// Since the field is updated via WithDefault, we check the wizard's current field
	currentField := wizard.currentField().(*mockBranchNameField)
	assert.NotEmpty(t, currentField.defaultValue, "Default should be set after transitioning to branch_name step")
	assert.Contains(t, currentField.defaultValue, "feature/PROJ-123", "Branch name should contain issue key")
	assert.Contains(t, currentField.defaultValue, "fix_bug", "Branch name should contain slugified summary")
}

// TestWizardBranchNamePreFill_WithCustomName tests that branch names are pre-filled from custom names.
func TestWizardBranchNamePreFill_WithCustomName(t *testing.T) {
	ctx := NewContext()

	// Create a feature-like workflow: worktree_name -> branch_name -> confirm
	branchField := &mockBranchNameField{key: "branch_name", title: "Branch Name"}
	steps := []Step{
		{
			Name:  "worktree_name",
			Field: newMockField("worktree_name", "Select Issue"),
		},
		{
			Name:  "branch_name",
			Field: branchField,
		},
		{
			Name:  "confirm",
			Field: newMockField("confirm", "Confirm"),
		},
	}

	wizard := NewWizard(steps, ctx)
	wizard.Init()

	// Step 1: Enter custom worktree name (not a JIRA issue)
	assert.Empty(t, branchField.defaultValue, "Initially no default")

	// Advance to branch_name step
	steps[0].Field.(*mockField).value = "my-feature"
	_, cmd := wizard.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		msg := cmd()
		wizard.Update(msg)
	}

	// Verify we're at the branch_name step
	assert.Equal(t, 1, wizard.current)

	// The branch name field should now have a default value based on custom name
	currentField := wizard.currentField().(*mockBranchNameField)
	assert.NotEmpty(t, currentField.defaultValue, "Default should be set from custom name")
	assert.Contains(t, currentField.defaultValue, "feature/my_feature", "Branch name should be based on custom name")
}

// TestWizardBranchNamePreFill_HotfixWorkflow tests that hotfix workflows use hotfix/ prefix.
func TestWizardBranchNamePreFill_HotfixWorkflow(t *testing.T) {
	ctx := NewContext()
	ctx.JiraService = &mockJiraService{
		issues: []JiraIssue{
			{Key: "PROJ-789", Summary: "Security patch for vulnerability"},
		},
	}

	// Create a hotfix-like workflow: worktree_name -> base_branch -> branch_name -> confirm
	// Note: base_branch comes BEFORE branch_name in hotfix workflow
	branchField := &mockBranchNameField{key: "branch_name", title: "Branch Name"}
	steps := []Step{
		{
			Name:  "worktree_name",
			Field: newMockField("worktree_name", "Select Issue"),
		},
		{
			Name:  "base_branch",
			Field: newMockField("base_branch", "Base Branch"),
		},
		{
			Name:  "branch_name",
			Field: branchField,
		},
		{
			Name:  "confirm",
			Field: newMockField("confirm", "Confirm"),
		},
	}

	wizard := NewWizard(steps, ctx)
	wizard.Init()

	// Step 1: Select JIRA issue
	steps[0].Field.(*mockField).value = "PROJ-789"
	_, cmd := wizard.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		msg := cmd()
		wizard.Update(msg)
	}

	// Step 2: Select base branch
	steps[1].Field.(*mockField).value = "main"
	_, cmd = wizard.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		msg := cmd()
		wizard.Update(msg)
	}

	// Step 3: Verify branch_name field has hotfix/ prefix
	assert.Equal(t, 2, wizard.current)
	currentField := wizard.currentField().(*mockBranchNameField)
	assert.NotEmpty(t, currentField.defaultValue, "Default should be set for hotfix")
	assert.Contains(t, currentField.defaultValue, "hotfix/PROJ-789", "Hotfix branch should have hotfix/ prefix")
	assert.Contains(t, currentField.defaultValue, "security_patch", "Branch name should contain slugified summary")
}

// TestWizardBranchNamePreFill_BugWorkflow tests that bug workflows use bug/ prefix.
func TestWizardBranchNamePreFill_BugWorkflow(t *testing.T) {
	ctx := NewContext()
	ctx.JiraService = &mockJiraService{
		issues: []JiraIssue{
			{Key: "BUG-456", Summary: "Fix memory leak in database"},
		},
	}

	// Create a bug-like workflow: worktree_name -> branch_name -> base_branch -> confirm
	// Note: branch_name comes BEFORE base_branch (like feature, unlike hotfix)
	branchField := &mockBranchNameField{key: "branch_name", title: "Branch Name"}
	steps := []Step{
		{
			Name:  "worktree_name",
			Field: newMockField("worktree_name", "Select Issue"),
		},
		{
			Name:  "branch_name",
			Field: branchField,
		},
		{
			Name:  "base_branch",
			Field: newMockField("base_branch", "Base Branch"),
		},
		{
			Name:  "confirm",
			Field: newMockField("confirm", "Confirm"),
		},
	}

	wizard := NewWizard(steps, ctx)
	// Set workflow type to "bug" to enable bug/ prefix
	wizard.ctx.State.WorkflowType = "bug"
	wizard.Init()

	// Step 1: Select JIRA issue
	steps[0].Field.(*mockField).value = "BUG-456"
	_, cmd := wizard.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		msg := cmd()
		wizard.Update(msg)
	}

	// Step 2: Verify branch_name field has bug/ prefix
	assert.Equal(t, 1, wizard.current)
	currentField := wizard.currentField().(*mockBranchNameField)
	assert.NotEmpty(t, currentField.defaultValue, "Default should be set for bug")
	assert.Contains(t, currentField.defaultValue, "bug/BUG-456", "Bug branch should have bug/ prefix")
	assert.Contains(t, currentField.defaultValue, "fix_memory_leak", "Branch name should contain slugified summary")
}

// TestWizardBranchNamePreFill_BugWorkflow_CustomName tests that bug workflows with custom names use bug/ prefix.
func TestWizardBranchNamePreFill_BugWorkflow_CustomName(t *testing.T) {
	ctx := NewContext()

	// Create a bug-like workflow: worktree_name -> branch_name -> base_branch -> confirm
	branchField := &mockBranchNameField{key: "branch_name", title: "Branch Name"}
	steps := []Step{
		{
			Name:  "worktree_name",
			Field: newMockField("worktree_name", "Select Issue"),
		},
		{
			Name:  "branch_name",
			Field: branchField,
		},
		{
			Name:  "base_branch",
			Field: newMockField("base_branch", "Base Branch"),
		},
		{
			Name:  "confirm",
			Field: newMockField("confirm", "Confirm"),
		},
	}

	wizard := NewWizard(steps, ctx)
	// Set workflow type to "bug" to enable bug/ prefix
	wizard.ctx.State.WorkflowType = "bug"
	wizard.Init()

	// Step 1: Enter custom bug name (not a JIRA issue)
	assert.Empty(t, branchField.defaultValue, "Initially no default")

	// Advance to branch_name step
	steps[0].Field.(*mockField).value = "fix-crash"
	_, cmd := wizard.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		msg := cmd()
		wizard.Update(msg)
	}

	// Step 2: Verify branch_name field has bug/ prefix based on custom name
	assert.Equal(t, 1, wizard.current)
	currentField := wizard.currentField().(*mockBranchNameField)
	assert.NotEmpty(t, currentField.defaultValue, "Default should be set from custom name")
	assert.Contains(t, currentField.defaultValue, "bug/fix_crash", "Bug branch should be based on custom name with bug/ prefix")
}

// mockBranchNameField is a mock field that tracks WithDefault calls.
type mockBranchNameField struct {
	value        any
	theme        *Theme
	key          string
	title        string
	defaultValue string
	width        int
	height       int
	complete     bool
}

func (m *mockBranchNameField) Init() tea.Cmd { return nil }
func (m *mockBranchNameField) Update(msg tea.Msg) (Field, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.Type == tea.KeyEnter {
			m.complete = true
			return m, func() tea.Msg { return NextStepMsg{} }
		}
	}
	return m, nil
}

func (m *mockBranchNameField) View() string { return m.title }
func (m *mockBranchNameField) Focus() tea.Cmd {
	// On focus, if we have a default value and the field is empty, use it
	if m.defaultValue != "" && m.value == "" {
		m.value = m.defaultValue
	}
	return nil
}
func (m *mockBranchNameField) Blur() tea.Cmd            { return nil }
func (m *mockBranchNameField) IsComplete() bool         { return m.complete }
func (m *mockBranchNameField) IsCancelled() bool        { return false }
func (m *mockBranchNameField) Error() error             { return nil }
func (m *mockBranchNameField) Skip() bool               { return false }
func (m *mockBranchNameField) WithTheme(t *Theme) Field { m.theme = t; return m }
func (m *mockBranchNameField) WithWidth(w int) Field    { m.width = w; return m }
func (m *mockBranchNameField) WithHeight(h int) Field   { m.height = h; return m }
func (m *mockBranchNameField) GetKey() string           { return m.key }
func (m *mockBranchNameField) GetValue() any            { return m.value }

// WithDefault is called by the wizard to set a default value.
func (m *mockBranchNameField) WithDefault(defaultValue string) Field {
	newField := *m
	newField.defaultValue = defaultValue
	return &newField
}

// mockJiraService for testing.
type mockJiraService struct {
	issues []JiraIssue
}

func (m *mockJiraService) FetchIssues() ([]JiraIssue, error) {
	return m.issues, nil
}

// TestBackBoundaryMsg_EmitAtStep0 verifies that BackBoundaryMsg is returned when pressing ESC at step 0.
func TestBackBoundaryMsg_EmitAtStep0(t *testing.T) {
	step1 := Step{
		Name:  "Step 1",
		Field: newMockField("test1", "First step"),
	}
	step2 := Step{
		Name:  "Step 2",
		Field: newMockField("test2", "Second step"),
	}

	wizard := NewWizard([]Step{step1, step2}, NewContext())
	wizard.Init()

	// At step 0, pressing ESC should return BackBoundaryMsg
	_, cmd := wizard.Update(tea.KeyMsg{Type: tea.KeyEsc})
	assert.NotNil(t, cmd, "Expected command to be returned")

	// Execute the command and verify it returns BackBoundaryMsg
	if cmd != nil {
		msg := cmd()
		_, isBackBoundaryMsg := msg.(BackBoundaryMsg)
		assert.True(t, isBackBoundaryMsg, "Expected BackBoundaryMsg from ESC at step 0")
	}
}

// TestBackBoundaryMsg_NoEmitAtStepN verifies that normal back navigation works at step N > 0.
func TestBackBoundaryMsg_NoEmitAtStepN(t *testing.T) {
	step1 := Step{
		Name:  "Step 1",
		Field: newMockField("test1", "First step"),
	}
	step2 := Step{
		Name:  "Step 2",
		Field: newMockField("test2", "Second step"),
	}

	wizard := NewWizard([]Step{step1, step2}, NewContext())
	wizard.Init()

	// Advance to step 2
	wizard.Update(NextStepMsg{})
	assert.Equal(t, 1, wizard.current, "Should be at step 1 (second step)")

	// Pressing ESC at step > 0 should go back, not emit BackBoundaryMsg
	_, cmd := wizard.Update(tea.KeyMsg{Type: tea.KeyEsc})

	// The cmd should exist but not return a BackBoundaryMsg when executed
	if cmd != nil {
		msg := cmd()
		_, isBackBoundaryMsg := msg.(BackBoundaryMsg)
		assert.False(t, isBackBoundaryMsg, "Should not emit BackBoundaryMsg when going back from step N > 0")
	}

	// Should now be back at step 0
	assert.Equal(t, 0, wizard.current, "Should be back at step 0 after ESC")
}

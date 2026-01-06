package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

// mockField is a simple test field that can be configured for testing
type mockField struct {
	key         string
	title       string
	value       interface{}
	complete    bool
	cancelled   bool
	err         error
	focused     bool
	width       int
	height      int
	theme       *Theme
	nextOnEnter bool // if true, pressing Enter generates NextStepMsg
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
func (m *mockField) GetValue() interface{}    { return m.value }

func (m *mockField) SetValue(v interface{}) { m.value = v }

// TestWizardNavigation_OneStepAtATime verifies that only the current step is rendered
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

// TestWizardNavigation_AdvanceOnEnter verifies advancing to next step on field completion
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

// TestWizardNavigation_StateUpdatedOnAdvance verifies WorkflowState is updated when advancing
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

// TestWizardNavigation_BackWithESC verifies ESC navigates back to previous step
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

// TestWizardNavigation_ESCOnFirstStep does nothing
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

// TestWizardNavigation_CancelWithCtrlC verifies Ctrl+C exits the wizard
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

// TestWizardNavigation_SkipLogic verifies steps with Skip functions are skipped
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

// TestWizardNavigation_SkipFunctionCanAccessState verifies Skip function accesses WorkflowState
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

// TestWizardNavigation_SkipLogicAppliesBackward verifies Skip logic applies to backward navigation
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

// TestWizardNavigation_CompleteOnLastStep verifies wizard completes when last step is finished
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

// TestWizardNavigation_IsCompleteReturnsFalseUntilAllStepsDone verifies incremental completion
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

// TestWizardNavigation_StatePopulatedWithAllValues verifies all values are collected
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

// TestWizardNavigation_AllStepsSkippedCompletesImmediately verifies all-skipped edge case
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

// TestWizardNavigation_WindowResizeUpdatesContext verifies terminal resize updates dimensions
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

// TestWizardNavigation_Integration tests a complex multi-step scenario
func TestWizardNavigation_Integration(t *testing.T) {
	step1 := Step{
		Name:  "Workflow Type",
		Field: newMockField("workflow_type", "Choose type"),
	}
	step2 := Step{
		Name:  "Branch Name",
		Field: newMockField("branch_name", "Branch"),
	}
	step3 := Step{
		Name:  "Base Branch (conditional)",
		Field: newMockField("base_branch", "Base"),
		Skip: func(state *WorkflowState) bool {
			// Skip if workflow type is feature
			return state.WorkflowType == "feature"
		},
	}

	ctx := NewContext()
	wizard := NewWizard([]Step{step1, step2, step3}, ctx)
	wizard.Init()

	// Set workflow type to "feature"
	field1 := step1.Field.(*mockField)
	field1.value = "feature"

	// Select "feature" and advance
	_, cmd := wizard.Update(tea.KeyMsg{Type: tea.KeyEnter})
	msg := cmd()
	wizard.Update(msg)
	assert.Equal(t, 1, wizard.current)
	assert.Equal(t, "feature", ctx.State.WorkflowType)

	// Select branch and advance
	_, cmd = wizard.Update(tea.KeyMsg{Type: tea.KeyEnter})
	msg = cmd()
	wizard.Update(msg)
	// Should skip step3 because it's a feature - so we're complete after just 2 steps
	assert.True(t, wizard.IsComplete())
}

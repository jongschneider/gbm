package tui

import (
	"bytes"
	"testing"
	"time"

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

// wizardModel wraps a Wizard for teatest, quitting on WorkflowCompleteMsg.
// This simulates what a parent model (like Navigator) would do in production.
type wizardModel struct {
	wizard *Wizard
}

func newWizardModel(w *Wizard) *wizardModel {
	return &wizardModel{wizard: w}
}

func (m *wizardModel) Init() tea.Cmd {
	return m.wizard.Init()
}

func (m *wizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle WorkflowCompleteMsg to quit the test program
	if _, ok := msg.(WorkflowCompleteMsg); ok {
		return m, tea.Quit
	}

	model, cmd := m.wizard.Update(msg)
	if w, ok := model.(*Wizard); ok {
		m.wizard = w
	}
	return m, cmd
}

func (m *wizardModel) View() string {
	return m.wizard.View()
}

// wizardTestField is a test field that renders distinctly and sends NextStepMsg on Enter.
type wizardTestField struct {
	key       string
	title     string
	value     any
	complete  bool
	cancelled bool
	focused   bool
	width     int
	height    int
	theme     *Theme
}

func newWizardTestField(key, title string) *wizardTestField {
	return &wizardTestField{
		key:   key,
		title: title,
		width: 80,
	}
}

func (f *wizardTestField) Init() tea.Cmd { return nil }

func (f *wizardTestField) Update(msg tea.Msg) (Field, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.Type == tea.KeyEnter {
			f.complete = true
			return f, func() tea.Msg { return NextStepMsg{} }
		}
	}
	return f, nil
}

func (f *wizardTestField) View() string {
	prefix := ""
	if f.focused {
		prefix = "> "
	}
	return prefix + f.title + "\n"
}

func (f *wizardTestField) Focus() tea.Cmd           { f.focused = true; return nil }
func (f *wizardTestField) Blur() tea.Cmd            { f.focused = false; return nil }
func (f *wizardTestField) IsComplete() bool         { return f.complete }
func (f *wizardTestField) IsCancelled() bool        { return f.cancelled }
func (f *wizardTestField) Error() error             { return nil }
func (f *wizardTestField) Skip() bool               { return false }
func (f *wizardTestField) WithTheme(t *Theme) Field { f.theme = t; return f }
func (f *wizardTestField) WithWidth(w int) Field    { f.width = w; return f }
func (f *wizardTestField) WithHeight(h int) Field   { f.height = h; return f }
func (f *wizardTestField) GetKey() string           { return f.key }
func (f *wizardTestField) GetValue() any            { return f.value }
func (f *wizardTestField) SetValue(v any)           { f.value = v }

// TestWizard_EnterAdvancesToNextStep verifies Enter key advances to the next step
// in a real Bubble Tea program context using teatest.
func TestWizard_EnterAdvancesToNextStep(t *testing.T) {
	step1Field := newWizardTestField("step1", "Step 1: Choose option")
	step2Field := newWizardTestField("step2", "Step 2: Enter name")
	step3Field := newWizardTestField("step3", "Step 3: Confirm")

	wizard := NewWizard([]Step{
		{Name: "step1", Field: step1Field},
		{Name: "step2", Field: step2Field},
		{Name: "step3", Field: step3Field},
	}, NewContext())

	tm := teatest.NewTestModel(t, wizard, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for initial render showing Step 1
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Step 1"))
	}, teatest.WithDuration(time.Second))

	// Press Enter to advance to Step 2
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Wait for Step 2 to appear
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Step 2"))
	}, teatest.WithDuration(time.Second))

	// Press Enter to advance to Step 3
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Wait for Step 3 to appear
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Step 3"))
	}, teatest.WithDuration(time.Second))

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// TestWizard_EscGoesBackToPreviousStep verifies Esc key navigates back to the previous step
// in a real Bubble Tea program context using teatest.
func TestWizard_EscGoesBackToPreviousStep(t *testing.T) {
	step1Field := newWizardTestField("step1", "Step 1: Choose option")
	step2Field := newWizardTestField("step2", "Step 2: Enter name")

	wizard := NewWizard([]Step{
		{Name: "step1", Field: step1Field},
		{Name: "step2", Field: step2Field},
	}, NewContext())

	tm := teatest.NewTestModel(t, wizard, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for initial render showing Step 1
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Step 1"))
	}, teatest.WithDuration(time.Second))

	// Press Enter to advance to Step 2
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Wait for Step 2 to appear
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Step 2"))
	}, teatest.WithDuration(time.Second))

	// Press Esc to go back to Step 1
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})

	// Wait for Step 1 to reappear
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Step 1"))
	}, teatest.WithDuration(time.Second))

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// TestWizard_CtrlCCancelsAndSetsIsCancelled verifies Ctrl+C cancels the wizard
// and sets IsCancelled() in a real Bubble Tea program context using teatest.
func TestWizard_CtrlCCancelsAndSetsIsCancelled(t *testing.T) {
	step1Field := newWizardTestField("step1", "Step 1: Choose option")

	wizard := NewWizard([]Step{
		{Name: "step1", Field: step1Field},
	}, NewContext())

	// Verify initial state
	assert.False(t, wizard.IsCancelled(), "wizard should not be cancelled initially")

	tm := teatest.NewTestModel(t, wizard, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Step 1"))
	}, teatest.WithDuration(time.Second))

	// Press Ctrl+C to cancel
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})

	// Wait for program to finish
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

	// Verify wizard is now cancelled
	assert.True(t, wizard.IsCancelled(), "wizard should be cancelled after Ctrl+C")
}

// TestWizard_StateUpdatesCorrectlyBetweenSteps verifies that field values are stored
// in WorkflowState when advancing between steps in a real Bubble Tea program context.
func TestWizard_StateUpdatesCorrectlyBetweenSteps(t *testing.T) {
	step1Field := newWizardTestField("worktree_name", "Step 1: Enter name")
	step1Field.value = "my-feature"

	step2Field := newWizardTestField("branch_name", "Step 2: Branch name")
	step2Field.value = "feature/my-feature"

	ctx := NewContext()
	wizard := NewWizard([]Step{
		{Name: "worktree_name", Field: step1Field},
		{Name: "branch_name", Field: step2Field},
	}, ctx)

	// Use wrapper model that quits on WorkflowCompleteMsg
	tm := teatest.NewTestModel(t, newWizardModel(wizard), teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Step 1"))
	}, teatest.WithDuration(time.Second))

	// Press Enter to advance - this should store step1Field.value in state
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Wait for Step 2 to appear
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Step 2"))
	}, teatest.WithDuration(time.Second))

	// Verify state was updated with step 1's value
	assert.Equal(t, "my-feature", ctx.State.WorktreeName,
		"WorkflowState.WorktreeName should be populated after advancing from step 1")

	// Press Enter again to advance - this should store step2Field.value
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Wait for wizard to complete (program will finish via WorkflowCompleteMsg)
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

	// Verify both values are in state
	assert.Equal(t, "my-feature", ctx.State.WorktreeName)
	assert.Equal(t, "feature/my-feature", ctx.State.BranchName,
		"WorkflowState.BranchName should be populated after advancing from step 2")
	assert.True(t, wizard.IsComplete(), "wizard should be complete after all steps")
}

// TestWizard_ViewDelegatesToCurrentFieldView verifies that View() delegates to
// the current field's View() method in a real Bubble Tea program context.
func TestWizard_ViewDelegatesToCurrentFieldView(t *testing.T) {
	step1Field := newWizardTestField("step1", "UNIQUE_STEP1_MARKER")
	step2Field := newWizardTestField("step2", "UNIQUE_STEP2_MARKER")

	wizard := NewWizard([]Step{
		{Name: "step1", Field: step1Field},
		{Name: "step2", Field: step2Field},
	}, NewContext())

	tm := teatest.NewTestModel(t, wizard, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for initial render - should show Step 1's unique marker
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("UNIQUE_STEP1_MARKER"))
	}, teatest.WithDuration(time.Second))

	// Verify Step 2's content is NOT shown
	time.Sleep(50 * time.Millisecond) // Small delay to ensure no extra render
	// Check current view does NOT contain step 2
	assert.Contains(t, wizard.View(), "UNIQUE_STEP1_MARKER",
		"View() should show step 1 content")
	assert.NotContains(t, wizard.View(), "UNIQUE_STEP2_MARKER",
		"View() should NOT show step 2 content when on step 1")

	// Press Enter to advance to Step 2
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Wait for Step 2's unique marker to appear
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("UNIQUE_STEP2_MARKER"))
	}, teatest.WithDuration(time.Second))

	// Verify View() now delegates to Step 2's field
	assert.Contains(t, wizard.View(), "UNIQUE_STEP2_MARKER",
		"View() should show step 2 content after advancing")
	assert.NotContains(t, wizard.View(), "UNIQUE_STEP1_MARKER",
		"View() should NOT show step 1 content when on step 2")

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// TestWizard_CompletionSendsWorkflowCompleteMsg verifies that completing all steps
// results in IsComplete() returning true in a real Bubble Tea program context.
func TestWizard_CompletionSendsWorkflowCompleteMsg(t *testing.T) {
	step1Field := newWizardTestField("step1", "Step 1")
	step2Field := newWizardTestField("step2", "Step 2")

	wizard := NewWizard([]Step{
		{Name: "step1", Field: step1Field},
		{Name: "step2", Field: step2Field},
	}, NewContext())

	// Use wrapper model that quits on WorkflowCompleteMsg
	tm := teatest.NewTestModel(t, newWizardModel(wizard), teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Step 1"))
	}, teatest.WithDuration(time.Second))

	assert.False(t, wizard.IsComplete(), "wizard should not be complete initially")

	// Advance through step 1
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Step 2"))
	}, teatest.WithDuration(time.Second))

	assert.False(t, wizard.IsComplete(), "wizard should not be complete at step 2")

	// Complete step 2 (last step)
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Wait for wizard to complete (program will finish via WorkflowCompleteMsg)
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

	assert.True(t, wizard.IsComplete(), "wizard should be complete after all steps")
}

// TestWizard_MultipleBackAndForth verifies multiple navigation cycles work correctly
// in a real Bubble Tea program context.
func TestWizard_MultipleBackAndForth(t *testing.T) {
	step1Field := newWizardTestField("step1", "STEP_ONE")
	step2Field := newWizardTestField("step2", "STEP_TWO")
	step3Field := newWizardTestField("step3", "STEP_THREE")

	wizard := NewWizard([]Step{
		{Name: "step1", Field: step1Field},
		{Name: "step2", Field: step2Field},
		{Name: "step3", Field: step3Field},
	}, NewContext())

	tm := teatest.NewTestModel(t, wizard, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for Step 1
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("STEP_ONE"))
	}, teatest.WithDuration(time.Second))

	// Forward to Step 2
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("STEP_TWO"))
	}, teatest.WithDuration(time.Second))

	// Forward to Step 3
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("STEP_THREE"))
	}, teatest.WithDuration(time.Second))

	// Back to Step 2
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("STEP_TWO"))
	}, teatest.WithDuration(time.Second))

	// Back to Step 1
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("STEP_ONE"))
	}, teatest.WithDuration(time.Second))

	// Forward again to Step 2
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("STEP_TWO"))
	}, teatest.WithDuration(time.Second))

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

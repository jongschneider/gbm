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
	time.Sleep(2 * time.Millisecond) // Brief wait for render (checking absence)
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

// =============================================================================
// TT-002: Wizard skip logic tests
// =============================================================================

// TestWizard_SkipStepOnForwardNavigation verifies that steps with Skip func returning true
// are skipped when navigating forward.
func TestWizard_SkipStepOnForwardNavigation(t *testing.T) {
	step1Field := newWizardTestField("step1", "STEP_ONE")
	step2Field := newWizardTestField("step2", "STEP_TWO_SKIPPED")
	step3Field := newWizardTestField("step3", "STEP_THREE")

	wizard := NewWizard([]Step{
		{Name: "step1", Field: step1Field},
		{Name: "step2", Field: step2Field, Skip: func(_ *WorkflowState) bool { return true }},
		{Name: "step3", Field: step3Field},
	}, NewContext())

	tm := teatest.NewTestModel(t, wizard, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for Step 1
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("STEP_ONE"))
	}, teatest.WithDuration(time.Second))

	// Press Enter - should skip Step 2 and go directly to Step 3
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Wait for Step 3 to appear (Step 2 should be skipped)
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("STEP_THREE"))
	}, teatest.WithDuration(time.Second))

	// Verify Step 2 content is NOT shown
	assert.NotContains(t, wizard.View(), "STEP_TWO_SKIPPED",
		"Skipped step should not appear in View()")

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// TestWizard_ShowStepWhenSkipReturnsFalse verifies that steps with Skip func returning false
// are shown normally.
func TestWizard_ShowStepWhenSkipReturnsFalse(t *testing.T) {
	step1Field := newWizardTestField("step1", "STEP_ONE")
	step2Field := newWizardTestField("step2", "STEP_TWO_SHOWN")
	step3Field := newWizardTestField("step3", "STEP_THREE")

	wizard := NewWizard([]Step{
		{Name: "step1", Field: step1Field},
		{Name: "step2", Field: step2Field, Skip: func(_ *WorkflowState) bool { return false }},
		{Name: "step3", Field: step3Field},
	}, NewContext())

	tm := teatest.NewTestModel(t, wizard, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for Step 1
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("STEP_ONE"))
	}, teatest.WithDuration(time.Second))

	// Press Enter - should go to Step 2 (Skip returns false)
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Wait for Step 2 to appear
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("STEP_TWO_SHOWN"))
	}, teatest.WithDuration(time.Second))

	// Verify we're at Step 2
	assert.Contains(t, wizard.View(), "STEP_TWO_SHOWN",
		"Step with Skip returning false should be shown")

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// TestWizard_SkipStepOnBackwardNavigation verifies that skipped steps are also skipped
// when navigating backward.
func TestWizard_SkipStepOnBackwardNavigation(t *testing.T) {
	step1Field := newWizardTestField("step1", "STEP_ONE")
	step2Field := newWizardTestField("step2", "STEP_TWO_SKIPPED")
	step3Field := newWizardTestField("step3", "STEP_THREE")

	wizard := NewWizard([]Step{
		{Name: "step1", Field: step1Field},
		{Name: "step2", Field: step2Field, Skip: func(_ *WorkflowState) bool { return true }},
		{Name: "step3", Field: step3Field},
	}, NewContext())

	tm := teatest.NewTestModel(t, wizard, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for Step 1
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("STEP_ONE"))
	}, teatest.WithDuration(time.Second))

	// Press Enter - should skip Step 2 and go to Step 3
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Wait for Step 3
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("STEP_THREE"))
	}, teatest.WithDuration(time.Second))

	// Press Esc - should skip Step 2 and go back to Step 1
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})

	// Wait for Step 1 to reappear (skipping Step 2)
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("STEP_ONE"))
	}, teatest.WithDuration(time.Second))

	// Verify we're back at Step 1, not Step 2
	assert.Contains(t, wizard.View(), "STEP_ONE",
		"Should go back to Step 1, skipping Step 2")
	assert.NotContains(t, wizard.View(), "STEP_TWO_SKIPPED",
		"Skipped step should not appear when navigating backward")

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// TestWizard_SkipLogicReEvaluatesBasedOnState verifies that skip logic re-evaluates
// based on current workflow state.
func TestWizard_SkipLogicReEvaluatesBasedOnState(t *testing.T) {
	step1Field := newWizardTestField("worktree_name", "STEP_ONE_NAME")
	step1Field.value = "my-feature"

	step2Field := newWizardTestField("step2", "STEP_TWO_CONDITIONAL")
	step3Field := newWizardTestField("step3", "STEP_THREE")

	ctx := NewContext()

	// Step 2 is skipped only when WorktreeName equals "skip-me"
	wizard := NewWizard([]Step{
		{Name: "worktree_name", Field: step1Field},
		{Name: "step2", Field: step2Field, Skip: func(state *WorkflowState) bool {
			return state.WorktreeName == "skip-me"
		}},
		{Name: "step3", Field: step3Field},
	}, ctx)

	tm := teatest.NewTestModel(t, wizard, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for Step 1
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("STEP_ONE_NAME"))
	}, teatest.WithDuration(time.Second))

	// Step 1 value is "my-feature", so Step 2 should NOT be skipped
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Wait for Step 2 (should appear because state.WorktreeName != "skip-me")
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("STEP_TWO_CONDITIONAL"))
	}, teatest.WithDuration(time.Second))

	// Verify state was stored
	assert.Equal(t, "my-feature", ctx.State.WorktreeName,
		"WorktreeName should be stored in state")

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// TestWizard_SkipLogicCausesStepToBeSkipped verifies that changing state causes
// the skip logic to evaluate differently.
func TestWizard_SkipLogicCausesStepToBeSkipped(t *testing.T) {
	step1Field := newWizardTestField("worktree_name", "STEP_ONE_NAME")
	step1Field.value = "skip-me" // This value will cause Step 2 to be skipped

	step2Field := newWizardTestField("step2", "STEP_TWO_SHOULD_BE_SKIPPED")
	step3Field := newWizardTestField("step3", "STEP_THREE_FINAL")

	ctx := NewContext()

	// Step 2 is skipped when WorktreeName equals "skip-me"
	wizard := NewWizard([]Step{
		{Name: "worktree_name", Field: step1Field},
		{Name: "step2", Field: step2Field, Skip: func(state *WorkflowState) bool {
			return state.WorktreeName == "skip-me"
		}},
		{Name: "step3", Field: step3Field},
	}, ctx)

	tm := teatest.NewTestModel(t, wizard, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for Step 1
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("STEP_ONE_NAME"))
	}, teatest.WithDuration(time.Second))

	// Step 1 value is "skip-me", so Step 2 should be skipped
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Wait for Step 3 (Step 2 should be skipped)
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("STEP_THREE_FINAL"))
	}, teatest.WithDuration(time.Second))

	// Verify Step 2 was skipped
	assert.NotContains(t, wizard.View(), "STEP_TWO_SHOULD_BE_SKIPPED",
		"Step 2 should be skipped because state.WorktreeName == 'skip-me'")
	assert.Equal(t, "skip-me", ctx.State.WorktreeName,
		"WorktreeName should be stored in state")

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// TestWizard_SkipFirstStepStartsAtSecond verifies that if the first step is skipped,
// the wizard starts at the second step.
func TestWizard_SkipFirstStepStartsAtSecond(t *testing.T) {
	step1Field := newWizardTestField("step1", "STEP_ONE_SKIPPED")
	step2Field := newWizardTestField("step2", "STEP_TWO_FIRST")
	step3Field := newWizardTestField("step3", "STEP_THREE")

	wizard := NewWizard([]Step{
		{Name: "step1", Field: step1Field, Skip: func(_ *WorkflowState) bool { return true }},
		{Name: "step2", Field: step2Field},
		{Name: "step3", Field: step3Field},
	}, NewContext())

	tm := teatest.NewTestModel(t, wizard, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Should start at Step 2 since Step 1 is skipped
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("STEP_TWO_FIRST"))
	}, teatest.WithDuration(time.Second))

	// Verify Step 1 is not shown
	assert.NotContains(t, wizard.View(), "STEP_ONE_SKIPPED",
		"First skipped step should not appear")
	assert.Contains(t, wizard.View(), "STEP_TWO_FIRST",
		"Wizard should start at first non-skipped step")

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// TestWizard_SkipMultipleConsecutiveSteps verifies that multiple consecutive skipped steps
// are all bypassed.
func TestWizard_SkipMultipleConsecutiveSteps(t *testing.T) {
	step1Field := newWizardTestField("step1", "STEP_ONE")
	step2Field := newWizardTestField("step2", "STEP_TWO_SKIPPED")
	step3Field := newWizardTestField("step3", "STEP_THREE_SKIPPED")
	step4Field := newWizardTestField("step4", "STEP_FOUR_FINAL")

	wizard := NewWizard([]Step{
		{Name: "step1", Field: step1Field},
		{Name: "step2", Field: step2Field, Skip: func(_ *WorkflowState) bool { return true }},
		{Name: "step3", Field: step3Field, Skip: func(_ *WorkflowState) bool { return true }},
		{Name: "step4", Field: step4Field},
	}, NewContext())

	tm := teatest.NewTestModel(t, wizard, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for Step 1
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("STEP_ONE"))
	}, teatest.WithDuration(time.Second))

	// Press Enter - should skip Steps 2 and 3, go directly to Step 4
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Wait for Step 4
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("STEP_FOUR_FINAL"))
	}, teatest.WithDuration(time.Second))

	// Verify Steps 2 and 3 are not shown
	assert.NotContains(t, wizard.View(), "STEP_TWO_SKIPPED")
	assert.NotContains(t, wizard.View(), "STEP_THREE_SKIPPED")
	assert.Contains(t, wizard.View(), "STEP_FOUR_FINAL")

	// Press Esc - should skip back over Steps 3 and 2 to Step 1
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})

	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("STEP_ONE"))
	}, teatest.WithDuration(time.Second))

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// =============================================================================
// TT-003: Wizard completion flow tests
// =============================================================================

// TestWizard_WorkflowCompleteMsgSentAfterLastStep verifies that WorkflowCompleteMsg is sent
// after pressing Enter on the last step. The wrapper model quits on WorkflowCompleteMsg,
// so program termination indicates the message was sent.
func TestWizard_WorkflowCompleteMsgSentAfterLastStep(t *testing.T) {
	step1Field := newWizardTestField("step1", "FIRST_STEP")
	step2Field := newWizardTestField("step2", "LAST_STEP")

	wizard := NewWizard([]Step{
		{Name: "step1", Field: step1Field},
		{Name: "step2", Field: step2Field},
	}, NewContext())

	// Track if WorkflowCompleteMsg was received
	completeMsgReceived := false
	wrapperModel := &workflowCompleteMsgTracker{
		wizard:              wizard,
		completeMsgReceived: &completeMsgReceived,
	}

	tm := teatest.NewTestModel(t, wrapperModel, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for first step
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("FIRST_STEP"))
	}, teatest.WithDuration(time.Second))

	// Advance to last step
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("LAST_STEP"))
	}, teatest.WithDuration(time.Second))

	// Press Enter on last step - should trigger WorkflowCompleteMsg
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Wait for program to finish (happens because wrapper quits on WorkflowCompleteMsg)
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

	// Verify WorkflowCompleteMsg was received
	assert.True(t, completeMsgReceived, "WorkflowCompleteMsg should be sent after last step Enter")
	assert.True(t, wizard.IsComplete(), "wizard should be complete after WorkflowCompleteMsg")
}

// workflowCompleteMsgTracker wraps a Wizard to track WorkflowCompleteMsg reception.
type workflowCompleteMsgTracker struct {
	wizard              *Wizard
	completeMsgReceived *bool
}

func (m *workflowCompleteMsgTracker) Init() tea.Cmd {
	return m.wizard.Init()
}

func (m *workflowCompleteMsgTracker) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(WorkflowCompleteMsg); ok {
		*m.completeMsgReceived = true
		return m, tea.Quit
	}
	model, cmd := m.wizard.Update(msg)
	if w, ok := model.(*Wizard); ok {
		m.wizard = w
	}
	return m, cmd
}

func (m *workflowCompleteMsgTracker) View() string {
	return m.wizard.View()
}

// TestWizard_IsCompleteAfterAllSteps verifies that IsComplete() returns true
// after completing all steps and false before completion.
func TestWizard_IsCompleteAfterAllSteps(t *testing.T) {
	step1Field := newWizardTestField("step1", "STEP_ONE")
	step2Field := newWizardTestField("step2", "STEP_TWO")
	step3Field := newWizardTestField("step3", "STEP_THREE")

	wizard := NewWizard([]Step{
		{Name: "step1", Field: step1Field},
		{Name: "step2", Field: step2Field},
		{Name: "step3", Field: step3Field},
	}, NewContext())

	tm := teatest.NewTestModel(t, newWizardModel(wizard), teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Initial state: not complete
	assert.False(t, wizard.IsComplete(), "wizard should not be complete initially")

	// Wait for Step 1
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("STEP_ONE"))
	}, teatest.WithDuration(time.Second))

	// Complete Step 1
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("STEP_TWO"))
	}, teatest.WithDuration(time.Second))
	assert.False(t, wizard.IsComplete(), "wizard should not be complete at step 2")

	// Complete Step 2
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("STEP_THREE"))
	}, teatest.WithDuration(time.Second))
	assert.False(t, wizard.IsComplete(), "wizard should not be complete at step 3")

	// Complete Step 3 (last step)
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Wait for wizard to complete
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

	// Final state: complete
	assert.True(t, wizard.IsComplete(), "wizard should be complete after all steps")
}

// TestWizard_AllFieldValuesStoredInState verifies that all field values are stored
// in WorkflowState after completing the wizard.
func TestWizard_AllFieldValuesStoredInState(t *testing.T) {
	// Create fields with distinct values
	worktreeField := newWizardTestField(FieldKeyWorktreeName, "Enter worktree name")
	worktreeField.value = "my-worktree"

	branchField := newWizardTestField(FieldKeyBranchName, "Enter branch name")
	branchField.value = "feature/my-branch"

	baseBranchField := newWizardTestField(FieldKeyBaseBranch, "Select base branch")
	baseBranchField.value = "develop"

	ctx := NewContext()
	wizard := NewWizard([]Step{
		{Name: FieldKeyWorktreeName, Field: worktreeField},
		{Name: FieldKeyBranchName, Field: branchField},
		{Name: FieldKeyBaseBranch, Field: baseBranchField},
	}, ctx)

	tm := teatest.NewTestModel(t, newWizardModel(wizard), teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Initial state: all values should be empty
	assert.Empty(t, ctx.State.WorktreeName, "WorktreeName should be empty initially")
	assert.Empty(t, ctx.State.BranchName, "BranchName should be empty initially")
	assert.Empty(t, ctx.State.BaseBranch, "BaseBranch should be empty initially")

	// Wait for first step
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Enter worktree name"))
	}, teatest.WithDuration(time.Second))

	// Complete step 1 - worktree name should be stored
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Enter branch name"))
	}, teatest.WithDuration(time.Second))
	assert.Equal(t, "my-worktree", ctx.State.WorktreeName, "WorktreeName should be stored after step 1")

	// Complete step 2 - branch name should be stored
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Select base branch"))
	}, teatest.WithDuration(time.Second))
	assert.Equal(t, "feature/my-branch", ctx.State.BranchName, "BranchName should be stored after step 2")

	// Complete step 3 - base branch should be stored
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Wait for wizard to complete
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

	// Verify all values are stored
	assert.Equal(t, "my-worktree", ctx.State.WorktreeName)
	assert.Equal(t, "feature/my-branch", ctx.State.BranchName)
	assert.Equal(t, "develop", ctx.State.BaseBranch)
	assert.True(t, wizard.IsComplete(), "wizard should be complete")
}

// TestWizard_EmptyStepsListCompletesImmediately verifies that a wizard with no steps
// handles the edge case gracefully by completing immediately.
func TestWizard_EmptyStepsListCompletesImmediately(t *testing.T) {
	ctx := NewContext()
	wizard := NewWizard([]Step{}, ctx)

	// Empty wizard should complete immediately on Init
	assert.False(t, wizard.IsComplete(), "wizard should not be complete before Init")

	tm := teatest.NewTestModel(t, wizard, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for program to finish (should happen immediately)
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

	// Verify wizard completed
	assert.True(t, wizard.IsComplete(), "empty wizard should be complete after Init")
	assert.False(t, wizard.IsCancelled(), "empty wizard should not be cancelled")

	// View should return empty string for empty wizard
	assert.Equal(t, "", wizard.View(), "empty wizard View() should return empty string")
}

// TestWizard_SingleStepCompletesOnEnter verifies that a single-step wizard
// completes correctly when Enter is pressed.
func TestWizard_SingleStepCompletesOnEnter(t *testing.T) {
	singleField := newWizardTestField("only_step", "ONLY_STEP_CONTENT")
	singleField.value = "single-value"

	ctx := NewContext()
	wizard := NewWizard([]Step{
		{Name: "only_step", Field: singleField},
	}, ctx)

	tm := teatest.NewTestModel(t, newWizardModel(wizard), teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for the single step
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("ONLY_STEP_CONTENT"))
	}, teatest.WithDuration(time.Second))

	assert.False(t, wizard.IsComplete(), "wizard should not be complete before Enter")

	// Press Enter to complete the only step
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Wait for wizard to complete
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

	assert.True(t, wizard.IsComplete(), "single-step wizard should be complete after Enter")
}

// =============================================================================
// TT-004: Wizard back boundary handling tests
// =============================================================================

// backBoundaryMsgTracker wraps a Wizard to track BackBoundaryMsg reception.
type backBoundaryMsgTracker struct {
	wizard               *Wizard
	backBoundaryReceived *bool
}

func (m *backBoundaryMsgTracker) Init() tea.Cmd {
	return m.wizard.Init()
}

func (m *backBoundaryMsgTracker) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(BackBoundaryMsg); ok {
		*m.backBoundaryReceived = true
		// Don't quit - just record that we received the message
	}
	// Handle Ctrl+C to quit
	if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.Type == tea.KeyCtrlC {
		return m, tea.Quit
	}
	model, cmd := m.wizard.Update(msg)
	if w, ok := model.(*Wizard); ok {
		m.wizard = w
	}
	return m, cmd
}

func (m *backBoundaryMsgTracker) View() string {
	return m.wizard.View()
}

// TestWizard_BackBoundaryMsgSentAtStep0 verifies that BackBoundaryMsg is sent
// when pressing Esc at step 0 (the first step).
func TestWizard_BackBoundaryMsgSentAtStep0(t *testing.T) {
	step1Field := newWizardTestField("step1", "FIRST_STEP")
	step2Field := newWizardTestField("step2", "SECOND_STEP")

	wizard := NewWizard([]Step{
		{Name: "step1", Field: step1Field},
		{Name: "step2", Field: step2Field},
	}, NewContext())

	// Track if BackBoundaryMsg was received
	backBoundaryReceived := false
	wrapperModel := &backBoundaryMsgTracker{
		wizard:               wizard,
		backBoundaryReceived: &backBoundaryReceived,
	}

	tm := teatest.NewTestModel(t, wrapperModel, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for first step
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("FIRST_STEP"))
	}, teatest.WithDuration(time.Second))

	// Press Esc at step 0 - should trigger BackBoundaryMsg
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})

	// Wait for BackBoundaryMsg to be received
	waitFor(t, func() bool { return backBoundaryReceived }, time.Second)

	// Verify BackBoundaryMsg was received
	assert.True(t, backBoundaryReceived, "BackBoundaryMsg should be sent when pressing Esc at step 0")

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// TestWizard_RemainsAtStep0AfterBackBoundary verifies that the wizard remains
// at step 0 after BackBoundaryMsg is sent (pressing Esc at first step doesn't navigate away).
func TestWizard_RemainsAtStep0AfterBackBoundary(t *testing.T) {
	step1Field := newWizardTestField("step1", "UNIQUE_FIRST_STEP")
	step2Field := newWizardTestField("step2", "UNIQUE_SECOND_STEP")

	wizard := NewWizard([]Step{
		{Name: "step1", Field: step1Field},
		{Name: "step2", Field: step2Field},
	}, NewContext())

	tm := teatest.NewTestModel(t, wizard, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for first step
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("UNIQUE_FIRST_STEP"))
	}, teatest.WithDuration(time.Second))

	// Press Esc at step 0 multiple times
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})

	// Brief wait for messages to be processed (checking absence - no state change expected)
	time.Sleep(2 * time.Millisecond)

	// Verify we're still at step 0 - view should still show first step content
	assert.Contains(t, wizard.View(), "UNIQUE_FIRST_STEP",
		"Wizard should remain at step 0 after pressing Esc at first step")
	assert.NotContains(t, wizard.View(), "UNIQUE_SECOND_STEP",
		"Wizard should not show step 2 content when at step 0")

	// Verify wizard state is still valid (not cancelled, not complete)
	assert.False(t, wizard.IsCancelled(), "Wizard should not be cancelled after Esc at step 0")
	assert.False(t, wizard.IsComplete(), "Wizard should not be complete after Esc at step 0")

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// TestWizard_FieldRemainsFocusedAfterBackBoundary verifies that the current field
// remains focused after BackBoundaryMsg is sent.
func TestWizard_FieldRemainsFocusedAfterBackBoundary(t *testing.T) {
	step1Field := newWizardTestField("step1", "FOCUSED_STEP")

	wizard := NewWizard([]Step{
		{Name: "step1", Field: step1Field},
	}, NewContext())

	tm := teatest.NewTestModel(t, wizard, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for first step - should show focused indicator "> "
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("> FOCUSED_STEP"))
	}, teatest.WithDuration(time.Second))

	// Verify field is focused (shows "> " prefix)
	assert.Contains(t, wizard.View(), "> FOCUSED_STEP",
		"Field should be focused initially (showing '> ' prefix)")

	// Press Esc at step 0
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})

	// Brief wait (checking absence - no focus change expected)
	time.Sleep(2 * time.Millisecond)

	// Verify field is still focused after BackBoundaryMsg
	// The wizardTestField shows "> " prefix when focused
	assert.Contains(t, wizard.View(), "> FOCUSED_STEP",
		"Field should remain focused after BackBoundaryMsg (showing '> ' prefix)")

	// Additional verification: the field's focused state should be true
	assert.True(t, step1Field.focused, "Field's focused flag should remain true after BackBoundaryMsg")

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// =============================================================================
// TT-005: Wizard window resize propagation tests
// =============================================================================

// resizeTrackingField tracks width/height updates for testing window resize propagation.
type resizeTrackingField struct {
	key         string
	title       string
	width       int
	height      int
	widthCalls  []int
	heightCalls []int
	complete    bool
	focused     bool
}

func newResizeTrackingField(key, title string) *resizeTrackingField {
	return &resizeTrackingField{
		key:         key,
		title:       title,
		width:       80,
		height:      24,
		widthCalls:  []int{},
		heightCalls: []int{},
	}
}

func (f *resizeTrackingField) Init() tea.Cmd { return nil }

func (f *resizeTrackingField) Update(msg tea.Msg) (Field, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.Type == tea.KeyEnter {
			f.complete = true
			return f, func() tea.Msg { return NextStepMsg{} }
		}
	}
	return f, nil
}

func (f *resizeTrackingField) View() string {
	return f.title + "\n"
}

func (f *resizeTrackingField) Focus() tea.Cmd           { f.focused = true; return nil }
func (f *resizeTrackingField) Blur() tea.Cmd            { f.focused = false; return nil }
func (f *resizeTrackingField) IsComplete() bool         { return f.complete }
func (f *resizeTrackingField) IsCancelled() bool        { return false }
func (f *resizeTrackingField) Error() error             { return nil }
func (f *resizeTrackingField) Skip() bool               { return false }
func (f *resizeTrackingField) WithTheme(t *Theme) Field { return f }
func (f *resizeTrackingField) WithWidth(w int) Field {
	f.width = w
	f.widthCalls = append(f.widthCalls, w)
	return f
}

func (f *resizeTrackingField) WithHeight(h int) Field {
	f.height = h
	f.heightCalls = append(f.heightCalls, h)
	return f
}
func (f *resizeTrackingField) GetKey() string { return f.key }
func (f *resizeTrackingField) GetValue() any  { return "" }
func (f *resizeTrackingField) SetValue(v any) {}

// TestWizard_WindowSizeMsgUpdatesContextDimensions verifies that WindowSizeMsg
// updates the wizard context width and height.
func TestWizard_WindowSizeMsgUpdatesContextDimensions(t *testing.T) {
	step1Field := newResizeTrackingField("step1", "STEP_ONE")

	ctx := NewContext()
	wizard := NewWizard([]Step{
		{Name: "step1", Field: step1Field},
	}, ctx)

	tm := teatest.NewTestModel(t, wizard, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("STEP_ONE"))
	}, teatest.WithDuration(time.Second))

	// Initial dimensions from teatest
	assert.Equal(t, 80, ctx.Width, "Initial context width should be 80")
	assert.Equal(t, 24, ctx.Height, "Initial context height should be 24")

	// Send window resize message
	tm.Send(tea.WindowSizeMsg{Width: 120, Height: 40})

	// Wait for context dimensions to be updated
	waitFor(t, func() bool { return ctx.Width == 120 && ctx.Height == 40 }, time.Second)

	// Verify context dimensions are updated
	assert.Equal(t, 120, ctx.Width, "Context width should be updated to 120 after WindowSizeMsg")
	assert.Equal(t, 40, ctx.Height, "Context height should be updated to 40 after WindowSizeMsg")

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// TestWizard_WindowSizeMsgCallsFieldWithWidthHeight verifies that WindowSizeMsg
// calls WithWidth and WithHeight on the current field.
func TestWizard_WindowSizeMsgCallsFieldWithWidthHeight(t *testing.T) {
	step1Field := newResizeTrackingField("step1", "STEP_ONE")

	wizard := NewWizard([]Step{
		{Name: "step1", Field: step1Field},
	}, NewContext())

	tm := teatest.NewTestModel(t, wizard, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("STEP_ONE"))
	}, teatest.WithDuration(time.Second))

	// Record initial call counts
	initialWidthCalls := len(step1Field.widthCalls)
	initialHeightCalls := len(step1Field.heightCalls)

	// Send window resize message
	tm.Send(tea.WindowSizeMsg{Width: 100, Height: 30})

	// Wait for WithWidth/WithHeight to be called
	waitFor(t, func() bool { return len(step1Field.widthCalls) > initialWidthCalls }, time.Second)

	// Verify WithWidth and WithHeight were called
	assert.Greater(t, len(step1Field.widthCalls), initialWidthCalls,
		"WithWidth should be called after WindowSizeMsg")
	assert.Greater(t, len(step1Field.heightCalls), initialHeightCalls,
		"WithHeight should be called after WindowSizeMsg")

	// Verify the new dimensions were passed
	if len(step1Field.widthCalls) > initialWidthCalls {
		lastWidthCall := step1Field.widthCalls[len(step1Field.widthCalls)-1]
		assert.Equal(t, 100, lastWidthCall, "WithWidth should receive new width 100")
	}
	if len(step1Field.heightCalls) > initialHeightCalls {
		lastHeightCall := step1Field.heightCalls[len(step1Field.heightCalls)-1]
		assert.Equal(t, 30, lastHeightCall, "WithHeight should receive new height 30")
	}

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// TestWizard_FieldReRendersWithNewDimensions verifies that after a window resize,
// the field's dimensions are updated and reflected in subsequent renders.
func TestWizard_FieldReRendersWithNewDimensions(t *testing.T) {
	step1Field := newResizeTrackingField("step1", "RESIZE_TEST")

	wizard := NewWizard([]Step{
		{Name: "step1", Field: step1Field},
	}, NewContext())

	tm := teatest.NewTestModel(t, wizard, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("RESIZE_TEST"))
	}, teatest.WithDuration(time.Second))

	// Verify field has initial dimensions
	assert.Equal(t, 80, step1Field.width, "Field should have initial width 80")
	assert.Equal(t, 24, step1Field.height, "Field should have initial height 24")

	// Send window resize message
	tm.Send(tea.WindowSizeMsg{Width: 150, Height: 50})

	// Wait for field dimensions to be updated
	waitFor(t, func() bool { return step1Field.width == 150 }, time.Second)

	// Verify field dimensions are updated
	assert.Equal(t, 150, step1Field.width, "Field width should be updated to 150")
	assert.Equal(t, 50, step1Field.height, "Field height should be updated to 50")

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// TestWizard_MultipleResizeMessages verifies that multiple consecutive resize messages
// are all processed correctly.
func TestWizard_MultipleResizeMessages(t *testing.T) {
	step1Field := newResizeTrackingField("step1", "MULTI_RESIZE")

	ctx := NewContext()
	wizard := NewWizard([]Step{
		{Name: "step1", Field: step1Field},
	}, ctx)

	tm := teatest.NewTestModel(t, wizard, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("MULTI_RESIZE"))
	}, teatest.WithDuration(time.Second))

	// Send multiple resize messages
	tm.Send(tea.WindowSizeMsg{Width: 100, Height: 30})
	waitFor(t, func() bool { return ctx.Width == 100 }, time.Second)
	tm.Send(tea.WindowSizeMsg{Width: 120, Height: 40})
	waitFor(t, func() bool { return ctx.Width == 120 }, time.Second)
	tm.Send(tea.WindowSizeMsg{Width: 160, Height: 60})

	// Wait for final resize to be processed
	waitFor(t, func() bool { return ctx.Width == 160 && ctx.Height == 60 }, time.Second)

	// Verify final dimensions match the last resize message
	assert.Equal(t, 160, ctx.Width, "Context width should match final resize")
	assert.Equal(t, 60, ctx.Height, "Context height should match final resize")
	assert.Equal(t, 160, step1Field.width, "Field width should match final resize")
	assert.Equal(t, 60, step1Field.height, "Field height should match final resize")

	// Verify all resize messages were processed (check call counts)
	assert.GreaterOrEqual(t, len(step1Field.widthCalls), 3,
		"WithWidth should be called at least 3 times for 3 resize messages")
	assert.GreaterOrEqual(t, len(step1Field.heightCalls), 3,
		"WithHeight should be called at least 3 times for 3 resize messages")

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// TestWizard_ResizeDoesNotAffectStepNavigation verifies that window resize messages
// don't interfere with step navigation.
func TestWizard_ResizeDoesNotAffectStepNavigation(t *testing.T) {
	step1Field := newResizeTrackingField("step1", "STEP_ONE_NAV")
	step2Field := newResizeTrackingField("step2", "STEP_TWO_NAV")

	wizard := NewWizard([]Step{
		{Name: "step1", Field: step1Field},
		{Name: "step2", Field: step2Field},
	}, NewContext())

	tm := teatest.NewTestModel(t, wizard, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for Step 1
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("STEP_ONE_NAV"))
	}, teatest.WithDuration(time.Second))

	// Send resize message
	tm.Send(tea.WindowSizeMsg{Width: 120, Height: 40})
	waitFor(t, func() bool { return step1Field.width == 120 }, time.Second)

	// Navigate to Step 2
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Wait for Step 2
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("STEP_TWO_NAV"))
	}, teatest.WithDuration(time.Second))

	// Verify navigation worked correctly
	assert.Contains(t, wizard.View(), "STEP_TWO_NAV",
		"Should be at Step 2 after navigation despite resize message")
	assert.NotContains(t, wizard.View(), "STEP_ONE_NAV",
		"Step 1 should not be visible after navigation")

	// Send another resize and verify Step 2 field gets the update
	tm.Send(tea.WindowSizeMsg{Width: 200, Height: 80})
	waitFor(t, func() bool { return step2Field.width == 200 }, time.Second)

	assert.Equal(t, 200, step2Field.width, "Step 2 field should receive resize after navigation")
	assert.Equal(t, 80, step2Field.height, "Step 2 field should receive resize after navigation")

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

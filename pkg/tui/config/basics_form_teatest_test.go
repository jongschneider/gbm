package config

import (
	"testing"
	"time"

	"gbm/pkg/tui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/muesli/termenv"
	"github.com/stretchr/testify/assert"
)

func initTestProfile() {
	lipgloss.SetColorProfile(termenv.Ascii)
}

// basicsFormModel wraps BasicsForm for teatest, quitting on BackBoundaryMsg.
type basicsFormModel struct {
	form *BasicsForm
}

func newBasicsFormModel(f *BasicsForm) *basicsFormModel {
	return &basicsFormModel{form: f}
}

func (m *basicsFormModel) Init() tea.Cmd {
	return m.form.Init()
}

func (m *basicsFormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle BackBoundaryMsg to quit the test program
	if _, ok := msg.(tui.BackBoundaryMsg); ok {
		return m, tea.Quit
	}

	model, cmd := m.form.Update(msg)
	if f, ok := model.(*BasicsForm); ok {
		m.form = f
	}
	return m, cmd
}

func (m *basicsFormModel) View() string {
	return m.form.View()
}

// TestBasicsForm_SaveFlow tests the save flow.
func TestBasicsForm_SaveFlow(t *testing.T) {
	t.Parallel()
	initTestProfile()

	saveCalled := false
	savedData := map[string]string{}

	config := BasicsFormConfig{
		DefaultBranch: "main",
		WorktreesDir:  "./worktrees",
		Theme:         tui.DefaultTheme(),
		OnSave: func(data map[string]string) error {
			saveCalled = true
			savedData = data

			return nil
		},
	}

	form := NewBasicsForm(config)
	model := newBasicsFormModel(form)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return len(bts) > 0
	}, teatest.WithDuration(time.Second))

	// Send 's' key to save
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})

	// Wait for quit
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

	// Verify save was called
	assert.True(t, saveCalled, "OnSave callback should have been called")
	assert.NotEmpty(t, savedData, "SaveData should not be empty")
}

// TestBasicsForm_DiscardFlow tests the discard confirmation flow.
func TestBasicsForm_DiscardFlow(t *testing.T) {
	t.Parallel()
	initTestProfile()

	config := BasicsFormConfig{
		DefaultBranch: "main",
		WorktreesDir:  "./worktrees",
		Theme:         tui.DefaultTheme(),
	}

	form := NewBasicsForm(config)
	model := newBasicsFormModel(form)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return len(bts) > 0
	}, teatest.WithDuration(time.Second))

	// Send 'q' key to start discard flow
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	// Wait for confirmation dialog to appear
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		output := string(bts)
		return output != ""
	}, teatest.WithDuration(time.Second))

	// Send Enter (or 'y') to confirm discard
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Wait for quit
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

	// Verify the form is cancelled
	assert.True(t, form.IsCancelled(), "Form should be cancelled after discard")
}

// TestBasicsForm_KeepEditingFlow tests cancelling the discard confirmation.
func TestBasicsForm_KeepEditingFlow(t *testing.T) {
	t.Parallel()
	initTestProfile()

	config := BasicsFormConfig{
		DefaultBranch: "main",
		WorktreesDir:  "./worktrees",
		Theme:         tui.DefaultTheme(),
	}

	form := NewBasicsForm(config)
	model := newBasicsFormModel(form)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return len(bts) > 0
	}, teatest.WithDuration(time.Second))

	// Send 'q' key to start discard flow
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	// Wait for confirmation dialog
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return string(bts) != ""
	}, teatest.WithDuration(time.Second))

	// Send 'n' to cancel discard (keep editing)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})

	// Wait for form to be back in view
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return len(bts) != 0 && !form.ShowConfirmDiscard()
	}, teatest.WithDuration(time.Second))

	// The form should NOT be cancelled
	assert.False(t, form.IsCancelled(), "Form should not be cancelled when keeping editing")
	assert.False(t, form.ShowConfirmDiscard(), "Confirmation dialog should be hidden")
}

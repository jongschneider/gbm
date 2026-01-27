package config

import (
	"gbm/pkg/tui"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/muesli/termenv"
	"github.com/stretchr/testify/assert"
)

// jiraFormModel wraps JiraForm for teatest, quitting on BackBoundaryMsg.
type jiraFormModel struct {
	form *JiraForm
}

func newJiraFormModel(f *JiraForm) *jiraFormModel {
	return &jiraFormModel{form: f}
}

func (m *jiraFormModel) Init() tea.Cmd {
	return m.form.Init()
}

func (m *jiraFormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle BackBoundaryMsg to quit the test program
	if _, ok := msg.(tui.BackBoundaryMsg); ok {
		return m, tea.Quit
	}

	model, cmd := m.form.Update(msg)
	if f, ok := model.(*JiraForm); ok {
		m.form = f
	}
	return m, cmd
}

func (m *jiraFormModel) View() string {
	return m.form.View()
}

// TestJiraForm_DisabledInitialRender tests rendering disabled JIRA form.
func TestJiraForm_DisabledInitialRender(t *testing.T) {
	t.Parallel()
	lipgloss.SetColorProfile(termenv.Ascii)

	config := JiraFormConfig{
		Enabled:  false,
		Host:     "",
		Username: "",
		APIToken: "",
		Theme:    tui.DefaultTheme(),
	}

	form := NewJiraForm(config)
	model := newJiraFormModel(form)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return len(bts) > 0
	}, teatest.WithDuration(time.Second))

	// Check that expected text appears
	finalOutput := ""
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		finalOutput = string(bts)
		return len(finalOutput) > 0
	}, teatest.WithDuration(100*time.Millisecond))

	assert.Contains(t, finalOutput, "JIRA Configuration")
	assert.NotContains(t, finalOutput, "Server Configuration")
}

// TestJiraForm_EnabledInitialRender tests rendering enabled JIRA form.
func TestJiraForm_EnabledInitialRender(t *testing.T) {
	t.Parallel()
	lipgloss.SetColorProfile(termenv.Ascii)

	config := JiraFormConfig{
		Enabled:  true,
		Host:     "https://jira.example.com",
		Username: "user@example.com",
		APIToken: "token123",
		Theme:    tui.DefaultTheme(),
	}

	form := NewJiraForm(config)
	model := newJiraFormModel(form)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return len(bts) > 0
	}, teatest.WithDuration(time.Second))

	// Check that expected text appears
	finalOutput := ""
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		finalOutput = string(bts)
		return len(finalOutput) > 0
	}, teatest.WithDuration(100*time.Millisecond))

	// When enabled, server configuration should be visible
	assert.Contains(t, finalOutput, "Server Configuration")
	assert.Contains(t, finalOutput, "JIRA Host")
}

// TestJiraForm_SaveFlow tests the save flow.
func TestJiraForm_SaveFlow(t *testing.T) {
	t.Parallel()
	lipgloss.SetColorProfile(termenv.Ascii)

	saveCalled := false
	var savedData map[string]interface{}

	config := JiraFormConfig{
		Enabled:  true,
		Host:     "https://jira.example.com",
		Username: "user@example.com",
		APIToken: "token123",
		Theme:    tui.DefaultTheme(),
		OnSave: func(data map[string]interface{}) error {
			saveCalled = true
			savedData = data
			return nil
		},
	}

	form := NewJiraForm(config)
	model := newJiraFormModel(form)

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
	assert.True(t, form.IsComplete(), "Form should be marked as submitted")
}

// TestJiraForm_DiscardFlow tests the discard confirmation flow.
func TestJiraForm_DiscardFlow(t *testing.T) {
	t.Parallel()
	lipgloss.SetColorProfile(termenv.Ascii)

	config := JiraFormConfig{
		Enabled: false,
		Theme:   tui.DefaultTheme(),
	}

	form := NewJiraForm(config)
	model := newJiraFormModel(form)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return len(bts) > 0
	}, teatest.WithDuration(time.Second))

	// Send 'q' key to show discard confirmation
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	// Wait a bit for confirmation to appear
	time.Sleep(50 * time.Millisecond)

	// Verify discard confirmation is shown
	assert.True(t, form.showConfirmDiscard, "Discard confirmation should be shown")

	// Send 'y' key to confirm discard
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})

	// Wait for quit
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

	// Verify cancelled
	assert.True(t, form.IsCancelled(), "Form should be marked as cancelled")
}

// TestJiraForm_KeepEditingFlow tests the keep editing flow in discard confirmation.
func TestJiraForm_KeepEditingFlow(t *testing.T) {
	t.Parallel()
	lipgloss.SetColorProfile(termenv.Ascii)

	config := JiraFormConfig{
		Enabled: false,
		Theme:   tui.DefaultTheme(),
	}

	form := NewJiraForm(config)
	model := newJiraFormModel(form)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return len(bts) > 0
	}, teatest.WithDuration(time.Second))

	// Send 'q' key to show discard confirmation
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	// Wait a bit for confirmation to appear
	time.Sleep(50 * time.Millisecond)

	// Send 'n' key to keep editing
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})

	// Wait a bit
	time.Sleep(50 * time.Millisecond)

	// Verify not cancelled and discard confirmation is hidden
	assert.False(t, form.IsCancelled(), "Form should not be cancelled")
	assert.False(t, form.showConfirmDiscard, "Discard confirmation should be hidden")
}

// TestJiraForm_TabNavigation tests tab navigation between fields.
func TestJiraForm_TabNavigation(t *testing.T) {
	t.Parallel()
	lipgloss.SetColorProfile(termenv.Ascii)

	config := JiraFormConfig{
		Enabled:  true,
		Host:     "https://jira.example.com",
		Username: "user@example.com",
		APIToken: "token123",
		Theme:    tui.DefaultTheme(),
	}

	form := NewJiraForm(config)
	model := newJiraFormModel(form)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return len(bts) > 0
	}, teatest.WithDuration(time.Second))

	initialIdx := form.focusedFieldIdx
	assert.Equal(t, 0, initialIdx, "Should start on first field")

	// Send Tab key
	tm.Send(tea.KeyMsg{Type: tea.KeyTab})

	// Wait for update
	time.Sleep(50 * time.Millisecond)

	// Check that focus moved
	newIdx := form.focusedFieldIdx
	assert.NotEqual(t, initialIdx, newIdx, "Focus should have moved to next field")
}

// TestJiraForm_EscapeKey tests escape for cancellation.
func TestJiraForm_EscapeKey(t *testing.T) {
	t.Parallel()
	lipgloss.SetColorProfile(termenv.Ascii)

	config := JiraFormConfig{
		Enabled: false,
		Theme:   tui.DefaultTheme(),
	}

	form := NewJiraForm(config)
	model := newJiraFormModel(form)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return len(bts) > 0
	}, teatest.WithDuration(time.Second))

	// Send Escape key
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})

	// Wait for quit
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

	// Verify cancelled
	assert.True(t, form.IsCancelled(), "Form should be marked as cancelled")
}

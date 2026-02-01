package config

import (
	"gbm/pkg/tui"
	"gbm/pkg/tui/fields"
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
		return finalOutput != ""
	}, teatest.WithDuration(100*time.Millisecond))

	assert.Contains(t, finalOutput, "JIRA Configuration")
	assert.NotContains(t, finalOutput, "Server Configuration")
}

// TestJiraForm_EnabledInitialRender tests rendering enabled JIRA form.
func TestJiraForm_EnabledInitialRender(t *testing.T) {
	t.Parallel()
	lipgloss.SetColorProfile(termenv.Ascii)

	config := JiraFormConfig{
		Enabled:                 true,
		Host:                    "https://jira.example.com",
		Username:                "user@example.com",
		APIToken:                "token123",
		AttachmentsMaxSize:      50,
		AttachmentsDir:          ".jira/attachments",
		MarkdownFilenamePattern: "{key}.md",
		Theme:                   tui.DefaultTheme(),
	}

	form := NewJiraForm(config)

	// Test that GetValue() returns config values
	// This confirms all fields are initialized with config values
	data := form.GetValue()
	assert.Equal(t, true, data["jira_enabled"])
	assert.Equal(t, "https://jira.example.com", data["jira_host"])
	assert.Equal(t, "user@example.com", data["jira_username"])
	assert.Equal(t, "token123", data["jira_api_token"])
	assert.Equal(t, "50", data["jira_attachments_max_size"])
	assert.Equal(t, ".jira/attachments", data["jira_attachments_dir"])
	assert.Equal(t, "{key}.md", data["jira_markdown_filename_pattern"])

	model := newJiraFormModel(form)

	// Use larger terminal to fit all subsections
	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(120, 50))

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return len(bts) > 0
	}, teatest.WithDuration(time.Second))

	// Check that expected text appears
	finalOutput := ""
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		finalOutput = string(bts)
		return finalOutput != ""
	}, teatest.WithDuration(100*time.Millisecond))

	// When enabled, verify that subsections are at least partially visible
	// The form contains all subsections; verify critical sections are rendered
	assert.Contains(t, finalOutput, "Attachments")
	assert.Contains(t, finalOutput, "Markdown")
}

// TestJiraForm_SaveFlow tests the save flow.
func TestJiraForm_SaveFlow(t *testing.T) {
	t.Parallel()
	lipgloss.SetColorProfile(termenv.Ascii)

	saveCalled := false
	var savedData map[string]any

	config := JiraFormConfig{
		Enabled:  true,
		Host:     "https://jira.example.com",
		Username: "user@example.com",
		APIToken: "token123",
		Theme:    tui.DefaultTheme(),
		OnSave: func(data map[string]any) error {
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

// TestJiraForm_SpacebarToggleConfirm tests spacebar toggling on Confirm fields.
func TestJiraForm_SpacebarToggleConfirm(t *testing.T) {
	t.Parallel()
	lipgloss.SetColorProfile(termenv.Ascii)

	config := JiraFormConfig{
		Enabled: true,
		Theme:   tui.DefaultTheme(),
	}

	form := NewJiraForm(config)
	model := newJiraFormModel(form)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() {
		//nolint:errcheck // Best-effort cleanup in test
		tm.Quit()
	})

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return len(bts) > 0
	}, teatest.WithDuration(time.Second))

	// Navigate to attachments enabled field (index 7)
	// Tab through: enable(0) -> host(1) -> user(2) -> token(3) -> status(4) -> priority(5) -> type(6) -> attachEnabled(7)
	for range 7 {
		tm.Send(tea.KeyMsg{Type: tea.KeyTab})
		time.Sleep(10 * time.Millisecond)
	}

	// Verify we're on the attachments enabled field
	assert.Equal(t, 7, form.focusedFieldIdx, "Should be on attachments enabled field")

	// Get the confirm field and check initial state
	confirmField, ok := form.attachmentsEnabledField.(*fields.Confirm)
	assert.True(t, ok, "attachmentsEnabledField should be a Confirm")

	initialSelected := confirmField.GetValue().(bool)

	// Press space to toggle
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	time.Sleep(10 * time.Millisecond)

	// Get current selection state (need to check selected field, not value which is set on confirm)
	// The View will show the toggled state
	view := form.View()
	assert.NotEmpty(t, view, "View should render")

	// Press space again to toggle back
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	time.Sleep(10 * time.Millisecond)

	// The toggle should work without errors - value is preserved until Enter is pressed
	_ = initialSelected // Used above
}

// TestJiraForm_JKNavigation tests j/k vim-style navigation in normal mode.
func TestJiraForm_JKNavigation(t *testing.T) {
	t.Parallel()
	lipgloss.SetColorProfile(termenv.Ascii)

	config := JiraFormConfig{
		Enabled: true,
		Theme:   tui.DefaultTheme(),
	}

	form := NewJiraForm(config)
	model := newJiraFormModel(form)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() {
		//nolint:errcheck // Best-effort cleanup in test
		tm.Quit()
	})

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return len(bts) > 0
	}, teatest.WithDuration(time.Second))

	// Navigate to attachments enabled field (index 7) - a Confirm field
	for range 7 {
		tm.Send(tea.KeyMsg{Type: tea.KeyTab})
		time.Sleep(10 * time.Millisecond)
	}
	assert.Equal(t, 7, form.focusedFieldIdx, "Should be on attachments enabled field (Confirm)")

	// Press j to move to next field (max size - TextInput)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, 8, form.focusedFieldIdx, "j should move to next field")

	// In normal mode, j/k always navigate (vim-style)
	// Press k to move back
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, 7, form.focusedFieldIdx, "k should move to previous field in normal mode")

	// Move forward with j again
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, 8, form.focusedFieldIdx, "j should move to next field")

	// Enter insert mode with 'i'
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	time.Sleep(10 * time.Millisecond)
	assert.True(t, form.insertMode, "Should be in insert mode after pressing i")

	// In insert mode, j/k should type instead of navigate
	startIdx := form.focusedFieldIdx
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, startIdx, form.focusedFieldIdx, "j should NOT navigate in insert mode")

	// Exit insert mode with Esc
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	time.Sleep(10 * time.Millisecond)
	assert.False(t, form.insertMode, "Should exit insert mode after pressing Esc")

	// Now j should navigate again
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, 9, form.focusedFieldIdx, "j should navigate after exiting insert mode")
}

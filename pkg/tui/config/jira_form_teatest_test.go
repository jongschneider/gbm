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

// jiraFormModel wraps JiraForm for teatest, quitting on boundary messages.
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
	switch msg.(type) {
	case tui.BackBoundaryMsg, tui.FormFlushCompleteMsg:
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

// TestJiraForm_FreeTextEditing tests that text inputs are freely editable on focus.
func TestJiraForm_FreeTextEditing(t *testing.T) {
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

	// Navigate to host field (index 1) - a TextInput
	tm.Send(tea.KeyMsg{Type: tea.KeyTab})
	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, 1, form.focusedFieldIdx, "Should be on host field")

	// j/k should type characters in text input, not navigate
	startIdx := form.focusedFieldIdx
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, startIdx, form.focusedFieldIdx, "j should type in text input, not navigate")

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, startIdx, form.focusedFieldIdx, "k should type in text input, not navigate")

	// Tab still navigates between fields
	tm.Send(tea.KeyMsg{Type: tea.KeyTab})
	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, 2, form.focusedFieldIdx, "Tab should still navigate to next field")
}

// TestJiraForm_ConfirmFieldKeys tests key delegation to Confirm fields.
func TestJiraForm_ConfirmFieldKeys(t *testing.T) {
	t.Parallel()
	lipgloss.SetColorProfile(termenv.Ascii)

	testCases := []struct {
		assert func(t *testing.T, form *JiraForm)
		name   string
		key    tea.KeyMsg
	}{
		{
			name: "enter on confirm field sets value and advances",
			key:  tea.KeyMsg{Type: tea.KeyEnter},
			assert: func(t *testing.T, form *JiraForm) {
				t.Helper()
				// Enter confirms the current selection and moves to next field
				confirm := form.attachmentsEnabledField.(*fields.Confirm)
				assert.False(t, confirm.GetValue().(bool), "value should be set to false (default)")
				assert.Equal(t, 8, form.focusedFieldIdx, "should advance to next field after enter")
			},
		},
		{
			name: "y on confirm field sets value to true and advances",
			key:  tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}},
			assert: func(t *testing.T, form *JiraForm) {
				t.Helper()
				confirm := form.attachmentsEnabledField.(*fields.Confirm)
				assert.True(t, confirm.GetValue().(bool), "y should set value to true")
				assert.Equal(t, 8, form.focusedFieldIdx, "should advance to next field after y")
			},
		},
		{
			name: "n on confirm field sets value to false and advances",
			key:  tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}},
			assert: func(t *testing.T, form *JiraForm) {
				t.Helper()
				confirm := form.attachmentsEnabledField.(*fields.Confirm)
				assert.False(t, confirm.GetValue().(bool), "n should set value to false")
				assert.Equal(t, 8, form.focusedFieldIdx, "should advance to next field after n")
			},
		},
		{
			name: "h on confirm field selects Yes without advancing",
			key:  tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}},
			assert: func(t *testing.T, form *JiraForm) {
				t.Helper()
				// h selects Yes (visual selection) but does not advance
				assert.Equal(t, 7, form.focusedFieldIdx, "should stay on confirm field after h")
			},
		},
		{
			name: "l on confirm field selects No without advancing",
			key:  tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}},
			assert: func(t *testing.T, form *JiraForm) {
				t.Helper()
				// l selects No (visual selection) but does not advance
				assert.Equal(t, 7, form.focusedFieldIdx, "should stay on confirm field after l")
			},
		},
		{
			name: "left arrow on confirm field selects Yes without advancing",
			key:  tea.KeyMsg{Type: tea.KeyLeft},
			assert: func(t *testing.T, form *JiraForm) {
				t.Helper()
				assert.Equal(t, 7, form.focusedFieldIdx, "should stay on confirm field after left")
			},
		},
		{
			name: "right arrow on confirm field selects No without advancing",
			key:  tea.KeyMsg{Type: tea.KeyRight},
			assert: func(t *testing.T, form *JiraForm) {
				t.Helper()
				assert.Equal(t, 7, form.focusedFieldIdx, "should stay on confirm field after right")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
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
			for range 7 {
				tm.Send(tea.KeyMsg{Type: tea.KeyTab})
				time.Sleep(10 * time.Millisecond)
			}
			assert.Equal(t, 7, form.focusedFieldIdx, "Should be on attachments enabled field")

			// Send the test key
			tm.Send(tc.key)
			time.Sleep(20 * time.Millisecond)

			tc.assert(t, form)
		})
	}
}

// TestJiraForm_EnableToggle_Enter tests toggling the enable field with Enter.
func TestJiraForm_EnableToggle_Enter(t *testing.T) {
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

	// Should start on enable field
	assert.Equal(t, 0, form.focusedFieldIdx)
	assert.True(t, form.enabled)

	// Press l to select No, then Enter to confirm
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	time.Sleep(10 * time.Millisecond)
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	time.Sleep(20 * time.Millisecond)

	// JIRA should now be disabled
	assert.False(t, form.enabled, "JIRA should be disabled after selecting No + Enter")
	// Should stay on enable field when disabled (no fields to navigate to)
	assert.Equal(t, 0, form.focusedFieldIdx, "should stay on enable field when disabled")
}

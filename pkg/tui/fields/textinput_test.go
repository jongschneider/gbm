package fields

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTextInput_FocusPreservesValue(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		setup  func() *TextInput
		assert func(t *testing.T, ti *TextInput)
	}{
		{
			name: "focus-blur-focus cycle preserves user edits",
			setup: func() *TextInput {
				ti := NewTextInput("branch", "Default Branch", "The default branch")
				ti.WithDefault("main")
				ti.Focus()
				// Simulate user editing the value
				ti.textInput.SetValue("develop")
				ti.Blur()
				// Re-focus (e.g., Tab back to this field)
				ti.Focus()
				return ti
			},
			assert: func(t *testing.T, ti *TextInput) {
				assert.Equal(t, "develop", ti.textInput.Value())
				assert.Equal(t, "develop", ti.GetValue())
			},
		},
		{
			name: "multiple focus-blur cycles preserve value",
			setup: func() *TextInput {
				ti := NewTextInput("host", "JIRA Host", "JIRA server URL")
				ti.WithDefault("https://jira.old.com")
				ti.Focus()
				ti.textInput.SetValue("https://jira.example.com")
				ti.Blur()
				// Cycle through several focus/blur rounds
				for range 5 {
					ti.Focus()
					ti.Blur()
				}
				ti.Focus()
				return ti
			},
			assert: func(t *testing.T, ti *TextInput) {
				assert.Equal(t, "https://jira.example.com", ti.textInput.Value())
			},
		},
		{
			name: "default value is populated on first focus",
			setup: func() *TextInput {
				ti := NewTextInput("branch", "Default Branch", "")
				ti.WithDefault("main")
				ti.Focus()
				return ti
			},
			assert: func(t *testing.T, ti *TextInput) {
				assert.Equal(t, "main", ti.textInput.Value())
				assert.Equal(t, "main", ti.GetValue())
			},
		},
		{
			name: "empty field stays empty after focus cycles",
			setup: func() *TextInput {
				ti := NewTextInput("key", "Title", "")
				ti.Focus()
				ti.Blur()
				ti.Focus()
				return ti
			},
			assert: func(t *testing.T, ti *TextInput) {
				assert.Equal(t, "", ti.textInput.Value())
			},
		},
		{
			name: "user clears default value and it stays cleared",
			setup: func() *TextInput {
				ti := NewTextInput("branch", "Default Branch", "")
				ti.WithDefault("main")
				ti.Focus()
				ti.textInput.SetValue("")
				ti.Blur()
				ti.Focus()
				return ti
			},
			assert: func(t *testing.T, ti *TextInput) {
				assert.Equal(t, "", ti.textInput.Value())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ti := tc.setup()
			tc.assert(t, ti)
		})
	}
}

func TestTextInput_FocusReturnsBlink(t *testing.T) {
	t.Parallel()
	ti := NewTextInput("key", "Title", "")
	cmd := ti.Focus()
	require.NotNil(t, cmd, "Focus() should return a blink command")
}

func TestTextInput_FocusSetsState(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		action func(ti *TextInput)
		assert func(t *testing.T, ti *TextInput)
	}{
		{
			name:   "Focus sets focused to true",
			action: func(ti *TextInput) { ti.Focus() },
			assert: func(t *testing.T, ti *TextInput) {
				assert.True(t, ti.focused)
			},
		},
		{
			name: "Blur sets focused to false",
			action: func(ti *TextInput) {
				ti.Focus()
				ti.Blur()
			},
			assert: func(t *testing.T, ti *TextInput) {
				assert.False(t, ti.focused)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ti := NewTextInput("key", "Title", "")
			tc.action(ti)
			tc.assert(t, ti)
		})
	}
}

func TestTextInput_GetValue(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		setup  func() *TextInput
		assert func(t *testing.T, val any)
	}{
		{
			name: "returns default value before any interaction",
			setup: func() *TextInput {
				ti := NewTextInput("key", "Title", "")
				ti.WithDefault("default-val")
				return ti
			},
			assert: func(t *testing.T, val any) {
				assert.Equal(t, "default-val", val)
			},
		},
		{
			name: "returns edited value after focus-edit-blur-focus",
			setup: func() *TextInput {
				ti := NewTextInput("key", "Title", "")
				ti.WithDefault("original")
				ti.Focus()
				ti.textInput.SetValue("edited")
				ti.Blur()
				ti.Focus()
				return ti
			},
			assert: func(t *testing.T, val any) {
				assert.Equal(t, "edited", val)
			},
		},
		{
			name: "returns confirmed value after Enter",
			setup: func() *TextInput {
				ti := NewTextInput("key", "Title", "")
				ti.Focus()
				ti.textInput.SetValue("submitted")
				ti.Update(tea.KeyMsg{Type: tea.KeyEnter})
				return ti
			},
			assert: func(t *testing.T, val any) {
				assert.Equal(t, "submitted", val)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ti := tc.setup()
			tc.assert(t, ti.GetValue())
		})
	}
}

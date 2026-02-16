package config

import (
	"gbm/pkg/tui"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestJiraForm_EscEmitsBackBoundaryMsg(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name            string
		focusedFieldIdx int
		enabled         bool
	}{
		{
			name:            "from enable field (first, confirm)",
			focusedFieldIdx: 0,
			enabled:         true,
		},
		{
			name:            "from host field (text input)",
			focusedFieldIdx: 1,
			enabled:         true,
		},
		{
			name:            "from attachments enabled field (confirm, middle)",
			focusedFieldIdx: 7,
			enabled:         true,
		},
		{
			name:            "from last field",
			focusedFieldIdx: 13,
			enabled:         true,
		},
		{
			name:            "from enable field when disabled",
			focusedFieldIdx: 0,
			enabled:         false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			form := NewJiraForm(JiraFormConfig{
				Enabled: tc.enabled,
				Host:    "https://jira.example.com",
				Theme:   tui.DefaultTheme(),
			})
			form.focusedFieldIdx = tc.focusedFieldIdx

			_, cmd := form.Update(tea.KeyMsg{Type: tea.KeyEsc})

			assert.NotNil(t, cmd, "Esc should return a command")
			msg := cmd()
			_, ok := msg.(tui.BackBoundaryMsg)
			assert.True(t, ok, "command should produce BackBoundaryMsg, got %T", msg)
		})
	}
}

func TestJiraForm_Create(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		assert func(t *testing.T, form *JiraForm)
		name   string
		config JiraFormConfig
	}{
		{
			name: "disabled form starts on enable field",
			config: JiraFormConfig{
				Enabled: false,
			},
			assert: func(t *testing.T, form *JiraForm) {
				t.Helper()
				assert.False(t, form.enabled)
				assert.Equal(t, 0, form.focusedFieldIdx)
			},
		},
		{
			name: "enabled form starts on enable field",
			config: JiraFormConfig{
				Enabled: true,
			},
			assert: func(t *testing.T, form *JiraForm) {
				t.Helper()
				assert.True(t, form.enabled)
				assert.Equal(t, 0, form.focusedFieldIdx)
			},
		},
		{
			name: "uses default theme when not provided",
			config: JiraFormConfig{
				Theme: nil,
			},
			assert: func(t *testing.T, form *JiraForm) {
				t.Helper()
				assert.NotNil(t, form.theme)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			form := NewJiraForm(tc.config)
			tc.assert(t, form)
		})
	}
}

func TestJiraForm_Validate(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		config     JiraFormConfig
		expectErrs int
	}{
		{
			name: "disabled form has no validation errors",
			config: JiraFormConfig{
				Enabled: false,
			},
			expectErrs: 0,
		},
		{
			name: "enabled form with valid values passes",
			config: JiraFormConfig{
				Enabled:  true,
				Host:     "https://jira.example.com",
				Username: "user@example.com",
				APIToken: "token123",
			},
			expectErrs: 0,
		},
		{
			name: "enabled form with empty fields fails",
			config: JiraFormConfig{
				Enabled:  true,
				Host:     "",
				Username: "",
				APIToken: "",
			},
			expectErrs: 3,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			form := NewJiraForm(tc.config)
			errs := form.Validate()
			assert.Len(t, errs, tc.expectErrs)
		})
	}
}

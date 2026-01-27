package config

import (
	"testing"

	"gbm/pkg/tui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestBasicsForm_Create(t *testing.T) {
	testCases := []struct {
		name   string
		config BasicsFormConfig
		expect func(t *testing.T, form *BasicsForm)
	}{
		{
			name: "creates with default values",
			config: BasicsFormConfig{
				DefaultBranch: "main",
				WorktreesDir:  "./worktrees",
			},
			expect: func(t *testing.T, form *BasicsForm) {
				t.Helper()

				assert.NotNil(t, form.defaultBranchField)
				assert.NotNil(t, form.worktreesDirField)
				assert.Equal(t, 0, form.focusedFieldIdx)
				assert.False(t, form.submitted)
				assert.False(t, form.cancelled)
			},
		},
		{
			name: "uses provided theme",
			config: BasicsFormConfig{
				Theme: tui.DefaultTheme(),
			},
			expect: func(t *testing.T, form *BasicsForm) {
				t.Helper()

				assert.NotNil(t, form.theme)
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			form := NewBasicsForm(tc.config)
			tc.expect(t, form)
		})
	}
}

func TestBasicsForm_GetValue(t *testing.T) {
	t.Parallel()

	form := NewBasicsForm(BasicsFormConfig{
		DefaultBranch: "develop",
		WorktreesDir:  "/home/user/worktrees",
	})

	// Note: Values are only set when user presses Enter.
	// Initially they should be empty.
	values := form.GetValue()
	assert.Empty(t, values["default_branch"])
	assert.Empty(t, values["worktrees_dir"])

	// Simulate user input and submission
	form.defaultBranchField = form.defaultBranchField.WithTheme(form.theme)
	form.worktreesDirField = form.worktreesDirField.WithTheme(form.theme)
}

func TestBasicsForm_View(t *testing.T) {
	t.Parallel()

	form := NewBasicsForm(BasicsFormConfig{
		DefaultBranch: "main",
		WorktreesDir:  "./worktrees",
	})

	view := form.View()
	assert.NotEmpty(t, view)
	assert.Contains(t, view, "Basic Settings")
	assert.Contains(t, view, "Default Branch")
	assert.Contains(t, view, "Worktrees Directory")
}

func TestBasicsForm_TabNavigation(t *testing.T) {
	t.Parallel()

	form := NewBasicsForm(BasicsFormConfig{})

	// Initially focused on first field
	assert.Equal(t, 0, form.focusedFieldIdx)

	// Tab moves to next field
	form.Update(tea.KeyMsg{Type: tea.KeyTab})
	assert.Equal(t, 1, form.focusedFieldIdx)

	// Tab again wraps to first field
	form.Update(tea.KeyMsg{Type: tea.KeyTab})
	assert.Equal(t, 0, form.focusedFieldIdx)
}

func TestBasicsForm_ShiftTabNavigation(t *testing.T) {
	t.Parallel()

	form := NewBasicsForm(BasicsFormConfig{})

	// Move to last field
	form.Update(tea.KeyMsg{Type: tea.KeyTab})
	assert.Equal(t, 1, form.focusedFieldIdx)

	// Shift+Tab goes back
	form.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	assert.Equal(t, 0, form.focusedFieldIdx)

	// Shift+Tab from first wraps to last
	form.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	assert.Equal(t, 1, form.focusedFieldIdx)
}

func TestBasicsForm_Complete(t *testing.T) {
	t.Parallel()

	form := NewBasicsForm(BasicsFormConfig{})

	assert.False(t, form.IsComplete())

	// Mark as submitted
	form.submitted = true
	assert.True(t, form.IsComplete())
}

func TestBasicsForm_Cancelled(t *testing.T) {
	t.Parallel()

	form := NewBasicsForm(BasicsFormConfig{})

	assert.False(t, form.IsCancelled())

	// Mark as cancelled
	form.cancelled = true
	assert.True(t, form.IsCancelled())
}

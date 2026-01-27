package config

import (
	"gbm/pkg/tui"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

// TestBasicsForm_HelpOverlay tests the help overlay functionality.
func TestBasicsForm_HelpOverlay(t *testing.T) {
	t.Parallel()

	t.Run("show help overlay with ? key", func(t *testing.T) {
		t.Parallel()
		form := NewBasicsForm(BasicsFormConfig{
			DefaultBranch: "main",
			WorktreesDir:  "./worktrees",
		})

		assert.False(t, form.showHelp)

		// Press '?' to show help
		form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
		assert.True(t, form.showHelp)
		assert.NotNil(t, form.helpOverlay)

		// View should show help overlay
		view := form.View()
		assert.Contains(t, view, "Navigation")
	})

	t.Run("dismiss help with Escape", func(t *testing.T) {
		t.Parallel()
		form := NewBasicsForm(BasicsFormConfig{
			DefaultBranch: "main",
			WorktreesDir:  "./worktrees",
		})

		// Show help
		form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
		assert.True(t, form.showHelp)

		// Dismiss with Escape
		form.Update(tea.KeyMsg{Type: tea.KeyEsc})
		assert.False(t, form.showHelp)
	})

	t.Run("dismiss help with ? key again", func(t *testing.T) {
		t.Parallel()
		form := NewBasicsForm(BasicsFormConfig{
			DefaultBranch: "main",
			WorktreesDir:  "./worktrees",
		})

		// Show help
		form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
		assert.True(t, form.showHelp)

		// Dismiss with '?' again
		form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
		assert.False(t, form.showHelp)
	})

	t.Run("dismiss help with Enter", func(t *testing.T) {
		t.Parallel()
		form := NewBasicsForm(BasicsFormConfig{
			DefaultBranch: "main",
			WorktreesDir:  "./worktrees",
		})

		// Show help
		form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
		assert.True(t, form.showHelp)

		// Dismiss with Enter
		form.Update(tea.KeyMsg{Type: tea.KeyEnter})
		assert.False(t, form.showHelp)
	})

	t.Run("ignore non-key messages in help overlay", func(t *testing.T) {
		t.Parallel()
		form := NewBasicsForm(BasicsFormConfig{
			DefaultBranch: "main",
			WorktreesDir:  "./worktrees",
		})

		// Show help
		form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
		assert.True(t, form.showHelp)

		// Send non-key message - should do nothing
		form.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
		assert.True(t, form.showHelp) // Still showing help
	})
}

// TestJiraForm_HelpOverlay tests the help overlay functionality.
func TestJiraForm_HelpOverlay(t *testing.T) {
	t.Parallel()

	t.Run("show help overlay with ? key", func(t *testing.T) {
		t.Parallel()
		form := NewJiraForm(JiraFormConfig{
			Theme: tui.DefaultTheme(),
		})

		assert.False(t, form.showHelp)

		// Press '?' to show help
		form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
		assert.True(t, form.showHelp)
		assert.NotNil(t, form.helpOverlay)
	})

	t.Run("dismiss help with Escape", func(t *testing.T) {
		t.Parallel()
		form := NewJiraForm(JiraFormConfig{
			Theme: tui.DefaultTheme(),
		})

		// Show help
		form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
		assert.True(t, form.showHelp)

		// Dismiss with Escape
		form.Update(tea.KeyMsg{Type: tea.KeyEsc})
		assert.False(t, form.showHelp)
	})

	t.Run("dismiss help with ? key again", func(t *testing.T) {
		t.Parallel()
		form := NewJiraForm(JiraFormConfig{
			Theme: tui.DefaultTheme(),
		})

		// Show help
		form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
		assert.True(t, form.showHelp)

		// Dismiss with '?' again
		form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
		assert.False(t, form.showHelp)
	})

	t.Run("dismiss help with Enter", func(t *testing.T) {
		t.Parallel()
		form := NewJiraForm(JiraFormConfig{
			Theme: tui.DefaultTheme(),
		})

		// Show help
		form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
		assert.True(t, form.showHelp)

		// Dismiss with Enter
		form.Update(tea.KeyMsg{Type: tea.KeyEnter})
		assert.False(t, form.showHelp)
	})

	t.Run("ignore non-key messages in help overlay", func(t *testing.T) {
		t.Parallel()
		form := NewJiraForm(JiraFormConfig{
			Theme: tui.DefaultTheme(),
		})

		// Show help
		form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
		assert.True(t, form.showHelp)

		// Send non-key message - should do nothing
		form.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
		assert.True(t, form.showHelp) // Still showing help
	})
}

// TestJiraForm_ValidationOverlay tests the validation overlay functionality.
func TestJiraForm_ValidationOverlay(t *testing.T) {
	t.Parallel()

	t.Run("show validation overlay on invalid save", func(t *testing.T) {
		t.Parallel()
		form := NewJiraForm(JiraFormConfig{
			Enabled: true,
			Theme:   tui.DefaultTheme(),
			// No host/username/token provided - should fail validation
		})

		assert.False(t, form.showValidationErrors)

		// Try to save - should trigger validation errors
		form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
		assert.True(t, form.showValidationErrors)
		assert.NotNil(t, form.validationOverlay)
	})

	t.Run("dismiss validation overlay with Escape", func(t *testing.T) {
		t.Parallel()
		form := NewJiraForm(JiraFormConfig{
			Enabled: true,
			Theme:   tui.DefaultTheme(),
		})

		// Trigger validation errors
		form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
		assert.True(t, form.showValidationErrors)

		// Dismiss with Escape
		form.Update(tea.KeyMsg{Type: tea.KeyEsc})
		assert.False(t, form.showValidationErrors)
	})

	t.Run("dismiss validation overlay with 'b' key", func(t *testing.T) {
		t.Parallel()
		form := NewJiraForm(JiraFormConfig{
			Enabled: true,
			Theme:   tui.DefaultTheme(),
		})

		// Trigger validation errors
		form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
		assert.True(t, form.showValidationErrors)

		// Dismiss with 'b'
		form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
		assert.False(t, form.showValidationErrors)
	})

	t.Run("dismiss validation overlay with Enter", func(t *testing.T) {
		t.Parallel()
		form := NewJiraForm(JiraFormConfig{
			Enabled: true,
			Theme:   tui.DefaultTheme(),
		})

		// Trigger validation errors
		form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
		assert.True(t, form.showValidationErrors)

		// Dismiss with Enter
		form.Update(tea.KeyMsg{Type: tea.KeyEnter})
		assert.False(t, form.showValidationErrors)
	})
}

// TestJiraForm_Enabled tests the Enabled() accessor.
func TestJiraForm_Enabled(t *testing.T) {
	t.Parallel()

	t.Run("returns false when disabled", func(t *testing.T) {
		t.Parallel()
		form := NewJiraForm(JiraFormConfig{
			Enabled: false,
			Theme:   tui.DefaultTheme(),
		})

		assert.False(t, form.Enabled())
	})

	t.Run("returns true when enabled", func(t *testing.T) {
		t.Parallel()
		form := NewJiraForm(JiraFormConfig{
			Enabled: true,
			Theme:   tui.DefaultTheme(),
		})

		assert.True(t, form.Enabled())
	})
}

// TestWorktreesForm_HelpOverlay tests the help overlay functionality.
func TestWorktreesForm_HelpOverlay(t *testing.T) {
	t.Parallel()

	t.Run("show help overlay with ? key", func(t *testing.T) {
		t.Parallel()
		form := NewWorktreesForm(WorktreesFormConfig{
			Theme: tui.DefaultTheme(),
		})

		assert.Equal(t, WorktreeModalNone, form.modalState)

		// Press '?' to show help
		form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
		assert.Equal(t, WorktreeModalHelp, form.modalState)
		assert.NotNil(t, form.helpOverlay)
	})

	t.Run("dismiss help with Escape", func(t *testing.T) {
		t.Parallel()
		form := NewWorktreesForm(WorktreesFormConfig{
			Theme: tui.DefaultTheme(),
		})

		// Show help
		form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
		assert.Equal(t, WorktreeModalHelp, form.modalState)

		// Dismiss with Escape
		form.Update(tea.KeyMsg{Type: tea.KeyEsc})
		assert.Equal(t, WorktreeModalNone, form.modalState)
		assert.Nil(t, form.helpOverlay)
	})

	t.Run("dismiss help with ? key again", func(t *testing.T) {
		t.Parallel()
		form := NewWorktreesForm(WorktreesFormConfig{
			Theme: tui.DefaultTheme(),
		})

		// Show help
		form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
		assert.Equal(t, WorktreeModalHelp, form.modalState)

		// Dismiss with '?' again
		form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
		assert.Equal(t, WorktreeModalNone, form.modalState)
	})

	t.Run("dismiss help with Enter", func(t *testing.T) {
		t.Parallel()
		form := NewWorktreesForm(WorktreesFormConfig{
			Theme: tui.DefaultTheme(),
		})

		// Show help
		form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
		assert.Equal(t, WorktreeModalHelp, form.modalState)

		// Dismiss with Enter
		form.Update(tea.KeyMsg{Type: tea.KeyEnter})
		assert.Equal(t, WorktreeModalNone, form.modalState)
	})

	t.Run("ignore non-key messages in help modal", func(t *testing.T) {
		t.Parallel()
		form := NewWorktreesForm(WorktreesFormConfig{
			Theme: tui.DefaultTheme(),
		})

		// Show help
		form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
		assert.Equal(t, WorktreeModalHelp, form.modalState)

		// Send non-key message - should do nothing
		form.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
		assert.Equal(t, WorktreeModalHelp, form.modalState) // Still showing help
	})
}

// TestWorktreesForm_Init tests the Init method.
func TestWorktreesForm_Init(t *testing.T) {
	t.Parallel()

	form := NewWorktreesForm(WorktreesFormConfig{
		Theme: tui.DefaultTheme(),
	})

	// Init should return the table's init command
	cmd := form.Init()
	// The table's init command may be nil or a batch command
	// Just ensure it doesn't panic
	_ = cmd
}

// TestFileCopyForm_HelpOverlay tests the help overlay functionality.
func TestFileCopyForm_HelpOverlay(t *testing.T) {
	t.Parallel()

	t.Run("show help overlay with ? key", func(t *testing.T) {
		t.Parallel()
		form := NewFileCopyForm(FileCopyFormConfig{
			Theme: tui.DefaultTheme(),
		})

		assert.Equal(t, ModalNone, form.modalState)

		// Press '?' to show help
		form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
		assert.Equal(t, ModalHelp, form.modalState)
		assert.NotNil(t, form.helpOverlay)
	})

	t.Run("dismiss help with Escape", func(t *testing.T) {
		t.Parallel()
		form := NewFileCopyForm(FileCopyFormConfig{
			Theme: tui.DefaultTheme(),
		})

		// Show help
		form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
		assert.Equal(t, ModalHelp, form.modalState)

		// Dismiss with Escape
		form.Update(tea.KeyMsg{Type: tea.KeyEsc})
		assert.Equal(t, ModalNone, form.modalState)
		assert.Nil(t, form.helpOverlay)
	})

	t.Run("dismiss help with ? key again", func(t *testing.T) {
		t.Parallel()
		form := NewFileCopyForm(FileCopyFormConfig{
			Theme: tui.DefaultTheme(),
		})

		// Show help
		form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
		assert.Equal(t, ModalHelp, form.modalState)

		// Dismiss with '?' again
		form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
		assert.Equal(t, ModalNone, form.modalState)
	})

	t.Run("dismiss help with Enter", func(t *testing.T) {
		t.Parallel()
		form := NewFileCopyForm(FileCopyFormConfig{
			Theme: tui.DefaultTheme(),
		})

		// Show help
		form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
		assert.Equal(t, ModalHelp, form.modalState)

		// Dismiss with Enter
		form.Update(tea.KeyMsg{Type: tea.KeyEnter})
		assert.Equal(t, ModalNone, form.modalState)
	})

	t.Run("ignore non-key messages in help modal", func(t *testing.T) {
		t.Parallel()
		form := NewFileCopyForm(FileCopyFormConfig{
			Theme: tui.DefaultTheme(),
		})

		// Show help
		form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
		assert.Equal(t, ModalHelp, form.modalState)

		// Send non-key message - should do nothing
		form.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
		assert.Equal(t, ModalHelp, form.modalState) // Still showing help
	})
}

package fields

import (
	"errors"
	"gbm/pkg/tui"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTextInput_BuilderMethods tests all builder/accessor methods on TextInput.
func TestTextInput_BuilderMethods(t *testing.T) {
	t.Parallel()

	t.Run("WithPlaceholder", func(t *testing.T) {
		t.Parallel()
		ti := NewTextInput("key", "title", "description")
		result := ti.WithPlaceholder("Enter text...")
		assert.Same(t, ti, result)
		assert.Equal(t, "Enter text...", ti.textInput.Placeholder)
	})

	t.Run("SetMasked true", func(t *testing.T) {
		t.Parallel()
		ti := NewTextInput("key", "title", "description")
		result := ti.SetMasked(true)
		assert.Same(t, ti, result)
		assert.True(t, ti.masked)
	})

	t.Run("SetMasked false", func(t *testing.T) {
		t.Parallel()
		ti := NewTextInput("key", "title", "description")
		ti.SetMasked(true)
		ti.SetMasked(false)
		assert.False(t, ti.masked)
	})

	t.Run("Error", func(t *testing.T) {
		t.Parallel()
		ti := NewTextInput("key", "title", "description")
		require.NoError(t, ti.Error())
		ti.err = errors.New("test error")
		require.EqualError(t, ti.Error(), "test error")
	})

	t.Run("Skip", func(t *testing.T) {
		t.Parallel()
		ti := NewTextInput("key", "title", "description")
		assert.False(t, ti.Skip())
	})

	t.Run("WithTheme", func(t *testing.T) {
		t.Parallel()
		ti := NewTextInput("key", "title", "description")
		theme := tui.DefaultTheme()
		result := ti.WithTheme(theme)
		assert.Equal(t, theme, ti.theme)
		// Verify fluent interface
		_, ok := result.(*TextInput)
		assert.True(t, ok)
	})

	t.Run("WithWidth", func(t *testing.T) {
		t.Parallel()
		ti := NewTextInput("key", "title", "description")
		result := ti.WithWidth(100)
		assert.Equal(t, 100, ti.width)
		assert.Equal(t, 96, ti.textInput.Width) // width - 4
		_, ok := result.(*TextInput)
		assert.True(t, ok)
	})

	t.Run("WithHeight", func(t *testing.T) {
		t.Parallel()
		ti := NewTextInput("key", "title", "description")
		result := ti.WithHeight(50)
		assert.Equal(t, 50, ti.height)
		_, ok := result.(*TextInput)
		assert.True(t, ok)
	})

	t.Run("GetKey", func(t *testing.T) {
		t.Parallel()
		ti := NewTextInput("my_key", "title", "description")
		assert.Equal(t, "my_key", ti.GetKey())
	})

	t.Run("RunValidator with no validator", func(t *testing.T) {
		t.Parallel()
		ti := NewTextInput("key", "title", "description")
		err := ti.RunValidator()
		assert.NoError(t, err)
	})

	t.Run("RunValidator with passing validator", func(t *testing.T) {
		t.Parallel()
		ti := NewTextInput("key", "title", "description")
		ti.WithValidator(func(s string) error {
			return nil
		})
		ti.textInput.SetValue("valid")
		err := ti.RunValidator()
		assert.NoError(t, err)
	})

	t.Run("RunValidator with failing validator", func(t *testing.T) {
		t.Parallel()
		ti := NewTextInput("key", "title", "description")
		ti.WithValidator(func(s string) error {
			if s == "" {
				return errors.New("value required")
			}
			return nil
		})
		err := ti.RunValidator()
		require.EqualError(t, err, "value required")
		require.EqualError(t, ti.Error(), "value required")
	})

	t.Run("SetError", func(t *testing.T) {
		t.Parallel()
		ti := NewTextInput("key", "title", "description")
		require.NoError(t, ti.Error())
		ti.SetError(errors.New("external error"))
		require.EqualError(t, ti.Error(), "external error")
	})
}

// TestConfirm_BuilderMethods tests all builder/accessor methods on Confirm.
func TestConfirm_BuilderMethods(t *testing.T) {
	t.Parallel()

	t.Run("SetValue true", func(t *testing.T) {
		t.Parallel()
		c := NewConfirm("key", "title")
		result := c.SetValue(true)
		assert.Same(t, c, result)
		assert.True(t, c.selected)
		assert.True(t, c.value)
	})

	t.Run("SetValue false", func(t *testing.T) {
		t.Parallel()
		c := NewConfirm("key", "title")
		result := c.SetValue(false)
		assert.Same(t, c, result)
		assert.False(t, c.selected)
		assert.False(t, c.value)
	})

	t.Run("Error", func(t *testing.T) {
		t.Parallel()
		c := NewConfirm("key", "title")
		// Confirm.Error() always returns nil
		assert.NoError(t, c.Error())
	})

	t.Run("Skip", func(t *testing.T) {
		t.Parallel()
		c := NewConfirm("key", "title")
		assert.False(t, c.Skip())
	})

	t.Run("WithTheme", func(t *testing.T) {
		t.Parallel()
		c := NewConfirm("key", "title")
		theme := tui.DefaultTheme()
		result := c.WithTheme(theme)
		assert.Equal(t, theme, c.theme)
		_, ok := result.(*Confirm)
		assert.True(t, ok)
	})

	t.Run("WithWidth", func(t *testing.T) {
		t.Parallel()
		c := NewConfirm("key", "title")
		result := c.WithWidth(100)
		assert.Equal(t, 100, c.width)
		_, ok := result.(*Confirm)
		assert.True(t, ok)
	})

	t.Run("WithHeight", func(t *testing.T) {
		t.Parallel()
		c := NewConfirm("key", "title")
		result := c.WithHeight(50)
		assert.Equal(t, 50, c.height)
		_, ok := result.(*Confirm)
		assert.True(t, ok)
	})

	t.Run("GetKey", func(t *testing.T) {
		t.Parallel()
		c := NewConfirm("my_key", "title")
		assert.Equal(t, "my_key", c.GetKey())
	})

	t.Run("ResetCompletion clears complete and cancelled", func(t *testing.T) {
		t.Parallel()
		c := NewConfirm("key", "title")
		c.Focus()

		// Simulate completing the field via Enter
		c.Update(tea.KeyMsg{Type: tea.KeyEnter})
		assert.True(t, c.IsComplete(), "should be complete after enter")

		// Reset
		c.ResetCompletion()
		assert.False(t, c.IsComplete(), "should not be complete after reset")
		assert.False(t, c.IsCancelled(), "should not be cancelled after reset")
	})

	t.Run("ResetCompletion clears cancelled from n key", func(t *testing.T) {
		t.Parallel()
		c := NewConfirm("key", "title")
		c.Focus()

		// Simulate cancelling via n key
		c.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
		assert.True(t, c.IsComplete(), "should be complete after n")
		assert.True(t, c.IsCancelled(), "should be cancelled after n")

		// Reset
		c.ResetCompletion()
		assert.False(t, c.IsComplete(), "should not be complete after reset")
		assert.False(t, c.IsCancelled(), "should not be cancelled after reset")
	})
}

// TestSelector_BuilderMethods tests all builder/accessor methods on Selector.
func TestSelector_BuilderMethods(t *testing.T) {
	t.Parallel()
	options := []Option{
		{Label: "Option 1", Value: "opt1"},
		{Label: "Option 2", Value: "opt2"},
	}

	t.Run("Error", func(t *testing.T) {
		t.Parallel()
		s := NewSelector("key", "title", options)
		// Selector.Error() always returns nil
		assert.NoError(t, s.Error())
	})

	t.Run("Skip", func(t *testing.T) {
		t.Parallel()
		s := NewSelector("key", "title", options)
		assert.False(t, s.Skip())
	})

	t.Run("WithTheme", func(t *testing.T) {
		t.Parallel()
		s := NewSelector("key", "title", options)
		theme := tui.DefaultTheme()
		result := s.WithTheme(theme)
		assert.Equal(t, theme, s.theme)
		_, ok := result.(*Selector)
		assert.True(t, ok)
	})

	t.Run("WithWidth", func(t *testing.T) {
		t.Parallel()
		s := NewSelector("key", "title", options)
		result := s.WithWidth(100)
		assert.Equal(t, 100, s.width)
		_, ok := result.(*Selector)
		assert.True(t, ok)
	})

	t.Run("WithHeight", func(t *testing.T) {
		t.Parallel()
		s := NewSelector("key", "title", options)
		result := s.WithHeight(50)
		assert.Equal(t, 50, s.height)
		_, ok := result.(*Selector)
		assert.True(t, ok)
	})

	t.Run("GetKey", func(t *testing.T) {
		t.Parallel()
		s := NewSelector("my_key", "title", options)
		assert.Equal(t, "my_key", s.GetKey())
	})
}

// TestFilterable_BuilderMethods tests builder/accessor methods on Filterable.
func TestFilterable_BuilderMethods(t *testing.T) {
	t.Parallel()
	options := []Option{
		{Label: "Item 1", Value: "item1"},
		{Label: "Item 2", Value: "item2"},
	}

	t.Run("Error", func(t *testing.T) {
		t.Parallel()
		f := NewFilterable("key", "title", "description", options)
		// Filterable.Error() always returns nil
		assert.NoError(t, f.Error())
	})

	t.Run("Skip", func(t *testing.T) {
		t.Parallel()
		f := NewFilterable("key", "title", "description", options)
		assert.False(t, f.Skip())
	})

	t.Run("WithTheme", func(t *testing.T) {
		t.Parallel()
		f := NewFilterable("key", "title", "description", options)
		theme := tui.DefaultTheme()
		result := f.WithTheme(theme)
		assert.Equal(t, theme, f.theme)
		_, ok := result.(*Filterable)
		assert.True(t, ok)
	})

	t.Run("WithWidth", func(t *testing.T) {
		t.Parallel()
		f := NewFilterable("key", "title", "description", options)
		result := f.WithWidth(100)
		assert.Equal(t, 100, f.width)
		_, ok := result.(*Filterable)
		assert.True(t, ok)
	})

	t.Run("GetKey", func(t *testing.T) {
		t.Parallel()
		f := NewFilterable("my_key", "title", "description", options)
		assert.Equal(t, "my_key", f.GetKey())
	})
}

// TestFilePicker_BuilderMethods tests builder/accessor methods on FilePicker.
func TestFilePicker_BuilderMethods(t *testing.T) {
	t.Parallel()

	t.Run("Init", func(t *testing.T) {
		t.Parallel()
		fp := NewFilePicker("key", "title", "/tmp")
		cmd := fp.Init()
		// FilePicker.Init() returns the filepicker's init command
		assert.NotNil(t, cmd)
	})

	t.Run("Update without focus", func(t *testing.T) {
		t.Parallel()
		fp := NewFilePicker("key", "title", "/tmp")
		// Not focused - should return without processing
		model, cmd := fp.Update(tea.KeyMsg{Type: tea.KeyEnter})
		assert.Same(t, fp, model)
		assert.Nil(t, cmd)
	})

	t.Run("Update with focus - unhandled key", func(t *testing.T) {
		t.Parallel()
		fp := NewFilePicker("key", "title", "/tmp")
		fp.Focus()
		// Unhandled key should be passed to filepicker
		_, _ = fp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
		// Just verify no panic
	})
}

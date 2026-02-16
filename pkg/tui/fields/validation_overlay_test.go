package fields

import (
	"gbm/pkg/tui"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidationOverlay_New(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		expect func(t *testing.T, v *ValidationOverlay)
		name   string
		errors []string
	}{
		{
			name:   "creates with single error",
			errors: []string{"field is required"},
			expect: func(t *testing.T, v *ValidationOverlay) {
				t.Helper()
				assert.Len(t, v.errors, 1)
				assert.Equal(t, "field is required", v.errors[0])
				assert.Equal(t, "Validation Errors", v.title)
				assert.NotNil(t, v.theme)
			},
		},
		{
			name:   "creates with multiple errors",
			errors: []string{"error 1", "error 2", "error 3"},
			expect: func(t *testing.T, v *ValidationOverlay) {
				t.Helper()
				assert.Len(t, v.errors, 3)
			},
		},
		{
			name:   "creates with empty errors",
			errors: []string{},
			expect: func(t *testing.T, v *ValidationOverlay) {
				t.Helper()
				assert.Empty(t, v.errors)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			overlay := NewValidationOverlay(tc.errors)
			tc.expect(t, overlay)
		})
	}
}

func TestValidationOverlay_WithTheme(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		theme  *tui.Theme
		expect func(t *testing.T, v *ValidationOverlay)
		name   string
	}{
		{
			name:  "sets custom theme",
			theme: tui.DefaultTheme(),
			expect: func(t *testing.T, v *ValidationOverlay) {
				t.Helper()
				assert.NotNil(t, v.theme)
			},
		},
		{
			name:  "ignores nil theme",
			theme: nil,
			expect: func(t *testing.T, v *ValidationOverlay) {
				t.Helper()
				// Should keep original default theme
				assert.NotNil(t, v.theme)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			overlay := NewValidationOverlay([]string{"error"})
			result := overlay.WithTheme(tc.theme)
			// Verify fluent interface returns same instance
			assert.Same(t, overlay, result)
			tc.expect(t, overlay)
		})
	}
}

func TestValidationOverlay_WithTitle(t *testing.T) {
	t.Parallel()
	overlay := NewValidationOverlay([]string{"error"})

	// Default title
	assert.Equal(t, "Validation Errors", overlay.title)

	// Set custom title
	result := overlay.WithTitle("Save Error")
	assert.Same(t, overlay, result)
	assert.Equal(t, "Save Error", overlay.title)
}

func TestValidationOverlay_WithWidth(t *testing.T) {
	t.Parallel()
	overlay := NewValidationOverlay([]string{"error"})

	assert.Equal(t, 0, overlay.width)

	result := overlay.WithWidth(100)
	assert.Same(t, overlay, result)
	assert.Equal(t, 100, overlay.width)
}

func TestValidationOverlay_WithHeight(t *testing.T) {
	t.Parallel()
	overlay := NewValidationOverlay([]string{"error"})

	assert.Equal(t, 0, overlay.height)

	result := overlay.WithHeight(50)
	assert.Same(t, overlay, result)
	assert.Equal(t, 50, overlay.height)
}

func TestValidationOverlay_Init(t *testing.T) {
	t.Parallel()
	overlay := NewValidationOverlay([]string{"error"})

	cmd := overlay.Init()
	assert.Nil(t, cmd)
}

func TestValidationOverlay_Update_Dismissal(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name string
		key  tea.KeyMsg
	}{
		{
			name: "dismiss with Escape",
			key:  tea.KeyMsg{Type: tea.KeyEsc},
		},
		{
			name: "dismiss with Enter",
			key:  tea.KeyMsg{Type: tea.KeyEnter},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			overlay := NewValidationOverlay([]string{"error"})

			model, cmd := overlay.Update(tc.key)
			assert.Same(t, overlay, model)
			require.NotNil(t, cmd)

			// Execute the command and verify it returns dismissal message
			msg := cmd()
			_, ok := msg.(ValidationOverlayDismissedMsg)
			assert.True(t, ok, "expected ValidationOverlayDismissedMsg")
		})
	}
}

func TestValidationOverlay_Update_WindowSize(t *testing.T) {
	t.Parallel()
	overlay := NewValidationOverlay([]string{"error"})

	assert.Equal(t, 0, overlay.width)
	assert.Equal(t, 0, overlay.height)

	model, cmd := overlay.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	assert.Same(t, overlay, model)
	assert.Nil(t, cmd)
	assert.Equal(t, 120, overlay.width)
	assert.Equal(t, 40, overlay.height)
}

func TestValidationOverlay_Update_UnhandledKey(t *testing.T) {
	t.Parallel()
	overlay := NewValidationOverlay([]string{"error"})

	// Unhandled key should do nothing
	model, cmd := overlay.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	assert.Same(t, overlay, model)
	assert.Nil(t, cmd)
}

func TestValidationOverlay_Update_BKeyDoesNotDismiss(t *testing.T) {
	t.Parallel()
	overlay := NewValidationOverlay([]string{"error"})

	model, cmd := overlay.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	assert.Same(t, overlay, model)
	assert.Nil(t, cmd, "'b' key should not dismiss the validation overlay")
}

func TestValidationOverlay_View(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name           string
		errors         []string
		title          string
		expectContains []string
	}{
		{
			name:           "renders single error",
			errors:         []string{"field is required"},
			title:          "Validation Errors",
			expectContains: []string{"Validation Errors", "field is required", "Press Escape or Enter to dismiss"},
		},
		{
			name:           "renders multiple errors",
			errors:         []string{"error 1", "error 2"},
			title:          "Validation Errors",
			expectContains: []string{"error 1", "error 2"},
		},
		{
			name:           "renders custom title",
			errors:         []string{"save failed"},
			title:          "Save Error",
			expectContains: []string{"Save Error", "save failed"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			overlay := NewValidationOverlay(tc.errors).WithTitle(tc.title)

			view := overlay.View()
			for _, expected := range tc.expectContains {
				assert.Contains(t, view, expected)
			}
		})
	}
}

func TestValidationOverlay_View_NarrowWidth(t *testing.T) {
	t.Parallel()
	overlay := NewValidationOverlay([]string{"error"}).WithWidth(40)

	// Should not panic and should render
	view := overlay.View()
	assert.NotEmpty(t, view)
}

func TestValidationOverlay_Errors(t *testing.T) {
	t.Parallel()
	errors := []string{"error 1", "error 2"}
	overlay := NewValidationOverlay(errors)

	result := overlay.Errors()
	assert.Equal(t, errors, result)
}

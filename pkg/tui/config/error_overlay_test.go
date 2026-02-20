package config

import (
	"gbm/pkg/tui"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sampleErrors() []ValidationError {
	return []ValidationError{
		{
			Tab:        TabGeneral,
			FieldKey:   "default_branch",
			FieldLabel: "Default Branch",
			Message:    "this field is required",
			FieldIndex: 0,
		},
		{
			Tab:        TabGeneral,
			FieldKey:   "worktrees_dir",
			FieldLabel: "Worktrees Directory",
			Message:    "invalid template variable '{foo}'",
			FieldIndex: 1,
		},
		{
			Tab:        TabJira,
			FieldKey:   "jira.attachments.max_size_mb",
			FieldLabel: "Max Size (MB)",
			Message:    "must be a positive integer",
			FieldIndex: 13,
		},
	}
}

func TestNewErrorOverlay(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, o *ErrorOverlay)
		name   string
		errs   []ValidationError
	}{
		{
			name: "with errors",
			errs: sampleErrors(),
			assert: func(t *testing.T, o *ErrorOverlay) {
				t.Helper()
				assert.True(t, o.HasErrors())
				assert.Len(t, o.Errors(), 3)
				assert.Equal(t, 0, o.Cursor())
			},
		},
		{
			name: "nil errors",
			errs: nil,
			assert: func(t *testing.T, o *ErrorOverlay) {
				t.Helper()
				assert.False(t, o.HasErrors())
				assert.Empty(t, o.Errors())
			},
		},
		{
			name: "nil theme uses default",
			errs: nil,
			assert: func(t *testing.T, o *ErrorOverlay) {
				t.Helper()
				require.NotNil(t, o.theme)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			o := NewErrorOverlay(tc.errs, nil)
			tc.assert(t, o)
		})
	}
}

func TestErrorOverlay_SetErrors(t *testing.T) {
	o := NewErrorOverlay(sampleErrors(), tui.DefaultTheme())

	// Navigate to position 2.
	o.HandleKey(runeKey('j'))
	o.HandleKey(runeKey('j'))
	assert.Equal(t, 2, o.Cursor())

	// Replace errors -- cursor should reset.
	newErrs := []ValidationError{
		{FieldLabel: "Host", Message: "required"},
	}
	o.SetErrors(newErrs)

	assert.Equal(t, 0, o.Cursor())
	assert.Len(t, o.Errors(), 1)
}

func TestErrorOverlay_Navigation(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, o *ErrorOverlay)
		name   string
		keys   []tea.KeyMsg
	}{
		{
			name: "down moves cursor forward",
			keys: []tea.KeyMsg{{Type: tea.KeyDown}},
			assert: func(t *testing.T, o *ErrorOverlay) {
				t.Helper()
				assert.Equal(t, 1, o.Cursor())
			},
		},
		{
			name: "j moves cursor forward",
			keys: []tea.KeyMsg{runeKey('j')},
			assert: func(t *testing.T, o *ErrorOverlay) {
				t.Helper()
				assert.Equal(t, 1, o.Cursor())
			},
		},
		{
			name: "up at start wraps to last",
			keys: []tea.KeyMsg{{Type: tea.KeyUp}},
			assert: func(t *testing.T, o *ErrorOverlay) {
				t.Helper()
				assert.Equal(t, 2, o.Cursor())
			},
		},
		{
			name: "k at start wraps to last",
			keys: []tea.KeyMsg{runeKey('k')},
			assert: func(t *testing.T, o *ErrorOverlay) {
				t.Helper()
				assert.Equal(t, 2, o.Cursor())
			},
		},
		{
			name: "down wraps from last to first",
			keys: []tea.KeyMsg{
				{Type: tea.KeyDown},
				{Type: tea.KeyDown},
				{Type: tea.KeyDown},
			},
			assert: func(t *testing.T, o *ErrorOverlay) {
				t.Helper()
				assert.Equal(t, 0, o.Cursor())
			},
		},
		{
			name: "down then up returns to start",
			keys: []tea.KeyMsg{
				{Type: tea.KeyDown},
				{Type: tea.KeyUp},
			},
			assert: func(t *testing.T, o *ErrorOverlay) {
				t.Helper()
				assert.Equal(t, 0, o.Cursor())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			o := NewErrorOverlay(sampleErrors(), tui.DefaultTheme())
			for _, k := range tc.keys {
				o.HandleKey(k)
			}
			tc.assert(t, o)
		})
	}
}

func TestErrorOverlay_HandleKey_Actions(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, action ErrorOverlayAction, o *ErrorOverlay)
		name   string
		key    tea.KeyMsg
	}{
		{
			name: "esc returns close action",
			key:  escKey(),
			assert: func(t *testing.T, action ErrorOverlayAction, _ *ErrorOverlay) {
				t.Helper()
				assert.Equal(t, ErrorActionClose, action)
			},
		},
		{
			name: "enter returns jump action",
			key:  enterKey(),
			assert: func(t *testing.T, action ErrorOverlayAction, _ *ErrorOverlay) {
				t.Helper()
				assert.Equal(t, ErrorActionJump, action)
			},
		},
		{
			name: "down returns none action",
			key:  tea.KeyMsg{Type: tea.KeyDown},
			assert: func(t *testing.T, action ErrorOverlayAction, _ *ErrorOverlay) {
				t.Helper()
				assert.Equal(t, ErrorActionNone, action)
			},
		},
		{
			name: "up returns none action",
			key:  tea.KeyMsg{Type: tea.KeyUp},
			assert: func(t *testing.T, action ErrorOverlayAction, _ *ErrorOverlay) {
				t.Helper()
				assert.Equal(t, ErrorActionNone, action)
			},
		},
		{
			name: "other key returns none action",
			key:  runeKey('x'),
			assert: func(t *testing.T, action ErrorOverlayAction, _ *ErrorOverlay) {
				t.Helper()
				assert.Equal(t, ErrorActionNone, action)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			o := NewErrorOverlay(sampleErrors(), tui.DefaultTheme())
			action := o.HandleKey(tc.key)
			tc.assert(t, action, o)
		})
	}
}

func TestErrorOverlay_HandleKey_EnterWithNoErrors(t *testing.T) {
	o := NewErrorOverlay(nil, tui.DefaultTheme())
	action := o.HandleKey(enterKey())
	assert.Equal(t, ErrorActionClose, action, "enter with no errors should close")
}

func TestErrorOverlay_SelectedError(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, sel ValidationError)
		setup  func(o *ErrorOverlay)
		name   string
	}{
		{
			name:  "first error selected by default",
			setup: func(_ *ErrorOverlay) {},
			assert: func(t *testing.T, sel ValidationError) {
				t.Helper()
				assert.Equal(t, "default_branch", sel.FieldKey)
			},
		},
		{
			name: "after navigating down",
			setup: func(o *ErrorOverlay) {
				o.HandleKey(runeKey('j'))
			},
			assert: func(t *testing.T, sel ValidationError) {
				t.Helper()
				assert.Equal(t, "worktrees_dir", sel.FieldKey)
			},
		},
		{
			name: "after navigating to last",
			setup: func(o *ErrorOverlay) {
				o.HandleKey(runeKey('j'))
				o.HandleKey(runeKey('j'))
			},
			assert: func(t *testing.T, sel ValidationError) {
				t.Helper()
				assert.Equal(t, "jira.attachments.max_size_mb", sel.FieldKey)
				assert.Equal(t, TabJira, sel.Tab)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			o := NewErrorOverlay(sampleErrors(), tui.DefaultTheme())
			tc.setup(o)
			sel := o.SelectedError()
			tc.assert(t, sel)
		})
	}
}

func TestErrorOverlay_SelectedError_Empty(t *testing.T) {
	o := NewErrorOverlay(nil, tui.DefaultTheme())
	sel := o.SelectedError()
	assert.Equal(t, -1, sel.FieldIndex)
}

func TestErrorOverlay_View(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, view string)
		name   string
		errs   []ValidationError
		width  int
		height int
	}{
		{
			name:   "shows title with error count",
			errs:   sampleErrors(),
			width:  80,
			height: 40,
			assert: func(t *testing.T, view string) {
				t.Helper()
				assert.Contains(t, view, "Validation Errors (3)")
			},
		},
		{
			name:   "shows close hint",
			errs:   sampleErrors(),
			width:  80,
			height: 40,
			assert: func(t *testing.T, view string) {
				t.Helper()
				assert.Contains(t, view, "esc to close")
				assert.Contains(t, view, "enter to jump")
			},
		},
		{
			name:   "shows error messages",
			errs:   sampleErrors(),
			width:  80,
			height: 40,
			assert: func(t *testing.T, view string) {
				t.Helper()
				assert.Contains(t, view, "Default Branch")
				assert.Contains(t, view, "this field is required")
				assert.Contains(t, view, "Worktrees Directory")
				assert.Contains(t, view, "Max Size (MB)")
			},
		},
		{
			name:   "shows tab labels in brackets",
			errs:   sampleErrors(),
			width:  80,
			height: 40,
			assert: func(t *testing.T, view string) {
				t.Helper()
				assert.Contains(t, view, "[General]")
				assert.Contains(t, view, "[JIRA]")
			},
		},
		{
			name:   "shows cursor indicator on selected error",
			errs:   sampleErrors(),
			width:  80,
			height: 40,
			assert: func(t *testing.T, view string) {
				t.Helper()
				assert.Contains(t, view, ">")
			},
		},
		{
			name:   "no errors shows success message",
			errs:   nil,
			width:  80,
			height: 40,
			assert: func(t *testing.T, view string) {
				t.Helper()
				assert.Contains(t, view, "No validation errors")
			},
		},
		{
			name:   "small viewport does not panic",
			errs:   sampleErrors(),
			width:  30,
			height: 5,
			assert: func(t *testing.T, view string) {
				t.Helper()
				assert.NotEmpty(t, view)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			o := NewErrorOverlay(tc.errs, tui.DefaultTheme())
			view := o.View(tc.width, tc.height)
			tc.assert(t, view)
		})
	}
}

func TestErrorOverlay_NavigationWithEmptyList(t *testing.T) {
	o := NewErrorOverlay(nil, tui.DefaultTheme())

	// Should not panic on navigation with empty list.
	o.HandleKey(runeKey('j'))
	o.HandleKey(runeKey('k'))
	o.HandleKey(tea.KeyMsg{Type: tea.KeyDown})
	o.HandleKey(tea.KeyMsg{Type: tea.KeyUp})

	assert.Equal(t, 0, o.Cursor())
}

func TestConfigModel_ErrorOverlayIntegration(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, m *ConfigModel)
		name   string
		setup  func(m *ConfigModel)
		keys   []tea.KeyMsg
	}{
		{
			name: "ShowErrorOverlay switches to errors state",
			setup: func(m *ConfigModel) {
				m.ShowErrorOverlay(sampleErrors())
			},
			keys: nil,
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.Equal(t, StateErrors, m.State())
				assert.True(t, m.tabBadges[TabGeneral])
				assert.True(t, m.tabBadges[TabJira])
			},
		},
		{
			name: "esc closes error overlay",
			setup: func(m *ConfigModel) {
				m.ShowErrorOverlay(sampleErrors())
			},
			keys: []tea.KeyMsg{escKey()},
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.Equal(t, StateBrowsing, m.State())
			},
		},
		{
			name: "enter on error jumps to tab",
			setup: func(m *ConfigModel) {
				m.ShowErrorOverlay(sampleErrors())
				// Navigate to the JIRA error (index 2).
				m.errorOverlay.HandleKey(runeKey('j'))
				m.errorOverlay.HandleKey(runeKey('j'))
			},
			keys: []tea.KeyMsg{enterKey()},
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.Equal(t, StateBrowsing, m.State())
				assert.Equal(t, TabJira, m.ActiveTab())
			},
		},
		{
			name: "navigation in errors state does not switch tabs",
			setup: func(m *ConfigModel) {
				m.ShowErrorOverlay(sampleErrors())
			},
			keys: []tea.KeyMsg{
				{Type: tea.KeyDown},
				{Type: tea.KeyDown},
			},
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.Equal(t, StateErrors, m.State())
				assert.Equal(t, TabGeneral, m.ActiveTab(), "tab should not change during errors navigation")
			},
		},
		{
			name: "empty errors is no-op",
			setup: func(m *ConfigModel) {
				m.ShowErrorOverlay(nil)
			},
			keys: nil,
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.Equal(t, StateBrowsing, m.State())
			},
		},
		{
			name: "ClearValidationErrors resets badges",
			setup: func(m *ConfigModel) {
				m.ShowErrorOverlay(sampleErrors())
				m.state = StateBrowsing // close overlay
				m.ClearValidationErrors()
			},
			keys: nil,
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				for i := range tabCount {
					assert.False(t, m.tabBadges[i], "tab %d badge should be cleared", i)
				}
				assert.False(t, m.errorOverlay.HasErrors())
			},
		},
		{
			name: "error overlay view renders when in errors state",
			setup: func(m *ConfigModel) {
				m.ShowErrorOverlay(sampleErrors())
			},
			keys: nil,
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				view := m.View()
				assert.Contains(t, view, "Validation Errors")
			},
		},
		{
			name: "tab bar shows badges after ShowErrorOverlay",
			setup: func(m *ConfigModel) {
				m.ShowErrorOverlay(sampleErrors())
				m.state = StateBrowsing // view the tab bar
			},
			keys: nil,
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				tabBar := m.viewTabBar()
				assert.Contains(t, tabBar, "General (!)")
				assert.Contains(t, tabBar, "JIRA (!)")
				assert.NotContains(t, tabBar, "File Copy (!)")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := NewConfigModel()
			m.width = 80
			m.height = 40
			tc.setup(m)

			var result tea.Model = m
			for _, k := range tc.keys {
				result, _ = result.Update(k)
			}
			updated := result.(*ConfigModel)
			tc.assert(t, updated)
		})
	}
}

func TestFieldRow_ErrorMarker(t *testing.T) {
	testCases := []struct {
		assert   func(t *testing.T, view string)
		name     string
		hasError bool
		dirty    bool
	}{
		{
			name:     "error marker shows exclamation",
			hasError: true,
			dirty:    false,
			assert: func(t *testing.T, view string) {
				t.Helper()
				stripped := stripAnsi(view)
				assert.Contains(t, stripped, "!")
			},
		},
		{
			name:     "dirty takes precedence over error",
			hasError: true,
			dirty:    true,
			assert: func(t *testing.T, view string) {
				t.Helper()
				stripped := stripAnsi(view)
				prefix := stripped[:4]
				assert.Contains(t, prefix, "*", "dirty marker should take precedence")
			},
		},
		{
			name:     "no error no dirty shows space",
			hasError: false,
			dirty:    false,
			assert: func(t *testing.T, view string) {
				t.Helper()
				stripped := stripAnsi(view)
				prefix := stripped[:4]
				assert.NotContains(t, prefix, "!")
				assert.NotContains(t, prefix, "*")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fr := newTestFieldRow(String)
			fr.SetValue("test")
			fr.SetHasError(tc.hasError)
			fr.SetDirty(tc.dirty)
			view := fr.View()
			tc.assert(t, view)
		})
	}
}

func TestFieldRow_ErrorAccessors(t *testing.T) {
	fr := newTestFieldRow(String)

	assert.False(t, fr.HasError())
	assert.Empty(t, fr.EditError())

	fr.SetHasError(true)
	assert.True(t, fr.HasError())

	fr.SetHasError(false)
	assert.False(t, fr.HasError())
}

func TestFieldRow_FocusedWithErrorMarker(t *testing.T) {
	fr := newTestFieldRow(String)
	fr.SetValue("test")
	fr.SetFocused(true)
	fr.SetHasError(true)

	view := fr.View()

	// Should still show cursor and the error marker.
	assert.Contains(t, view, ">")
	stripped := stripAnsi(view)
	assert.Contains(t, stripped, "!")
}

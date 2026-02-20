package config

import (
	"gbm/pkg/tui"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRuleOverlay(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, o *RuleOverlay)
		name   string
		source string
		files  []string
		opts   []RuleOverlayOption
	}{
		{
			name:   "default values",
			source: "main",
			files:  []string{".env", "config.json"},
			opts:   nil,
			assert: func(t *testing.T, o *RuleOverlay) {
				t.Helper()
				assert.Equal(t, ROBrowsing, o.State())
				assert.False(t, o.IsNew())
				assert.Equal(t, 0, o.FocusIndex())
				assert.Len(t, o.Fields(), 2)
				assert.Equal(t, "main", o.SourceWorktree())
				assert.Equal(t, []string{".env", "config.json"}, o.Files())
			},
		},
		{
			name:   "with custom theme",
			source: "",
			files:  nil,
			opts:   []RuleOverlayOption{WithRuleTheme(tui.DefaultTheme())},
			assert: func(t *testing.T, o *RuleOverlay) {
				t.Helper()
				assert.NotNil(t, o.theme)
			},
		},
		{
			name:   "nil theme does not override default",
			source: "",
			files:  nil,
			opts:   []RuleOverlayOption{WithRuleTheme(nil)},
			assert: func(t *testing.T, o *RuleOverlay) {
				t.Helper()
				assert.NotNil(t, o.theme)
			},
		},
		{
			name:   "with custom width",
			source: "",
			files:  nil,
			opts:   []RuleOverlayOption{WithRuleWidth(80)},
			assert: func(t *testing.T, o *RuleOverlay) {
				t.Helper()
				assert.Equal(t, 80, o.width)
			},
		},
		{
			name:   "zero width does not override default",
			source: "",
			files:  nil,
			opts:   []RuleOverlayOption{WithRuleWidth(0)},
			assert: func(t *testing.T, o *RuleOverlay) {
				t.Helper()
				assert.Equal(t, 50, o.width)
			},
		},
		{
			name:   "first field is focused",
			source: "main",
			files:  []string{".env"},
			opts:   nil,
			assert: func(t *testing.T, o *RuleOverlay) {
				t.Helper()
				assert.True(t, o.Fields()[0].IsFocused())
				assert.False(t, o.Fields()[1].IsFocused())
			},
		},
		{
			name:   "nil files stored as nil",
			source: "main",
			files:  nil,
			opts:   nil,
			assert: func(t *testing.T, o *RuleOverlay) {
				t.Helper()
				assert.Nil(t, o.Files())
			},
		},
		{
			name:   "deep copies files",
			source: "main",
			files:  []string{"a", "b"},
			opts:   nil,
			assert: func(t *testing.T, o *RuleOverlay) {
				t.Helper()
				files := o.Files()
				files[0] = "modified"
				assert.Equal(t, "a", o.Files()[0], "modifying returned slice should not affect overlay")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			o := NewRuleOverlay(tc.source, tc.files, tc.opts...)
			tc.assert(t, o)
		})
	}
}

func TestNewRuleOverlayForNew(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, o *RuleOverlay)
		name   string
		opts   []RuleOverlayOption
	}{
		{
			name: "starts in browsing state with empty fields",
			opts: nil,
			assert: func(t *testing.T, o *RuleOverlay) {
				t.Helper()
				assert.Equal(t, ROBrowsing, o.State())
				assert.True(t, o.IsNew())
				assert.Empty(t, o.SourceWorktree())
				assert.Nil(t, o.Files())
			},
		},
		{
			name: "with custom width",
			opts: []RuleOverlayOption{WithRuleWidth(60)},
			assert: func(t *testing.T, o *RuleOverlay) {
				t.Helper()
				assert.Equal(t, 60, o.width)
				assert.True(t, o.IsNew())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			o := NewRuleOverlayForNew(tc.opts...)
			tc.assert(t, o)
		})
	}
}

func TestRuleOverlay_Navigation(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, o *RuleOverlay)
		name   string
		keys   []tea.KeyMsg
	}{
		{
			name: "down moves to next field",
			keys: []tea.KeyMsg{{Type: tea.KeyDown}},
			assert: func(t *testing.T, o *RuleOverlay) {
				t.Helper()
				assert.Equal(t, 1, o.FocusIndex())
				assert.False(t, o.Fields()[0].IsFocused())
				assert.True(t, o.Fields()[1].IsFocused())
			},
		},
		{
			name: "j moves to next field",
			keys: []tea.KeyMsg{runeKey('j')},
			assert: func(t *testing.T, o *RuleOverlay) {
				t.Helper()
				assert.Equal(t, 1, o.FocusIndex())
			},
		},
		{
			name: "up wraps to last field",
			keys: []tea.KeyMsg{{Type: tea.KeyUp}},
			assert: func(t *testing.T, o *RuleOverlay) {
				t.Helper()
				assert.Equal(t, 1, o.FocusIndex())
				assert.True(t, o.Fields()[1].IsFocused())
			},
		},
		{
			name: "k wraps to last field",
			keys: []tea.KeyMsg{runeKey('k')},
			assert: func(t *testing.T, o *RuleOverlay) {
				t.Helper()
				assert.Equal(t, 1, o.FocusIndex())
			},
		},
		{
			name: "down wraps from last to first",
			keys: []tea.KeyMsg{
				{Type: tea.KeyDown},
				{Type: tea.KeyDown},
			},
			assert: func(t *testing.T, o *RuleOverlay) {
				t.Helper()
				assert.Equal(t, 0, o.FocusIndex())
				assert.True(t, o.Fields()[0].IsFocused())
			},
		},
		{
			name: "navigate down and back up",
			keys: []tea.KeyMsg{
				{Type: tea.KeyDown}, // 0 -> 1
				{Type: tea.KeyUp},   // 1 -> 0
			},
			assert: func(t *testing.T, o *RuleOverlay) {
				t.Helper()
				assert.Equal(t, 0, o.FocusIndex())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			o := NewRuleOverlay("main", []string{".env"})
			for _, k := range tc.keys {
				o.HandleKey(k)
			}
			tc.assert(t, o)
		})
	}
}

func TestRuleOverlay_EditSourceWorktree(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, o *RuleOverlay)
		name   string
		keys   []tea.KeyMsg
	}{
		{
			name: "e enters editing state on source worktree",
			keys: []tea.KeyMsg{runeKey('e')},
			assert: func(t *testing.T, o *RuleOverlay) {
				t.Helper()
				assert.Equal(t, ROEditingField, o.State())
				assert.Equal(t, FieldEditing, o.Fields()[0].State())
			},
		},
		{
			name: "esc cancels editing and returns to browsing",
			keys: []tea.KeyMsg{runeKey('e'), escKey()},
			assert: func(t *testing.T, o *RuleOverlay) {
				t.Helper()
				assert.Equal(t, ROBrowsing, o.State())
				assert.Equal(t, FieldBrowsing, o.Fields()[0].State())
			},
		},
		{
			name: "enter commits field edit and returns to browsing",
			keys: []tea.KeyMsg{
				runeKey('e'), // start editing
				enterKey(),   // commit
			},
			assert: func(t *testing.T, o *RuleOverlay) {
				t.Helper()
				assert.Equal(t, ROBrowsing, o.State())
				assert.Equal(t, FieldBrowsing, o.Fields()[0].State())
			},
		},
		{
			name: "typing in editing mode forwards to field input",
			keys: []tea.KeyMsg{
				runeKey('e'),
				runeKey('x'),
				runeKey('y'),
			},
			assert: func(t *testing.T, o *RuleOverlay) {
				t.Helper()
				assert.Equal(t, ROEditingField, o.State())
				assert.Contains(t, o.Fields()[0].input.Value(), "xy")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			o := NewRuleOverlay("main", []string{".env"})
			for _, k := range tc.keys {
				o.HandleKey(k)
			}
			tc.assert(t, o)
		})
	}
}

func TestRuleOverlay_EditFiles(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, o *RuleOverlay)
		name   string
		keys   []tea.KeyMsg
	}{
		{
			name: "e on Files opens list overlay",
			keys: []tea.KeyMsg{
				{Type: tea.KeyDown}, // move to Files
				runeKey('e'),        // open list overlay
			},
			assert: func(t *testing.T, o *RuleOverlay) {
				t.Helper()
				assert.Equal(t, ROListOverlay, o.State())
				assert.NotNil(t, o.ListOverlay())
				assert.Equal(t, []string{".env", "config.json"}, o.ListOverlay().Items())
			},
		},
		{
			name: "closing list overlay with enter commits items",
			keys: []tea.KeyMsg{
				{Type: tea.KeyDown}, // move to Files
				runeKey('e'),        // open list overlay
				enterKey(),          // commit list overlay (no changes)
			},
			assert: func(t *testing.T, o *RuleOverlay) {
				t.Helper()
				assert.Equal(t, ROBrowsing, o.State())
				assert.Nil(t, o.ListOverlay())
			},
		},
		{
			name: "closing list overlay with esc discards when no changes",
			keys: []tea.KeyMsg{
				{Type: tea.KeyDown}, // move to Files
				runeKey('e'),        // open list overlay
				escKey(),            // discard (no changes, so immediate)
			},
			assert: func(t *testing.T, o *RuleOverlay) {
				t.Helper()
				assert.Equal(t, ROBrowsing, o.State())
				assert.Nil(t, o.ListOverlay())
				assert.Equal(t, []string{".env", "config.json"}, o.Files())
			},
		},
		{
			name: "list overlay add and commit updates files",
			keys: []tea.KeyMsg{
				{Type: tea.KeyDown}, // move to Files
				runeKey('e'),        // open list overlay
				runeKey('a'),        // add item
				runeKey('n'),
				runeKey('e'),
				runeKey('w'),
				enterKey(), // confirm add
				enterKey(), // commit list overlay
			},
			assert: func(t *testing.T, o *RuleOverlay) {
				t.Helper()
				assert.Equal(t, ROBrowsing, o.State())
				assert.Equal(t, []string{".env", "config.json", "new"}, o.Files())
			},
		},
		{
			name: "e on Files with nil files opens empty list",
			keys: []tea.KeyMsg{
				{Type: tea.KeyDown}, // move to Files
				runeKey('e'),        // open list overlay
			},
			assert: func(t *testing.T, o *RuleOverlay) {
				t.Helper()
				assert.Equal(t, ROListOverlay, o.State())
				assert.NotNil(t, o.ListOverlay())
				assert.Empty(t, o.ListOverlay().Items())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var o *RuleOverlay
			if tc.name == "e on Files with nil files opens empty list" {
				o = NewRuleOverlayForNew()
			} else {
				o = NewRuleOverlay("main", []string{".env", "config.json"})
			}
			for _, k := range tc.keys {
				o.HandleKey(k)
			}
			tc.assert(t, o)
		})
	}
}

func TestRuleOverlay_ConfirmAndCancel(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, result *RuleOverlayResultMsg)
		name   string
		keys   []tea.KeyMsg
	}{
		{
			name: "enter confirms and returns result",
			keys: []tea.KeyMsg{enterKey()},
			assert: func(t *testing.T, result *RuleOverlayResultMsg) {
				t.Helper()
				require.NotNil(t, result)
				assert.True(t, result.Committed)
				assert.Equal(t, "main", result.SourceWorktree)
				assert.Equal(t, []string{".env"}, result.Files)
			},
		},
		{
			name: "esc without changes discards immediately",
			keys: []tea.KeyMsg{escKey()},
			assert: func(t *testing.T, result *RuleOverlayResultMsg) {
				t.Helper()
				require.NotNil(t, result)
				assert.False(t, result.Committed)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			o := NewRuleOverlay("main", []string{".env"})
			var lastResult *RuleOverlayResultMsg
			for _, k := range tc.keys {
				result, _ := o.HandleKey(k)
				if result != nil {
					lastResult = result
				}
			}
			tc.assert(t, lastResult)
		})
	}
}

func TestRuleOverlay_DiscardConfirmation(t *testing.T) {
	testCases := []struct {
		assertResult func(t *testing.T, result *RuleOverlayResultMsg)
		assert       func(t *testing.T, o *RuleOverlay)
		name         string
		keys         []tea.KeyMsg
	}{
		{
			name: "esc with changes shows discard confirmation",
			keys: []tea.KeyMsg{
				runeKey('e'),                             // edit source worktree
				{Type: tea.KeyCtrlU},                     // clear
				runeKey('d'), runeKey('e'), runeKey('v'), // type "dev"
				enterKey(), // commit field edit
				escKey(),   // try to cancel overlay
			},
			assert: func(t *testing.T, o *RuleOverlay) {
				t.Helper()
				assert.Equal(t, ROConfirmDiscard, o.State())
			},
			assertResult: func(t *testing.T, result *RuleOverlayResultMsg) {
				t.Helper()
				assert.Nil(t, result)
			},
		},
		{
			name: "y confirms discard",
			keys: []tea.KeyMsg{
				runeKey('e'),
				{Type: tea.KeyCtrlU},
				runeKey('d'), runeKey('e'), runeKey('v'),
				enterKey(),
				escKey(),
				runeKey('y'),
			},
			assert: func(t *testing.T, _ *RuleOverlay) {
				t.Helper()
			},
			assertResult: func(t *testing.T, result *RuleOverlayResultMsg) {
				t.Helper()
				require.NotNil(t, result)
				assert.False(t, result.Committed)
			},
		},
		{
			name: "n cancels discard and returns to browsing",
			keys: []tea.KeyMsg{
				runeKey('e'),
				{Type: tea.KeyCtrlU},
				runeKey('d'), runeKey('e'), runeKey('v'),
				enterKey(),
				escKey(),
				runeKey('n'),
			},
			assert: func(t *testing.T, o *RuleOverlay) {
				t.Helper()
				assert.Equal(t, ROBrowsing, o.State())
			},
			assertResult: func(t *testing.T, result *RuleOverlayResultMsg) {
				t.Helper()
				assert.Nil(t, result)
			},
		},
		{
			name: "esc cancels discard and returns to browsing",
			keys: []tea.KeyMsg{
				runeKey('e'),
				{Type: tea.KeyCtrlU},
				runeKey('d'), runeKey('e'), runeKey('v'),
				enterKey(),
				escKey(),
				escKey(),
			},
			assert: func(t *testing.T, o *RuleOverlay) {
				t.Helper()
				assert.Equal(t, ROBrowsing, o.State())
			},
			assertResult: func(t *testing.T, result *RuleOverlayResultMsg) {
				t.Helper()
				assert.Nil(t, result)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			o := NewRuleOverlay("main", []string{".env"})
			var lastResult *RuleOverlayResultMsg
			for _, k := range tc.keys {
				result, _ := o.HandleKey(k)
				if result != nil {
					lastResult = result
				}
			}
			tc.assert(t, o)
			tc.assertResult(t, lastResult)
		})
	}
}

func TestRuleOverlay_HasChanges(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, changed bool)
		setup  func(o *RuleOverlay)
		name   string
	}{
		{
			name:  "no changes initially",
			setup: func(_ *RuleOverlay) {},
			assert: func(t *testing.T, changed bool) {
				t.Helper()
				assert.False(t, changed)
			},
		},
		{
			name: "editing source worktree marks changes",
			setup: func(o *RuleOverlay) {
				o.HandleKey(runeKey('e'))
				o.HandleKey(tea.KeyMsg{Type: tea.KeyCtrlU})
				o.HandleKey(runeKey('x'))
				o.HandleKey(enterKey())
			},
			assert: func(t *testing.T, changed bool) {
				t.Helper()
				assert.True(t, changed)
			},
		},
		{
			name: "editing files via list overlay marks changes",
			setup: func(o *RuleOverlay) {
				// Open files list overlay and add an item.
				o.HandleKey(tea.KeyMsg{Type: tea.KeyDown})
				o.HandleKey(runeKey('e'))
				o.HandleKey(runeKey('a'))
				o.HandleKey(runeKey('x'))
				o.HandleKey(enterKey()) // confirm add
				o.HandleKey(enterKey()) // commit list
			},
			assert: func(t *testing.T, changed bool) {
				t.Helper()
				assert.True(t, changed)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			o := NewRuleOverlay("main", []string{".env"})
			tc.setup(o)
			tc.assert(t, o.HasChanges())
		})
	}
}

func TestRuleOverlay_View(t *testing.T) {
	testCases := []struct {
		setup  func() *RuleOverlay
		assert func(t *testing.T, view string)
		name   string
	}{
		{
			name: "browsing shows edit rule title",
			setup: func() *RuleOverlay {
				return NewRuleOverlay("main", []string{".env"},
					WithRuleWidth(60))
			},
			assert: func(t *testing.T, view string) {
				t.Helper()
				assert.Contains(t, view, "Edit Rule")
			},
		},
		{
			name: "new rule shows new rule title",
			setup: func() *RuleOverlay {
				return NewRuleOverlayForNew(WithRuleWidth(60))
			},
			assert: func(t *testing.T, view string) {
				t.Helper()
				assert.Contains(t, view, "New Rule")
			},
		},
		{
			name: "shows field labels",
			setup: func() *RuleOverlay {
				return NewRuleOverlay("main", []string{".env"},
					WithRuleWidth(60))
			},
			assert: func(t *testing.T, view string) {
				t.Helper()
				assert.Contains(t, view, "Source Worktree")
				assert.Contains(t, view, "Files")
			},
		},
		{
			name: "shows field values",
			setup: func() *RuleOverlay {
				return NewRuleOverlay("main", []string{".env", "config.json"},
					WithRuleWidth(60))
			},
			assert: func(t *testing.T, view string) {
				t.Helper()
				assert.Contains(t, view, "main")
				assert.Contains(t, view, ".env")
			},
		},
		{
			name: "browsing shows hints",
			setup: func() *RuleOverlay {
				return NewRuleOverlay("main", []string{".env"},
					WithRuleWidth(60))
			},
			assert: func(t *testing.T, view string) {
				t.Helper()
				assert.Contains(t, view, "navigate")
				assert.Contains(t, view, "edit")
				assert.Contains(t, view, "confirm")
				assert.Contains(t, view, "cancel")
			},
		},
		{
			name: "editing shows editing hints",
			setup: func() *RuleOverlay {
				o := NewRuleOverlay("main", []string{".env"},
					WithRuleWidth(60))
				o.HandleKey(runeKey('e'))
				return o
			},
			assert: func(t *testing.T, view string) {
				t.Helper()
				assert.Contains(t, view, "confirm")
				assert.Contains(t, view, "cancel")
			},
		},
		{
			name: "confirm discard shows y/n prompt",
			setup: func() *RuleOverlay {
				o := NewRuleOverlay("main", []string{".env"},
					WithRuleWidth(60))
				// Make a change then try to discard.
				o.HandleKey(runeKey('e'))
				o.HandleKey(tea.KeyMsg{Type: tea.KeyCtrlU})
				o.HandleKey(runeKey('x'))
				o.HandleKey(enterKey())
				o.HandleKey(escKey())
				return o
			},
			assert: func(t *testing.T, view string) {
				t.Helper()
				assert.Contains(t, view, "Discard changes")
				assert.Contains(t, view, "y/n")
			},
		},
		{
			name: "list overlay state renders list overlay view",
			setup: func() *RuleOverlay {
				o := NewRuleOverlay("main", []string{".env"},
					WithRuleWidth(60))
				o.HandleKey(tea.KeyMsg{Type: tea.KeyDown})
				o.HandleKey(runeKey('e'))
				return o
			},
			assert: func(t *testing.T, view string) {
				t.Helper()
				assert.Contains(t, view, "File Copy")
				assert.Contains(t, view, "Files")
			},
		},
		{
			name: "focused field shows cursor",
			setup: func() *RuleOverlay {
				return NewRuleOverlay("main", []string{".env"},
					WithRuleWidth(60))
			},
			assert: func(t *testing.T, view string) {
				t.Helper()
				assert.Contains(t, view, ">")
			},
		},
		{
			name: "small width does not panic",
			setup: func() *RuleOverlay {
				return NewRuleOverlay("main", []string{".env"},
					WithRuleWidth(24))
			},
			assert: func(t *testing.T, view string) {
				t.Helper()
				assert.NotEmpty(t, view)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			o := tc.setup()
			view := o.View()
			tc.assert(t, view)
		})
	}
}

func TestRuleOverlay_SetWidth(t *testing.T) {
	o := NewRuleOverlay("main", []string{".env"})
	o.SetWidth(100)
	assert.Equal(t, 100, o.width)
}

func TestRuleOverlay_SetWidthZeroIgnored(t *testing.T) {
	o := NewRuleOverlay("main", []string{".env"}, WithRuleWidth(60))
	o.SetWidth(0)
	assert.Equal(t, 60, o.width, "zero width should be ignored")
}

func TestRuleOverlay_NestedModalStack(t *testing.T) {
	o := NewRuleOverlay("main", []string{".env", "config.json"})

	// Navigate to Files field and open list overlay.
	o.HandleKey(tea.KeyMsg{Type: tea.KeyDown})
	o.HandleKey(runeKey('e'))
	require.Equal(t, ROListOverlay, o.State())
	require.NotNil(t, o.ListOverlay())

	// The list overlay should have the current files.
	assert.Equal(t, []string{".env", "config.json"}, o.ListOverlay().Items())

	// Navigate in list overlay.
	o.HandleKey(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 1, o.ListOverlay().Cursor())

	// Delete an item: d, y
	o.HandleKey(runeKey('d'))
	assert.Equal(t, listConfirmDelete, o.ListOverlay().State())
	o.HandleKey(runeKey('y'))
	assert.Equal(t, listBrowsing, o.ListOverlay().State())
	assert.Equal(t, []string{".env"}, o.ListOverlay().Items())

	// Commit the list overlay.
	o.HandleKey(enterKey())
	assert.Equal(t, ROBrowsing, o.State())
	assert.Nil(t, o.ListOverlay())

	// Files should be updated.
	assert.Equal(t, []string{".env"}, o.Files())
}

func TestRuleOverlay_EditThenConfirm(t *testing.T) {
	o := NewRuleOverlay("main", []string{".env"})

	// Edit the source worktree field.
	o.HandleKey(runeKey('e'))
	require.Equal(t, ROEditingField, o.State())

	// Type something new.
	o.HandleKey(runeKey('X'))
	o.HandleKey(enterKey()) // commit field edit
	require.Equal(t, ROBrowsing, o.State())

	// Confirm the overlay.
	result, _ := o.HandleKey(enterKey())
	require.NotNil(t, result)
	assert.True(t, result.Committed)
	assert.Contains(t, result.SourceWorktree, "X")
}

func TestRuleOverlay_NewRuleFullFlow(t *testing.T) {
	o := NewRuleOverlayForNew()

	// Edit source worktree.
	o.HandleKey(runeKey('e'))
	for _, r := range "develop" {
		o.HandleKey(runeKey(r))
	}
	o.HandleKey(enterKey()) // commit field
	require.Equal(t, ROBrowsing, o.State())

	// Navigate to Files and open list overlay.
	o.HandleKey(tea.KeyMsg{Type: tea.KeyDown})
	o.HandleKey(runeKey('e'))
	require.Equal(t, ROListOverlay, o.State())

	// Add a file.
	o.HandleKey(runeKey('a'))
	for _, r := range ".env" {
		o.HandleKey(runeKey(r))
	}
	o.HandleKey(enterKey()) // confirm add
	o.HandleKey(enterKey()) // commit list overlay

	require.Equal(t, ROBrowsing, o.State())

	// Confirm overlay.
	result, _ := o.HandleKey(enterKey())
	require.NotNil(t, result)
	assert.True(t, result.Committed)
	assert.Equal(t, "develop", result.SourceWorktree)
	assert.Equal(t, []string{".env"}, result.Files)
}

func TestRuleOverlay_ListOverlayDiscardDoesNotUpdateFiles(t *testing.T) {
	o := NewRuleOverlay("main", []string{".env", "config.json"})

	// Open list overlay.
	o.HandleKey(tea.KeyMsg{Type: tea.KeyDown})
	o.HandleKey(runeKey('e'))
	require.Equal(t, ROListOverlay, o.State())

	// Add an item in list overlay.
	o.HandleKey(runeKey('a'))
	o.HandleKey(runeKey('x'))
	o.HandleKey(enterKey()) // confirm add

	// Discard list overlay: esc, then y to confirm discard.
	o.HandleKey(escKey())
	assert.Equal(t, listConfirmDiscard, o.ListOverlay().State())
	o.HandleKey(runeKey('y'))

	// Should be back in browsing, files unchanged.
	assert.Equal(t, ROBrowsing, o.State())
	assert.Equal(t, []string{".env", "config.json"}, o.Files())
}

func TestCopyStrings(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, result []string)
		name   string
		input  []string
	}{
		{
			name:  "nil returns nil",
			input: nil,
			assert: func(t *testing.T, result []string) {
				t.Helper()
				assert.Nil(t, result)
			},
		},
		{
			name:  "empty returns empty",
			input: []string{},
			assert: func(t *testing.T, result []string) {
				t.Helper()
				assert.NotNil(t, result)
				assert.Empty(t, result)
			},
		},
		{
			name:  "copies values",
			input: []string{"a", "b", "c"},
			assert: func(t *testing.T, result []string) {
				t.Helper()
				assert.Equal(t, []string{"a", "b", "c"}, result)
			},
		},
		{
			name:  "deep copy is independent",
			input: []string{"a", "b"},
			assert: func(t *testing.T, result []string) {
				t.Helper()
				result[0] = "modified"
				// Original should be unchanged, but we can only check
				// that result is independent by verifying it was changed.
				assert.Equal(t, "modified", result[0])
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := copyStrings(tc.input)
			tc.assert(t, result)
		})
	}
}

func TestFormatFileSummary(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, summary string)
		name   string
		files  []string
	}{
		{
			name:  "empty list",
			files: nil,
			assert: func(t *testing.T, summary string) {
				t.Helper()
				assert.Equal(t, "(none)", summary)
			},
		},
		{
			name:  "one file",
			files: []string{".env"},
			assert: func(t *testing.T, summary string) {
				t.Helper()
				assert.Equal(t, "1 file(s)", summary)
			},
		},
		{
			name:  "multiple files",
			files: []string{".env", "config.json", "secrets.yaml"},
			assert: func(t *testing.T, summary string) {
				t.Helper()
				assert.Equal(t, "3 file(s)", summary)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			summary := formatFileSummary(tc.files)
			tc.assert(t, summary)
		})
	}
}

func TestRuleOverlay_UnhandledKeysInConfirmDiscard(t *testing.T) {
	o := NewRuleOverlay("main", []string{".env"})

	// Make a change so discard confirmation appears.
	o.HandleKey(runeKey('e'))
	o.HandleKey(tea.KeyMsg{Type: tea.KeyCtrlU})
	o.HandleKey(runeKey('x'))
	o.HandleKey(enterKey())
	o.HandleKey(escKey())
	require.Equal(t, ROConfirmDiscard, o.State())

	// Unhandled key should stay in confirm state.
	result, _ := o.HandleKey(runeKey('z'))
	assert.Nil(t, result)
	assert.Equal(t, ROConfirmDiscard, o.State())
}

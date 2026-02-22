package config

import (
	"gbm/pkg/tui"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testWorktreeValues() [3]string {
	return [3]string{"feature/auth", "main", "Auth feature"}
}

func TestNewWorktreeOverlay(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, o *WorktreeOverlay)
		name   string
		opts   []WorktreeOverlayOption
	}{
		{
			name: "default values",
			opts: nil,
			assert: func(t *testing.T, o *WorktreeOverlay) {
				t.Helper()
				assert.Equal(t, WOBrowsing, o.State())
				assert.Equal(t, "feature/auth", o.Name())
				assert.False(t, o.IsNew())
				assert.False(t, o.IsConfirmed())
				assert.Equal(t, 0, o.FocusIndex())
				assert.Len(t, o.Fields(), 3)
			},
		},
		{
			name: "with custom theme",
			opts: []WorktreeOverlayOption{WithWorktreeTheme(tui.DefaultTheme())},
			assert: func(t *testing.T, o *WorktreeOverlay) {
				t.Helper()
				assert.NotNil(t, o.theme)
			},
		},
		{
			name: "nil theme does not override default",
			opts: []WorktreeOverlayOption{WithWorktreeTheme(nil)},
			assert: func(t *testing.T, o *WorktreeOverlay) {
				t.Helper()
				assert.NotNil(t, o.theme)
			},
		},
		{
			name: "with custom width",
			opts: []WorktreeOverlayOption{WithWorktreeWidth(80)},
			assert: func(t *testing.T, o *WorktreeOverlay) {
				t.Helper()
				assert.Equal(t, 80, o.width)
			},
		},
		{
			name: "zero width does not override default",
			opts: []WorktreeOverlayOption{WithWorktreeWidth(0)},
			assert: func(t *testing.T, o *WorktreeOverlay) {
				t.Helper()
				assert.Equal(t, 50, o.width)
			},
		},
		{
			name: "with existing names",
			opts: []WorktreeOverlayOption{WithExistingNames([]string{"main", "develop"})},
			assert: func(t *testing.T, o *WorktreeOverlay) {
				t.Helper()
				assert.Equal(t, []string{"main", "develop"}, o.existingNames)
			},
		},
		{
			name: "field values loaded correctly",
			opts: nil,
			assert: func(t *testing.T, o *WorktreeOverlay) {
				t.Helper()
				vals := o.Values()
				assert.Equal(t, "feature/auth", vals[0])
				assert.Equal(t, "main", vals[1])
				assert.Equal(t, "Auth feature", vals[2])
			},
		},
		{
			name: "first field is focused",
			opts: nil,
			assert: func(t *testing.T, o *WorktreeOverlay) {
				t.Helper()
				assert.True(t, o.Fields()[0].IsFocused())
				assert.False(t, o.Fields()[1].IsFocused())
				assert.False(t, o.Fields()[2].IsFocused())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			o := NewWorktreeOverlay("feature/auth", testWorktreeValues(), tc.opts...)
			tc.assert(t, o)
		})
	}
}

func TestNewWorktreeOverlayForNew(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, o *WorktreeOverlay)
		name   string
		opts   []WorktreeOverlayOption
	}{
		{
			name: "starts in prompting name state",
			opts: nil,
			assert: func(t *testing.T, o *WorktreeOverlay) {
				t.Helper()
				assert.Equal(t, WOPromptingName, o.State())
				assert.True(t, o.IsNew())
				assert.Empty(t, o.Name())
			},
		},
		{
			name: "fields are empty",
			opts: nil,
			assert: func(t *testing.T, o *WorktreeOverlay) {
				t.Helper()
				vals := o.Values()
				assert.Empty(t, vals[0])
				assert.Empty(t, vals[1])
				assert.Empty(t, vals[2])
			},
		},
		{
			name: "with existing names for validation",
			opts: []WorktreeOverlayOption{WithExistingNames([]string{"main"})},
			assert: func(t *testing.T, o *WorktreeOverlay) {
				t.Helper()
				assert.Equal(t, []string{"main"}, o.existingNames)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			o := NewWorktreeOverlayForNew(tc.opts...)
			tc.assert(t, o)
		})
	}
}

func TestWorktreeOverlay_Navigation(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, o *WorktreeOverlay)
		name   string
		keys   []tea.KeyMsg
	}{
		{
			name: "down moves to next field",
			keys: []tea.KeyMsg{{Type: tea.KeyDown}},
			assert: func(t *testing.T, o *WorktreeOverlay) {
				t.Helper()
				assert.Equal(t, 1, o.FocusIndex())
				assert.False(t, o.Fields()[0].IsFocused())
				assert.True(t, o.Fields()[1].IsFocused())
			},
		},
		{
			name: "j moves to next field",
			keys: []tea.KeyMsg{runeKey('j')},
			assert: func(t *testing.T, o *WorktreeOverlay) {
				t.Helper()
				assert.Equal(t, 1, o.FocusIndex())
			},
		},
		{
			name: "up wraps to last field",
			keys: []tea.KeyMsg{{Type: tea.KeyUp}},
			assert: func(t *testing.T, o *WorktreeOverlay) {
				t.Helper()
				assert.Equal(t, 2, o.FocusIndex())
				assert.True(t, o.Fields()[2].IsFocused())
			},
		},
		{
			name: "k wraps to last field",
			keys: []tea.KeyMsg{runeKey('k')},
			assert: func(t *testing.T, o *WorktreeOverlay) {
				t.Helper()
				assert.Equal(t, 2, o.FocusIndex())
			},
		},
		{
			name: "down wraps from last to first",
			keys: []tea.KeyMsg{
				{Type: tea.KeyDown},
				{Type: tea.KeyDown},
				{Type: tea.KeyDown},
			},
			assert: func(t *testing.T, o *WorktreeOverlay) {
				t.Helper()
				assert.Equal(t, 0, o.FocusIndex())
				assert.True(t, o.Fields()[0].IsFocused())
			},
		},
		{
			name: "navigate through all fields and back",
			keys: []tea.KeyMsg{
				{Type: tea.KeyDown}, // 0 -> 1
				{Type: tea.KeyDown}, // 1 -> 2
				{Type: tea.KeyUp},   // 2 -> 1
			},
			assert: func(t *testing.T, o *WorktreeOverlay) {
				t.Helper()
				assert.Equal(t, 1, o.FocusIndex())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			o := NewWorktreeOverlay("test", testWorktreeValues())
			for _, k := range tc.keys {
				o.HandleKey(k)
			}
			tc.assert(t, o)
		})
	}
}

func TestWorktreeOverlay_EditField(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, o *WorktreeOverlay)
		name   string
		setup  func(o *WorktreeOverlay)
		keys   []tea.KeyMsg
	}{
		{
			name:  "e enters editing state",
			setup: func(_ *WorktreeOverlay) {},
			keys:  []tea.KeyMsg{runeKey('e')},
			assert: func(t *testing.T, o *WorktreeOverlay) {
				t.Helper()
				assert.Equal(t, WOEditing, o.State())
				assert.Equal(t, FieldEditing, o.Fields()[0].State())
			},
		},
		{
			name:  "esc cancels editing and returns to browsing",
			setup: func(_ *WorktreeOverlay) {},
			keys:  []tea.KeyMsg{runeKey('e'), escKey()},
			assert: func(t *testing.T, o *WorktreeOverlay) {
				t.Helper()
				assert.Equal(t, WOBrowsing, o.State())
				assert.Equal(t, FieldBrowsing, o.Fields()[0].State())
			},
		},
		{
			name: "enter commits field edit and returns to browsing",
			setup: func(o *WorktreeOverlay) {
				// Move to second field, then edit.
			},
			keys: []tea.KeyMsg{
				{Type: tea.KeyDown}, // move to merge_into
				runeKey('e'),        // start editing
				enterKey(),          // commit
			},
			assert: func(t *testing.T, o *WorktreeOverlay) {
				t.Helper()
				assert.Equal(t, WOBrowsing, o.State())
				assert.Equal(t, FieldBrowsing, o.Fields()[1].State())
			},
		},
		{
			name:  "typing in editing mode forwards to field input",
			setup: func(_ *WorktreeOverlay) {},
			keys: []tea.KeyMsg{
				runeKey('e'),
				runeKey('x'),
				runeKey('y'),
			},
			assert: func(t *testing.T, o *WorktreeOverlay) {
				t.Helper()
				assert.Equal(t, WOEditing, o.State())
				// The input should contain the original value plus "xy"
				assert.Contains(t, o.Fields()[0].input.Value(), "xy")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			o := NewWorktreeOverlay("test", testWorktreeValues())
			tc.setup(o)
			for _, k := range tc.keys {
				o.HandleKey(k)
			}
			tc.assert(t, o)
		})
	}
}

func TestWorktreeOverlay_Rename(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, o *WorktreeOverlay)
		name   string
		keys   []tea.KeyMsg
		others []string
	}{
		{
			name:   "r enters renaming state",
			keys:   []tea.KeyMsg{runeKey('r')},
			others: nil,
			assert: func(t *testing.T, o *WorktreeOverlay) {
				t.Helper()
				assert.Equal(t, WORenaming, o.State())
			},
		},
		{
			name:   "esc cancels rename",
			keys:   []tea.KeyMsg{runeKey('r'), escKey()},
			others: nil,
			assert: func(t *testing.T, o *WorktreeOverlay) {
				t.Helper()
				assert.Equal(t, WOBrowsing, o.State())
				assert.Equal(t, "feature/auth", o.Name())
				assert.Empty(t, o.NameError())
			},
		},
		{
			name: "enter with valid name confirms rename",
			keys: []tea.KeyMsg{
				runeKey('r'),
				// Clear existing and type new name.
				{Type: tea.KeyCtrlU}, // clear
				runeKey('n'),
				runeKey('e'),
				runeKey('w'),
				enterKey(),
			},
			others: nil,
			assert: func(t *testing.T, o *WorktreeOverlay) {
				t.Helper()
				assert.Equal(t, WOBrowsing, o.State())
				assert.Equal(t, "new", o.Name())
			},
		},
		{
			name: "rename rejects duplicate name",
			keys: []tea.KeyMsg{
				runeKey('r'),
				{Type: tea.KeyCtrlU},
				runeKey('m'),
				runeKey('a'),
				runeKey('i'),
				runeKey('n'),
				enterKey(),
			},
			others: []string{"main", "develop"},
			assert: func(t *testing.T, o *WorktreeOverlay) {
				t.Helper()
				assert.Equal(t, WORenaming, o.State())
				assert.Contains(t, o.NameError(), "already exists")
			},
		},
		{
			name: "rename rejects empty name",
			keys: []tea.KeyMsg{
				runeKey('r'),
				{Type: tea.KeyCtrlU}, // clear
				enterKey(),
			},
			others: nil,
			assert: func(t *testing.T, o *WorktreeOverlay) {
				t.Helper()
				assert.Equal(t, WORenaming, o.State())
				assert.Contains(t, o.NameError(), "required")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			o := NewWorktreeOverlay("feature/auth", testWorktreeValues(),
				WithExistingNames(tc.others))
			for _, k := range tc.keys {
				o.HandleKey(k)
			}
			tc.assert(t, o)
		})
	}
}

func TestWorktreeOverlay_ConfirmAndCancel(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, o *WorktreeOverlay, closed bool)
		name   string
		keys   []tea.KeyMsg
	}{
		{
			name: "enter confirms and closes",
			keys: []tea.KeyMsg{enterKey()},
			assert: func(t *testing.T, o *WorktreeOverlay, closed bool) {
				t.Helper()
				assert.True(t, closed)
				assert.True(t, o.IsConfirmed())
			},
		},
		{
			name: "esc discards and closes",
			keys: []tea.KeyMsg{escKey()},
			assert: func(t *testing.T, o *WorktreeOverlay, closed bool) {
				t.Helper()
				assert.True(t, closed)
				assert.False(t, o.IsConfirmed())
			},
		},
		{
			name: "esc restores original values",
			keys: []tea.KeyMsg{
				runeKey('e'), // edit branch field
				runeKey('Z'),
				runeKey('Z'),
				enterKey(), // commit field edit
				escKey(),   // discard overlay
			},
			assert: func(t *testing.T, o *WorktreeOverlay, closed bool) {
				t.Helper()
				assert.True(t, closed)
				vals := o.Values()
				assert.Equal(t, "feature/auth", vals[0])
				assert.Equal(t, "main", vals[1])
				assert.Equal(t, "Auth feature", vals[2])
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			o := NewWorktreeOverlay("feature/auth", testWorktreeValues())
			var closed bool
			for _, k := range tc.keys {
				_, c := o.HandleKey(k)
				if c {
					closed = true
				}
			}
			tc.assert(t, o, closed)
		})
	}
}

func TestWorktreeOverlay_NewWorktreeNamePrompt(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, o *WorktreeOverlay, closed bool)
		name   string
		keys   []tea.KeyMsg
		others []string
	}{
		{
			name: "enter with valid name transitions to browsing",
			keys: []tea.KeyMsg{
				runeKey('m'),
				runeKey('y'),
				runeKey('w'),
				runeKey('t'),
				enterKey(),
			},
			others: nil,
			assert: func(t *testing.T, o *WorktreeOverlay, closed bool) {
				t.Helper()
				assert.False(t, closed)
				assert.Equal(t, WOBrowsing, o.State())
				assert.Equal(t, "mywt", o.Name())
			},
		},
		{
			name:   "esc closes the overlay",
			keys:   []tea.KeyMsg{escKey()},
			others: nil,
			assert: func(t *testing.T, o *WorktreeOverlay, closed bool) {
				t.Helper()
				assert.True(t, closed)
			},
		},
		{
			name: "enter with duplicate name shows error",
			keys: []tea.KeyMsg{
				runeKey('m'),
				runeKey('a'),
				runeKey('i'),
				runeKey('n'),
				enterKey(),
			},
			others: []string{"main"},
			assert: func(t *testing.T, o *WorktreeOverlay, closed bool) {
				t.Helper()
				assert.False(t, closed)
				assert.Equal(t, WOPromptingName, o.State())
				assert.Contains(t, o.NameError(), "already exists")
			},
		},
		{
			name:   "enter with empty name shows error",
			keys:   []tea.KeyMsg{enterKey()},
			others: nil,
			assert: func(t *testing.T, o *WorktreeOverlay, closed bool) {
				t.Helper()
				assert.False(t, closed)
				assert.Equal(t, WOPromptingName, o.State())
				assert.Contains(t, o.NameError(), "required")
			},
		},
		{
			name: "typing clears error",
			keys: []tea.KeyMsg{
				enterKey(),   // trigger error
				runeKey('a'), // type to clear
			},
			others: nil,
			assert: func(t *testing.T, o *WorktreeOverlay, closed bool) {
				t.Helper()
				assert.Empty(t, o.NameError())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			o := NewWorktreeOverlayForNew(WithExistingNames(tc.others))
			var closed bool
			for _, k := range tc.keys {
				_, c := o.HandleKey(k)
				if c {
					closed = true
				}
			}
			tc.assert(t, o, closed)
		})
	}
}

func TestWorktreeOverlay_NameValidation(t *testing.T) {
	testCases := []struct {
		assertError func(t *testing.T, err error)
		name        string
		input       string
		others      []string
	}{
		{
			name:   "valid simple name",
			input:  "feature/auth",
			others: nil,
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name:   "empty name rejected",
			input:  "",
			others: nil,
			assertError: func(t *testing.T, err error) {
				t.Helper()
				require.Error(t, err)
				assert.Contains(t, err.Error(), "required")
			},
		},
		{
			name:   "duplicate name rejected",
			input:  "main",
			others: []string{"main", "develop"},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				require.Error(t, err)
				assert.Contains(t, err.Error(), "already exists")
			},
		},
		{
			name:   "name with space rejected",
			input:  "my worktree",
			others: nil,
			assertError: func(t *testing.T, err error) {
				t.Helper()
				require.Error(t, err)
				assert.Contains(t, err.Error(), "\" \"")
			},
		},
		{
			name:   "name with tilde rejected",
			input:  "feature~1",
			others: nil,
			assertError: func(t *testing.T, err error) {
				t.Helper()
				require.Error(t, err)
				assert.Contains(t, err.Error(), "\"~\"")
			},
		},
		{
			name:   "name with caret rejected",
			input:  "feature^2",
			others: nil,
			assertError: func(t *testing.T, err error) {
				t.Helper()
				require.Error(t, err)
				assert.Contains(t, err.Error(), "\"^\"")
			},
		},
		{
			name:   "name with colon rejected",
			input:  "feature:branch",
			others: nil,
			assertError: func(t *testing.T, err error) {
				t.Helper()
				require.Error(t, err)
				assert.Contains(t, err.Error(), "\":\"")
			},
		},
		{
			name:   "name with question mark rejected",
			input:  "feature?branch",
			others: nil,
			assertError: func(t *testing.T, err error) {
				t.Helper()
				require.Error(t, err)
			},
		},
		{
			name:   "name with asterisk rejected",
			input:  "feature*branch",
			others: nil,
			assertError: func(t *testing.T, err error) {
				t.Helper()
				require.Error(t, err)
			},
		},
		{
			name:   "name with bracket rejected",
			input:  "feature[branch",
			others: nil,
			assertError: func(t *testing.T, err error) {
				t.Helper()
				require.Error(t, err)
			},
		},
		{
			name:   "name with backslash rejected",
			input:  "feature\\branch",
			others: nil,
			assertError: func(t *testing.T, err error) {
				t.Helper()
				require.Error(t, err)
			},
		},
		{
			name:   "name starting with dash rejected",
			input:  "-feature",
			others: nil,
			assertError: func(t *testing.T, err error) {
				t.Helper()
				require.Error(t, err)
				assert.Contains(t, err.Error(), "start with '-'")
			},
		},
		{
			name:   "name starting with dot rejected",
			input:  ".feature",
			others: nil,
			assertError: func(t *testing.T, err error) {
				t.Helper()
				require.Error(t, err)
				assert.Contains(t, err.Error(), "start with '.'")
			},
		},
		{
			name:   "name ending with dot rejected",
			input:  "feature.",
			others: nil,
			assertError: func(t *testing.T, err error) {
				t.Helper()
				require.Error(t, err)
				assert.Contains(t, err.Error(), "end with '.'")
			},
		},
		{
			name:   "name ending with .lock rejected",
			input:  "feature.lock",
			others: nil,
			assertError: func(t *testing.T, err error) {
				t.Helper()
				require.Error(t, err)
				assert.Contains(t, err.Error(), ".lock")
			},
		},
		{
			name:   "name with double dot rejected",
			input:  "feature..branch",
			others: nil,
			assertError: func(t *testing.T, err error) {
				t.Helper()
				require.Error(t, err)
				assert.Contains(t, err.Error(), "'..'")
			},
		},
		{
			name:   "name with control char rejected",
			input:  "feature\x01branch",
			others: nil,
			assertError: func(t *testing.T, err error) {
				t.Helper()
				require.Error(t, err)
				assert.Contains(t, err.Error(), "control")
			},
		},
		{
			name:   "valid name with slash",
			input:  "feature/PROJ-123",
			others: nil,
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name:   "valid name with dots",
			input:  "release.1.0",
			others: nil,
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name:   "valid name with dashes",
			input:  "my-feature-branch",
			others: nil,
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			o := NewWorktreeOverlay("test", [3]string{},
				WithExistingNames(tc.others))
			err := o.validateName(tc.input)
			tc.assertError(t, err)
		})
	}
}

func TestWorktreeOverlay_View(t *testing.T) {
	testCases := []struct {
		setup  func() *WorktreeOverlay
		assert func(t *testing.T, view string)
		name   string
	}{
		{
			name: "browsing shows title with name",
			setup: func() *WorktreeOverlay {
				return NewWorktreeOverlay("feature/auth", testWorktreeValues(),
					WithWorktreeWidth(60))
			},
			assert: func(t *testing.T, view string) {
				t.Helper()
				assert.Contains(t, view, "Edit")
				assert.Contains(t, view, "feature/auth")
			},
		},
		{
			name: "browsing shows field labels",
			setup: func() *WorktreeOverlay {
				return NewWorktreeOverlay("test", testWorktreeValues(),
					WithWorktreeWidth(60))
			},
			assert: func(t *testing.T, view string) {
				t.Helper()
				assert.Contains(t, view, "Branch")
				assert.Contains(t, view, "Merge Into")
				assert.Contains(t, view, "Description")
			},
		},
		{
			name: "browsing shows field values",
			setup: func() *WorktreeOverlay {
				return NewWorktreeOverlay("test", testWorktreeValues(),
					WithWorktreeWidth(60))
			},
			assert: func(t *testing.T, view string) {
				t.Helper()
				assert.Contains(t, view, "feature/auth")
				assert.Contains(t, view, "main")
				assert.Contains(t, view, "Auth feature")
			},
		},
		{
			name: "browsing shows hints",
			setup: func() *WorktreeOverlay {
				return NewWorktreeOverlay("test", testWorktreeValues(),
					WithWorktreeWidth(60))
			},
			assert: func(t *testing.T, view string) {
				t.Helper()
				assert.Contains(t, view, "navigate")
				assert.Contains(t, view, "edit")
				assert.Contains(t, view, "rename")
				assert.Contains(t, view, "confirm")
				assert.Contains(t, view, "cancel")
			},
		},
		{
			name: "prompting name shows new title",
			setup: func() *WorktreeOverlay {
				return NewWorktreeOverlayForNew(WithWorktreeWidth(60))
			},
			assert: func(t *testing.T, view string) {
				t.Helper()
				assert.Contains(t, view, "New")
				assert.Contains(t, view, "Enter worktree name")
			},
		},
		{
			name: "renaming shows rename title",
			setup: func() *WorktreeOverlay {
				o := NewWorktreeOverlay("feature/auth", testWorktreeValues(),
					WithWorktreeWidth(60))
				o.HandleKey(runeKey('r'))
				return o
			},
			assert: func(t *testing.T, view string) {
				t.Helper()
				assert.Contains(t, view, "Rename")
				assert.Contains(t, view, "Name")
			},
		},
		{
			name: "prompting name with error shows error",
			setup: func() *WorktreeOverlay {
				o := NewWorktreeOverlayForNew(WithWorktreeWidth(60))
				o.HandleKey(enterKey()) // trigger empty name error
				return o
			},
			assert: func(t *testing.T, view string) {
				t.Helper()
				assert.Contains(t, view, "required")
			},
		},
		{
			name: "focused field shows cursor",
			setup: func() *WorktreeOverlay {
				return NewWorktreeOverlay("test", testWorktreeValues(),
					WithWorktreeWidth(60))
			},
			assert: func(t *testing.T, view string) {
				t.Helper()
				assert.Contains(t, view, ">")
			},
		},
		{
			name: "small width does not panic",
			setup: func() *WorktreeOverlay {
				return NewWorktreeOverlay("test", testWorktreeValues(),
					WithWorktreeWidth(24))
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

func TestWorktreeOverlay_SetWidth(t *testing.T) {
	o := NewWorktreeOverlay("test", testWorktreeValues())
	o.SetWidth(100)
	assert.Equal(t, 100, o.width)
}

func TestWorktreeOverlay_SetWidthZeroIgnored(t *testing.T) {
	o := NewWorktreeOverlay("test", testWorktreeValues(), WithWorktreeWidth(60))
	o.SetWidth(0)
	assert.Equal(t, 60, o.width, "zero width should be ignored")
}

func TestWorktreeOverlay_EditThenConfirmClosesOverlay(t *testing.T) {
	o := NewWorktreeOverlay("test", testWorktreeValues())

	// Edit the branch field.
	o.HandleKey(runeKey('e'))
	require.Equal(t, WOEditing, o.State())

	// Type something new.
	o.HandleKey(runeKey('X'))
	o.HandleKey(enterKey()) // commit field edit
	require.Equal(t, WOBrowsing, o.State())

	// Now confirm the overlay.
	_, closed := o.HandleKey(enterKey())
	assert.True(t, closed)
	assert.True(t, o.IsConfirmed())

	vals := o.Values()
	assert.Contains(t, vals[0], "X") // branch should have the edit
}

func TestWorktreeOverlay_NewWorktreeFullFlow(t *testing.T) {
	o := NewWorktreeOverlayForNew(WithExistingNames([]string{"main"}))

	// Name prompt: type "feature/x".
	for _, r := range "feature/x" {
		o.HandleKey(runeKey(r))
	}
	o.HandleKey(enterKey())
	require.Equal(t, WOBrowsing, o.State())
	assert.Equal(t, "feature/x", o.Name())

	// Edit the branch field.
	o.HandleKey(runeKey('e'))
	for _, r := range "feature/x" {
		o.HandleKey(runeKey(r))
	}
	o.HandleKey(enterKey()) // commit field
	require.Equal(t, WOBrowsing, o.State())

	// Move to merge_into and edit.
	o.HandleKey(tea.KeyMsg{Type: tea.KeyDown})
	o.HandleKey(runeKey('e'))
	for _, r := range "main" {
		o.HandleKey(runeKey(r))
	}
	o.HandleKey(enterKey()) // commit field

	// Confirm overlay.
	_, closed := o.HandleKey(enterKey())
	assert.True(t, closed)
	assert.True(t, o.IsConfirmed())

	vals := o.Values()
	assert.Equal(t, "feature/x", vals[0])
	assert.Equal(t, "main", vals[1])
}

func TestWorktreeOverlay_RenameClearsErrorOnInput(t *testing.T) {
	o := NewWorktreeOverlay("test", testWorktreeValues(),
		WithExistingNames([]string{"main"}))

	// Enter rename mode.
	o.HandleKey(runeKey('r'))
	require.Equal(t, WORenaming, o.State())

	// Clear and type duplicate name.
	o.HandleKey(tea.KeyMsg{Type: tea.KeyCtrlU})
	for _, r := range "main" {
		o.HandleKey(runeKey(r))
	}
	o.HandleKey(enterKey()) // triggers duplicate error
	require.NotEmpty(t, o.NameError())

	// Typing should clear the error.
	o.HandleKey(runeKey('x'))
	assert.Empty(t, o.NameError())
}

func TestWorktreeOverlay_BranchSuggestions(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, o *WorktreeOverlay)
		name   string
		names  []string
	}{
		{
			name:  "branch field receives branch names as suggestions",
			names: []string{"main", "develop", "feature/auth"},
			assert: func(t *testing.T, o *WorktreeOverlay) {
				t.Helper()
				require.Len(t, o.Fields(), 3)
				require.NotNil(t, o.Fields()[0].meta.Suggestions)
				assert.Equal(t, []string{"main", "develop", "feature/auth"}, o.Fields()[0].meta.Suggestions())
			},
		},
		{
			name:  "no suggestions when branch names are empty",
			names: nil,
			assert: func(t *testing.T, o *WorktreeOverlay) {
				t.Helper()
				require.Len(t, o.Fields(), 3)
				assert.Nil(t, o.Fields()[0].meta.Suggestions)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			o := NewWorktreeOverlay("test", testWorktreeValues(),
				WithBranchNames(tc.names))
			tc.assert(t, o)
		})
	}
}

func TestWorktreeOverlay_MergeIntoSuggestions(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, o *WorktreeOverlay)
		name   string
		names  []string
	}{
		{
			name:  "merge_into field receives worktree names as suggestions",
			names: []string{"main", "develop", "staging"},
			assert: func(t *testing.T, o *WorktreeOverlay) {
				t.Helper()
				require.Len(t, o.Fields(), 3)
				require.NotNil(t, o.Fields()[1].meta.Suggestions)
				assert.Equal(t, []string{"main", "develop", "staging"}, o.Fields()[1].meta.Suggestions())
			},
		},
		{
			name:  "no suggestions when worktree names are empty",
			names: nil,
			assert: func(t *testing.T, o *WorktreeOverlay) {
				t.Helper()
				require.Len(t, o.Fields(), 3)
				assert.Nil(t, o.Fields()[1].meta.Suggestions)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			o := NewWorktreeOverlay("test", testWorktreeValues(),
				WithWorktreeNames(tc.names))
			tc.assert(t, o)
		})
	}
}

func TestValidateGitBranchChars(t *testing.T) {
	testCases := []struct {
		assertError func(t *testing.T, err error)
		name        string
		input       string
	}{
		{
			name:  "valid simple name",
			input: "main",
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name:  "valid with slash",
			input: "feature/auth",
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name:  "valid with dash",
			input: "my-feature",
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name:  "valid with underscore",
			input: "my_feature",
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name:  "rejects null byte",
			input: "feat\x00ure",
			assertError: func(t *testing.T, err error) {
				t.Helper()
				require.Error(t, err)
				assert.Contains(t, err.Error(), "control")
			},
		},
		{
			name:  "rejects DEL character",
			input: "feat\x7fure",
			assertError: func(t *testing.T, err error) {
				t.Helper()
				require.Error(t, err)
				assert.Contains(t, err.Error(), "control")
			},
		},
		{
			name:  "rejects tab",
			input: "feat\ture",
			assertError: func(t *testing.T, err error) {
				t.Helper()
				require.Error(t, err)
				assert.Contains(t, err.Error(), "control")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateGitBranchChars(tc.input)
			tc.assertError(t, err)
		})
	}
}

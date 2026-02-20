package config

import (
	"gbm/pkg/tui"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewListOverlay(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, l *ListOverlay)
		name   string
		title  string
		items  []string
	}{
		{
			name:  "creates with items",
			title: "JIRA > Filters > Status",
			items: []string{"In Dev", "Open", "Closed"},
			assert: func(t *testing.T, l *ListOverlay) {
				t.Helper()
				assert.Equal(t, "JIRA > Filters > Status", l.title)
				assert.Equal(t, []string{"In Dev", "Open", "Closed"}, l.Items())
				assert.Equal(t, 0, l.Cursor())
				assert.Equal(t, listBrowsing, l.State())
			},
		},
		{
			name:  "creates with empty items",
			title: "Empty List",
			items: nil,
			assert: func(t *testing.T, l *ListOverlay) {
				t.Helper()
				assert.Empty(t, l.Items())
				assert.Equal(t, 0, l.Cursor())
			},
		},
		{
			name:  "deep copies items",
			title: "Copy Test",
			items: []string{"a", "b"},
			assert: func(t *testing.T, l *ListOverlay) {
				t.Helper()
				items := l.Items()
				items[0] = "modified"
				assert.Equal(t, "a", l.items[0], "modifying returned slice should not affect overlay")
			},
		},
		{
			name:  "nil theme uses default",
			title: "Default Theme",
			items: []string{"x"},
			assert: func(t *testing.T, l *ListOverlay) {
				t.Helper()
				assert.NotNil(t, l.theme)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			l := NewListOverlay(tc.title, tc.items, tui.DefaultTheme())
			tc.assert(t, l)
		})
	}
}

func TestListOverlay_NilTheme(t *testing.T) {
	l := NewListOverlay("test", []string{"a"}, nil)
	require.NotNil(t, l)
	assert.NotNil(t, l.theme)
}

func TestListOverlay_HasChanges(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, changed bool)
		setup  func(l *ListOverlay)
		name   string
	}{
		{
			name:  "no changes initially",
			setup: func(_ *ListOverlay) {},
			assert: func(t *testing.T, changed bool) {
				t.Helper()
				assert.False(t, changed)
			},
		},
		{
			name: "adding an item marks changes",
			setup: func(l *ListOverlay) {
				l.items = append(l.items, "new")
			},
			assert: func(t *testing.T, changed bool) {
				t.Helper()
				assert.True(t, changed)
			},
		},
		{
			name: "removing an item marks changes",
			setup: func(l *ListOverlay) {
				l.items = l.items[:1]
			},
			assert: func(t *testing.T, changed bool) {
				t.Helper()
				assert.True(t, changed)
			},
		},
		{
			name: "modifying an item marks changes",
			setup: func(l *ListOverlay) {
				l.items[0] = "modified"
			},
			assert: func(t *testing.T, changed bool) {
				t.Helper()
				assert.True(t, changed)
			},
		},
		{
			name: "reverting to original shows no changes",
			setup: func(l *ListOverlay) {
				l.items[0] = "modified"
				l.items[0] = "In Dev"
			},
			assert: func(t *testing.T, changed bool) {
				t.Helper()
				assert.False(t, changed)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			l := NewListOverlay("test", []string{"In Dev", "Open"}, tui.DefaultTheme())
			tc.setup(l)
			tc.assert(t, l.HasChanges())
		})
	}
}

func TestListOverlay_Navigation(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, l *ListOverlay)
		name   string
		keys   []tea.KeyMsg
	}{
		{
			name: "down moves cursor forward",
			keys: []tea.KeyMsg{{Type: tea.KeyDown}},
			assert: func(t *testing.T, l *ListOverlay) {
				t.Helper()
				assert.Equal(t, 1, l.Cursor())
			},
		},
		{
			name: "up from first wraps to last",
			keys: []tea.KeyMsg{{Type: tea.KeyUp}},
			assert: func(t *testing.T, l *ListOverlay) {
				t.Helper()
				assert.Equal(t, 2, l.Cursor())
			},
		},
		{
			name: "down wraps from last to first",
			keys: []tea.KeyMsg{
				{Type: tea.KeyDown},
				{Type: tea.KeyDown},
				{Type: tea.KeyDown},
			},
			assert: func(t *testing.T, l *ListOverlay) {
				t.Helper()
				assert.Equal(t, 0, l.Cursor())
			},
		},
		{
			name: "multiple navigations",
			keys: []tea.KeyMsg{
				{Type: tea.KeyDown},
				{Type: tea.KeyDown},
				{Type: tea.KeyUp},
			},
			assert: func(t *testing.T, l *ListOverlay) {
				t.Helper()
				assert.Equal(t, 1, l.Cursor())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			l := NewListOverlay("test", []string{"a", "b", "c"}, tui.DefaultTheme())
			for _, k := range tc.keys {
				l.Update(k)
			}
			tc.assert(t, l)
		})
	}
}

func TestListOverlay_NavigationEmptyList(t *testing.T) {
	l := NewListOverlay("test", nil, tui.DefaultTheme())

	// Navigation on empty list should not panic.
	l.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 0, l.Cursor())

	l.Update(tea.KeyMsg{Type: tea.KeyUp})
	assert.Equal(t, 0, l.Cursor())
}

func TestListOverlay_AddItem(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, l *ListOverlay, result *ListOverlayResultMsg)
		name   string
		keys   []tea.KeyMsg
	}{
		{
			name: "a enters adding state",
			keys: []tea.KeyMsg{runeKey('a')},
			assert: func(t *testing.T, l *ListOverlay, _ *ListOverlayResultMsg) {
				t.Helper()
				assert.Equal(t, listAdding, l.State())
			},
		},
		{
			name: "add item with enter confirms",
			keys: []tea.KeyMsg{
				runeKey('a'),
				// Type "new item" character by character
				runeKey('n'), runeKey('e'), runeKey('w'),
				enterKey(),
			},
			assert: func(t *testing.T, l *ListOverlay, _ *ListOverlayResultMsg) {
				t.Helper()
				assert.Equal(t, listBrowsing, l.State())
				items := l.Items()
				assert.Len(t, items, 3)
				assert.Equal(t, "new", items[2])
				assert.Equal(t, 2, l.Cursor(), "cursor should move to new item")
			},
		},
		{
			name: "add empty item is ignored",
			keys: []tea.KeyMsg{
				runeKey('a'),
				enterKey(),
			},
			assert: func(t *testing.T, l *ListOverlay, _ *ListOverlayResultMsg) {
				t.Helper()
				assert.Equal(t, listBrowsing, l.State())
				assert.Len(t, l.Items(), 2)
			},
		},
		{
			name: "esc cancels add",
			keys: []tea.KeyMsg{
				runeKey('a'),
				runeKey('x'),
				escKey(),
			},
			assert: func(t *testing.T, l *ListOverlay, _ *ListOverlayResultMsg) {
				t.Helper()
				assert.Equal(t, listBrowsing, l.State())
				assert.Len(t, l.Items(), 2)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			l := NewListOverlay("test", []string{"first", "second"}, tui.DefaultTheme())
			var lastResult *ListOverlayResultMsg
			for _, k := range tc.keys {
				result, _ := l.Update(k)
				if result != nil {
					lastResult = result
				}
			}
			tc.assert(t, l, lastResult)
		})
	}
}

func TestListOverlay_DeleteItem(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, l *ListOverlay)
		name   string
		keys   []tea.KeyMsg
	}{
		{
			name: "d enters confirm delete state",
			keys: []tea.KeyMsg{runeKey('d')},
			assert: func(t *testing.T, l *ListOverlay) {
				t.Helper()
				assert.Equal(t, listConfirmDelete, l.State())
			},
		},
		{
			name: "d on empty list is noop",
			keys: []tea.KeyMsg{runeKey('d')},
			assert: func(t *testing.T, l *ListOverlay) {
				t.Helper()
				assert.Equal(t, listBrowsing, l.State())
			},
		},
		{
			name: "y confirms delete",
			keys: []tea.KeyMsg{runeKey('d'), runeKey('y')},
			assert: func(t *testing.T, l *ListOverlay) {
				t.Helper()
				assert.Equal(t, listBrowsing, l.State())
				assert.Equal(t, []string{"second", "third"}, l.Items())
			},
		},
		{
			name: "n cancels delete",
			keys: []tea.KeyMsg{runeKey('d'), runeKey('n')},
			assert: func(t *testing.T, l *ListOverlay) {
				t.Helper()
				assert.Equal(t, listBrowsing, l.State())
				assert.Len(t, l.Items(), 3)
			},
		},
		{
			name: "esc cancels delete",
			keys: []tea.KeyMsg{runeKey('d'), escKey()},
			assert: func(t *testing.T, l *ListOverlay) {
				t.Helper()
				assert.Equal(t, listBrowsing, l.State())
				assert.Len(t, l.Items(), 3)
			},
		},
		{
			name: "delete last item adjusts cursor",
			keys: []tea.KeyMsg{
				{Type: tea.KeyDown}, // cursor=1
				{Type: tea.KeyDown}, // cursor=2
				runeKey('d'),
				runeKey('y'),
			},
			assert: func(t *testing.T, l *ListOverlay) {
				t.Helper()
				assert.Equal(t, listBrowsing, l.State())
				assert.Equal(t, []string{"first", "second"}, l.Items())
				assert.Equal(t, 1, l.Cursor(), "cursor should clamp to new last item")
			},
		},
		{
			name: "delete middle item keeps cursor",
			keys: []tea.KeyMsg{
				{Type: tea.KeyDown}, // cursor=1
				runeKey('d'),
				runeKey('y'),
			},
			assert: func(t *testing.T, l *ListOverlay) {
				t.Helper()
				assert.Equal(t, []string{"first", "third"}, l.Items())
				assert.Equal(t, 1, l.Cursor())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var l *ListOverlay
			if tc.name == "d on empty list is noop" {
				l = NewListOverlay("test", nil, tui.DefaultTheme())
			} else {
				l = NewListOverlay("test", []string{"first", "second", "third"}, tui.DefaultTheme())
			}
			for _, k := range tc.keys {
				l.Update(k)
			}
			tc.assert(t, l)
		})
	}
}

func TestListOverlay_Commit(t *testing.T) {
	l := NewListOverlay("test", []string{"a", "b"}, tui.DefaultTheme())

	result, _ := l.Update(enterKey())
	require.NotNil(t, result)
	assert.True(t, result.Committed)
	assert.Equal(t, []string{"a", "b"}, result.Items)
}

func TestListOverlay_CommitAfterEdits(t *testing.T) {
	l := NewListOverlay("test", []string{"a", "b"}, tui.DefaultTheme())

	// Add an item.
	l.Update(runeKey('a'))
	l.Update(runeKey('c'))
	l.Update(enterKey())

	// Commit.
	result, _ := l.Update(enterKey())
	require.NotNil(t, result)
	assert.True(t, result.Committed)
	assert.Equal(t, []string{"a", "b", "c"}, result.Items)
}

func TestListOverlay_EscNoChanges(t *testing.T) {
	l := NewListOverlay("test", []string{"a", "b"}, tui.DefaultTheme())

	result, _ := l.Update(escKey())
	require.NotNil(t, result)
	assert.False(t, result.Committed)
}

func TestListOverlay_EscWithChanges(t *testing.T) {
	testCases := []struct {
		assertResult func(t *testing.T, result *ListOverlayResultMsg)
		assert       func(t *testing.T, l *ListOverlay)
		name         string
		keys         []tea.KeyMsg
	}{
		{
			name: "esc with changes shows discard confirmation",
			keys: []tea.KeyMsg{
				runeKey('a'), runeKey('x'), enterKey(), // add item
				escKey(), // try to discard
			},
			assert: func(t *testing.T, l *ListOverlay) {
				t.Helper()
				assert.Equal(t, listConfirmDiscard, l.State())
			},
			assertResult: func(t *testing.T, result *ListOverlayResultMsg) {
				t.Helper()
				assert.Nil(t, result)
			},
		},
		{
			name: "y confirms discard",
			keys: []tea.KeyMsg{
				runeKey('a'), runeKey('x'), enterKey(),
				escKey(),
				runeKey('y'),
			},
			assert: func(t *testing.T, _ *ListOverlay) {
				t.Helper()
			},
			assertResult: func(t *testing.T, result *ListOverlayResultMsg) {
				t.Helper()
				require.NotNil(t, result)
				assert.False(t, result.Committed)
			},
		},
		{
			name: "n cancels discard",
			keys: []tea.KeyMsg{
				runeKey('a'), runeKey('x'), enterKey(),
				escKey(),
				runeKey('n'),
			},
			assert: func(t *testing.T, l *ListOverlay) {
				t.Helper()
				assert.Equal(t, listBrowsing, l.State())
				assert.Len(t, l.Items(), 3)
			},
			assertResult: func(t *testing.T, result *ListOverlayResultMsg) {
				t.Helper()
				assert.Nil(t, result)
			},
		},
		{
			name: "esc cancels discard",
			keys: []tea.KeyMsg{
				runeKey('a'), runeKey('x'), enterKey(),
				escKey(),
				escKey(),
			},
			assert: func(t *testing.T, l *ListOverlay) {
				t.Helper()
				assert.Equal(t, listBrowsing, l.State())
			},
			assertResult: func(t *testing.T, result *ListOverlayResultMsg) {
				t.Helper()
				assert.Nil(t, result)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			l := NewListOverlay("test", []string{"a", "b"}, tui.DefaultTheme())
			var lastResult *ListOverlayResultMsg
			for _, k := range tc.keys {
				result, _ := l.Update(k)
				if result != nil {
					lastResult = result
				}
			}
			tc.assert(t, l)
			tc.assertResult(t, lastResult)
		})
	}
}

func TestListOverlay_View(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, view string)
		setup  func(l *ListOverlay)
		name   string
		title  string
		items  []string
		width  int
		height int
	}{
		{
			name:   "contains title",
			title:  "JIRA > Filters > Status",
			items:  []string{"In Dev", "Open"},
			width:  80,
			height: 30,
			setup:  func(_ *ListOverlay) {},
			assert: func(t *testing.T, view string) {
				t.Helper()
				assert.Contains(t, view, "JIRA > Filters > Status")
			},
		},
		{
			name:   "shows numbered items",
			title:  "Test",
			items:  []string{"Alpha", "Beta", "Gamma"},
			width:  80,
			height: 30,
			setup:  func(_ *ListOverlay) {},
			assert: func(t *testing.T, view string) {
				t.Helper()
				assert.Contains(t, view, "1.")
				assert.Contains(t, view, "Alpha")
				assert.Contains(t, view, "2.")
				assert.Contains(t, view, "Beta")
				assert.Contains(t, view, "3.")
				assert.Contains(t, view, "Gamma")
			},
		},
		{
			name:   "shows cursor on selected item",
			title:  "Test",
			items:  []string{"Alpha", "Beta"},
			width:  80,
			height: 30,
			setup:  func(_ *ListOverlay) {},
			assert: func(t *testing.T, view string) {
				t.Helper()
				assert.Contains(t, view, ">")
			},
		},
		{
			name:   "shows empty placeholder",
			title:  "Empty",
			items:  nil,
			width:  80,
			height: 30,
			setup:  func(_ *ListOverlay) {},
			assert: func(t *testing.T, view string) {
				t.Helper()
				assert.Contains(t, view, "(empty list)")
			},
		},
		{
			name:   "shows footer hints in browsing",
			title:  "Test",
			items:  []string{"x"},
			width:  80,
			height: 30,
			setup:  func(_ *ListOverlay) {},
			assert: func(t *testing.T, view string) {
				t.Helper()
				assert.Contains(t, view, "add")
				assert.Contains(t, view, "delete")
				assert.Contains(t, view, "confirm")
				assert.Contains(t, view, "discard")
			},
		},
		{
			name:   "shows delete confirmation prompt",
			title:  "Test",
			items:  []string{"victim"},
			width:  80,
			height: 30,
			setup: func(l *ListOverlay) {
				l.Update(runeKey('d'))
			},
			assert: func(t *testing.T, view string) {
				t.Helper()
				assert.Contains(t, view, "Delete")
				assert.Contains(t, view, "victim")
				assert.Contains(t, view, "y/n")
			},
		},
		{
			name:   "shows discard confirmation prompt",
			title:  "Test",
			items:  []string{"a"},
			width:  80,
			height: 30,
			setup: func(l *ListOverlay) {
				l.items = append(l.items, "new") // mutate to have changes
				l.Update(escKey())
			},
			assert: func(t *testing.T, view string) {
				t.Helper()
				assert.Contains(t, view, "Discard changes")
				assert.Contains(t, view, "y/n")
			},
		},
		{
			name:   "handles small viewport gracefully",
			title:  "Tiny",
			items:  []string{"a", "b", "c"},
			width:  30,
			height: 10,
			setup:  func(_ *ListOverlay) {},
			assert: func(t *testing.T, view string) {
				t.Helper()
				assert.NotEmpty(t, view)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			l := NewListOverlay(tc.title, tc.items, tui.DefaultTheme())
			tc.setup(l)
			view := l.View(tc.width, tc.height)
			tc.assert(t, view)
		})
	}
}

func TestListOverlay_SetSize(t *testing.T) {
	l := NewListOverlay("test", []string{"a"}, tui.DefaultTheme())
	l.SetSize(100, 50)
	assert.Equal(t, 100, l.width)
	assert.Equal(t, 50, l.height)
}

func TestListOverlay_NonKeyMsgIgnored(t *testing.T) {
	l := NewListOverlay("test", []string{"a"}, tui.DefaultTheme())
	result, cmd := l.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	assert.Nil(t, result)
	assert.Nil(t, cmd)
}

func TestListOverlay_UnhandledKeysInConfirmStates(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, l *ListOverlay)
		name   string
		setup  func(l *ListOverlay)
		key    tea.KeyMsg
	}{
		{
			name: "unhandled key in confirm delete stays in state",
			setup: func(l *ListOverlay) {
				l.Update(runeKey('d'))
			},
			key: runeKey('x'),
			assert: func(t *testing.T, l *ListOverlay) {
				t.Helper()
				assert.Equal(t, listConfirmDelete, l.State())
			},
		},
		{
			name: "unhandled key in confirm discard stays in state",
			setup: func(l *ListOverlay) {
				l.items = append(l.items, "new")
				l.Update(escKey())
			},
			key: runeKey('x'),
			assert: func(t *testing.T, l *ListOverlay) {
				t.Helper()
				assert.Equal(t, listConfirmDiscard, l.State())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			l := NewListOverlay("test", []string{"a", "b"}, tui.DefaultTheme())
			tc.setup(l)
			l.Update(tc.key)
			tc.assert(t, l)
		})
	}
}

func TestListOverlay_DeleteTruncatesLongName(t *testing.T) {
	longName := "this is a very long item name that exceeds twenty characters"
	l := NewListOverlay("test", []string{longName}, tui.DefaultTheme())
	l.Update(runeKey('d'))

	view := l.View(80, 30)
	assert.Contains(t, view, "...")
	assert.Contains(t, view, "Delete")
}

func TestListOverlay_AddToEmptyList(t *testing.T) {
	l := NewListOverlay("test", nil, tui.DefaultTheme())

	l.Update(runeKey('a'))
	assert.Equal(t, listAdding, l.State())

	l.Update(runeKey('x'))
	l.Update(enterKey())

	assert.Equal(t, listBrowsing, l.State())
	assert.Equal(t, []string{"x"}, l.Items())
	assert.Equal(t, 0, l.Cursor())
}

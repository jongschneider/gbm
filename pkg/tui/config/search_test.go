package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearchFilter_NewSearchFilter(t *testing.T) {
	testCases := []struct {
		name   string
		assert func(t *testing.T, sf *SearchFilter)
	}{
		{
			name: "default state is inactive",
			assert: func(t *testing.T, sf *SearchFilter) {
				t.Helper()
				assert.False(t, sf.IsActive())
				assert.Empty(t, sf.Query())
			},
		},
		{
			name: "nil theme uses default",
			assert: func(t *testing.T, _ *SearchFilter) {
				t.Helper()
				sf := NewSearchFilter(nil)
				assert.NotNil(t, sf.theme)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sf := NewSearchFilter(nil)
			tc.assert(t, sf)
		})
	}
}

func TestSearchFilter_OpenClose(t *testing.T) {
	testCases := []struct {
		name   string
		action func(sf *SearchFilter)
		assert func(t *testing.T, sf *SearchFilter)
	}{
		{
			name: "open activates with empty query",
			action: func(sf *SearchFilter) {
				sf.Open()
			},
			assert: func(t *testing.T, sf *SearchFilter) {
				t.Helper()
				assert.True(t, sf.IsActive())
				assert.Empty(t, sf.Query())
			},
		},
		{
			name: "close deactivates and clears query",
			action: func(sf *SearchFilter) {
				sf.Open()
				sf.HandleRune('h')
				sf.HandleRune('i')
				sf.Close()
			},
			assert: func(t *testing.T, sf *SearchFilter) {
				t.Helper()
				assert.False(t, sf.IsActive())
				assert.Empty(t, sf.Query())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sf := NewSearchFilter(nil)
			tc.action(sf)
			tc.assert(t, sf)
		})
	}
}

func TestSearchFilter_HandleRune(t *testing.T) {
	testCases := []struct {
		name   string
		runes  []rune
		assert func(t *testing.T, sf *SearchFilter)
	}{
		{
			name:  "appends characters",
			runes: []rune{'h', 'e', 'l', 'l', 'o'},
			assert: func(t *testing.T, sf *SearchFilter) {
				t.Helper()
				assert.Equal(t, "hello", sf.Query())
			},
		},
		{
			name:  "single character",
			runes: []rune{'x'},
			assert: func(t *testing.T, sf *SearchFilter) {
				t.Helper()
				assert.Equal(t, "x", sf.Query())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sf := NewSearchFilter(nil)
			sf.Open()
			for _, r := range tc.runes {
				sf.HandleRune(r)
			}
			tc.assert(t, sf)
		})
	}
}

func TestSearchFilter_HandleBackspace(t *testing.T) {
	testCases := []struct {
		name   string
		setup  func(sf *SearchFilter)
		assert func(t *testing.T, sf *SearchFilter)
	}{
		{
			name: "removes last character",
			setup: func(sf *SearchFilter) {
				sf.HandleRune('a')
				sf.HandleRune('b')
				sf.HandleRune('c')
				sf.HandleBackspace()
			},
			assert: func(t *testing.T, sf *SearchFilter) {
				t.Helper()
				assert.Equal(t, "ab", sf.Query())
			},
		},
		{
			name: "no-op on empty query",
			setup: func(sf *SearchFilter) {
				sf.HandleBackspace()
			},
			assert: func(t *testing.T, sf *SearchFilter) {
				t.Helper()
				assert.Empty(t, sf.Query())
			},
		},
		{
			name: "removes all characters one by one",
			setup: func(sf *SearchFilter) {
				sf.HandleRune('x')
				sf.HandleBackspace()
			},
			assert: func(t *testing.T, sf *SearchFilter) {
				t.Helper()
				assert.Empty(t, sf.Query())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sf := NewSearchFilter(nil)
			sf.Open()
			tc.setup(sf)
			tc.assert(t, sf)
		})
	}
}

func TestSearchFilter_FilterRows(t *testing.T) {
	groupedRows := []Row{
		{Label: "Connection", Group: "Connection", Kind: RowGroupHeader, FieldIndex: -1, EntryIndex: -1},
		{Label: "Host", Group: "Connection", Kind: RowField, FieldIndex: 0, EntryIndex: -1},
		{Label: "Username", Group: "Connection", Kind: RowField, FieldIndex: 1, EntryIndex: -1},
		{Label: "Filters", Group: "Filters", Kind: RowGroupHeader, FieldIndex: -1, EntryIndex: -1},
		{Label: "Priority", Group: "Filters", Kind: RowField, FieldIndex: 2, EntryIndex: -1},
		{Label: "Type", Group: "Filters", Kind: RowField, FieldIndex: 3, EntryIndex: -1},
		{Label: "Reverse", Group: "Filters", Kind: RowField, FieldIndex: 4, EntryIndex: -1},
		{Label: "Markdown", Group: "Markdown", Kind: RowGroupHeader, FieldIndex: -1, EntryIndex: -1},
		{Label: "Max Depth", Group: "Markdown", Kind: RowField, FieldIndex: 5, EntryIndex: -1},
	}

	testCases := []struct {
		name   string
		query  string
		rows   []Row
		active bool
		assert func(t *testing.T, filtered []Row)
	}{
		{
			name:   "inactive returns all rows",
			rows:   groupedRows,
			active: false,
			query:  "",
			assert: func(t *testing.T, filtered []Row) {
				t.Helper()
				assert.Len(t, filtered, len(groupedRows))
			},
		},
		{
			name:   "empty query returns all rows",
			rows:   groupedRows,
			active: true,
			query:  "",
			assert: func(t *testing.T, filtered []Row) {
				t.Helper()
				assert.Len(t, filtered, len(groupedRows))
			},
		},
		{
			name:   "case-insensitive match",
			rows:   groupedRows,
			active: true,
			query:  "host",
			assert: func(t *testing.T, filtered []Row) {
				t.Helper()
				// Should include: Connection header + Host field
				require.Len(t, filtered, 2)
				assert.Equal(t, RowGroupHeader, filtered[0].Kind)
				assert.Equal(t, "Connection", filtered[0].Label)
				assert.Equal(t, RowField, filtered[1].Kind)
				assert.Equal(t, "Host", filtered[1].Label)
			},
		},
		{
			name:   "uppercase query matches lowercase label",
			rows:   groupedRows,
			active: true,
			query:  "HOST",
			assert: func(t *testing.T, filtered []Row) {
				t.Helper()
				require.Len(t, filtered, 2)
				assert.Equal(t, "Host", filtered[1].Label)
			},
		},
		{
			name:   "substring match across groups",
			rows:   groupedRows,
			active: true,
			query:  "er",
			assert: func(t *testing.T, filtered []Row) {
				t.Helper()
				// Matches: Username (Connection), Reverse (Filters)
				// So: Connection header, Username, Filters header, Reverse
				require.Len(t, filtered, 4)
				assert.Equal(t, "Connection", filtered[0].Label)
				assert.Equal(t, "Username", filtered[1].Label)
				assert.Equal(t, "Filters", filtered[2].Label)
				assert.Equal(t, "Reverse", filtered[3].Label)
			},
		},
		{
			name:   "no matches returns empty",
			rows:   groupedRows,
			active: true,
			query:  "zzzzz",
			assert: func(t *testing.T, filtered []Row) {
				t.Helper()
				assert.Empty(t, filtered)
			},
		},
		{
			name: "entry rows excluded during search",
			rows: append(groupedRows,
				Row{Label: "Rules", Group: "Rules", Kind: RowGroupHeader, FieldIndex: -1, EntryIndex: -1},
				Row{Label: "rule1", Group: "Rules", Kind: RowEntry, FieldIndex: -1, EntryIndex: 0},
			),
			active: true,
			query:  "rule",
			assert: func(t *testing.T, filtered []Row) {
				t.Helper()
				// Entry rows are excluded during search.
				assert.Empty(t, filtered)
			},
		},
		{
			name: "empty rows excluded during search",
			rows: append(groupedRows,
				Row{Label: "(empty)", Group: "Rules", Kind: RowEmpty, FieldIndex: -1, EntryIndex: -1},
			),
			active: true,
			query:  "empty",
			assert: func(t *testing.T, filtered []Row) {
				t.Helper()
				assert.Empty(t, filtered)
			},
		},
		{
			name:   "groups with no matching fields are hidden",
			rows:   groupedRows,
			active: true,
			query:  "depth",
			assert: func(t *testing.T, filtered []Row) {
				t.Helper()
				// Only Markdown group has "Max Depth"
				require.Len(t, filtered, 2)
				assert.Equal(t, RowGroupHeader, filtered[0].Kind)
				assert.Equal(t, "Markdown", filtered[0].Label)
				assert.Equal(t, "Max Depth", filtered[1].Label)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sf := NewSearchFilter(nil)
			if tc.active {
				sf.Open()
			}
			for _, r := range tc.query {
				sf.HandleRune(r)
			}
			filtered := sf.FilterRows(tc.rows)
			tc.assert(t, filtered)
		})
	}
}

func TestSearchFilter_View(t *testing.T) {
	testCases := []struct {
		name   string
		setup  func(sf *SearchFilter)
		assert func(t *testing.T, view string)
	}{
		{
			name: "inactive returns empty",
			setup: func(_ *SearchFilter) {
				// Don't open.
			},
			assert: func(t *testing.T, view string) {
				t.Helper()
				assert.Empty(t, view)
			},
		},
		{
			name: "active with empty query shows prompt and cursor",
			setup: func(sf *SearchFilter) {
				sf.Open()
			},
			assert: func(t *testing.T, view string) {
				t.Helper()
				assert.Contains(t, view, "/")
				assert.Contains(t, view, "_")
			},
		},
		{
			name: "active with query shows prompt query and cursor",
			setup: func(sf *SearchFilter) {
				sf.Open()
				sf.HandleRune('h')
				sf.HandleRune('o')
				sf.HandleRune('s')
				sf.HandleRune('t')
			},
			assert: func(t *testing.T, view string) {
				t.Helper()
				assert.Contains(t, view, "/")
				assert.Contains(t, view, "host")
				assert.Contains(t, view, "_")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sf := NewSearchFilter(nil)
			tc.setup(sf)
			view := sf.View(72)
			tc.assert(t, view)
		})
	}
}

// --- SectionModel search integration tests ---.

func TestSectionModel_OpenSearch(t *testing.T) {
	s := NewSectionModel(testGroupedFields(), WithViewportHeight(20), WithWidth(72))

	s.OpenSearch()

	assert.True(t, s.IsSearchActive())
	assert.Empty(t, s.Search().Query())
}

func TestSectionModel_CloseSearch(t *testing.T) {
	s := NewSectionModel(testGroupedFields(), WithViewportHeight(20), WithWidth(72))

	s.OpenSearch()
	s.SearchHandleRune('h')
	s.CloseSearch()

	assert.False(t, s.IsSearchActive())
	assert.Empty(t, s.Search().Query())
	// Focus should be restored to first focusable row.
	assert.Equal(t, "Host", s.FocusedRow().Label)
}

func TestSectionModel_SearchFiltersFields(t *testing.T) {
	s := NewSectionModel(testGroupedFields(), WithViewportHeight(20), WithWidth(72))

	s.OpenSearch()
	s.SearchHandleRune('r')
	s.SearchHandleRune('e')
	s.SearchHandleRune('v')

	// Should filter to "Reverse" in Filters group.
	rows := s.Rows()
	fieldRows := 0
	for _, r := range rows {
		if r.Kind == RowField {
			fieldRows++
		}
	}
	assert.Equal(t, 1, fieldRows, "only Reverse should match")
	assert.Equal(t, 1, s.FieldCount())
	assert.Equal(t, "Reverse", s.FocusedRow().Label)
}

func TestSectionModel_SearchPositionIndicator(t *testing.T) {
	s := NewSectionModel(testGroupedFields(), WithViewportHeight(20), WithWidth(72))

	s.OpenSearch()
	s.SearchHandleRune('e') // Matches: Reverse, Type, Username (all contain 'e')

	// Position indicator should reflect filtered count.
	assert.Equal(t, 1, s.FocusPosition())
	total := s.FieldCount()
	assert.Greater(t, total, 0)

	view := s.View()
	// Should contain the position indicator with filtered count.
	assert.Contains(t, view, "/")
}

func TestSectionModel_SearchHidesNonMatchingGroups(t *testing.T) {
	s := NewSectionModel(testGroupedFields(), WithViewportHeight(20), WithWidth(72))

	s.OpenSearch()
	s.SearchHandleRune('d')
	s.SearchHandleRune('e')
	s.SearchHandleRune('p')
	s.SearchHandleRune('t')
	s.SearchHandleRune('h')

	// Only "Max Depth" should match, in Markdown group.
	rows := s.Rows()
	for _, r := range rows {
		if r.Kind == RowGroupHeader {
			assert.Equal(t, "Markdown", r.Label,
				"only Markdown group header should be visible")
		}
	}
}

func TestSectionModel_SearchBackspaceWidensResults(t *testing.T) {
	s := NewSectionModel(testGroupedFields(), WithViewportHeight(20), WithWidth(72))

	s.OpenSearch()
	s.SearchHandleRune('h')
	s.SearchHandleRune('o')
	s.SearchHandleRune('s')
	s.SearchHandleRune('t')

	countBefore := s.FieldCount()

	s.SearchHandleBackspace() // "hos"
	s.SearchHandleBackspace() // "ho"
	s.SearchHandleBackspace() // "h"

	countAfter := s.FieldCount()
	assert.GreaterOrEqual(t, countAfter, countBefore)
}

func TestSectionModel_SearchNavigationOnFilteredList(t *testing.T) {
	s := NewSectionModel(testGroupedFields(), WithViewportHeight(20), WithWidth(72))

	s.OpenSearch()
	// "e" matches multiple fields across groups.
	s.SearchHandleRune('e')

	firstLabel := s.FocusedRow().Label

	s.MoveFocusDown()
	secondLabel := s.FocusedRow().Label
	assert.NotEqual(t, firstLabel, secondLabel, "should move to next matching field")

	s.MoveFocusUp()
	assert.Equal(t, firstLabel, s.FocusedRow().Label, "should return to first matching field")
}

func TestSectionModel_SearchViewRendersSearchBar(t *testing.T) {
	s := NewSectionModel(testGroupedFields(), WithViewportHeight(20), WithWidth(72))

	s.OpenSearch()
	s.SearchHandleRune('h')

	view := s.View()
	lines := strings.Split(view, "\n")

	// First line should be the search bar containing "/" and the query.
	require.NotEmpty(t, lines)
	assert.Contains(t, lines[0], "/")
	assert.Contains(t, lines[0], "h")
}

func TestSectionModel_SearchNoMatchesShowsSearchBar(t *testing.T) {
	s := NewSectionModel(testGroupedFields(), WithViewportHeight(20), WithWidth(72))

	s.OpenSearch()
	s.SearchHandleRune('z')
	s.SearchHandleRune('z')
	s.SearchHandleRune('z')

	view := s.View()
	// Even with no matches, the search bar should be visible.
	assert.Contains(t, view, "/")
	assert.Contains(t, view, "zzz")
}

func TestSectionModel_SearchGroupJumping(t *testing.T) {
	s := NewSectionModel(testGroupedFields(), WithViewportHeight(20), WithWidth(72))

	s.OpenSearch()
	// "e" matches fields in Connection, Filters, and Markdown groups.
	s.SearchHandleRune('e')

	initialGroup := s.FocusedRow().Group

	s.JumpToNextGroup()
	nextGroup := s.FocusedRow().Group
	assert.NotEqual(t, initialGroup, nextGroup, "should jump to a different group")

	s.JumpToPrevGroup()
	assert.Equal(t, initialGroup, s.FocusedRow().Group, "should return to initial group")
}

func TestSectionModel_SearchJumpToFirstLast(t *testing.T) {
	s := NewSectionModel(testGroupedFields(), WithViewportHeight(20), WithWidth(72))

	s.OpenSearch()
	s.SearchHandleRune('e') // multiple matches

	s.JumpToLast()
	lastLabel := s.FocusedRow().Label

	s.JumpToFirst()
	firstLabel := s.FocusedRow().Label

	assert.NotEqual(t, firstLabel, lastLabel, "first and last should differ when multiple matches")
}

func TestSectionModel_SearchEmptyQueryShowsAllRows(t *testing.T) {
	s := NewSectionModel(testGroupedFields(), WithViewportHeight(20), WithWidth(72))

	s.OpenSearch()
	// With empty query, all rows should be visible.
	assert.Equal(t, 7, s.FieldCount())

	s.SearchHandleRune('h')
	filteredCount := s.FieldCount()
	assert.Less(t, filteredCount, 7)

	s.SearchHandleBackspace()
	// Back to empty query, all rows should return.
	assert.Equal(t, 7, s.FieldCount())
}

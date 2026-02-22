package config

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testFields returns a small set of fields for testing.
func testFields() []FieldMeta {
	return []FieldMeta{
		{Key: "default_branch", Label: "Default Branch", Type: String},
		{Key: "worktrees_dir", Label: "Worktrees Dir", Type: String},
	}
}

// testGroupedFields returns fields with groups to test group headers and navigation.
func testGroupedFields() []FieldMeta {
	return []FieldMeta{
		{Key: "jira.host", Label: "Host", Type: String, Group: "Connection"},
		{Key: "jira.me", Label: "Username", Type: String, Group: "Connection"},
		{Key: "jira.filters.priority", Label: "Priority", Type: String, Group: "Filters"},
		{Key: "jira.filters.type", Label: "Type", Type: String, Group: "Filters"},
		{Key: "jira.filters.reverse", Label: "Reverse", Type: Bool, Group: "Filters"},
		{Key: "jira.markdown.max_depth", Label: "Max Depth", Type: Int, Group: "Markdown"},
		{Key: "jira.markdown.include_comments", Label: "Include Comments", Type: Bool, Group: "Markdown"},
	}
}

func TestNewSectionModel(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, s *SectionModel)
		name   string
		fields []FieldMeta
		opts   []SectionOption
	}{
		{
			name:   "basic fields without groups",
			fields: testFields(),
			assert: func(t *testing.T, s *SectionModel) {
				t.Helper()
				assert.Len(t, s.rows, 2, "should have 2 field rows")
				assert.Equal(t, RowField, s.rows[0].Kind)
				assert.Equal(t, RowField, s.rows[1].Kind)
				assert.Equal(t, 0, s.focusIndex, "focus should start at first field")
			},
		},
		{
			name:   "grouped fields have headers",
			fields: testGroupedFields(),
			assert: func(t *testing.T, s *SectionModel) {
				t.Helper()
				// 3 group headers + 7 fields = 10 rows
				assert.Len(t, s.rows, 10)

				// First row should be a group header.
				assert.Equal(t, RowGroupHeader, s.rows[0].Kind)
				assert.Equal(t, "Connection", s.rows[0].Label)

				// Second row should be a field.
				assert.Equal(t, RowField, s.rows[1].Kind)
				assert.Equal(t, "Host", s.rows[1].Label)

				// Focus should be on first field, not header.
				assert.Equal(t, 1, s.focusIndex)
			},
		},
		{
			name:   "with entry list and entries",
			fields: testFields(),
			opts: []SectionOption{
				WithEntryList("Rules", []string{
					"main -> .env, config/",
					"develop -> .vscode/settings.json",
				}, "(no rules configured) -- press a to add"),
			},
			assert: func(t *testing.T, s *SectionModel) {
				t.Helper()
				// 2 fields + 1 group header for "Rules" + 2 entry rows = 5
				assert.Len(t, s.rows, 5)

				// Last two should be entry rows.
				assert.Equal(t, RowEntry, s.rows[3].Kind)
				assert.Equal(t, 0, s.rows[3].EntryIndex)
				assert.Equal(t, RowEntry, s.rows[4].Kind)
				assert.Equal(t, 1, s.rows[4].EntryIndex)
			},
		},
		{
			name:   "with empty entry list shows placeholder",
			fields: testFields(),
			opts: []SectionOption{
				WithEntryList("Rules", nil, "(no rules configured) -- press a to add"),
			},
			assert: func(t *testing.T, s *SectionModel) {
				t.Helper()
				// 2 fields + 1 group header + 1 empty row = 4
				assert.Len(t, s.rows, 4)
				assert.Equal(t, RowEmpty, s.rows[3].Kind)
				assert.Equal(t, "(no rules configured) -- press a to add", s.rows[3].Label)
			},
		},
		{
			name:   "nil theme option does not override default",
			fields: testFields(),
			opts:   []SectionOption{WithSectionTheme(nil)},
			assert: func(t *testing.T, s *SectionModel) {
				t.Helper()
				assert.NotNil(t, s.theme)
			},
		},
		{
			name:   "viewport height option",
			fields: testFields(),
			opts:   []SectionOption{WithViewportHeight(10)},
			assert: func(t *testing.T, s *SectionModel) {
				t.Helper()
				assert.Equal(t, 10, s.viewportHeight)
			},
		},
		{
			name:   "width option",
			fields: testFields(),
			opts:   []SectionOption{WithWidth(80)},
			assert: func(t *testing.T, s *SectionModel) {
				t.Helper()
				assert.Equal(t, 80, s.width)
			},
		},
		{
			name:   "zero viewport height ignored",
			fields: testFields(),
			opts:   []SectionOption{WithViewportHeight(0)},
			assert: func(t *testing.T, s *SectionModel) {
				t.Helper()
				assert.Equal(t, 20, s.viewportHeight, "should keep default")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s := NewSectionModel(tc.fields, tc.opts...)
			tc.assert(t, s)
		})
	}
}

func TestSectionModel_RowIsFocusable(t *testing.T) {
	testCases := []struct {
		name     string
		kind     RowKind
		expected bool
	}{
		{name: "field is focusable", kind: RowField, expected: true},
		{name: "entry is focusable", kind: RowEntry, expected: true},
		{name: "group header is not focusable", kind: RowGroupHeader, expected: false},
		{name: "empty is not focusable", kind: RowEmpty, expected: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := Row{Kind: tc.kind}
			assert.Equal(t, tc.expected, r.IsFocusable())
		})
	}
}

func TestSectionModel_FocusedRow(t *testing.T) {
	t.Run("returns focused row", func(t *testing.T) {
		s := NewSectionModel(testGroupedFields())
		r := s.FocusedRow()
		assert.Equal(t, RowField, r.Kind)
		assert.Equal(t, "Host", r.Label)
	})

	t.Run("empty section returns zero row", func(t *testing.T) {
		s := NewSectionModel(nil)
		r := s.FocusedRow()
		assert.Equal(t, -1, r.FieldIndex)
		assert.Equal(t, -1, r.EntryIndex)
	})
}

func TestSectionModel_FieldCount(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, count int)
		name   string
		fields []FieldMeta
		opts   []SectionOption
	}{
		{
			name:   "ungrouped fields",
			fields: testFields(),
			assert: func(t *testing.T, count int) {
				t.Helper()
				assert.Equal(t, 2, count)
			},
		},
		{
			name:   "grouped fields exclude headers",
			fields: testGroupedFields(),
			assert: func(t *testing.T, count int) {
				t.Helper()
				assert.Equal(t, 7, count)
			},
		},
		{
			name:   "with entries",
			fields: testFields(),
			opts: []SectionOption{
				WithEntryList("Rules", []string{"entry1", "entry2"}, ""),
			},
			assert: func(t *testing.T, count int) {
				t.Helper()
				assert.Equal(t, 4, count, "2 fields + 2 entries")
			},
		},
		{
			name:   "empty entries do not count",
			fields: testFields(),
			opts: []SectionOption{
				WithEntryList("Rules", nil, "(empty)"),
			},
			assert: func(t *testing.T, count int) {
				t.Helper()
				assert.Equal(t, 2, count, "only 2 fields, empty row not focusable")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s := NewSectionModel(tc.fields, tc.opts...)
			tc.assert(t, s.FieldCount())
		})
	}
}

func TestSectionModel_FocusPosition(t *testing.T) {
	s := NewSectionModel(testGroupedFields())

	// Initial position should be 1 (first focusable).
	assert.Equal(t, 1, s.FocusPosition())

	// Move down and check position increments.
	s.MoveFocusDown()
	assert.Equal(t, 2, s.FocusPosition())

	s.MoveFocusDown()
	assert.Equal(t, 3, s.FocusPosition())
}

func TestSectionModel_MoveFocusDown(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, s *SectionModel)
		name   string
		fields []FieldMeta
		moves  int
	}{
		{
			name:   "moves to next field",
			fields: testFields(),
			moves:  1,
			assert: func(t *testing.T, s *SectionModel) {
				t.Helper()
				assert.Equal(t, "Worktrees Dir", s.FocusedRow().Label)
			},
		},
		{
			name:   "skips group headers",
			fields: testGroupedFields(),
			moves:  2, // Host -> Username -> skip Filters header -> Priority
			assert: func(t *testing.T, s *SectionModel) {
				t.Helper()
				assert.Equal(t, "Priority", s.FocusedRow().Label)
			},
		},
		{
			name:   "wraps from last to first",
			fields: testFields(),
			moves:  2,
			assert: func(t *testing.T, s *SectionModel) {
				t.Helper()
				assert.Equal(t, "Default Branch", s.FocusedRow().Label)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s := NewSectionModel(tc.fields)
			for range tc.moves {
				s.MoveFocusDown()
			}
			tc.assert(t, s)
		})
	}
}

func TestSectionModel_MoveFocusUp(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, s *SectionModel)
		name   string
		fields []FieldMeta
		moves  int
	}{
		{
			name:   "wraps from first to last",
			fields: testFields(),
			moves:  1,
			assert: func(t *testing.T, s *SectionModel) {
				t.Helper()
				assert.Equal(t, "Worktrees Dir", s.FocusedRow().Label)
			},
		},
		{
			name:   "skips group headers going up",
			fields: testGroupedFields(),
			moves:  1, // Host is first focused, up wraps to last field = Include Comments
			assert: func(t *testing.T, s *SectionModel) {
				t.Helper()
				assert.Equal(t, "Include Comments", s.FocusedRow().Label)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s := NewSectionModel(tc.fields)
			for range tc.moves {
				s.MoveFocusUp()
			}
			tc.assert(t, s)
		})
	}
}

func TestSectionModel_JumpToFirst(t *testing.T) {
	s := NewSectionModel(testGroupedFields())
	// Move to the last field.
	s.JumpToLast()
	require.Equal(t, "Include Comments", s.FocusedRow().Label)

	// Jump back to first.
	s.JumpToFirst()
	assert.Equal(t, "Host", s.FocusedRow().Label)
}

func TestSectionModel_JumpToLast(t *testing.T) {
	s := NewSectionModel(testGroupedFields())
	s.JumpToLast()
	assert.Equal(t, "Include Comments", s.FocusedRow().Label)
}

func TestSectionModel_JumpToNextGroup(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, s *SectionModel)
		setup  func(s *SectionModel)
		name   string
	}{
		{
			name:  "from Connection jumps to Filters",
			setup: func(_ *SectionModel) {},
			assert: func(t *testing.T, s *SectionModel) {
				t.Helper()
				assert.Equal(t, "Priority", s.FocusedRow().Label)
				assert.Equal(t, "Filters", s.FocusedRow().Group)
			},
		},
		{
			name: "from Filters jumps to Markdown",
			setup: func(s *SectionModel) {
				// Move to a Filters field first.
				s.JumpToNextGroup() // Connection -> Filters
			},
			assert: func(t *testing.T, s *SectionModel) {
				t.Helper()
				assert.Equal(t, "Max Depth", s.FocusedRow().Label)
				assert.Equal(t, "Markdown", s.FocusedRow().Group)
			},
		},
		{
			name: "from last group wraps to first",
			setup: func(s *SectionModel) {
				s.JumpToLast() // last field is in Markdown group
			},
			assert: func(t *testing.T, s *SectionModel) {
				t.Helper()
				assert.Equal(t, "Host", s.FocusedRow().Label)
				assert.Equal(t, "Connection", s.FocusedRow().Group)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s := NewSectionModel(testGroupedFields())
			tc.setup(s)
			s.JumpToNextGroup()
			tc.assert(t, s)
		})
	}
}

func TestSectionModel_JumpToPrevGroup(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, s *SectionModel)
		setup  func(s *SectionModel)
		name   string
	}{
		{
			name:  "from Connection wraps to Markdown (last group)",
			setup: func(_ *SectionModel) {},
			assert: func(t *testing.T, s *SectionModel) {
				t.Helper()
				assert.Equal(t, "Max Depth", s.FocusedRow().Label)
				assert.Equal(t, "Markdown", s.FocusedRow().Group)
			},
		},
		{
			name: "from Filters jumps to Connection",
			setup: func(s *SectionModel) {
				s.JumpToNextGroup() // Connection -> Filters
			},
			assert: func(t *testing.T, s *SectionModel) {
				t.Helper()
				assert.Equal(t, "Host", s.FocusedRow().Label)
				assert.Equal(t, "Connection", s.FocusedRow().Group)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s := NewSectionModel(testGroupedFields())
			tc.setup(s)
			s.JumpToPrevGroup()
			tc.assert(t, s)
		})
	}
}

func TestSectionModel_Scrolling(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, s *SectionModel)
		name   string
	}{
		{
			name: "scroll down when focus moves below viewport",
			assert: func(t *testing.T, s *SectionModel) {
				t.Helper()
				// Viewport is 3 rows. Grouped fields = 10 rows total.
				// Move down enough to go past viewport.
				for range 5 {
					s.MoveFocusDown()
				}
				// Focus should be visible (scroll offset adjusted).
				fi := s.FocusIndex()
				so := s.ScrollOffset()
				assert.GreaterOrEqual(t, fi, so, "focused row should be >= scroll offset")
				assert.Less(t, fi, so+3, "focused row should be within viewport")
			},
		},
		{
			name: "scroll up when focus wraps to bottom",
			assert: func(t *testing.T, s *SectionModel) {
				t.Helper()
				// Move up from first field -> wraps to last.
				s.MoveFocusUp()
				fi := s.FocusIndex()
				so := s.ScrollOffset()
				assert.GreaterOrEqual(t, fi, so)
				assert.Less(t, fi, so+3)
			},
		},
		{
			name: "jump to last scrolls to bottom",
			assert: func(t *testing.T, s *SectionModel) {
				t.Helper()
				s.JumpToLast()
				fi := s.FocusIndex()
				so := s.ScrollOffset()
				assert.GreaterOrEqual(t, fi, so)
				assert.Less(t, fi, so+3)
			},
		},
		{
			name: "jump to first scrolls to top",
			assert: func(t *testing.T, s *SectionModel) {
				t.Helper()
				s.JumpToLast()
				s.JumpToFirst()
				assert.Equal(t, 0, s.ScrollOffset())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s := NewSectionModel(testGroupedFields(), WithViewportHeight(3))
			tc.assert(t, s)
		})
	}
}

func TestSectionModel_SetViewportHeight(t *testing.T) {
	s := NewSectionModel(testGroupedFields(), WithViewportHeight(3))
	s.JumpToLast() // scrolls down

	oldOffset := s.ScrollOffset()
	s.SetViewportHeight(20) // now everything fits
	// Scroll offset should be clamped.
	assert.LessOrEqual(t, s.ScrollOffset(), oldOffset)
}

func TestSectionModel_SetFieldValue(t *testing.T) {
	s := NewSectionModel(testFields())
	s.SetFieldValue(0, "main")
	s.SetFieldValue(1, "worktrees")

	// Find the field rows and check values.
	for _, r := range s.Rows() {
		if r.FieldIndex == 0 {
			assert.Equal(t, "main", r.Value)
		}
		if r.FieldIndex == 1 {
			assert.Equal(t, "worktrees", r.Value)
		}
	}
}

func TestSectionModel_UpdateEntries(t *testing.T) {
	s := NewSectionModel(testFields(), WithEntryList("Rules", nil, "(empty)"))

	// Initially has empty placeholder.
	hasEmpty := false
	for _, r := range s.Rows() {
		if r.Kind == RowEmpty {
			hasEmpty = true
		}
	}
	require.True(t, hasEmpty, "should have empty placeholder initially")

	// Update with entries.
	s.UpdateEntries([]string{"rule1", "rule2"})

	hasEntry := false
	hasEmptyAfter := false
	for _, r := range s.Rows() {
		if r.Kind == RowEntry {
			hasEntry = true
		}
		if r.Kind == RowEmpty {
			hasEmptyAfter = true
		}
	}
	assert.True(t, hasEntry, "should have entry rows after update")
	assert.False(t, hasEmptyAfter, "should not have empty placeholder after adding entries")
}

func TestSectionModel_ViewRendersGroupHeaders(t *testing.T) {
	s := NewSectionModel(testGroupedFields(), WithViewportHeight(20), WithWidth(72))

	view := s.View()

	assert.Contains(t, view, "Connection")
	assert.Contains(t, view, "Filters")
	assert.Contains(t, view, "Markdown")
	assert.Contains(t, view, "--") // dash separators
}

func TestSectionModel_ViewRendersFieldRows(t *testing.T) {
	s := NewSectionModel(testFields(), WithViewportHeight(20), WithWidth(72))
	s.SetFieldValue(0, "main")
	s.SetFieldValue(1, "worktrees")

	view := s.View()

	assert.Contains(t, view, "Default Branch")
	assert.Contains(t, view, "main")
	assert.Contains(t, view, "Worktrees Dir")
	assert.Contains(t, view, "worktrees")
}

func TestSectionModel_ViewRendersPositionIndicator(t *testing.T) {
	s := NewSectionModel(testGroupedFields(), WithViewportHeight(20), WithWidth(72))

	view := s.View()

	// Should contain "1/7" since we have 7 focusable fields and focus is on first.
	assert.Contains(t, view, "1/7")
}

func TestSectionModel_ViewRendersEntryRows(t *testing.T) {
	s := NewSectionModel(testFields(),
		WithEntryList("Rules", []string{
			"main -> .env, config/",
			"develop -> .vscode/settings.json",
		}, "(no rules configured)"),
		WithViewportHeight(20), WithWidth(72),
	)

	view := s.View()

	assert.Contains(t, view, "Rules")
	assert.Contains(t, view, "1. main -> .env, config/")
	assert.Contains(t, view, "2. develop -> .vscode/settings.json")
}

func TestSectionModel_ViewRendersEmptyState(t *testing.T) {
	s := NewSectionModel(testFields(),
		WithEntryList("Rules", nil, "(no rules configured) -- press a to add"),
		WithViewportHeight(20), WithWidth(72),
	)

	view := s.View()

	assert.Contains(t, view, "Rules")
	assert.Contains(t, view, "(no rules configured) -- press a to add")
}

func TestSectionModel_ViewEmptyFields(t *testing.T) {
	s := NewSectionModel(nil, WithViewportHeight(20))
	view := s.View()
	assert.Empty(t, view)
}

func TestSectionModel_FieldRowShowsDashForEmptyValue(t *testing.T) {
	s := NewSectionModel(testFields(), WithViewportHeight(20), WithWidth(72))
	// Don't set any values -- default should show "--".

	view := s.View()
	assert.Contains(t, view, "--")
}

func TestSectionModel_FocusIndicator(t *testing.T) {
	s := NewSectionModel(testFields(), WithViewportHeight(20), WithWidth(72))

	view := s.View()

	// The focused row should have the ">" cursor.
	assert.Contains(t, view, ">")
}

func TestSectionModel_NavigationOnEmptySection(t *testing.T) {
	s := NewSectionModel(nil)

	// None of these should panic.
	s.MoveFocusDown()
	s.MoveFocusUp()
	s.JumpToFirst()
	s.JumpToLast()
	s.JumpToNextGroup()
	s.JumpToPrevGroup()

	assert.Equal(t, 0, s.FocusIndex())
	assert.Equal(t, 0, s.FieldCount())
	assert.Equal(t, 0, s.FocusPosition())
}

func TestSectionModel_GroupNavigationWithEntryList(t *testing.T) {
	s := NewSectionModel(
		testGroupedFields(),
		WithEntryList("Extra", []string{"entry1"}, ""),
	)

	// Jump through all groups: Connection -> Filters -> Markdown -> Extra -> Connection.
	s.JumpToNextGroup()
	assert.Equal(t, "Filters", s.FocusedRow().Group)

	s.JumpToNextGroup()
	assert.Equal(t, "Markdown", s.FocusedRow().Group)

	s.JumpToNextGroup()
	assert.Equal(t, "Extra", s.FocusedRow().Group)
	assert.Equal(t, RowEntry, s.FocusedRow().Kind)

	s.JumpToNextGroup()
	assert.Equal(t, "Connection", s.FocusedRow().Group)
}

func TestSectionModel_SetWidth(t *testing.T) {
	s := NewSectionModel(testFields())
	s.SetWidth(100)
	assert.Equal(t, 100, s.width)
}

func TestSectionModel_SetWidthZeroIgnored(t *testing.T) {
	s := NewSectionModel(testFields(), WithWidth(80))
	s.SetWidth(0)
	assert.Equal(t, 80, s.width, "zero width should be ignored")
}

func TestSectionModel_Accessors(t *testing.T) {
	s := NewSectionModel(testGroupedFields(), WithViewportHeight(5))

	assert.Equal(t, 5, s.viewportHeight)
	assert.NotNil(t, s.Rows())
	assert.Len(t, s.Rows(), 10)

	// FocusIndex should point to first focusable row.
	assert.Equal(t, 1, s.FocusIndex()) // row 0 is Connection header
}

func TestSectionModel_EntryListOnlySection(t *testing.T) {
	// A section with no fields, only an entry list (like Worktrees tab).
	s := NewSectionModel(nil,
		WithEntryList("", []string{
			"main -> branch: main, merge: --",
			"feature/auth -> branch: feature/auth, merge: main",
		}, "(no worktrees configured)"),
		WithViewportHeight(20), WithWidth(72),
	)

	// Should have entries (no group header when label is empty).
	assert.Equal(t, 2, s.FieldCount())
	assert.Equal(t, RowEntry, s.FocusedRow().Kind)
}

func TestSectionModel_UngroupedFieldsNoHeaders(t *testing.T) {
	s := NewSectionModel(testFields())

	// No group headers should be inserted for ungrouped fields.
	for _, r := range s.Rows() {
		assert.NotEqual(t, RowGroupHeader, r.Kind,
			"ungrouped fields should not generate group headers")
	}
}

func TestSectionModel_FieldIndex(t *testing.T) {
	s := NewSectionModel(testGroupedFields())

	// Verify FieldIndex maps correctly.
	fieldCount := 0
	for _, r := range s.Rows() {
		if r.Kind == RowField {
			assert.Equal(t, fieldCount, r.FieldIndex)
			fieldCount++
		}
	}
	assert.Equal(t, 7, fieldCount)
}

func TestSectionModel_ScrollOffsetClamps(t *testing.T) {
	s := NewSectionModel(testFields(), WithViewportHeight(10))

	// With only 2 rows and viewport of 10, scroll offset should be 0.
	s.JumpToLast()
	assert.Equal(t, 0, s.ScrollOffset(), "should not scroll when all rows fit in viewport")
}

func TestSectionModel_SetFocusByFieldIndex(t *testing.T) {
	testCases := []struct {
		setup       func(s *SectionModel)
		assert      func(t *testing.T, s *SectionModel)
		assertFound func(t *testing.T, found bool)
		name        string
		fields      []FieldMeta
		fieldIndex  int
	}{
		{
			name:       "focuses matching field in ungrouped section",
			fields:     testFields(),
			fieldIndex: 1,
			setup:      func(_ *SectionModel) {},
			assertFound: func(t *testing.T, found bool) {
				t.Helper()
				assert.True(t, found)
			},
			assert: func(t *testing.T, s *SectionModel) {
				t.Helper()
				assert.Equal(t, "Worktrees Dir", s.FocusedRow().Label)
				assert.Equal(t, 1, s.FocusedRow().FieldIndex)
			},
		},
		{
			name:       "focuses matching field in grouped section skipping headers",
			fields:     testGroupedFields(),
			fieldIndex: 3,
			setup:      func(_ *SectionModel) {},
			assertFound: func(t *testing.T, found bool) {
				t.Helper()
				assert.True(t, found)
			},
			assert: func(t *testing.T, s *SectionModel) {
				t.Helper()
				assert.Equal(t, "Type", s.FocusedRow().Label)
				assert.Equal(t, 3, s.FocusedRow().FieldIndex)
			},
		},
		{
			name:       "scrolls viewport when target is below visible area",
			fields:     testGroupedFields(),
			fieldIndex: 6,
			setup:      func(_ *SectionModel) {},
			assertFound: func(t *testing.T, found bool) {
				t.Helper()
				assert.True(t, found)
			},
			assert: func(t *testing.T, s *SectionModel) {
				t.Helper()
				fi := s.FocusIndex()
				so := s.ScrollOffset()
				assert.GreaterOrEqual(t, fi, so, "focused row should be at or after scroll offset")
				assert.Less(t, fi, so+3, "focused row should be within viewport")
				assert.Equal(t, "Include Comments", s.FocusedRow().Label)
			},
		},
		{
			name:       "returns false for nonexistent field index",
			fields:     testGroupedFields(),
			fieldIndex: 99,
			setup:      func(_ *SectionModel) {},
			assertFound: func(t *testing.T, found bool) {
				t.Helper()
				assert.False(t, found)
			},
			assert: func(t *testing.T, s *SectionModel) {
				t.Helper()
				// Focus should remain unchanged (first focusable row).
				assert.Equal(t, "Host", s.FocusedRow().Label)
			},
		},
		{
			name:       "returns false when field is filtered out by search",
			fields:     testGroupedFields(),
			fieldIndex: 5, // "Max Depth" in Markdown group
			setup: func(s *SectionModel) {
				s.OpenSearch()
				s.SearchHandleRune('H')
				s.SearchHandleRune('o')
				s.SearchHandleRune('s')
				s.SearchHandleRune('t')
				// Only "Host" should match the filter.
			},
			assertFound: func(t *testing.T, found bool) {
				t.Helper()
				assert.False(t, found, "field not in filtered results should return false")
			},
			assert: func(t *testing.T, s *SectionModel) {
				t.Helper()
				// Focus should remain on the search result.
				assert.Equal(t, "Host", s.FocusedRow().Label)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s := NewSectionModel(tc.fields, WithViewportHeight(3))
			tc.setup(s)
			found := s.SetFocusByFieldIndex(tc.fieldIndex)
			tc.assertFound(t, found)
			tc.assert(t, s)
		})
	}
}

func TestSectionModel_OverlayRight(t *testing.T) {
	testCases := []struct {
		assert  func(t *testing.T, result string)
		name    string
		line    string
		overlay string
		width   int
	}{
		{
			name:    "plain text truncated to make room for overlay",
			line:    strings.Repeat("x", 40),
			overlay: "1/7",
			width:   40,
			assert: func(t *testing.T, result string) {
				t.Helper()
				assert.Contains(t, result, "1/7")
				assert.Equal(t, 40, lipgloss.Width(result))
			},
		},
		{
			name:    "ANSI styled line preserves escape sequences",
			line:    "\x1b[31m" + strings.Repeat("R", 30) + "\x1b[0m",
			overlay: "3/5",
			width:   40,
			assert: func(t *testing.T, result string) {
				t.Helper()
				assert.Contains(t, result, "3/5")
				// The result must not contain broken escape sequences.
				// A broken sequence would have \x1b followed by a truncation
				// that cuts off the closing letter (e.g., \x1b[3 without the 'm').
				assert.NotContains(t, result, "\x1b[3/", "should not slice through ANSI escape")
				assert.Equal(t, 40, lipgloss.Width(result))
			},
		},
		{
			name:    "short line is padded before overlay",
			line:    "short",
			overlay: "2/4",
			width:   40,
			assert: func(t *testing.T, result string) {
				t.Helper()
				assert.Contains(t, result, "2/4")
				assert.Equal(t, 40, lipgloss.Width(result))
			},
		},
		{
			name:    "overlay wider than width returns overlay only",
			line:    "some content",
			overlay: strings.Repeat("O", 50),
			width:   40,
			assert: func(t *testing.T, result string) {
				t.Helper()
				assert.Equal(t, strings.Repeat("O", 50), result)
			},
		},
		{
			name:    "ANSI with multiple style codes intact after truncation",
			line:    "\x1b[1m\x1b[34mBold Blue Text Here\x1b[0m" + strings.Repeat(" ", 20),
			overlay: "5/9",
			width:   40,
			assert: func(t *testing.T, result string) {
				t.Helper()
				assert.Contains(t, result, "5/9")
				assert.Equal(t, 40, lipgloss.Width(result))
				// Verify no partial escape sequences: every \x1b must be
				// followed by [ and eventually a letter.
				for i := range len(result) {
					if result[i] == '\x1b' {
						// Must have room for at least \x1b[Xm
						require.Greater(t, len(result)-i, 2,
							"escape sequence truncated at end of string")
						assert.Equal(t, byte('['), result[i+1],
							"escape sequence must have [ after ESC")
					}
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s := NewSectionModel(testFields(), WithWidth(tc.width))
			result := s.overlayRight(tc.line, tc.overlay)
			tc.assert(t, result)
		})
	}
}

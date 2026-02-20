package config

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testConfigAccessor implements ConfigAccessor for integration tests.
// It stores values in a map and supports get/set by dot-path key.
type testConfigAccessor struct {
	values map[string]any
}

func (a *testConfigAccessor) GetValue(key string) any {
	return a.values[key]
}

func (a *testConfigAccessor) SetValue(key string, value any) error {
	a.values[key] = value
	return nil
}

// fileCopyRule mirrors service.FileCopyRule for integration tests.
// It has a slice field (Files), making it non-comparable with ==.
type fileCopyRule struct {
	SourceWorktree string
	Files          []string
}

// worktreeConfig mirrors service.WorktreeConfig for integration tests.
type worktreeConfig struct {
	Branch      string
	MergeInto   string
	Description string
}

// seedValues returns a realistic set of values for all config keys used
// by the section field definitions. This provides a complete dataset for
// integration tests, including non-comparable types (struct slices and maps)
// that exercise the dirty tracker's reflect.DeepEqual fallback.
func seedValues() map[string]any {
	return map[string]any{
		// General
		"default_branch": "main",
		"worktrees_dir":  "worktrees",

		// JIRA > Connection
		"jira.host": "https://jira.example.com",
		"jira.me":   "testuser",

		// JIRA > Filters
		"jira.filters.priority":    "High",
		"jira.filters.type":        "Story",
		"jira.filters.component":   "backend",
		"jira.filters.reporter":    "",
		"jira.filters.assignee":    "testuser",
		"jira.filters.order_by":    "priority",
		"jira.filters.status":      []string{"Open", "In Progress"},
		"jira.filters.labels":      []string{"team-alpha"},
		"jira.filters.custom_args": []string{},
		"jira.filters.reverse":     false,

		// JIRA > Markdown
		"jira.markdown.filename_pattern":      "{key}.md",
		"jira.markdown.max_depth":             3,
		"jira.markdown.include_comments":      true,
		"jira.markdown.include_attachments":   false,
		"jira.markdown.use_relative_links":    true,
		"jira.markdown.include_linked_issues": false,

		// JIRA > Attachments
		"jira.attachments.enabled":                  true,
		"jira.attachments.max_size_mb":              10,
		"jira.attachments.directory":                ".attachments",
		"jira.attachments.download_timeout_seconds": 30,
		"jira.attachments.retry_attempts":           3,
		"jira.attachments.retry_backoff_ms":         1000,

		// File Copy > Rules (non-comparable: []fileCopyRule contains []string)
		"file_copy.rules": []fileCopyRule{
			{SourceWorktree: "main", Files: []string{".env", ".env.local"}},
			{SourceWorktree: "dev", Files: []string{"Makefile"}},
		},

		// File Copy > Auto
		"file_copy.auto.enabled":         true,
		"file_copy.auto.source_worktree": "{default}",
		"file_copy.auto.copy_ignored":    false,
		"file_copy.auto.copy_untracked":  false,
		"file_copy.auto.exclude":         []string{"*.log", "node_modules/"},

		// Worktrees (non-comparable: map[string]worktreeConfig)
		"worktrees": map[string]worktreeConfig{
			"feature-x": {Branch: "feature/x", MergeInto: "main", Description: "Feature X"},
			"hotfix-1":  {Branch: "hotfix/1", MergeInto: "main", Description: "Hotfix 1"},
		},
	}
}

// newIntegrationModel creates a fully wired ConfigModel for integration tests.
// It sets up a testConfigAccessor, DirtyTracker, window size, and initializes
// sections from the accessor values.
func newIntegrationModel(values map[string]any) (*ConfigModel, *testConfigAccessor) {
	accessor := &testConfigAccessor{values: values}
	dt := NewDirtyTracker(values)
	m := NewConfigModel(
		WithAccessor(accessor),
		WithDirtyTracker(dt),
		WithFilePath("/tmp/test-config.yaml"),
		WithNewFile(true),
	)
	// Send WindowSizeMsg to set dimensions before InitSections.
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m.InitSections()
	return m, accessor
}

// sendKeys sends a sequence of key messages to the model, returning the final
// model pointer.
func sendKeys(m *ConfigModel, keys ...tea.KeyMsg) *ConfigModel {
	var result tea.Model = m
	for _, k := range keys {
		result, _ = result.Update(k)
	}
	return result.(*ConfigModel)
}

// downKey returns a tea.KeyMsg for the down arrow key.
func downKey() tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyDown} }

// upKey returns a tea.KeyMsg for the up arrow key.
func upKey() tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyUp} }

// --- Integration tests ---.

func TestIntegration_ConfigLoadsAndDisplaysValues(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, m *ConfigModel)
		name   string
	}{
		{
			name: "General tab shows field labels and values",
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				view := m.View()
				assert.Contains(t, view, "Default Branch")
				assert.Contains(t, view, "main")
				assert.Contains(t, view, "Worktrees Dir")
				assert.Contains(t, view, "worktrees")
			},
		},
		{
			name: "model starts on General tab in browsing state",
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.Equal(t, TabGeneral, m.ActiveTab())
				assert.Equal(t, StateBrowsing, m.State())
			},
		},
		{
			name: "sections are initialized with values from accessor",
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				sections := m.Sections()
				for i, s := range sections {
					require.NotNil(t, s, "section %d should be initialized", i)
				}
				// General section should have 2 focusable fields.
				assert.Equal(t, 2, sections[TabGeneral].FieldCount())
			},
		},
		{
			name: "JIRA tab shows field values when switched to",
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				m = sendKeys(m, tabKey()) // switch to JIRA
				assert.Equal(t, TabJira, m.ActiveTab())
				view := m.View()
				assert.Contains(t, view, "Host")
				assert.Contains(t, view, "Connection")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m, _ := newIntegrationModel(seedValues())
			tc.assert(t, m)
		})
	}
}

func TestIntegration_NavigationAcrossTabs(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, m *ConfigModel)
		name   string
		keys   []tea.KeyMsg
	}{
		{
			name: "tab advances to JIRA",
			keys: []tea.KeyMsg{tabKey()},
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.Equal(t, TabJira, m.ActiveTab())
			},
		},
		{
			name: "two tabs advances to File Copy",
			keys: []tea.KeyMsg{tabKey(), tabKey()},
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.Equal(t, TabFileCopy, m.ActiveTab())
			},
		},
		{
			name: "three tabs advances to Worktrees",
			keys: []tea.KeyMsg{tabKey(), tabKey(), tabKey()},
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.Equal(t, TabWorktrees, m.ActiveTab())
			},
		},
		{
			name: "four tabs wraps to General",
			keys: []tea.KeyMsg{tabKey(), tabKey(), tabKey(), tabKey()},
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.Equal(t, TabGeneral, m.ActiveTab())
			},
		},
		{
			name: "down moves focus to next field",
			keys: []tea.KeyMsg{downKey()},
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				row := m.activeSection().FocusedRow()
				assert.Equal(t, "Worktrees Directory", row.Label)
			},
		},
		{
			name: "group jump on JIRA tab moves between groups",
			keys: []tea.KeyMsg{tabKey(), runeKey('}')},
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				s := m.activeSection()
				row := s.FocusedRow()
				// Should have jumped from Connection to Filters group.
				assert.Equal(t, "Filters", row.Group)
			},
		},
		{
			name: "g jumps to first field",
			keys: []tea.KeyMsg{tabKey(), downKey(), downKey(), downKey(), runeKey('g')},
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				s := m.activeSection()
				row := s.FocusedRow()
				assert.Equal(t, "Host", row.Label)
			},
		},
		{
			name: "G jumps to last field",
			keys: []tea.KeyMsg{tabKey(), runeKey('G')},
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				s := m.activeSection()
				row := s.FocusedRow()
				// Last JIRA field is in Attachments group.
				assert.Equal(t, "Attachments", row.Group)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m, _ := newIntegrationModel(seedValues())
			m = sendKeys(m, tc.keys...)
			tc.assert(t, m)
		})
	}
}

func TestIntegration_FieldEditLifecycle(t *testing.T) {
	testCases := []struct {
		assert      func(t *testing.T, m *ConfigModel, accessor *testConfigAccessor)
		name        string
		editValue   string
		expectDirty int
	}{
		{
			name:        "edit string field changes value and marks dirty",
			editValue:   "develop",
			expectDirty: 1,
			assert: func(t *testing.T, m *ConfigModel, accessor *testConfigAccessor) {
				t.Helper()
				assert.Equal(t, StateBrowsing, m.State())
				assert.Equal(t, "develop", accessor.values["default_branch"])
				assert.True(t, m.dirty.IsKeyDirty("default_branch"))
			},
		},
		{
			name:        "edit to same value does not mark dirty",
			editValue:   "main",
			expectDirty: 0,
			assert: func(t *testing.T, m *ConfigModel, accessor *testConfigAccessor) {
				t.Helper()
				assert.Equal(t, "main", accessor.values["default_branch"])
				assert.False(t, m.dirty.IsKeyDirty("default_branch"))
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m, accessor := newIntegrationModel(seedValues())

			// Press 'e' to enter editing on "Default Branch".
			m = sendKeys(m, runeKey('e'))
			require.Equal(t, StateEditing, m.State())

			// Set the new value in the text input.
			fr := m.activeFieldRow()
			require.NotNil(t, fr)
			fr.input.SetValue(tc.editValue)

			// Press enter to confirm.
			m = sendKeys(m, enterKey())

			assert.Equal(t, tc.expectDirty, m.dirty.DirtyCount())
			tc.assert(t, m, accessor)
		})
	}
}

func TestIntegration_BoolToggle(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, m *ConfigModel, accessor *testConfigAccessor)
		name   string
	}{
		{
			name: "toggle bool field from false to true",
			assert: func(t *testing.T, m *ConfigModel, accessor *testConfigAccessor) {
				t.Helper()
				// jira.filters.reverse starts as false; should be true after toggle.
				assert.Equal(t, true, accessor.values["jira.filters.reverse"])
				assert.Equal(t, StateBrowsing, m.State())
				assert.True(t, m.dirty.IsKeyDirty("jira.filters.reverse"))
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m, accessor := newIntegrationModel(seedValues())

			// Switch to JIRA tab.
			m = sendKeys(m, tabKey())
			require.Equal(t, TabJira, m.ActiveTab())

			// Navigate to the "Reverse" bool field.
			s := m.activeSection()
			rows := s.Rows()
			var targetIdx int
			for i, r := range rows {
				if r.Kind == RowField && r.FieldIndex >= 0 {
					if jiraFields[r.FieldIndex].Key == "jira.filters.reverse" {
						targetIdx = i
						break
					}
				}
			}
			// Navigate to the target row.
			for s.FocusIndex() != targetIdx {
				s.MoveFocusDown()
			}
			m.syncFocusedField()

			// Press 'e' to toggle.
			m = sendKeys(m, runeKey('e'))

			tc.assert(t, m, accessor)
		})
	}
}

func TestIntegration_StringListOpensListOverlay(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, m *ConfigModel, accessor *testConfigAccessor)
		action func(t *testing.T, m *ConfigModel)
		name   string
	}{
		{
			name: "esc closes overlay without changes",
			action: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				// Press esc to close (no changes made, so it closes directly).
				sendKeys(m, escKey())
			},
			assert: func(t *testing.T, m *ConfigModel, accessor *testConfigAccessor) {
				t.Helper()
				assert.Equal(t, StateBrowsing, m.State())
				// Original value preserved.
				val := accessor.values["jira.filters.status"]
				sl, ok := val.([]string)
				require.True(t, ok)
				assert.Equal(t, []string{"Open", "In Progress"}, sl)
			},
		},
		{
			name: "add item and confirm writes back",
			action: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				// Press 'a' to add a new item.
				result, _ := m.Update(runeKey('a'))
				m = result.(*ConfigModel)
				require.NotNil(t, m.listOverlay)
				assert.Equal(t, listAdding, m.listOverlay.State())

				// Type the new item value via the overlay's text input.
				m.listOverlay.input.SetValue("Done")

				// Press enter to confirm the new item.
				result, _ = m.Update(enterKey())
				m = result.(*ConfigModel)

				// Press enter to confirm the list overlay.
				result, _ = m.Update(enterKey())
				_ = result.(*ConfigModel)
			},
			assert: func(t *testing.T, m *ConfigModel, accessor *testConfigAccessor) {
				t.Helper()
				assert.Equal(t, StateBrowsing, m.State())
				val := accessor.values["jira.filters.status"]
				sl, ok := val.([]string)
				require.True(t, ok)
				assert.Contains(t, sl, "Done")
				assert.Contains(t, sl, "Open")
				assert.Contains(t, sl, "In Progress")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m, accessor := newIntegrationModel(seedValues())

			// Switch to JIRA tab.
			m = sendKeys(m, tabKey())

			// Navigate to "Status" (StringList field in Filters group).
			s := m.activeSection()
			rows := s.Rows()
			var targetIdx int
			for i, r := range rows {
				if r.Kind == RowField && r.FieldIndex >= 0 {
					if jiraFields[r.FieldIndex].Key == "jira.filters.status" {
						targetIdx = i
						break
					}
				}
			}
			for s.FocusIndex() != targetIdx {
				s.MoveFocusDown()
			}
			m.syncFocusedField()

			// Press 'e' to open the list overlay.
			result, _ := m.Update(runeKey('e'))
			m = result.(*ConfigModel)
			require.Equal(t, StateOverlay, m.State())
			require.NotNil(t, m.listOverlay)

			tc.action(t, m)
			tc.assert(t, m, accessor)
		})
	}
}

func TestIntegration_SearchFiltersFields(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, m *ConfigModel)
		name   string
	}{
		{
			name: "search activates and filters visible fields",
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				// Open search.
				m = sendKeys(m, runeKey('/'))
				require.Equal(t, StateSearch, m.State())

				s := m.activeSection()
				require.NotNil(t, s)
				assert.True(t, s.IsSearchActive())

				// Type "host" to filter.
				sendKeys(m, runeKey('h'), runeKey('o'), runeKey('s'), runeKey('t'))
				assert.Equal(t, "host", s.Search().Query())

				// Check that filtered rows contain only matching fields.
				visibleRows := s.Rows()
				fieldCount := 0
				for _, r := range visibleRows {
					if r.Kind == RowField {
						fieldCount++
						assert.Contains(t, r.Label, "Host",
							"filtered field should contain search term")
					}
				}
				assert.Positive(t, fieldCount,
					"should have at least one matching field")
			},
		},
		{
			name: "esc closes search and restores all fields",
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				// Open search and type a filter.
				m = sendKeys(m, runeKey('/'), runeKey('h'), runeKey('o'))

				s := m.activeSection()
				filteredCount := s.FieldCount()

				// Close search.
				m = sendKeys(m, escKey())
				assert.Equal(t, StateBrowsing, m.State())
				assert.False(t, s.IsSearchActive())

				// All fields should be visible again.
				allCount := s.FieldCount()
				assert.Greater(t, allCount, filteredCount,
					"unfiltered field count should be greater than filtered")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m, _ := newIntegrationModel(seedValues())
			// Switch to JIRA tab for more fields to search through.
			m = sendKeys(m, tabKey())
			require.Equal(t, TabJira, m.ActiveTab())
			tc.assert(t, m)
		})
	}
}

func TestIntegration_DirtyTrackingAndSaveFlow(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, m *ConfigModel)
		name   string
	}{
		{
			name: "edit one field yields dirty count 1",
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				// Edit Default Branch.
				m = sendKeys(m, runeKey('e'))
				fr := m.activeFieldRow()
				require.NotNil(t, fr)
				fr.input.SetValue("develop")
				m = sendKeys(m, enterKey())

				assert.Equal(t, 1, m.dirty.DirtyCount())
			},
		},
		{
			name: "edit two fields yields dirty count 2",
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				// Edit first field.
				m = sendKeys(m, runeKey('e'))
				fr1 := m.activeFieldRow()
				require.NotNil(t, fr1)
				fr1.input.SetValue("develop")
				m = sendKeys(m, enterKey())

				// Navigate down, edit second field.
				m = sendKeys(m, downKey(), runeKey('e'))
				fr2 := m.activeFieldRow()
				require.NotNil(t, fr2)
				fr2.input.SetValue("wt")
				m = sendKeys(m, enterKey())

				assert.Equal(t, 2, m.dirty.DirtyCount())
			},
		},
		{
			name: "reset one field reduces dirty count",
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				// Edit first field.
				m = sendKeys(m, runeKey('e'))
				fr1 := m.activeFieldRow()
				require.NotNil(t, fr1)
				fr1.input.SetValue("develop")
				m = sendKeys(m, enterKey())

				// Edit second field.
				m = sendKeys(m, downKey(), runeKey('e'))
				fr2 := m.activeFieldRow()
				require.NotNil(t, fr2)
				fr2.input.SetValue("wt")
				m = sendKeys(m, enterKey())
				require.Equal(t, 2, m.dirty.DirtyCount())

				// Navigate back to first field.
				m = sendKeys(m, upKey())
				m.syncFocusedField()

				// Reset (r -> y).
				m = sendKeys(m, runeKey('r'))
				require.Equal(t, StateResetConfirm, m.State())
				m = sendKeys(m, runeKey('y'))

				assert.Equal(t, 1, m.dirty.DirtyCount())
				assert.Equal(t, "main", fr1.Value())
			},
		},
		{
			name: "save clears dirty count via MarkClean",
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				// Edit a field.
				m = sendKeys(m, runeKey('e'))
				fr := m.activeFieldRow()
				require.NotNil(t, fr)
				fr.input.SetValue("develop")
				m = sendKeys(m, enterKey())
				require.Equal(t, 1, m.dirty.DirtyCount())

				// Directly invoke MarkClean to simulate a successful save.
				// (Full save flow requires file I/O, tested separately.)
				m.dirty.MarkClean()

				assert.Equal(t, 0, m.dirty.DirtyCount())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m, _ := newIntegrationModel(seedValues())
			tc.assert(t, m)
		})
	}
}

func TestIntegration_QuitGuardWithDirtyChanges(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, m *ConfigModel, cmd tea.Cmd)
		action func(t *testing.T, m *ConfigModel) (*ConfigModel, tea.Cmd)
		name   string
	}{
		{
			name: "q with no dirty changes quits immediately",
			action: func(t *testing.T, m *ConfigModel) (*ConfigModel, tea.Cmd) {
				t.Helper()
				result, cmd := m.Update(runeKey('q'))
				return result.(*ConfigModel), cmd
			},
			assert: func(t *testing.T, m *ConfigModel, cmd tea.Cmd) {
				t.Helper()
				require.NotNil(t, cmd)
				msg := cmd()
				_, isQuit := msg.(tea.QuitMsg)
				assert.True(t, isQuit)
			},
		},
		{
			name: "q with dirty changes shows quit confirmation",
			action: func(t *testing.T, m *ConfigModel) (*ConfigModel, tea.Cmd) {
				t.Helper()
				// Make a dirty change.
				m = sendKeys(m, runeKey('e'))
				fr := m.activeFieldRow()
				require.NotNil(t, fr)
				fr.input.SetValue("develop")
				m = sendKeys(m, enterKey())
				require.Equal(t, 1, m.dirty.DirtyCount())

				// Press q.
				result, cmd := m.Update(runeKey('q'))
				return result.(*ConfigModel), cmd
			},
			assert: func(t *testing.T, m *ConfigModel, cmd tea.Cmd) {
				t.Helper()
				assert.Equal(t, StateQuitConfirm, m.State())
				assert.Nil(t, cmd)
			},
		},
		{
			name: "esc on quit confirmation returns to browsing",
			action: func(t *testing.T, m *ConfigModel) (*ConfigModel, tea.Cmd) {
				t.Helper()
				// Make dirty, press q, then esc.
				m = sendKeys(m, runeKey('e'))
				fr := m.activeFieldRow()
				require.NotNil(t, fr)
				fr.input.SetValue("develop")
				m = sendKeys(m, enterKey(), runeKey('q'))
				require.Equal(t, StateQuitConfirm, m.State())

				result, cmd := m.Update(escKey())
				return result.(*ConfigModel), cmd
			},
			assert: func(t *testing.T, m *ConfigModel, cmd tea.Cmd) {
				t.Helper()
				assert.Equal(t, StateBrowsing, m.State())
				assert.Nil(t, cmd)
			},
		},
		{
			name: "d on quit confirmation discards and quits",
			action: func(t *testing.T, m *ConfigModel) (*ConfigModel, tea.Cmd) {
				t.Helper()
				// Make dirty, press q, then d to discard.
				m = sendKeys(m, runeKey('e'))
				fr := m.activeFieldRow()
				require.NotNil(t, fr)
				fr.input.SetValue("develop")
				m = sendKeys(m, enterKey(), runeKey('q'))
				require.Equal(t, StateQuitConfirm, m.State())

				result, cmd := m.Update(runeKey('d'))
				return result.(*ConfigModel), cmd
			},
			assert: func(t *testing.T, m *ConfigModel, cmd tea.Cmd) {
				t.Helper()
				require.NotNil(t, cmd)
				msg := cmd()
				_, isQuit := msg.(tea.QuitMsg)
				assert.True(t, isQuit, "d should trigger quit")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m, _ := newIntegrationModel(seedValues())
			m, cmd := tc.action(t, m)
			tc.assert(t, m, cmd)
		})
	}
}

func TestIntegration_ViewRenderingEndToEnd(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, m *ConfigModel)
		name   string
	}{
		{
			name: "view contains tab bar with all four tabs",
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				view := m.View()
				assert.Contains(t, view, "General")
				assert.Contains(t, view, "JIRA")
				assert.Contains(t, view, "File Copy")
				assert.Contains(t, view, "Worktrees")
			},
		},
		{
			name: "JIRA tab view shows group headers",
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				m = sendKeys(m, tabKey())
				view := m.View()
				assert.Contains(t, view, "Connection")
				assert.Contains(t, view, "Filters")
			},
		},
		{
			name: "File Copy tab view shows auto-copy fields and rules label",
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				m = sendKeys(m, tabKey(), tabKey())
				view := m.View()
				assert.Contains(t, view, "Auto Copy")
				assert.Contains(t, view, "Enabled")
				assert.Contains(t, view, "Rules")
			},
		},
		{
			name: "Worktrees tab view shows entry list header",
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				m = sendKeys(m, tabKey(), tabKey(), tabKey())
				view := m.View()
				assert.Contains(t, view, "Worktrees")
			},
		},
		{
			name: "editing state shows text input in view",
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				m = sendKeys(m, runeKey('e'))
				require.Equal(t, StateEditing, m.State())
				view := m.View()
				// The view should still contain the tab bar.
				assert.Contains(t, view, "General")
			},
		},
		{
			name: "dirty count appears in status bar after edit",
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				// Clear isNewFile so the dirty count is shown instead of [new file].
				m.isNewFile = false

				m = sendKeys(m, runeKey('e'))
				fr := m.activeFieldRow()
				require.NotNil(t, fr)
				fr.input.SetValue("develop")
				m = sendKeys(m, enterKey())

				view := m.View()
				assert.Contains(t, view, "[1 modified]")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m, _ := newIntegrationModel(seedValues())
			tc.assert(t, m)
		})
	}
}

func TestIntegration_HelpOverlay(t *testing.T) {
	m, _ := newIntegrationModel(seedValues())

	// Press '?' to open help.
	m = sendKeys(m, runeKey('?'))
	assert.Equal(t, StateHelp, m.State())

	view := m.View()
	assert.Contains(t, view, "Keybinding Reference")

	// Press esc to close.
	m = sendKeys(m, escKey())
	assert.Equal(t, StateBrowsing, m.State())
}

func TestIntegration_ResetAllFlow(t *testing.T) {
	m, accessor := newIntegrationModel(seedValues())

	// Edit two fields.
	m = sendKeys(m, runeKey('e'))
	fr1 := m.activeFieldRow()
	require.NotNil(t, fr1)
	fr1.input.SetValue("develop")
	m = sendKeys(m, enterKey())

	m = sendKeys(m, downKey(), runeKey('e'))
	fr2 := m.activeFieldRow()
	require.NotNil(t, fr2)
	fr2.input.SetValue("wt")
	m = sendKeys(m, enterKey())

	require.Equal(t, 2, m.dirty.DirtyCount())

	// Press R -> should enter reset-all confirm.
	m = sendKeys(m, runeKey('R'))
	require.Equal(t, StateResetAllConfirm, m.State())

	// Confirm with y.
	m = sendKeys(m, runeKey('y'))

	assert.Equal(t, StateBrowsing, m.State())
	assert.Equal(t, 0, m.dirty.DirtyCount())
	assert.Equal(t, "main", accessor.values["default_branch"])
	assert.Equal(t, "worktrees", accessor.values["worktrees_dir"])
}

func TestIntegration_EditCancelPreservesValue(t *testing.T) {
	m, accessor := newIntegrationModel(seedValues())

	// Enter edit mode.
	m = sendKeys(m, runeKey('e'))
	require.Equal(t, StateEditing, m.State())

	fr := m.activeFieldRow()
	require.NotNil(t, fr)

	// Type something different.
	fr.input.SetValue("changed")

	// Cancel with esc.
	m = sendKeys(m, escKey())
	assert.Equal(t, StateBrowsing, m.State())

	// Value should be unchanged.
	assert.Equal(t, "main", accessor.values["default_branch"])
	assert.Equal(t, 0, m.dirty.DirtyCount())
}

func TestIntegration_FullEditRoundTrip(t *testing.T) {
	// This test exercises the complete data flow:
	// accessor -> ConfigModel -> sections -> field rows -> editing -> commit
	// -> accessor writeback -> dirty tracking
	m, accessor := newIntegrationModel(seedValues())

	// Verify initial state.
	assert.Equal(t, "main", accessor.values["default_branch"])
	assert.Equal(t, 0, m.dirty.DirtyCount())

	// Edit default_branch.
	m = sendKeys(m, runeKey('e'))
	fr := m.activeFieldRow()
	require.NotNil(t, fr)
	assert.Equal(t, "default_branch", fr.Meta().Key)
	fr.input.SetValue("release/v2")
	m = sendKeys(m, enterKey())

	// Verify writeback.
	assert.Equal(t, "release/v2", accessor.values["default_branch"])
	assert.Equal(t, 1, m.dirty.DirtyCount())
	assert.True(t, m.dirty.IsKeyDirty("default_branch"))

	// Verify the field row has the new value.
	assert.Equal(t, "release/v2", fr.Value())
	assert.True(t, fr.IsDirty())

	// Verify the section display was updated.
	s := m.activeSection()
	focusedRow := s.FocusedRow()
	assert.Equal(t, "release/v2", focusedRow.Value)

	// Reset the field.
	m.syncFocusedField()
	m = sendKeys(m, runeKey('r'))
	require.Equal(t, StateResetConfirm, m.State())
	m = sendKeys(m, runeKey('y'))

	// Verify reset.
	assert.Equal(t, "main", accessor.values["default_branch"])
	assert.Equal(t, 0, m.dirty.DirtyCount())
	assert.Equal(t, "main", fr.Value())
	assert.False(t, fr.IsDirty())
}

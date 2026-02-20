package config

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// keyMsg helpers for creating tea.KeyMsg values in tests.
func tabKey() tea.KeyMsg      { return tea.KeyMsg{Type: tea.KeyTab} }
func shiftTabKey() tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyShiftTab} }
func escKey() tea.KeyMsg      { return tea.KeyMsg{Type: tea.KeyEsc} }
func enterKey() tea.KeyMsg    { return tea.KeyMsg{Type: tea.KeyEnter} }
func ctrlCKey() tea.KeyMsg    { return tea.KeyMsg{Type: tea.KeyCtrlC} }

func runeKey(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

func TestNewConfigModel(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, m *ConfigModel)
		name   string
		opts   []ConfigModelOption
	}{
		{
			name: "default values",
			opts: nil,
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.Equal(t, TabGeneral, m.activeTab)
				assert.Equal(t, StateBrowsing, m.state)
				assert.NotNil(t, m.theme)
				assert.NotNil(t, m.dirty)
				assert.False(t, m.isNewFile)
				assert.Empty(t, m.filePath)
				assert.Empty(t, m.flashMessage)
			},
		},
		{
			name: "with file path",
			opts: []ConfigModelOption{WithFilePath("/tmp/config.yaml")},
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.Equal(t, "/tmp/config.yaml", m.filePath)
			},
		},
		{
			name: "with new file flag",
			opts: []ConfigModelOption{WithNewFile(true)},
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.True(t, m.isNewFile)
			},
		},
		{
			name: "with dirty tracker",
			opts: []ConfigModelOption{WithDirtyTracker(NewDirtyTracker(map[string]any{"key": "val"}))},
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.NotNil(t, m.dirty)
				assert.Equal(t, "val", m.dirty.GetOriginal("key"))
			},
		},
		{
			name: "nil theme option does not override default",
			opts: []ConfigModelOption{WithTheme(nil)},
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.NotNil(t, m.theme)
			},
		},
		{
			name: "nil dirty tracker option does not override default",
			opts: []ConfigModelOption{WithDirtyTracker(nil)},
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.NotNil(t, m.dirty)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := NewConfigModel(tc.opts...)
			tc.assert(t, m)
		})
	}
}

func TestConfigModel_Init(t *testing.T) {
	m := NewConfigModel()
	cmd := m.Init()
	assert.Nil(t, cmd)
}

func TestConfigModel_ImplementsTeaModel(t *testing.T) {
	// Verify at compile time that ConfigModel satisfies tea.Model.
	var _ tea.Model = (*ConfigModel)(nil)
}

func TestConfigModel_WindowSizeMsg(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, m *ConfigModel)
		name   string
		width  int
		height int
	}{
		{
			name:   "normal size",
			width:  80,
			height: 24,
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.Equal(t, 80, m.Width())
				assert.Equal(t, 24, m.Height())
			},
		},
		{
			name:   "minimum size",
			width:  60,
			height: 16,
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.Equal(t, 60, m.Width())
				assert.Equal(t, 16, m.Height())
			},
		},
		{
			name:   "too small",
			width:  40,
			height: 10,
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.Equal(t, 40, m.Width())
				assert.Equal(t, 10, m.Height())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := NewConfigModel()
			result, cmd := m.Update(tea.WindowSizeMsg{Width: tc.width, Height: tc.height})
			updated := result.(*ConfigModel)
			assert.Nil(t, cmd)
			tc.assert(t, updated)
		})
	}
}

func TestConfigModel_TabSwitching(t *testing.T) {
	testCases := []struct {
		assert    func(t *testing.T, m *ConfigModel)
		name      string
		keys      []tea.KeyMsg
		initState ModelState
	}{
		{
			name:      "tab advances to next tab",
			keys:      []tea.KeyMsg{tabKey()},
			initState: StateBrowsing,
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.Equal(t, TabJira, m.ActiveTab())
			},
		},
		{
			name:      "multiple tabs cycle through all tabs",
			keys:      []tea.KeyMsg{tabKey(), tabKey(), tabKey()},
			initState: StateBrowsing,
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.Equal(t, TabWorktrees, m.ActiveTab())
			},
		},
		{
			name:      "tab wraps around from last to first",
			keys:      []tea.KeyMsg{tabKey(), tabKey(), tabKey(), tabKey()},
			initState: StateBrowsing,
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.Equal(t, TabGeneral, m.ActiveTab())
			},
		},
		{
			name:      "shift-tab goes to previous tab (wraps to last)",
			keys:      []tea.KeyMsg{shiftTabKey()},
			initState: StateBrowsing,
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.Equal(t, TabWorktrees, m.ActiveTab())
			},
		},
		{
			name:      "tab ignored during editing state",
			keys:      []tea.KeyMsg{tabKey()},
			initState: StateEditing,
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.Equal(t, TabGeneral, m.ActiveTab())
			},
		},
		{
			name:      "shift-tab ignored during editing state",
			keys:      []tea.KeyMsg{shiftTabKey()},
			initState: StateEditing,
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.Equal(t, TabGeneral, m.ActiveTab())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := NewConfigModel()
			m.state = tc.initState
			m.width = 80
			m.height = 24

			var result tea.Model = m
			for _, k := range tc.keys {
				result, _ = result.Update(k)
			}
			updated := result.(*ConfigModel)
			tc.assert(t, updated)
		})
	}
}

func TestConfigModel_ViewTooSmall(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, view string)
		name   string
		width  int
		height int
	}{
		{
			name:   "width too small",
			width:  59,
			height: 20,
			assert: func(t *testing.T, view string) {
				t.Helper()
				assert.Contains(t, view, "Terminal too small")
				assert.Contains(t, view, "59x20")
				assert.Contains(t, view, "60x16")
			},
		},
		{
			name:   "height too small",
			width:  80,
			height: 15,
			assert: func(t *testing.T, view string) {
				t.Helper()
				assert.Contains(t, view, "Terminal too small")
				assert.Contains(t, view, "80x15")
			},
		},
		{
			name:   "both too small",
			width:  30,
			height: 10,
			assert: func(t *testing.T, view string) {
				t.Helper()
				assert.Contains(t, view, "Terminal too small")
				assert.Contains(t, view, "30x10")
			},
		},
		{
			name:   "exactly minimum size renders normally",
			width:  60,
			height: 16,
			assert: func(t *testing.T, view string) {
				t.Helper()
				assert.NotContains(t, view, "Terminal too small")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := NewConfigModel()
			m.width = tc.width
			m.height = tc.height
			view := m.View()
			tc.assert(t, view)
		})
	}
}

func TestConfigModel_TabBarRendering(t *testing.T) {
	testCases := []struct {
		setup  func(m *ConfigModel)
		assert func(t *testing.T, tabBar string)
		name   string
	}{
		{
			name:  "general tab active by default",
			setup: func(_ *ConfigModel) {},
			assert: func(t *testing.T, tabBar string) {
				t.Helper()
				assert.Contains(t, tabBar, "[General]")
				assert.Contains(t, tabBar, "JIRA")
				assert.Contains(t, tabBar, "File Copy")
				assert.Contains(t, tabBar, "Worktrees")
			},
		},
		{
			name: "jira tab active",
			setup: func(m *ConfigModel) {
				m.activeTab = TabJira
			},
			assert: func(t *testing.T, tabBar string) {
				t.Helper()
				assert.Contains(t, tabBar, "[JIRA]")
				assert.Contains(t, tabBar, "General")
				// "General" should not be bracketed.
				assert.NotContains(t, tabBar, "[General]")
			},
		},
		{
			name: "tab with error badge",
			setup: func(m *ConfigModel) {
				m.SetTabBadge(TabJira, true)
			},
			assert: func(t *testing.T, tabBar string) {
				t.Helper()
				assert.Contains(t, tabBar, "JIRA (!)")
			},
		},
		{
			name: "multiple tabs with badges",
			setup: func(m *ConfigModel) {
				m.SetTabBadge(TabGeneral, true)
				m.SetTabBadge(TabFileCopy, true)
			},
			assert: func(t *testing.T, tabBar string) {
				t.Helper()
				assert.Contains(t, tabBar, "General (!)")
				assert.Contains(t, tabBar, "File Copy (!)")
				assert.NotContains(t, tabBar, "JIRA (!)")
			},
		},
		{
			name: "clear badge",
			setup: func(m *ConfigModel) {
				m.SetTabBadge(TabJira, true)
				m.SetTabBadge(TabJira, false)
			},
			assert: func(t *testing.T, tabBar string) {
				t.Helper()
				assert.NotContains(t, tabBar, "JIRA (!)")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := NewConfigModel()
			m.width = 80
			m.height = 24
			tc.setup(m)
			tabBar := m.viewTabBar()
			tc.assert(t, tabBar)
		})
	}
}

func TestConfigModel_StatusBarRendering(t *testing.T) {
	testCases := []struct {
		setup  func(m *ConfigModel)
		assert func(t *testing.T, statusBar string)
		name   string
	}{
		{
			name:  "no dirty fields shows keybinding hints only",
			setup: func(_ *ConfigModel) {},
			assert: func(t *testing.T, statusBar string) {
				t.Helper()
				assert.NotContains(t, statusBar, "modified")
				assert.NotContains(t, statusBar, "new file")
				assert.Contains(t, statusBar, "tab")
				assert.Contains(t, statusBar, "help")
			},
		},
		{
			name: "new file indicator",
			setup: func(m *ConfigModel) {
				m.isNewFile = true
			},
			assert: func(t *testing.T, statusBar string) {
				t.Helper()
				assert.Contains(t, statusBar, "[new file]")
			},
		},
		{
			name: "dirty count shown",
			setup: func(m *ConfigModel) {
				m.dirty.Set("key1", "new_value")
			},
			assert: func(t *testing.T, statusBar string) {
				t.Helper()
				assert.Contains(t, statusBar, "[1 modified]")
			},
		},
		{
			name: "multiple dirty fields",
			setup: func(m *ConfigModel) {
				m.dirty.Set("key1", "new_value")
				m.dirty.Set("key2", "new_value")
				m.dirty.Set("key3", "new_value")
			},
			assert: func(t *testing.T, statusBar string) {
				t.Helper()
				assert.Contains(t, statusBar, "[3 modified]")
			},
		},
		{
			name: "flash message replaces keybinding hints",
			setup: func(m *ConfigModel) {
				m.flashMessage = "ok saved .gbm/config.yaml"
			},
			assert: func(t *testing.T, statusBar string) {
				t.Helper()
				assert.Contains(t, statusBar, "ok saved .gbm/config.yaml")
			},
		},
		{
			name: "editing state shows editing keybindings",
			setup: func(m *ConfigModel) {
				m.state = StateEditing
			},
			assert: func(t *testing.T, statusBar string) {
				t.Helper()
				assert.Contains(t, statusBar, "enter")
				assert.Contains(t, statusBar, "confirm")
				assert.Contains(t, statusBar, "esc")
				assert.Contains(t, statusBar, "cancel")
			},
		},
		{
			name: "field-type-aware edit verb for bool",
			setup: func(m *ConfigModel) {
				m.focusedFieldType = Bool
			},
			assert: func(t *testing.T, statusBar string) {
				t.Helper()
				assert.Contains(t, statusBar, "toggle")
			},
		},
		{
			name: "field-type-aware edit verb for string list",
			setup: func(m *ConfigModel) {
				m.focusedFieldType = StringList
			},
			assert: func(t *testing.T, statusBar string) {
				t.Helper()
				assert.Contains(t, statusBar, "open")
			},
		},
		{
			name: "field-type-aware edit verb for string",
			setup: func(m *ConfigModel) {
				m.focusedFieldType = String
			},
			assert: func(t *testing.T, statusBar string) {
				t.Helper()
				assert.Contains(t, statusBar, "edit")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := NewConfigModel()
			m.width = 80
			m.height = 24
			tc.setup(m)
			statusBar := m.viewStatusBar()
			tc.assert(t, statusBar)
		})
	}
}

func TestConfigModel_FlashMessage(t *testing.T) {
	m := NewConfigModel()
	cmd := m.SetFlash("test message")

	assert.Equal(t, "test message", m.flashMessage)
	require.NotNil(t, cmd, "SetFlash should return a tea.Cmd for auto-clear")

	// Simulate the flash clear message.
	result, _ := m.Update(flashClearMsg{})
	updated := result.(*ConfigModel)
	assert.Empty(t, updated.flashMessage)
}

func TestConfigModel_ViewFourTabs(t *testing.T) {
	m := NewConfigModel()
	m.width = 80
	m.height = 24

	view := m.View()
	assert.Contains(t, view, "General")
	assert.Contains(t, view, "JIRA")
	assert.Contains(t, view, "File Copy")
	assert.Contains(t, view, "Worktrees")
}

func TestConfigModel_ActiveTabVisuallyDistinct(t *testing.T) {
	m := NewConfigModel()
	m.width = 80
	m.height = 24

	view := m.View()
	assert.Contains(t, view, "[General]")

	m.activeTab = TabJira
	view = m.View()
	assert.Contains(t, view, "[JIRA]")
	assert.NotContains(t, view, "[General]")
}

func TestConfigModel_ContentShowsActiveTab(t *testing.T) {
	testCases := []struct {
		name     string
		expected string
		tab      SectionTab
	}{
		{name: "general tab shows field labels", tab: TabGeneral, expected: "Default Branch"},
		{name: "jira tab shows group headers", tab: TabJira, expected: "Connection"},
		{name: "file copy tab shows auto-copy fields", tab: TabFileCopy, expected: "Auto Copy"},
		{name: "worktrees tab shows entry list", tab: TabWorktrees, expected: "Worktrees"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := NewConfigModel()
			m.width = 80
			m.height = 24
			m.activeTab = tc.tab
			view := m.View()
			assert.Contains(t, view, tc.expected)
		})
	}
}

func TestConfigModel_QuitFromBrowsing(t *testing.T) {
	testCases := []struct {
		name string
		key  tea.KeyMsg
	}{
		{
			name: "q quits",
			key:  runeKey('q'),
		},
		{
			name: "ctrl-c quits",
			key:  ctrlCKey(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := NewConfigModel()
			m.width = 80
			m.height = 24

			_, cmd := m.Update(tc.key)
			require.NotNil(t, cmd)

			msg := cmd()
			_, isQuit := msg.(tea.QuitMsg)
			assert.True(t, isQuit, "expected tea.QuitMsg, got %T", msg)
		})
	}
}

func TestConfigModel_EditingStateCancels(t *testing.T) {
	testCases := []struct {
		name    string
		key     tea.KeyMsg
		wantCmd bool
	}{
		{
			name:    "esc cancels editing",
			key:     escKey(),
			wantCmd: false,
		},
		{
			name:    "enter confirms editing",
			key:     enterKey(),
			wantCmd: false,
		},
		{
			name:    "ctrl-c cancels editing (does not quit)",
			key:     ctrlCKey(),
			wantCmd: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := NewConfigModel()
			m.state = StateEditing
			m.width = 80
			m.height = 24

			result, cmd := m.Update(tc.key)
			updated := result.(*ConfigModel)

			assert.Equal(t, StateBrowsing, updated.State(),
				"should return to browsing state")
			assert.Nil(t, cmd, "editing key should not produce a command")
		})
	}
}

func TestConfigModel_SetFocusedFieldType(t *testing.T) {
	m := NewConfigModel()
	m.width = 80
	m.height = 24

	m.SetFocusedFieldType(Bool)
	assert.Equal(t, Bool, m.focusedFieldType)

	statusBar := m.viewStatusBar()
	assert.Contains(t, statusBar, "toggle")
}

func TestConfigModel_SetTabBadgeBoundsCheck(t *testing.T) {
	m := NewConfigModel()

	// Setting badge on an invalid tab should not panic.
	m.SetTabBadge(SectionTab(-1), true)
	m.SetTabBadge(SectionTab(tabCount), true)
	m.SetTabBadge(SectionTab(100), true)

	// Valid tabs should work.
	m.SetTabBadge(TabGeneral, true)
	assert.True(t, m.tabBadges[TabGeneral])
}

func TestFieldTypeToString(t *testing.T) {
	testCases := []struct {
		name     string
		expected string
		ft       FieldType
	}{
		{name: "string", ft: String, expected: "string"},
		{name: "sensitive string", ft: SensitiveString, expected: "sensitive_string"},
		{name: "int", ft: Int, expected: "int"},
		{name: "bool", ft: Bool, expected: "bool"},
		{name: "string list", ft: StringList, expected: "string_list"},
		{name: "object list", ft: ObjectList, expected: "object_list"},
		{name: "unknown defaults to string", ft: FieldType(99), expected: "string"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := fieldTypeToString(tc.ft)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestConfigModel_ViewContainsSeparators(t *testing.T) {
	m := NewConfigModel()
	m.width = 80
	m.height = 24

	view := m.View()
	assert.Contains(t, view, "\u2500",
		"view should contain horizontal line separator characters")
}

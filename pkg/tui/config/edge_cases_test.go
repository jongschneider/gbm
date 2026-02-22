package config

import (
	"maps"
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- EmptyState tests ---.

func TestNewEmptyState(t *testing.T) {
	testCases := []struct {
		assert   func(t *testing.T, es *EmptyState)
		name     string
		defaults []FieldMeta
		isEmpty  bool
	}{
		{
			name:    "empty state with defaults",
			isEmpty: true,
			defaults: []FieldMeta{
				{Key: "jira.host", Label: "Host", Type: String},
			},
			assert: func(t *testing.T, es *EmptyState) {
				t.Helper()
				assert.True(t, es.IsEmpty())
				assert.Len(t, es.Defaults(), 1)
				assert.Equal(t, "jira.host", es.Defaults()[0].Key)
			},
		},
		{
			name:    "non-empty state",
			isEmpty: false,
			defaults: []FieldMeta{
				{Key: "jira.host", Label: "Host", Type: String},
			},
			assert: func(t *testing.T, es *EmptyState) {
				t.Helper()
				assert.False(t, es.IsEmpty())
				assert.Len(t, es.Defaults(), 1)
			},
		},
		{
			name:     "empty with no defaults",
			isEmpty:  true,
			defaults: nil,
			assert: func(t *testing.T, es *EmptyState) {
				t.Helper()
				assert.True(t, es.IsEmpty())
				assert.Nil(t, es.Defaults())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			es := NewEmptyState(tc.isEmpty, tc.defaults)
			tc.assert(t, es)
		})
	}
}

func TestEmptyState_SetEmpty(t *testing.T) {
	es := NewEmptyState(true, nil)
	require.True(t, es.IsEmpty())

	es.SetEmpty(false)
	assert.False(t, es.IsEmpty())

	es.SetEmpty(true)
	assert.True(t, es.IsEmpty())
}

func TestPlaceholderText(t *testing.T) {
	text := PlaceholderText()
	assert.Contains(t, text, "not configured")
	assert.Contains(t, text, "press e to set up")
}

// --- SectionEnabled tests ---.

func TestNewSectionEnabled(t *testing.T) {
	testCases := []struct {
		assert  func(t *testing.T, se *SectionEnabled)
		name    string
		key     string
		enabled bool
	}{
		{
			name:    "enabled section",
			key:     "jira.enabled",
			enabled: true,
			assert: func(t *testing.T, se *SectionEnabled) {
				t.Helper()
				assert.True(t, se.IsEnabled())
				assert.Equal(t, "jira.enabled", se.Key())
			},
		},
		{
			name:    "disabled section",
			key:     "jira.enabled",
			enabled: false,
			assert: func(t *testing.T, se *SectionEnabled) {
				t.Helper()
				assert.False(t, se.IsEnabled())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			se := NewSectionEnabled(tc.key, tc.enabled)
			tc.assert(t, se)
		})
	}
}

func TestSectionEnabled_Toggle(t *testing.T) {
	se := NewSectionEnabled("jira.enabled", true)
	require.True(t, se.IsEnabled())

	se.Toggle()
	assert.False(t, se.IsEnabled())

	se.Toggle()
	assert.True(t, se.IsEnabled())
}

func TestSectionEnabled_VisibleFieldCount(t *testing.T) {
	testCases := []struct {
		assert     func(t *testing.T, count int)
		name       string
		enabled    bool
		fieldCount int
	}{
		{
			name:       "enabled shows all fields",
			enabled:    true,
			fieldCount: 10,
			assert: func(t *testing.T, count int) {
				t.Helper()
				assert.Equal(t, 10, count)
			},
		},
		{
			name:       "disabled shows zero fields",
			enabled:    false,
			fieldCount: 10,
			assert: func(t *testing.T, count int) {
				t.Helper()
				assert.Equal(t, 0, count)
			},
		},
		{
			name:       "enabled with zero fields",
			enabled:    true,
			fieldCount: 0,
			assert: func(t *testing.T, count int) {
				t.Helper()
				assert.Equal(t, 0, count)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			se := NewSectionEnabled("jira.enabled", tc.enabled)
			tc.assert(t, se.VisibleFieldCount(tc.fieldCount))
		})
	}
}

// --- CorruptConfigState tests ---.

func TestNewCorruptConfigState(t *testing.T) {
	cs := NewCorruptConfigState("yaml: line 5: bad indentation", "/tmp/config.yaml")

	assert.Equal(t, "yaml: line 5: bad indentation", cs.ParseError())
	assert.Equal(t, "/tmp/config.yaml", cs.FilePath())
}

func TestRenderCorruptConfig(t *testing.T) {
	testCases := []struct {
		assert     func(t *testing.T, output string)
		name       string
		parseError string
		width      int
		height     int
	}{
		{
			name:       "renders error message and hints",
			width:      80,
			height:     24,
			parseError: "yaml: line 5: bad indentation",
			assert: func(t *testing.T, output string) {
				t.Helper()
				assert.Contains(t, output, "Corrupt Config File")
				assert.Contains(t, output, "yaml: line 5: bad indentation")
				assert.Contains(t, output, "open in $EDITOR")
				assert.Contains(t, output, "quit")
			},
		},
		{
			name:       "small terminal still renders",
			width:      40,
			height:     12,
			parseError: "parse error",
			assert: func(t *testing.T, output string) {
				t.Helper()
				assert.Contains(t, output, "Corrupt Config File")
				assert.Contains(t, output, "parse error")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			theme := CorruptConfigTheme{}
			output := RenderCorruptConfig(tc.width, tc.height, tc.parseError, theme)
			tc.assert(t, output)
		})
	}
}

// --- ConfigModel corrupt config integration tests ---.

func TestConfigModel_SetCorruptConfig(t *testing.T) {
	m := NewConfigModel(WithFilePath("/tmp/config.yaml"))
	m.width = 80
	m.height = 24

	m.SetCorruptConfig("yaml: line 5: bad indentation", "/tmp/config.yaml")

	assert.Equal(t, StateCorruptConfig, m.State())
	require.NotNil(t, m.CorruptConfig())
	assert.Equal(t, "yaml: line 5: bad indentation", m.CorruptConfig().ParseError())
	assert.Equal(t, "/tmp/config.yaml", m.CorruptConfig().FilePath())
}

func TestConfigModel_CorruptConfigView(t *testing.T) {
	m := NewConfigModel(WithFilePath("/tmp/config.yaml"))
	m.width = 80
	m.height = 24
	m.SetCorruptConfig("yaml: line 5: bad indentation", "/tmp/config.yaml")

	view := m.View()
	assert.Contains(t, view, "Corrupt Config File")
	assert.Contains(t, view, "yaml: line 5: bad indentation")
	assert.Contains(t, view, "open in $EDITOR")
}

func TestConfigModel_CorruptConfigKeyHandling(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, m *ConfigModel, cmd tea.Cmd)
		name   string
		key    tea.KeyMsg
	}{
		{
			name: "q quits",
			key:  runeKey('q'),
			assert: func(t *testing.T, m *ConfigModel, cmd tea.Cmd) {
				t.Helper()
				require.NotNil(t, cmd)
				msg := cmd()
				_, isQuit := msg.(tea.QuitMsg)
				assert.True(t, isQuit)
			},
		},
		{
			name: "esc quits",
			key:  escKey(),
			assert: func(t *testing.T, m *ConfigModel, cmd tea.Cmd) {
				t.Helper()
				require.NotNil(t, cmd)
				msg := cmd()
				_, isQuit := msg.(tea.QuitMsg)
				assert.True(t, isQuit)
			},
		},
		{
			name: "ctrl-c quits",
			key:  ctrlCKey(),
			assert: func(t *testing.T, m *ConfigModel, cmd tea.Cmd) {
				t.Helper()
				require.NotNil(t, cmd)
				msg := cmd()
				_, isQuit := msg.(tea.QuitMsg)
				assert.True(t, isQuit)
			},
		},
		{
			name: "unhandled key is ignored",
			key:  runeKey('x'),
			assert: func(t *testing.T, m *ConfigModel, cmd tea.Cmd) {
				t.Helper()
				assert.Nil(t, cmd)
				assert.Equal(t, StateCorruptConfig, m.State())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := NewConfigModel()
			m.width = 80
			m.height = 24
			m.SetCorruptConfig("parse error", "/tmp/config.yaml")

			result, cmd := m.Update(tc.key)
			updated := result.(*ConfigModel)
			tc.assert(t, updated, cmd)
		})
	}
}

// reloadableTestAccessor implements ConfigAccessor for editor-reload tests.
// ReloadFromFile replaces the internal values with reloadValues, simulating
// a real accessor re-reading from disk.
type reloadableTestAccessor struct {
	values       map[string]any
	reloadValues map[string]any
	reloadCalled bool
}

func (a *reloadableTestAccessor) GetValue(key string) any { return a.values[key] }
func (a *reloadableTestAccessor) SetValue(key string, value any) error {
	a.values[key] = value
	return nil
}

func (a *reloadableTestAccessor) ReloadFromFile(_ string) error {
	a.reloadCalled = true
	maps.Copy(a.values, a.reloadValues)
	return nil
}

func TestConfigModel_CorruptConfigEditorReload_Success(t *testing.T) {
	// Create a valid YAML config file for the reload to parse.
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	err := os.WriteFile(configPath, []byte("default_branch: main\n"), 0o644)
	require.NoError(t, err)

	m := NewConfigModel(WithFilePath(configPath))
	m.width = 80
	m.height = 24
	m.SetCorruptConfig("yaml: parse error", configPath)

	// Simulate successful editor close.
	result, cmd := m.Update(editorReloadMsg{err: nil})
	updated := result.(*ConfigModel)

	assert.Equal(t, StateBrowsing, updated.State())
	assert.Nil(t, updated.CorruptConfig())
	assert.NotNil(t, updated.root)
	require.NotNil(t, cmd, "should return flash command")
}

func TestConfigModel_EditorReload_ReinitializesAccessorAndSections(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	err := os.WriteFile(configPath, []byte("default_branch: develop\nworktrees_dir: wt\n"), 0o644)
	require.NoError(t, err)

	// Accessor starts with "stale" values (simulating corrupt/pre-editor state).
	staleValues := map[string]any{
		"default_branch": "",
		"worktrees_dir":  "",
	}
	// After reload, accessor returns "fresh" values.
	freshValues := map[string]any{
		"default_branch": "develop",
		"worktrees_dir":  "wt",
	}
	accessor := &reloadableTestAccessor{
		values:       staleValues,
		reloadValues: freshValues,
	}

	dt := NewDirtyTracker(staleValues)
	m := NewConfigModel(
		WithAccessor(accessor),
		WithDirtyTracker(dt),
		WithFilePath(configPath),
	)
	m.width = 80
	m.height = 24
	m.InitSections()

	// Put model into corrupt config state.
	m.SetCorruptConfig("yaml: parse error", configPath)
	require.Equal(t, StateCorruptConfig, m.State())

	// Simulate successful editor close (editor fixed the file).
	result, cmd := m.Update(editorReloadMsg{err: nil})
	updated := result.(*ConfigModel)

	// Verify state transition.
	assert.Equal(t, StateBrowsing, updated.State())
	assert.Nil(t, updated.CorruptConfig())
	require.NotNil(t, cmd, "should return flash command")

	// Verify accessor was reloaded.
	assert.True(t, accessor.reloadCalled, "ReloadFromFile should have been called")
	assert.Equal(t, "develop", accessor.GetValue("default_branch"))

	// Verify dirty tracker was rebuilt (all clean).
	assert.Equal(t, 0, updated.dirty.DirtyCount(),
		"dirty tracker should be clean after reload")

	// Verify sections were rebuilt with fresh values.
	sections := updated.Sections()
	require.NotNil(t, sections[TabGeneral])
	generalSection := sections[TabGeneral]
	// The first field in the general section should show the reloaded value.
	focusedRow := generalSection.FocusedRow()
	assert.Equal(t, "develop", focusedRow.Value,
		"section field should show reloaded value, not stale value")
}

func TestConfigModel_CorruptConfigEditorReload_StillCorrupt(t *testing.T) {
	// Create an invalid YAML file.
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	err := os.WriteFile(configPath, []byte("{{bad yaml\n"), 0o644)
	require.NoError(t, err)

	m := NewConfigModel(WithFilePath(configPath))
	m.width = 80
	m.height = 24
	m.SetCorruptConfig("yaml: parse error", configPath)

	// Simulate editor close -- file is still corrupt.
	result, _ := m.Update(editorReloadMsg{err: nil})
	updated := result.(*ConfigModel)

	assert.Equal(t, StateCorruptConfig, updated.State())
	require.NotNil(t, updated.CorruptConfig())
	assert.Contains(t, updated.CorruptConfig().ParseError(), "parse config YAML")
}

func TestConfigModel_CorruptConfigEditorReload_EditorError(t *testing.T) {
	m := NewConfigModel()
	m.width = 80
	m.height = 24
	m.SetCorruptConfig("yaml: parse error", "/tmp/config.yaml")

	// Simulate editor failing to launch.
	result, cmd := m.Update(editorReloadMsg{err: os.ErrNotExist})
	updated := result.(*ConfigModel)

	assert.Equal(t, StateCorruptConfig, updated.State())
	require.NotNil(t, updated.CorruptConfig())
	assert.Contains(t, updated.CorruptConfig().ParseError(), "editor error")
	assert.Nil(t, cmd)
}

// --- Terminal too small tests ---.

func TestConfigModel_TerminalTooSmall_Recovery(t *testing.T) {
	testCases := []struct {
		assert   func(t *testing.T, m *ConfigModel, view string)
		name     string
		sequence []tea.WindowSizeMsg
	}{
		{
			name: "shrink then grow restores normal UI",
			sequence: []tea.WindowSizeMsg{
				{Width: 80, Height: 24}, // normal
				{Width: 40, Height: 10}, // too small
				{Width: 80, Height: 24}, // normal again
			},
			assert: func(t *testing.T, m *ConfigModel, view string) {
				t.Helper()
				assert.NotContains(t, view, "Terminal too small")
				assert.Contains(t, view, "General")
			},
		},
		{
			name: "exactly at minimum shows normal UI",
			sequence: []tea.WindowSizeMsg{
				{Width: 40, Height: 10}, // too small
				{Width: 60, Height: 16}, // exactly minimum
			},
			assert: func(t *testing.T, m *ConfigModel, view string) {
				t.Helper()
				assert.NotContains(t, view, "Terminal too small")
			},
		},
		{
			name: "width at minimum but height below shows too small",
			sequence: []tea.WindowSizeMsg{
				{Width: 60, Height: 15},
			},
			assert: func(t *testing.T, m *ConfigModel, view string) {
				t.Helper()
				assert.Contains(t, view, "Terminal too small")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := NewConfigModel()

			var result tea.Model = m
			for _, sizeMsg := range tc.sequence {
				result, _ = result.Update(sizeMsg)
			}
			updated := result.(*ConfigModel)
			view := updated.View()
			tc.assert(t, updated, view)
		})
	}
}

func TestConfigModel_IsTerminalTooSmall(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, tooSmall bool)
		name   string
		width  int
		height int
	}{
		{
			name:   "normal size is not too small",
			width:  80,
			height: 24,
			assert: func(t *testing.T, tooSmall bool) {
				t.Helper()
				assert.False(t, tooSmall)
			},
		},
		{
			name:   "exactly minimum is not too small",
			width:  60,
			height: 16,
			assert: func(t *testing.T, tooSmall bool) {
				t.Helper()
				assert.False(t, tooSmall)
			},
		},
		{
			name:   "width below minimum",
			width:  59,
			height: 24,
			assert: func(t *testing.T, tooSmall bool) {
				t.Helper()
				assert.True(t, tooSmall)
			},
		},
		{
			name:   "height below minimum",
			width:  80,
			height: 15,
			assert: func(t *testing.T, tooSmall bool) {
				t.Helper()
				assert.True(t, tooSmall)
			},
		},
		{
			name:   "zero dimensions",
			width:  0,
			height: 0,
			assert: func(t *testing.T, tooSmall bool) {
				t.Helper()
				assert.True(t, tooSmall)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := NewConfigModel()
			m.width = tc.width
			m.height = tc.height
			tc.assert(t, m.IsTerminalTooSmall())
		})
	}
}

func TestTooSmallMessage(t *testing.T) {
	msg := TooSmallMessage(40, 10)
	assert.Contains(t, msg, "Terminal too small")
	assert.Contains(t, msg, "40x10")
	assert.Contains(t, msg, "60x16")
}

// --- SectionModel empty state tests ---.

func TestSectionModel_EmptyState(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, s *SectionModel)
		name   string
		opts   []SectionOption
	}{
		{
			name: "section without empty state",
			opts: nil,
			assert: func(t *testing.T, s *SectionModel) {
				t.Helper()
				assert.False(t, s.IsEmpty())
				assert.Nil(t, s.EmptyState())
			},
		},
		{
			name: "section with empty state true",
			opts: []SectionOption{
				WithEmptyState(NewEmptyState(true, jiraFields)),
			},
			assert: func(t *testing.T, s *SectionModel) {
				t.Helper()
				assert.True(t, s.IsEmpty())
				assert.NotNil(t, s.EmptyState())
			},
		},
		{
			name: "section with empty state false",
			opts: []SectionOption{
				WithEmptyState(NewEmptyState(false, jiraFields)),
			},
			assert: func(t *testing.T, s *SectionModel) {
				t.Helper()
				assert.False(t, s.IsEmpty())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s := NewSectionModel(testFields(), tc.opts...)
			tc.assert(t, s)
		})
	}
}

func TestSectionModel_EmptyStateView(t *testing.T) {
	s := NewSectionModel(nil,
		WithEmptyState(NewEmptyState(true, jiraFields)),
		WithViewportHeight(20),
		WithWidth(72),
	)

	view := s.View()
	assert.Contains(t, view, "not configured")
	assert.Contains(t, view, "press e to set up")
}

func TestSectionModel_PopulateDefaults(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, s *SectionModel, populated bool)
		name   string
		fields []FieldMeta
		opts   []SectionOption
	}{
		{
			name:   "populate defaults on empty section",
			fields: nil,
			opts: []SectionOption{
				WithEmptyState(NewEmptyState(true, testFields())),
			},
			assert: func(t *testing.T, s *SectionModel, populated bool) {
				t.Helper()
				assert.True(t, populated)
				assert.False(t, s.IsEmpty())
				assert.Equal(t, 2, s.FieldCount())
				assert.Equal(t, "Default Branch", s.FocusedRow().Label)
			},
		},
		{
			name:   "no-op on non-empty section",
			fields: testFields(),
			opts: []SectionOption{
				WithEmptyState(NewEmptyState(false, jiraFields)),
			},
			assert: func(t *testing.T, s *SectionModel, populated bool) {
				t.Helper()
				assert.False(t, populated)
				assert.Equal(t, 2, s.FieldCount())
			},
		},
		{
			name:   "no-op when no empty state configured",
			fields: testFields(),
			opts:   nil,
			assert: func(t *testing.T, s *SectionModel, populated bool) {
				t.Helper()
				assert.False(t, populated)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s := NewSectionModel(tc.fields, tc.opts...)
			populated := s.PopulateDefaults()
			tc.assert(t, s, populated)
		})
	}
}

func TestSectionModel_EmptyToPopulatedViewTransition(t *testing.T) {
	s := NewSectionModel(nil,
		WithEmptyState(NewEmptyState(true, testFields())),
		WithViewportHeight(20),
		WithWidth(72),
	)

	// Before populate: shows placeholder.
	view1 := s.View()
	assert.Contains(t, view1, "not configured")

	// Populate defaults.
	populated := s.PopulateDefaults()
	require.True(t, populated)

	// After populate: shows fields.
	view2 := s.View()
	assert.NotContains(t, view2, "not configured")
	assert.Contains(t, view2, "Default Branch")
}

// --- RenderEmptySection tests ---.

func TestRenderEmptySection(t *testing.T) {
	output := RenderEmptySection(80, 20, nil)
	assert.Contains(t, output, "not configured")
	assert.Contains(t, output, "press e to set up")
}

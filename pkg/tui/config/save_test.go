package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// newTestYAMLRoot creates a minimal YAML document node for testing.
func newTestYAMLRoot() *yaml.Node {
	return &yaml.Node{
		Kind: yaml.DocumentNode,
		Content: []*yaml.Node{
			{
				Kind: yaml.MappingNode,
				Tag:  "!!map",
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Tag: "!!str", Value: "default_branch"},
					{Kind: yaml.ScalarNode, Tag: "!!str", Value: "main"},
				},
			},
		},
	}
}

func TestSaveFlow_Validate(t *testing.T) {
	testCases := []struct {
		name        string
		accessor    ConfigAccessor
		assertError func(t *testing.T, errs []ValidationError)
	}{
		{
			name: "no errors with valid config",
			accessor: &stubAccessor{values: map[string]any{
				"default_branch": "main",
			}},
			assertError: func(t *testing.T, errs []ValidationError) {
				t.Helper()
				assert.Empty(t, errs)
			},
		},
		{
			name:     "nil accessor values produce no errors",
			accessor: &stubAccessor{values: map[string]any{}},
			assertError: func(t *testing.T, errs []ValidationError) {
				t.Helper()
				// With no values set, optional fields pass; required fields
				// return nil from GetValue which is treated as "not set".
				assert.Empty(t, errs)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sf := NewSaveFlow("/tmp/test.yaml", time.Now(), newTestYAMLRoot(), NewDirtyTracker(nil), tc.accessor, false)
			errs := sf.Validate()
			tc.assertError(t, errs)
		})
	}
}

func TestSaveFlow_NeedsOverwriteConfirmation(t *testing.T) {
	testCases := []struct {
		name        string
		setup       func(t *testing.T) (string, time.Time, bool)
		assert      func(t *testing.T, needs bool)
		assertError func(t *testing.T, err error)
	}{
		{
			name: "new file never needs confirmation",
			setup: func(_ *testing.T) (string, time.Time, bool) {
				return "/tmp/nonexistent.yaml", time.Now(), true
			},
			assert: func(t *testing.T, needs bool) {
				t.Helper()
				assert.False(t, needs)
			},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name: "unchanged file does not need confirmation",
			setup: func(t *testing.T) (string, time.Time, bool) {
				t.Helper()
				dir := t.TempDir()
				path := filepath.Join(dir, "config.yaml")
				require.NoError(t, os.WriteFile(path, []byte("key: value\n"), 0o644))
				info, err := os.Stat(path)
				require.NoError(t, err)
				return path, info.ModTime(), false
			},
			assert: func(t *testing.T, needs bool) {
				t.Helper()
				assert.False(t, needs)
			},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name: "externally changed file needs confirmation",
			setup: func(t *testing.T) (string, time.Time, bool) {
				t.Helper()
				dir := t.TempDir()
				path := filepath.Join(dir, "config.yaml")
				require.NoError(t, os.WriteFile(path, []byte("key: value\n"), 0o644))
				// Record a mod time from "the past".
				oldTime := time.Now().Add(-10 * time.Second)
				require.NoError(t, os.Chtimes(path, oldTime, oldTime))
				// Write again to change the mod time.
				require.NoError(t, os.WriteFile(path, []byte("key: updated\n"), 0o644))
				return path, oldTime, false
			},
			assert: func(t *testing.T, needs bool) {
				t.Helper()
				assert.True(t, needs)
			},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name: "deleted file needs confirmation",
			setup: func(t *testing.T) (string, time.Time, bool) {
				t.Helper()
				dir := t.TempDir()
				path := filepath.Join(dir, "deleted.yaml")
				// File does not exist on disk but we have a mod time from load.
				return path, time.Now(), false
			},
			assert: func(t *testing.T, needs bool) {
				t.Helper()
				assert.True(t, needs)
			},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			path, modTime, isNew := tc.setup(t)
			sf := NewSaveFlow(path, modTime, newTestYAMLRoot(), NewDirtyTracker(nil), &stubAccessor{values: map[string]any{}}, isNew)
			needs, err := sf.NeedsOverwriteConfirmation()
			tc.assertError(t, err)
			tc.assert(t, needs)
		})
	}
}

func TestSaveFlow_Execute(t *testing.T) {
	testCases := []struct {
		name        string
		setup       func(t *testing.T) (*SaveFlow, string)
		assert      func(t *testing.T, modTime time.Time, dir string)
		assertError func(t *testing.T, err error)
	}{
		{
			name: "saves to existing file with backup",
			setup: func(t *testing.T) (*SaveFlow, string) {
				t.Helper()
				dir := t.TempDir()
				configDir := filepath.Join(dir, ".gbm")
				require.NoError(t, os.MkdirAll(configDir, 0o755))
				path := filepath.Join(configDir, "config.yaml")
				require.NoError(t, os.WriteFile(path, []byte("default_branch: main\n"), 0o644))

				root := newTestYAMLRoot()
				dirty := NewDirtyTracker(map[string]any{"default_branch": "main"})
				dirty.Set("default_branch", "develop")
				accessor := &stubAccessor{values: map[string]any{"default_branch": "develop"}}

				sf := NewSaveFlow(path, time.Now(), root, dirty, accessor, false)
				return sf, dir
			},
			assert: func(t *testing.T, _ time.Time, dir string) {
				t.Helper()
				configDir := filepath.Join(dir, ".gbm")
				// Backup should exist.
				bakPath := filepath.Join(configDir, "config.yaml.bak")
				assert.FileExists(t, bakPath)
				bakData, err := os.ReadFile(bakPath)
				require.NoError(t, err)
				assert.Contains(t, string(bakData), "default_branch: main")

				// Main file should have updated content.
				data, err := os.ReadFile(filepath.Join(configDir, "config.yaml"))
				require.NoError(t, err)
				assert.Contains(t, string(data), "develop")
			},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name: "creates new file and parent directory",
			setup: func(t *testing.T) (*SaveFlow, string) {
				t.Helper()
				dir := t.TempDir()
				configDir := filepath.Join(dir, ".gbm")
				path := filepath.Join(configDir, "config.yaml")
				// Directory does not exist yet.

				root := newTestYAMLRoot()
				dirty := NewDirtyTracker(nil)
				dirty.Set("default_branch", "main")
				accessor := &stubAccessor{values: map[string]any{"default_branch": "main"}}

				sf := NewSaveFlow(path, time.Time{}, root, dirty, accessor, true)
				return sf, dir
			},
			assert: func(t *testing.T, _ time.Time, dir string) {
				t.Helper()
				configDir := filepath.Join(dir, ".gbm")
				path := filepath.Join(configDir, "config.yaml")
				assert.FileExists(t, path)
				// No backup for new files.
				bakPath := path + ".bak"
				_, err := os.Stat(bakPath)
				assert.True(t, os.IsNotExist(err), "backup should not exist for new file")
			},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name: "marks dirty tracker clean after save",
			setup: func(t *testing.T) (*SaveFlow, string) {
				t.Helper()
				dir := t.TempDir()
				path := filepath.Join(dir, "config.yaml")
				require.NoError(t, os.WriteFile(path, []byte("default_branch: main\n"), 0o644))

				root := newTestYAMLRoot()
				dirty := NewDirtyTracker(map[string]any{"default_branch": "main"})
				dirty.Set("default_branch", "develop")
				accessor := &stubAccessor{values: map[string]any{"default_branch": "develop"}}

				sf := NewSaveFlow(path, time.Now(), root, dirty, accessor, false)
				return sf, dir
			},
			assert: func(t *testing.T, _ time.Time, _ string) {
				t.Helper()
				// The dirty tracker state is verified in the test body below.
			},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name: "write to read-only directory fails",
			setup: func(t *testing.T) (*SaveFlow, string) {
				t.Helper()
				dir := t.TempDir()
				readOnlyDir := filepath.Join(dir, "readonly")
				require.NoError(t, os.MkdirAll(readOnlyDir, 0o555))
				path := filepath.Join(readOnlyDir, "subdir", "config.yaml")

				root := newTestYAMLRoot()
				dirty := NewDirtyTracker(nil)
				accessor := &stubAccessor{values: map[string]any{}}

				sf := NewSaveFlow(path, time.Time{}, root, dirty, accessor, true)
				return sf, dir
			},
			assert: func(t *testing.T, _ time.Time, _ string) {
				t.Helper()
			},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.Error(t, err)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sf, dir := tc.setup(t)
			modTime, err := sf.Execute()
			tc.assertError(t, err)
			tc.assert(t, modTime, dir)

			// Special check for dirty tracker test.
			if tc.name == "marks dirty tracker clean after save" {
				assert.False(t, sf.dirty.IsDirty(), "dirty tracker should be clean after save")
			}
		})
	}
}

func TestSaveFlow_Execute_PreservesYAMLComments(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	yamlContent := "# Top-level comment\ndefault_branch: main # inline comment\n"
	require.NoError(t, os.WriteFile(path, []byte(yamlContent), 0o644))

	// Load the file to get a node tree with comments.
	cf, err := LoadConfigFile(path)
	require.NoError(t, err)

	dirty := NewDirtyTracker(map[string]any{"default_branch": "main"})
	dirty.Set("default_branch", "develop")
	accessor := &stubAccessor{values: map[string]any{"default_branch": "develop"}}

	sf := NewSaveFlow(path, cf.ModTime, cf.Root, dirty, accessor, false)
	_, err = sf.Execute()
	require.NoError(t, err)

	// Read back and verify comments are preserved.
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, "# Top-level comment")
	assert.Contains(t, content, "inline comment")
	assert.Contains(t, content, "develop")
}

func TestConfigModel_SaveKeyTriggersSaveFlow(t *testing.T) {
	testCases := []struct {
		name   string
		key    tea.KeyMsg
		assert func(t *testing.T, m *ConfigModel, cmd tea.Cmd)
	}{
		{
			name: "s key triggers save without quit",
			key:  runeKey('s'),
			assert: func(t *testing.T, m *ConfigModel, _ tea.Cmd) {
				t.Helper()
				// Without accessor, save skips validation and proceeds.
				// Without filePath, it will enter StateSaving.
				assert.Equal(t, StateSaving, m.State())
				assert.False(t, m.quitAfterSave)
			},
		},
		{
			name: "enter key triggers save-and-quit",
			key:  enterKey(),
			assert: func(t *testing.T, m *ConfigModel, _ tea.Cmd) {
				t.Helper()
				assert.Equal(t, StateSaving, m.State())
				assert.True(t, m.quitAfterSave)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := NewConfigModel(
				WithFilePath("/tmp/nonexistent.yaml"),
				WithYAMLRoot(newTestYAMLRoot()),
				WithNewFile(true),
			)
			m.width = 80
			m.height = 24

			result, cmd := m.Update(tc.key)
			updated := result.(*ConfigModel)
			tc.assert(t, updated, cmd)
		})
	}
}

func TestConfigModel_SaveWithValidationErrors(t *testing.T) {
	// Create an accessor that triggers a validation error by setting
	// a required field to empty string.
	accessor := &stubAccessor{values: map[string]any{
		"default_branch": "",
	}}

	m := NewConfigModel(
		WithFilePath("/tmp/test.yaml"),
		WithAccessor(accessor),
		WithYAMLRoot(newTestYAMLRoot()),
	)
	m.width = 80
	m.height = 24

	result, cmd := m.Update(runeKey('s'))
	updated := result.(*ConfigModel)

	assert.Equal(t, StateErrors, updated.State())
	assert.Nil(t, cmd)
	assert.True(t, updated.ErrorOverlay().HasErrors())
}

func TestConfigModel_SaveExternalChangeShowsOverwrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("default_branch: main\n"), 0o644))

	// Use an old mod time to simulate external change.
	oldModTime := time.Now().Add(-1 * time.Hour)

	m := NewConfigModel(
		WithFilePath(path),
		WithYAMLRoot(newTestYAMLRoot()),
		WithModTime(oldModTime),
	)
	m.width = 80
	m.height = 24

	result, cmd := m.Update(runeKey('s'))
	updated := result.(*ConfigModel)

	assert.Equal(t, StateOverwriteConfirm, updated.State())
	assert.Nil(t, cmd)
}

func TestConfigModel_OverwriteConfirmKey(t *testing.T) {
	testCases := []struct {
		name   string
		key    tea.KeyMsg
		assert func(t *testing.T, m *ConfigModel, cmd tea.Cmd)
	}{
		{
			name: "y confirms overwrite and starts save",
			key:  runeKey('y'),
			assert: func(t *testing.T, m *ConfigModel, cmd tea.Cmd) {
				t.Helper()
				assert.Equal(t, StateSaving, m.State())
				assert.NotNil(t, cmd)
			},
		},
		{
			name: "n cancels overwrite",
			key:  runeKey('n'),
			assert: func(t *testing.T, m *ConfigModel, cmd tea.Cmd) {
				t.Helper()
				assert.Equal(t, StateBrowsing, m.State())
				assert.Nil(t, cmd)
			},
		},
		{
			name: "esc cancels overwrite",
			key:  escKey(),
			assert: func(t *testing.T, m *ConfigModel, cmd tea.Cmd) {
				t.Helper()
				assert.Equal(t, StateBrowsing, m.State())
				assert.Nil(t, cmd)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := NewConfigModel(
				WithFilePath("/tmp/test.yaml"),
				WithYAMLRoot(newTestYAMLRoot()),
				WithNewFile(true),
			)
			m.state = StateOverwriteConfirm
			m.width = 80
			m.height = 24

			result, cmd := m.Update(tc.key)
			updated := result.(*ConfigModel)
			tc.assert(t, updated, cmd)
		})
	}
}

func TestConfigModel_WriteErrorOverlay(t *testing.T) {
	m := NewConfigModel()
	m.state = StateWriteError
	m.writeErrorMsg = "permission denied"
	m.width = 80
	m.height = 24

	// View should contain the error message.
	view := m.View()
	assert.Contains(t, view, "Write Error")
	assert.Contains(t, view, "permission denied")

	// Esc should close the overlay.
	result, cmd := m.Update(escKey())
	updated := result.(*ConfigModel)
	assert.Equal(t, StateBrowsing, updated.State())
	assert.Empty(t, updated.WriteErrorMsg())
	assert.Nil(t, cmd)
}

func TestConfigModel_OverwriteConfirmView(t *testing.T) {
	m := NewConfigModel()
	m.state = StateOverwriteConfirm
	m.width = 80
	m.height = 24

	view := m.View()
	assert.Contains(t, view, "File changed externally")
	assert.Contains(t, view, "Overwrite")
}

func TestConfigModel_HandleSaveResult(t *testing.T) {
	testCases := []struct {
		name   string
		msg    SaveResultMsg
		quit   bool
		assert func(t *testing.T, m *ConfigModel, cmd tea.Cmd)
	}{
		{
			name: "successful save shows flash and returns to browsing",
			msg:  SaveResultMsg{Path: "/tmp/.gbm/config.yaml"},
			quit: false,
			assert: func(t *testing.T, m *ConfigModel, cmd tea.Cmd) {
				t.Helper()
				assert.Equal(t, StateBrowsing, m.State())
				assert.Contains(t, m.flashMessage, "ok saved")
				assert.Contains(t, m.flashMessage, "config.yaml")
				assert.False(t, m.IsNewFile())
				assert.NotNil(t, cmd, "should return flash clear timer cmd")
			},
		},
		{
			name: "save error shows write error overlay",
			msg:  SaveResultMsg{Err: os.ErrPermission, Path: "/tmp/test.yaml"},
			quit: false,
			assert: func(t *testing.T, m *ConfigModel, cmd tea.Cmd) {
				t.Helper()
				assert.Equal(t, StateWriteError, m.State())
				assert.Contains(t, m.WriteErrorMsg(), "permission")
				assert.Nil(t, cmd)
			},
		},
		{
			name: "successful save-and-quit returns quit command",
			msg:  SaveResultMsg{Path: "/tmp/.gbm/config.yaml"},
			quit: true,
			assert: func(t *testing.T, m *ConfigModel, cmd tea.Cmd) {
				t.Helper()
				assert.Equal(t, StateBrowsing, m.State())
				assert.NotNil(t, cmd, "should return batch cmd with quit")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "config.yaml")
			require.NoError(t, os.WriteFile(path, []byte("test\n"), 0o644))

			m := NewConfigModel(WithFilePath(path))
			m.state = StateSaving
			m.quitAfterSave = tc.quit
			m.width = 80
			m.height = 24

			result, cmd := m.Update(tc.msg)
			updated := result.(*ConfigModel)
			tc.assert(t, updated, cmd)
		})
	}
}

func TestConfigModel_SavingStateIgnoresKeys(t *testing.T) {
	m := NewConfigModel()
	m.state = StateSaving
	m.width = 80
	m.height = 24

	keys := []tea.KeyMsg{
		runeKey('s'),
		runeKey('q'),
		escKey(),
		tabKey(),
		enterKey(),
	}

	for _, k := range keys {
		result, cmd := m.Update(k)
		updated := result.(*ConfigModel)
		assert.Equal(t, StateSaving, updated.State(), "key %q should not change saving state", k.String())
		assert.Nil(t, cmd)
	}
}

func TestSaveFlow_BackupCreation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	originalContent := "# Important config\ndefault_branch: main\nworktrees_dir: worktrees\n"
	require.NoError(t, os.WriteFile(path, []byte(originalContent), 0o644))

	root := newTestYAMLRoot()
	dirty := NewDirtyTracker(map[string]any{"default_branch": "main"})
	dirty.Set("default_branch", "develop")
	accessor := &stubAccessor{values: map[string]any{"default_branch": "develop"}}

	sf := NewSaveFlow(path, time.Now(), root, dirty, accessor, false)
	_, err := sf.Execute()
	require.NoError(t, err)

	// Verify backup contains original content.
	bakPath := path + ".bak"
	assert.FileExists(t, bakPath)
	bakData, err := os.ReadFile(bakPath)
	require.NoError(t, err)
	assert.Equal(t, originalContent, string(bakData))
}

package config

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Quit guard tests ---.

func TestHandleQuit_CleanExitsImmediately(t *testing.T) {
	testCases := []struct {
		name string
		key  tea.KeyMsg
	}{
		{name: "q with clean state", key: runeKey('q')},
		{name: "ctrl-c with clean state", key: ctrlCKey()},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := NewConfigModel()
			m.width = 80
			m.height = 24

			result, cmd := m.Update(tc.key)
			updated := result.(*ConfigModel)

			assert.Equal(t, StateBrowsing, updated.State())
			require.NotNil(t, cmd, "should return a quit command")

			msg := cmd()
			_, isQuit := msg.(tea.QuitMsg)
			assert.True(t, isQuit, "expected tea.QuitMsg, got %T", msg)
		})
	}
}

func TestHandleQuit_DirtyShowsConfirmOverlay(t *testing.T) {
	testCases := []struct {
		name string
		key  tea.KeyMsg
	}{
		{name: "q with dirty state", key: runeKey('q')},
		{name: "ctrl-c with dirty state", key: ctrlCKey()},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dt := NewDirtyTracker(map[string]any{"default_branch": "main"})
			dt.Set("default_branch", "develop")

			m := NewConfigModel(WithDirtyTracker(dt))
			m.width = 80
			m.height = 24

			result, cmd := m.Update(tc.key)
			updated := result.(*ConfigModel)

			assert.Equal(t, StateQuitConfirm, updated.State())
			assert.Nil(t, cmd, "should not return a command")
		})
	}
}

func TestQuitConfirmKey(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, m *ConfigModel, cmd tea.Cmd)
		name   string
		key    tea.KeyMsg
	}{
		{
			name: "s triggers save-and-quit flow",
			key:  runeKey('s'),
			assert: func(t *testing.T, m *ConfigModel, cmd tea.Cmd) {
				t.Helper()
				// startSave sets state to StateSaving when no validation errors
				// and no external changes (new file = true skips both).
				assert.Equal(t, StateSaving, m.State())
				assert.True(t, m.quitAfterSave)
				assert.NotNil(t, cmd)
			},
		},
		{
			name: "d discards changes and quits",
			key:  runeKey('d'),
			assert: func(t *testing.T, m *ConfigModel, cmd tea.Cmd) {
				t.Helper()
				require.NotNil(t, cmd)
				msg := cmd()
				_, isQuit := msg.(tea.QuitMsg)
				assert.True(t, isQuit, "expected tea.QuitMsg, got %T", msg)
			},
		},
		{
			name: "esc cancels and returns to browsing",
			key:  escKey(),
			assert: func(t *testing.T, m *ConfigModel, cmd tea.Cmd) {
				t.Helper()
				assert.Equal(t, StateBrowsing, m.State())
				assert.Nil(t, cmd)
			},
		},
		{
			name: "unrelated key is ignored",
			key:  runeKey('x'),
			assert: func(t *testing.T, m *ConfigModel, cmd tea.Cmd) {
				t.Helper()
				assert.Equal(t, StateQuitConfirm, m.State())
				assert.Nil(t, cmd)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dt := NewDirtyTracker(map[string]any{"default_branch": "main"})
			dt.Set("default_branch", "develop")

			m := NewConfigModel(
				WithDirtyTracker(dt),
				WithFilePath("/tmp/test.yaml"),
				WithYAMLRoot(newTestYAMLRoot()),
				WithNewFile(true),
			)
			m.state = StateQuitConfirm
			m.width = 80
			m.height = 24

			result, cmd := m.Update(tc.key)
			updated := result.(*ConfigModel)
			tc.assert(t, updated, cmd)
		})
	}
}

func TestQuitConfirmView(t *testing.T) {
	testCases := []struct {
		setup  func(m *ConfigModel)
		assert func(t *testing.T, view string)
		name   string
	}{
		{
			name: "shows title and dirty fields",
			setup: func(m *ConfigModel) {
				m.dirty.Set("default_branch", "develop")
			},
			assert: func(t *testing.T, view string) {
				t.Helper()
				assert.Contains(t, view, "Unsaved Changes")
				assert.Contains(t, view, "Default Branch")
				assert.Contains(t, view, "1 field(s) modified")
				assert.Contains(t, view, "Save & Quit")
				assert.Contains(t, view, "Discard")
				assert.Contains(t, view, "Cancel")
			},
		},
		{
			name: "shows multiple dirty fields",
			setup: func(m *ConfigModel) {
				m.dirty.Set("default_branch", "develop")
				m.dirty.Set("worktrees_dir", "/new/path")
			},
			assert: func(t *testing.T, view string) {
				t.Helper()
				assert.Contains(t, view, "2 field(s) modified")
				assert.Contains(t, view, "Default Branch")
				assert.Contains(t, view, "Worktrees Directory")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dt := NewDirtyTracker(map[string]any{
				"default_branch": "main",
				"worktrees_dir":  "worktrees",
			})

			m := NewConfigModel(WithDirtyTracker(dt))
			m.state = StateQuitConfirm
			m.width = 80
			m.height = 24
			tc.setup(m)

			view := m.View()
			tc.assert(t, view)
		})
	}
}

// --- Enter in browsing triggers save-and-quit ---.

func TestEnterInBrowsingTriggersSaveQuit(t *testing.T) {
	m := NewConfigModel(
		WithFilePath("/tmp/test.yaml"),
		WithYAMLRoot(newTestYAMLRoot()),
		WithNewFile(true),
	)
	m.width = 80
	m.height = 24

	result, cmd := m.Update(enterKey())
	updated := result.(*ConfigModel)

	assert.Equal(t, StateSaving, updated.State())
	assert.True(t, updated.quitAfterSave)
	assert.NotNil(t, cmd)
}

// --- Ctrl-C during editing cancels edit first ---.

func TestCtrlCDuringEditingCancelsEdit(t *testing.T) {
	dt := NewDirtyTracker(map[string]any{"default_branch": "main"})
	dt.Set("default_branch", "develop")

	m := NewConfigModel(WithDirtyTracker(dt))
	m.state = StateEditing
	m.width = 80
	m.height = 24

	// First ctrl-c cancels the edit.
	result, cmd := m.Update(ctrlCKey())
	updated := result.(*ConfigModel)
	assert.Equal(t, StateBrowsing, updated.State(), "first ctrl-c should cancel edit")
	assert.Nil(t, cmd)

	// Second ctrl-c (now in browsing with dirty state) shows quit confirm.
	result, cmd = updated.Update(ctrlCKey())
	updated = result.(*ConfigModel)
	assert.Equal(t, StateQuitConfirm, updated.State(), "second ctrl-c should show quit confirm")
	assert.Nil(t, cmd)
}

// --- Single-field reset tests ---.

func TestHandleResetField(t *testing.T) {
	testCases := []struct {
		setup  func(m *ConfigModel)
		assert func(t *testing.T, m *ConfigModel)
		name   string
	}{
		{
			name: "noop when nothing is dirty",
			setup: func(m *ConfigModel) {
				m.focusedFieldKey = "default_branch"
			},
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.Equal(t, StateBrowsing, m.State())
			},
		},
		{
			name: "noop when focused field is not dirty",
			setup: func(m *ConfigModel) {
				m.dirty.Set("worktrees_dir", "/new/path")
				m.focusedFieldKey = "default_branch"
			},
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.Equal(t, StateBrowsing, m.State())
			},
		},
		{
			name: "noop when no focused field key",
			setup: func(m *ConfigModel) {
				m.dirty.Set("default_branch", "develop")
				m.focusedFieldKey = ""
			},
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.Equal(t, StateBrowsing, m.State())
			},
		},
		{
			name: "shows reset confirm when focused field is dirty",
			setup: func(m *ConfigModel) {
				m.dirty.Set("default_branch", "develop")
				m.focusedFieldKey = "default_branch"
			},
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.Equal(t, StateResetConfirm, m.State())
				assert.Equal(t, "default_branch", m.ResetKey())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dt := NewDirtyTracker(map[string]any{
				"default_branch": "main",
				"worktrees_dir":  "worktrees",
			})

			m := NewConfigModel(WithDirtyTracker(dt))
			m.width = 80
			m.height = 24
			tc.setup(m)

			result, _ := m.Update(runeKey('r'))
			updated := result.(*ConfigModel)
			tc.assert(t, updated)
		})
	}
}

func TestResetConfirmKey(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, m *ConfigModel)
		name   string
		key    tea.KeyMsg
	}{
		{
			name: "y resets field and returns to browsing",
			key:  runeKey('y'),
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.Equal(t, StateBrowsing, m.State())
				assert.False(t, m.dirty.IsKeyDirty("default_branch"),
					"field should be reset to original")
				assert.Empty(t, m.ResetKey())
			},
		},
		{
			name: "n cancels and returns to browsing",
			key:  runeKey('n'),
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.Equal(t, StateBrowsing, m.State())
				assert.True(t, m.dirty.IsKeyDirty("default_branch"),
					"field should still be dirty")
				assert.Empty(t, m.ResetKey())
			},
		},
		{
			name: "esc cancels and returns to browsing",
			key:  escKey(),
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.Equal(t, StateBrowsing, m.State())
				assert.True(t, m.dirty.IsKeyDirty("default_branch"),
					"field should still be dirty")
				assert.Empty(t, m.ResetKey())
			},
		},
		{
			name: "unrelated key is ignored",
			key:  runeKey('x'),
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.Equal(t, StateResetConfirm, m.State())
				assert.Equal(t, "default_branch", m.ResetKey())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dt := NewDirtyTracker(map[string]any{"default_branch": "main"})
			dt.Set("default_branch", "develop")

			m := NewConfigModel(WithDirtyTracker(dt))
			m.state = StateResetConfirm
			m.resetKey = "default_branch"
			m.width = 80
			m.height = 24

			result, _ := m.Update(tc.key)
			updated := result.(*ConfigModel)
			tc.assert(t, updated)
		})
	}
}

// --- Reset all tests ---.

func TestHandleResetAll(t *testing.T) {
	testCases := []struct {
		setup  func(m *ConfigModel)
		assert func(t *testing.T, m *ConfigModel)
		name   string
	}{
		{
			name:  "noop when nothing is dirty",
			setup: func(_ *ConfigModel) {},
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.Equal(t, StateBrowsing, m.State())
			},
		},
		{
			name: "shows reset-all confirm when fields are dirty",
			setup: func(m *ConfigModel) {
				m.dirty.Set("default_branch", "develop")
				m.dirty.Set("worktrees_dir", "/new/path")
			},
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.Equal(t, StateResetAllConfirm, m.State())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dt := NewDirtyTracker(map[string]any{
				"default_branch": "main",
				"worktrees_dir":  "worktrees",
			})

			m := NewConfigModel(WithDirtyTracker(dt))
			m.width = 80
			m.height = 24
			tc.setup(m)

			result, _ := m.Update(runeKey('R'))
			updated := result.(*ConfigModel)
			tc.assert(t, updated)
		})
	}
}

func TestResetAllConfirmKey(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, m *ConfigModel)
		name   string
		key    tea.KeyMsg
	}{
		{
			name: "y resets all fields",
			key:  runeKey('y'),
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.Equal(t, StateBrowsing, m.State())
				assert.False(t, m.dirty.IsDirty(), "all fields should be reset")
				assert.Equal(t, 0, m.dirty.DirtyCount())
			},
		},
		{
			name: "n cancels reset-all",
			key:  runeKey('n'),
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.Equal(t, StateBrowsing, m.State())
				assert.True(t, m.dirty.IsDirty(), "fields should still be dirty")
			},
		},
		{
			name: "esc cancels reset-all",
			key:  escKey(),
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.Equal(t, StateBrowsing, m.State())
				assert.True(t, m.dirty.IsDirty(), "fields should still be dirty")
			},
		},
		{
			name: "unrelated key is ignored",
			key:  runeKey('x'),
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.Equal(t, StateResetAllConfirm, m.State())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dt := NewDirtyTracker(map[string]any{
				"default_branch": "main",
				"worktrees_dir":  "worktrees",
			})
			dt.Set("default_branch", "develop")
			dt.Set("worktrees_dir", "/new/path")

			m := NewConfigModel(WithDirtyTracker(dt))
			m.state = StateResetAllConfirm
			m.width = 80
			m.height = 24

			result, _ := m.Update(tc.key)
			updated := result.(*ConfigModel)
			tc.assert(t, updated)
		})
	}
}

func TestResetAllConfirmView(t *testing.T) {
	dt := NewDirtyTracker(map[string]any{
		"default_branch": "main",
		"worktrees_dir":  "worktrees",
	})
	dt.Set("default_branch", "develop")
	dt.Set("worktrees_dir", "/new/path")

	m := NewConfigModel(WithDirtyTracker(dt))
	m.state = StateResetAllConfirm
	m.width = 80
	m.height = 24

	view := m.View()
	assert.Contains(t, view, "Reset All Fields")
	assert.Contains(t, view, "2 field(s) modified")
	assert.Contains(t, view, "Default Branch")
	assert.Contains(t, view, "Worktrees Directory")
	assert.Contains(t, view, "Reset")
	assert.Contains(t, view, "Cancel")
}

// --- dirtyKeysToLabels tests ---.

func TestDirtyKeysToLabels(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, labels []string)
		name   string
		keys   []string
	}{
		{
			name: "resolves known keys to labels",
			keys: []string{"default_branch", "worktrees_dir"},
			assert: func(t *testing.T, labels []string) {
				t.Helper()
				assert.Equal(t, []string{"Default Branch", "Worktrees Directory"}, labels)
			},
		},
		{
			name: "unknown key falls back to raw key",
			keys: []string{"unknown.key"},
			assert: func(t *testing.T, labels []string) {
				t.Helper()
				assert.Equal(t, []string{"unknown.key"}, labels)
			},
		},
		{
			name: "mixed known and unknown keys",
			keys: []string{"default_branch", "some.custom.key"},
			assert: func(t *testing.T, labels []string) {
				t.Helper()
				assert.Equal(t, []string{"Default Branch", "some.custom.key"}, labels)
			},
		},
		{
			name: "empty keys returns empty",
			keys: []string{},
			assert: func(t *testing.T, labels []string) {
				t.Helper()
				assert.Empty(t, labels)
			},
		},
		{
			name: "jira field keys resolve correctly",
			keys: []string{"jira.host", "jira.filters.priority"},
			assert: func(t *testing.T, labels []string) {
				t.Helper()
				assert.Equal(t, []string{"Host", "Priority"}, labels)
			},
		},
		{
			name: "file copy field keys resolve correctly",
			keys: []string{"file_copy.auto.enabled"},
			assert: func(t *testing.T, labels []string) {
				t.Helper()
				assert.Equal(t, []string{"Enabled"}, labels)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			labels := dirtyKeysToLabels(tc.keys)
			tc.assert(t, labels)
		})
	}
}

// --- SetFocusedFieldKey tests ---.

func TestSetFocusedFieldKey(t *testing.T) {
	m := NewConfigModel()
	assert.Empty(t, m.FocusedFieldKey())

	m.SetFocusedFieldKey("default_branch")
	assert.Equal(t, "default_branch", m.FocusedFieldKey())

	m.SetFocusedFieldKey("")
	assert.Empty(t, m.FocusedFieldKey())
}

// --- Full flow integration tests ---.

func TestFullFlow_QuitSaveAndQuit(t *testing.T) {
	// Start with dirty model, trigger quit, choose save & quit.
	dir := t.TempDir()
	path := dir + "/config.yaml"

	dt := NewDirtyTracker(map[string]any{"default_branch": "main"})
	dt.Set("default_branch", "develop")

	m := NewConfigModel(
		WithDirtyTracker(dt),
		WithFilePath(path),
		WithYAMLRoot(newTestYAMLRoot()),
		WithNewFile(true),
	)
	m.width = 80
	m.height = 24

	// Step 1: Press q -> should show quit confirm.
	result, cmd := m.Update(runeKey('q'))
	updated := result.(*ConfigModel)
	assert.Equal(t, StateQuitConfirm, updated.State())
	assert.Nil(t, cmd)

	// Step 2: Press s -> should start save-and-quit flow.
	result, cmd = updated.Update(runeKey('s'))
	updated = result.(*ConfigModel)
	assert.Equal(t, StateSaving, updated.State())
	assert.True(t, updated.quitAfterSave)
	require.NotNil(t, cmd)
}

func TestFullFlow_QuitDiscard(t *testing.T) {
	dt := NewDirtyTracker(map[string]any{"default_branch": "main"})
	dt.Set("default_branch", "develop")

	m := NewConfigModel(WithDirtyTracker(dt))
	m.width = 80
	m.height = 24

	// Step 1: Press q -> shows quit confirm.
	result, _ := m.Update(runeKey('q'))
	updated := result.(*ConfigModel)
	assert.Equal(t, StateQuitConfirm, updated.State())

	// Step 2: Press d -> quits without saving.
	result, cmd := updated.Update(runeKey('d'))
	_ = result.(*ConfigModel)
	require.NotNil(t, cmd)
	msg := cmd()
	_, isQuit := msg.(tea.QuitMsg)
	assert.True(t, isQuit)
}

func TestFullFlow_QuitCancel(t *testing.T) {
	dt := NewDirtyTracker(map[string]any{"default_branch": "main"})
	dt.Set("default_branch", "develop")

	m := NewConfigModel(WithDirtyTracker(dt))
	m.width = 80
	m.height = 24

	// Step 1: Press q -> shows quit confirm.
	result, _ := m.Update(runeKey('q'))
	updated := result.(*ConfigModel)
	assert.Equal(t, StateQuitConfirm, updated.State())

	// Step 2: Press esc -> returns to browsing, changes still dirty.
	result, cmd := updated.Update(escKey())
	updated = result.(*ConfigModel)
	assert.Equal(t, StateBrowsing, updated.State())
	assert.Nil(t, cmd)
	assert.True(t, updated.dirty.IsDirty())
}

func TestFullFlow_ResetFieldThenQuit(t *testing.T) {
	dt := NewDirtyTracker(map[string]any{"default_branch": "main"})
	dt.Set("default_branch", "develop")

	m := NewConfigModel(WithDirtyTracker(dt))
	m.width = 80
	m.height = 24
	m.focusedFieldKey = "default_branch"

	// Step 1: Press r -> shows reset confirm.
	result, _ := m.Update(runeKey('r'))
	updated := result.(*ConfigModel)
	assert.Equal(t, StateResetConfirm, updated.State())

	// Step 2: Press y -> resets field.
	result, _ = updated.Update(runeKey('y'))
	updated = result.(*ConfigModel)
	assert.Equal(t, StateBrowsing, updated.State())
	assert.False(t, updated.dirty.IsDirty())

	// Step 3: Press q -> exits immediately (clean).
	result, cmd := updated.Update(runeKey('q'))
	_ = result.(*ConfigModel)
	require.NotNil(t, cmd)
	msg := cmd()
	_, isQuit := msg.(tea.QuitMsg)
	assert.True(t, isQuit)
}

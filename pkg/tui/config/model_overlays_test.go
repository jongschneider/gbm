package config

import (
	"maps"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// overlayTestAccessor implements ConfigAccessor for overlay unit tests.
type overlayTestAccessor struct {
	values map[string]any
}

func (a *overlayTestAccessor) GetValue(key string) any          { return a.values[key] }
func (a *overlayTestAccessor) ReloadFromFile(_ string) error    { return nil }
func (a *overlayTestAccessor) SetValue(key string, v any) error { a.values[key] = v; return nil }

// newOverlayTestModel creates a ConfigModel with an accessor pre-loaded
// with worktree data. Returns both the model and the accessor for assertions.
func newOverlayTestModel(worktrees map[string]worktreeConfig) (*ConfigModel, *overlayTestAccessor) {
	accessor := &overlayTestAccessor{values: map[string]any{
		"worktrees": worktrees,
	}}
	dt := NewDirtyTracker(accessor.values)
	m := NewConfigModel(
		WithAccessor(accessor),
		WithDirtyTracker(dt),
	)
	m.width = 80
	m.height = 24
	return m, accessor
}

// getWorktreeMap extracts the worktrees map from the accessor for assertions.
func getWorktreeMap(t *testing.T, accessor *overlayTestAccessor) map[string]worktreeConfig {
	t.Helper()
	val := accessor.values["worktrees"]
	require.NotNil(t, val, "worktrees should not be nil")

	rv := reflect.ValueOf(val)
	require.Equal(t, reflect.Map, rv.Kind(), "worktrees should be a map")

	result := make(map[string]worktreeConfig)
	for _, key := range rv.MapKeys() {
		entry := rv.MapIndex(key)
		if entry.Kind() == reflect.Ptr {
			entry = entry.Elem()
		}
		result[key.String()] = worktreeConfig{
			Branch:      reflectStringField(entry, "Branch"),
			MergeInto:   reflectStringField(entry, "MergeInto"),
			Description: reflectStringField(entry, "Description"),
		}
	}
	return result
}

func TestUpdateWorktreeEntry(t *testing.T) {
	testCases := []struct {
		initial map[string]worktreeConfig
		assert  func(t *testing.T, accessor *overlayTestAccessor)
		values  [3]string
		name    string
		oldName string
		newName string
	}{
		{
			name: "same-name update modifies entry in place",
			initial: map[string]worktreeConfig{
				"feature-x": {Branch: "feature/x", MergeInto: "main", Description: "Feature X"},
			},
			oldName: "feature-x",
			newName: "feature-x",
			values:  [3]string{"feature/x-v2", "develop", "Updated Feature X"},
			assert: func(t *testing.T, accessor *overlayTestAccessor) {
				t.Helper()
				wt := getWorktreeMap(t, accessor)
				require.Contains(t, wt, "feature-x")
				assert.Equal(t, "feature/x-v2", wt["feature-x"].Branch)
				assert.Equal(t, "develop", wt["feature-x"].MergeInto)
				assert.Equal(t, "Updated Feature X", wt["feature-x"].Description)
				assert.Len(t, wt, 1, "map should still have exactly one entry")
			},
		},
		{
			name: "rename deletes old key and creates new entry with updated values",
			initial: map[string]worktreeConfig{
				"feature-x": {Branch: "feature/x", MergeInto: "main", Description: "Feature X"},
			},
			oldName: "feature-x",
			newName: "feature-y",
			values:  [3]string{"feature/y", "develop", "Feature Y"},
			assert: func(t *testing.T, accessor *overlayTestAccessor) {
				t.Helper()
				wt := getWorktreeMap(t, accessor)
				assert.NotContains(t, wt, "feature-x", "old key should be deleted")
				require.Contains(t, wt, "feature-y", "new key should exist")
				assert.Equal(t, "feature/y", wt["feature-y"].Branch)
				assert.Equal(t, "develop", wt["feature-y"].MergeInto)
				assert.Equal(t, "Feature Y", wt["feature-y"].Description)
				assert.Len(t, wt, 1, "map should still have exactly one entry")
			},
		},
		{
			name: "rename preserves other entries in map",
			initial: map[string]worktreeConfig{
				"feature-x": {Branch: "feature/x", MergeInto: "main", Description: "Feature X"},
				"hotfix-1":  {Branch: "hotfix/1", MergeInto: "main", Description: "Hotfix 1"},
			},
			oldName: "feature-x",
			newName: "feature-z",
			values:  [3]string{"feature/z", "main", "Feature Z"},
			assert: func(t *testing.T, accessor *overlayTestAccessor) {
				t.Helper()
				wt := getWorktreeMap(t, accessor)
				assert.NotContains(t, wt, "feature-x", "old key should be deleted")
				require.Contains(t, wt, "feature-z", "new key should exist")
				assert.Equal(t, "feature/z", wt["feature-z"].Branch)

				// Other entry should be untouched.
				require.Contains(t, wt, "hotfix-1", "other entry should still exist")
				assert.Equal(t, "hotfix/1", wt["hotfix-1"].Branch)
				assert.Equal(t, "main", wt["hotfix-1"].MergeInto)
				assert.Equal(t, "Hotfix 1", wt["hotfix-1"].Description)
				assert.Len(t, wt, 2)
			},
		},
		{
			name: "same-name update preserves other entries",
			initial: map[string]worktreeConfig{
				"feature-x": {Branch: "feature/x", MergeInto: "main", Description: "Feature X"},
				"hotfix-1":  {Branch: "hotfix/1", MergeInto: "main", Description: "Hotfix 1"},
			},
			oldName: "hotfix-1",
			newName: "hotfix-1",
			values:  [3]string{"hotfix/1-fixed", "release", "Hotfix 1 Fixed"},
			assert: func(t *testing.T, accessor *overlayTestAccessor) {
				t.Helper()
				wt := getWorktreeMap(t, accessor)
				require.Contains(t, wt, "hotfix-1")
				assert.Equal(t, "hotfix/1-fixed", wt["hotfix-1"].Branch)
				assert.Equal(t, "release", wt["hotfix-1"].MergeInto)
				assert.Equal(t, "Hotfix 1 Fixed", wt["hotfix-1"].Description)

				// Other entry untouched.
				require.Contains(t, wt, "feature-x")
				assert.Equal(t, "feature/x", wt["feature-x"].Branch)
				assert.Len(t, wt, 2)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			initial := make(map[string]worktreeConfig, len(tc.initial))
			maps.Copy(initial, tc.initial)

			m, accessor := newOverlayTestModel(initial)
			m.updateWorktreeEntry(tc.oldName, tc.newName, tc.values)
			tc.assert(t, accessor)
		})
	}
}

func TestUpdateWorktreeEntry_NonMapValue(t *testing.T) {
	// When the accessor returns a non-map value for "worktrees",
	// updateWorktreeEntry should return early without panicking.
	accessor := &overlayTestAccessor{values: map[string]any{
		"worktrees": "not-a-map",
	}}
	m := NewConfigModel(WithAccessor(accessor))
	m.width = 80
	m.height = 24

	// Should not panic.
	m.updateWorktreeEntry("x", "y", [3]string{"b", "m", "d"})
	assert.Equal(t, "not-a-map", accessor.values["worktrees"])
}

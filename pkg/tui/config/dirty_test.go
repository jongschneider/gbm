package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDirtyTracker(t *testing.T) {
	tests := []struct {
		input  map[string]any
		assert func(t *testing.T, dt *DirtyTracker)
		name   string
	}{
		{
			name:  "creates tracker from values",
			input: map[string]any{"branch": "main", "enabled": true},
			assert: func(t *testing.T, dt *DirtyTracker) {
				t.Helper()
				assert.Equal(t, "main", dt.GetOriginal("branch"))
				assert.Equal(t, true, dt.GetOriginal("enabled"))
				assert.False(t, dt.IsDirty())
			},
		},
		{
			name:  "nil input creates empty tracker",
			input: nil,
			assert: func(t *testing.T, dt *DirtyTracker) {
				t.Helper()
				assert.False(t, dt.IsDirty())
				assert.Equal(t, 0, dt.DirtyCount())
			},
		},
		{
			name:  "empty input creates clean tracker",
			input: map[string]any{},
			assert: func(t *testing.T, dt *DirtyTracker) {
				t.Helper()
				assert.False(t, dt.IsDirty())
				assert.Empty(t, dt.DirtyKeys())
			},
		},
		{
			name:  "deep copies original map",
			input: map[string]any{"items": []string{"a", "b"}},
			assert: func(t *testing.T, dt *DirtyTracker) {
				t.Helper()
				// Mutating the returned original should not affect the tracker.
				orig := dt.GetOriginal("items").([]string)
				assert.Equal(t, []string{"a", "b"}, orig)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dt := NewDirtyTracker(tc.input)
			require.NotNil(t, dt)
			tc.assert(t, dt)
		})
	}
}

func TestNewDirtyTracker_does_not_alias_input(t *testing.T) {
	input := map[string]any{"branch": "main", "items": []string{"a"}}
	dt := NewDirtyTracker(input)

	// Mutate the input map after construction.
	input["branch"] = "develop"
	input["items"].([]string)[0] = "z"

	assert.Equal(t, "main", dt.GetOriginal("branch"),
		"mutating input map should not affect original snapshot")
	assert.Equal(t, []string{"a"}, dt.GetOriginal("items"),
		"mutating input slice should not affect original snapshot")
}

func TestDirtyTracker_Set_and_IsDirty(t *testing.T) {
	tests := []struct {
		setup  func(dt *DirtyTracker)
		assert func(t *testing.T, dt *DirtyTracker)
		name   string
	}{
		{
			name: "string change marks dirty",
			setup: func(dt *DirtyTracker) {
				dt.Set("branch", "develop")
			},
			assert: func(t *testing.T, dt *DirtyTracker) {
				t.Helper()
				assert.True(t, dt.IsDirty())
				assert.True(t, dt.IsKeyDirty("branch"))
			},
		},
		{
			name: "string set to same value stays clean",
			setup: func(dt *DirtyTracker) {
				dt.Set("branch", "main")
			},
			assert: func(t *testing.T, dt *DirtyTracker) {
				t.Helper()
				assert.False(t, dt.IsDirty())
				assert.False(t, dt.IsKeyDirty("branch"))
			},
		},
		{
			name: "int change marks dirty",
			setup: func(dt *DirtyTracker) {
				dt.Set("depth", 10)
			},
			assert: func(t *testing.T, dt *DirtyTracker) {
				t.Helper()
				assert.True(t, dt.IsKeyDirty("depth"))
			},
		},
		{
			name: "int set to same value stays clean",
			setup: func(dt *DirtyTracker) {
				dt.Set("depth", 5)
			},
			assert: func(t *testing.T, dt *DirtyTracker) {
				t.Helper()
				assert.False(t, dt.IsKeyDirty("depth"))
			},
		},
		{
			name: "bool change marks dirty",
			setup: func(dt *DirtyTracker) {
				dt.Set("enabled", false)
			},
			assert: func(t *testing.T, dt *DirtyTracker) {
				t.Helper()
				assert.True(t, dt.IsKeyDirty("enabled"))
			},
		},
		{
			name: "bool set to same value stays clean",
			setup: func(dt *DirtyTracker) {
				dt.Set("enabled", true)
			},
			assert: func(t *testing.T, dt *DirtyTracker) {
				t.Helper()
				assert.False(t, dt.IsKeyDirty("enabled"))
			},
		},
		{
			name: "string list change marks dirty",
			setup: func(dt *DirtyTracker) {
				dt.Set("items", []string{"a", "c"})
			},
			assert: func(t *testing.T, dt *DirtyTracker) {
				t.Helper()
				assert.True(t, dt.IsKeyDirty("items"))
			},
		},
		{
			name: "string list set to same values stays clean",
			setup: func(dt *DirtyTracker) {
				dt.Set("items", []string{"a", "b"})
			},
			assert: func(t *testing.T, dt *DirtyTracker) {
				t.Helper()
				assert.False(t, dt.IsKeyDirty("items"))
			},
		},
		{
			name: "new key not in original marks dirty",
			setup: func(dt *DirtyTracker) {
				dt.Set("new_key", "value")
			},
			assert: func(t *testing.T, dt *DirtyTracker) {
				t.Helper()
				assert.True(t, dt.IsKeyDirty("new_key"))
				assert.True(t, dt.IsDirty())
			},
		},
		{
			name: "change then revert stays clean",
			setup: func(dt *DirtyTracker) {
				dt.Set("branch", "develop")
				dt.Set("branch", "main")
			},
			assert: func(t *testing.T, dt *DirtyTracker) {
				t.Helper()
				assert.False(t, dt.IsKeyDirty("branch"))
				assert.False(t, dt.IsDirty())
			},
		},
	}

	originals := map[string]any{
		"branch":  "main",
		"depth":   5,
		"enabled": true,
		"items":   []string{"a", "b"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dt := NewDirtyTracker(originals)
			tc.setup(dt)
			tc.assert(t, dt)
		})
	}
}

func TestDirtyTracker_nil_vs_empty_slice(t *testing.T) {
	tests := []struct {
		current any
		orig    map[string]any
		assert  func(t *testing.T, dt *DirtyTracker)
		name    string
	}{
		{
			name:    "nil original vs empty slice current is clean",
			orig:    map[string]any{"items": nil},
			current: []string{},
			assert: func(t *testing.T, dt *DirtyTracker) {
				t.Helper()
				assert.False(t, dt.IsKeyDirty("items"))
			},
		},
		{
			name:    "empty slice original vs nil current is clean",
			orig:    map[string]any{"items": []string{}},
			current: nil,
			assert: func(t *testing.T, dt *DirtyTracker) {
				t.Helper()
				assert.False(t, dt.IsKeyDirty("items"))
			},
		},
		{
			name:    "nil original vs non-empty slice is dirty",
			orig:    map[string]any{"items": nil},
			current: []string{"x"},
			assert: func(t *testing.T, dt *DirtyTracker) {
				t.Helper()
				assert.True(t, dt.IsKeyDirty("items"))
			},
		},
		{
			name:    "empty slice original vs non-empty is dirty",
			orig:    map[string]any{"items": []string{}},
			current: []string{"x"},
			assert: func(t *testing.T, dt *DirtyTracker) {
				t.Helper()
				assert.True(t, dt.IsKeyDirty("items"))
			},
		},
		{
			name:    "both nil is clean",
			orig:    map[string]any{"items": nil},
			current: nil,
			assert: func(t *testing.T, dt *DirtyTracker) {
				t.Helper()
				assert.False(t, dt.IsKeyDirty("items"))
			},
		},
		{
			name:    "both empty slice is clean",
			orig:    map[string]any{"items": []string{}},
			current: []string{},
			assert: func(t *testing.T, dt *DirtyTracker) {
				t.Helper()
				assert.False(t, dt.IsKeyDirty("items"))
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dt := NewDirtyTracker(tc.orig)
			dt.Set("items", tc.current)
			tc.assert(t, dt)
		})
	}
}

func TestDirtyTracker_DirtyCount(t *testing.T) {
	tests := []struct {
		setup  func(dt *DirtyTracker)
		assert func(t *testing.T, count int)
		name   string
	}{
		{
			name:  "no changes returns zero",
			setup: func(_ *DirtyTracker) {},
			assert: func(t *testing.T, count int) {
				t.Helper()
				assert.Equal(t, 0, count)
			},
		},
		{
			name: "one change returns one",
			setup: func(dt *DirtyTracker) {
				dt.Set("branch", "develop")
			},
			assert: func(t *testing.T, count int) {
				t.Helper()
				assert.Equal(t, 1, count)
			},
		},
		{
			name: "multiple changes returns correct count",
			setup: func(dt *DirtyTracker) {
				dt.Set("branch", "develop")
				dt.Set("enabled", false)
				dt.Set("depth", 99)
			},
			assert: func(t *testing.T, count int) {
				t.Helper()
				assert.Equal(t, 3, count)
			},
		},
		{
			name: "reverted change not counted",
			setup: func(dt *DirtyTracker) {
				dt.Set("branch", "develop")
				dt.Set("branch", "main") // revert
				dt.Set("enabled", false)
			},
			assert: func(t *testing.T, count int) {
				t.Helper()
				assert.Equal(t, 1, count)
			},
		},
	}

	originals := map[string]any{
		"branch":  "main",
		"depth":   5,
		"enabled": true,
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dt := NewDirtyTracker(originals)
			tc.setup(dt)
			tc.assert(t, dt.DirtyCount())
		})
	}
}

func TestDirtyTracker_DirtyKeys(t *testing.T) {
	tests := []struct {
		setup  func(dt *DirtyTracker)
		assert func(t *testing.T, keys []string)
		name   string
	}{
		{
			name:  "no changes returns empty",
			setup: func(_ *DirtyTracker) {},
			assert: func(t *testing.T, keys []string) {
				t.Helper()
				assert.Empty(t, keys)
			},
		},
		{
			name: "returns sorted keys",
			setup: func(dt *DirtyTracker) {
				dt.Set("depth", 99)
				dt.Set("branch", "develop")
			},
			assert: func(t *testing.T, keys []string) {
				t.Helper()
				assert.Equal(t, []string{"branch", "depth"}, keys)
			},
		},
		{
			name: "excludes clean keys",
			setup: func(dt *DirtyTracker) {
				dt.Set("branch", "develop")
				dt.Set("enabled", true) // same as original
			},
			assert: func(t *testing.T, keys []string) {
				t.Helper()
				assert.Equal(t, []string{"branch"}, keys)
			},
		},
	}

	originals := map[string]any{
		"branch":  "main",
		"depth":   5,
		"enabled": true,
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dt := NewDirtyTracker(originals)
			tc.setup(dt)
			tc.assert(t, dt.DirtyKeys())
		})
	}
}

func TestDirtyTracker_ResetKey(t *testing.T) {
	tests := []struct {
		setup  func(dt *DirtyTracker)
		assert func(t *testing.T, dt *DirtyTracker)
		name   string
	}{
		{
			name: "restores string to original",
			setup: func(dt *DirtyTracker) {
				dt.Set("branch", "develop")
				dt.ResetKey("branch")
			},
			assert: func(t *testing.T, dt *DirtyTracker) {
				t.Helper()
				assert.False(t, dt.IsKeyDirty("branch"))
			},
		},
		{
			name: "restores slice to original",
			setup: func(dt *DirtyTracker) {
				dt.Set("items", []string{"x", "y"})
				dt.ResetKey("items")
			},
			assert: func(t *testing.T, dt *DirtyTracker) {
				t.Helper()
				assert.False(t, dt.IsKeyDirty("items"))
			},
		},
		{
			name: "only resets specified key",
			setup: func(dt *DirtyTracker) {
				dt.Set("branch", "develop")
				dt.Set("enabled", false)
				dt.ResetKey("branch")
			},
			assert: func(t *testing.T, dt *DirtyTracker) {
				t.Helper()
				assert.False(t, dt.IsKeyDirty("branch"))
				assert.True(t, dt.IsKeyDirty("enabled"))
				assert.Equal(t, 1, dt.DirtyCount())
			},
		},
		{
			name: "reset key not in original removes from current",
			setup: func(dt *DirtyTracker) {
				dt.Set("new_key", "value")
				dt.ResetKey("new_key")
			},
			assert: func(t *testing.T, dt *DirtyTracker) {
				t.Helper()
				assert.False(t, dt.IsKeyDirty("new_key"))
				assert.False(t, dt.IsDirty())
			},
		},
		{
			name: "reset noop on clean key",
			setup: func(dt *DirtyTracker) {
				dt.ResetKey("branch")
			},
			assert: func(t *testing.T, dt *DirtyTracker) {
				t.Helper()
				assert.False(t, dt.IsKeyDirty("branch"))
				assert.False(t, dt.IsDirty())
			},
		},
	}

	originals := map[string]any{
		"branch":  "main",
		"enabled": true,
		"items":   []string{"a", "b"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dt := NewDirtyTracker(originals)
			tc.setup(dt)
			tc.assert(t, dt)
		})
	}
}

func TestDirtyTracker_ResetAll(t *testing.T) {
	dt := NewDirtyTracker(map[string]any{
		"branch":  "main",
		"enabled": true,
		"depth":   5,
		"items":   []string{"a"},
	})

	dt.Set("branch", "develop")
	dt.Set("enabled", false)
	dt.Set("depth", 99)
	dt.Set("items", []string{"x", "y"})
	require.True(t, dt.IsDirty())
	require.Equal(t, 4, dt.DirtyCount())

	dt.ResetAll()

	assert.False(t, dt.IsDirty())
	assert.Equal(t, 0, dt.DirtyCount())
	assert.Empty(t, dt.DirtyKeys())
}

func TestDirtyTracker_MarkClean(t *testing.T) {
	dt := NewDirtyTracker(map[string]any{
		"branch":  "main",
		"enabled": true,
	})

	dt.Set("branch", "develop")
	dt.Set("enabled", false)
	require.True(t, dt.IsDirty())

	dt.MarkClean()

	assert.False(t, dt.IsDirty(), "should be clean after MarkClean")
	assert.Equal(t, "develop", dt.GetOriginal("branch"),
		"original should be updated to current after MarkClean")
	assert.Equal(t, false, dt.GetOriginal("enabled"),
		"original should be updated to current after MarkClean")

	// Further edits are compared against the new baseline.
	dt.Set("branch", "main")
	assert.True(t, dt.IsKeyDirty("branch"),
		"changing from new baseline should be dirty")
}

func TestDirtyTracker_GetOriginal(t *testing.T) {
	tests := []struct {
		assert func(t *testing.T, val any)
		name   string
		key    string
	}{
		{
			name: "returns string original",
			key:  "branch",
			assert: func(t *testing.T, val any) {
				t.Helper()
				assert.Equal(t, "main", val)
			},
		},
		{
			name: "returns int original",
			key:  "depth",
			assert: func(t *testing.T, val any) {
				t.Helper()
				assert.Equal(t, 5, val)
			},
		},
		{
			name: "returns bool original",
			key:  "enabled",
			assert: func(t *testing.T, val any) {
				t.Helper()
				assert.Equal(t, true, val)
			},
		},
		{
			name: "returns slice original",
			key:  "items",
			assert: func(t *testing.T, val any) {
				t.Helper()
				assert.Equal(t, []string{"a", "b"}, val)
			},
		},
		{
			name: "returns nil for unknown key",
			key:  "nonexistent",
			assert: func(t *testing.T, val any) {
				t.Helper()
				assert.Nil(t, val)
			},
		},
	}

	originals := map[string]any{
		"branch":  "main",
		"depth":   5,
		"enabled": true,
		"items":   []string{"a", "b"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dt := NewDirtyTracker(originals)
			// Modify current to verify GetOriginal returns the original, not current.
			dt.Set("branch", "develop")
			dt.Set("depth", 99)
			tc.assert(t, dt.GetOriginal(tc.key))
		})
	}
}

// testRule is a struct containing a slice, making it non-comparable.
// This mirrors service.FileCopyRule for dirty-tracking tests.
type testRule struct {
	Source string
	Files  []string
}

// testWorktree is a simple struct for map value tests.
type testWorktree struct {
	Branch string
}

func TestValuesEqual(t *testing.T) {
	tests := []struct {
		a      any
		b      any
		assert func(t *testing.T, equal bool)
		name   string
	}{
		{
			name: "both nil",
			a:    nil,
			b:    nil,
			assert: func(t *testing.T, equal bool) {
				t.Helper()
				assert.True(t, equal)
			},
		},
		{
			name: "equal strings",
			a:    "main",
			b:    "main",
			assert: func(t *testing.T, equal bool) {
				t.Helper()
				assert.True(t, equal)
			},
		},
		{
			name: "different strings",
			a:    "main",
			b:    "develop",
			assert: func(t *testing.T, equal bool) {
				t.Helper()
				assert.False(t, equal)
			},
		},
		{
			name: "equal ints",
			a:    42,
			b:    42,
			assert: func(t *testing.T, equal bool) {
				t.Helper()
				assert.True(t, equal)
			},
		},
		{
			name: "different ints",
			a:    42,
			b:    99,
			assert: func(t *testing.T, equal bool) {
				t.Helper()
				assert.False(t, equal)
			},
		},
		{
			name: "equal bools",
			a:    true,
			b:    true,
			assert: func(t *testing.T, equal bool) {
				t.Helper()
				assert.True(t, equal)
			},
		},
		{
			name: "different bools",
			a:    true,
			b:    false,
			assert: func(t *testing.T, equal bool) {
				t.Helper()
				assert.False(t, equal)
			},
		},
		{
			name: "equal slices",
			a:    []string{"a", "b"},
			b:    []string{"a", "b"},
			assert: func(t *testing.T, equal bool) {
				t.Helper()
				assert.True(t, equal)
			},
		},
		{
			name: "different slices",
			a:    []string{"a", "b"},
			b:    []string{"a", "c"},
			assert: func(t *testing.T, equal bool) {
				t.Helper()
				assert.False(t, equal)
			},
		},
		{
			name: "different length slices",
			a:    []string{"a"},
			b:    []string{"a", "b"},
			assert: func(t *testing.T, equal bool) {
				t.Helper()
				assert.False(t, equal)
			},
		},
		{
			name: "nil vs empty slice",
			a:    nil,
			b:    []string{},
			assert: func(t *testing.T, equal bool) {
				t.Helper()
				assert.True(t, equal)
			},
		},
		{
			name: "empty slice vs nil",
			a:    []string{},
			b:    nil,
			assert: func(t *testing.T, equal bool) {
				t.Helper()
				assert.True(t, equal)
			},
		},
		{
			name: "nil vs non-empty slice",
			a:    nil,
			b:    []string{"x"},
			assert: func(t *testing.T, equal bool) {
				t.Helper()
				assert.False(t, equal)
			},
		},
		{
			name: "string vs nil",
			a:    "hello",
			b:    nil,
			assert: func(t *testing.T, equal bool) {
				t.Helper()
				assert.False(t, equal)
			},
		},
		{
			name: "nil vs string",
			a:    nil,
			b:    "hello",
			assert: func(t *testing.T, equal bool) {
				t.Helper()
				assert.False(t, equal)
			},
		},
		// Non-comparable types: struct slices (mirrors []FileCopyRule).
		{
			name: "equal struct slices",
			a:    []testRule{{Source: "main", Files: []string{".env"}}},
			b:    []testRule{{Source: "main", Files: []string{".env"}}},
			assert: func(t *testing.T, equal bool) {
				t.Helper()
				assert.True(t, equal)
			},
		},
		{
			name: "different struct slices",
			a:    []testRule{{Source: "main", Files: []string{".env"}}},
			b:    []testRule{{Source: "dev", Files: []string{".env"}}},
			assert: func(t *testing.T, equal bool) {
				t.Helper()
				assert.False(t, equal)
			},
		},
		{
			name: "nil vs empty struct slice",
			a:    nil,
			b:    []testRule{},
			assert: func(t *testing.T, equal bool) {
				t.Helper()
				assert.False(t, equal, "nil vs typed empty slice is not equal via DeepEqual")
			},
		},
		// Non-comparable types: maps (mirrors map[string]WorktreeConfig).
		{
			name: "equal maps",
			a:    map[string]testWorktree{"feat": {Branch: "feature/x"}},
			b:    map[string]testWorktree{"feat": {Branch: "feature/x"}},
			assert: func(t *testing.T, equal bool) {
				t.Helper()
				assert.True(t, equal)
			},
		},
		{
			name: "different maps",
			a:    map[string]testWorktree{"feat": {Branch: "feature/x"}},
			b:    map[string]testWorktree{"feat": {Branch: "feature/y"}},
			assert: func(t *testing.T, equal bool) {
				t.Helper()
				assert.False(t, equal)
			},
		},
		{
			name: "nil vs empty map",
			a:    nil,
			b:    map[string]testWorktree{},
			assert: func(t *testing.T, equal bool) {
				t.Helper()
				assert.False(t, equal, "nil vs typed empty map is not equal via DeepEqual")
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.assert(t, valuesEqual(tc.a, tc.b))
		})
	}
}

func TestDirtyTracker_NonComparableTypes(t *testing.T) {
	rules := []testRule{
		{Source: "main", Files: []string{".env", ".config"}},
		{Source: "dev", Files: []string{"Makefile"}},
	}
	worktrees := map[string]testWorktree{
		"feat-x": {Branch: "feature/x"},
		"feat-y": {Branch: "feature/y"},
	}

	tests := []struct {
		setup  func(dt *DirtyTracker)
		assert func(t *testing.T, dt *DirtyTracker)
		name   string
	}{
		{
			name:  "struct slice unchanged stays clean",
			setup: func(_ *DirtyTracker) {},
			assert: func(t *testing.T, dt *DirtyTracker) {
				t.Helper()
				assert.False(t, dt.IsKeyDirty("rules"))
			},
		},
		{
			name: "struct slice changed marks dirty",
			setup: func(dt *DirtyTracker) {
				dt.Set("rules", []testRule{{Source: "other", Files: []string{"x"}}})
			},
			assert: func(t *testing.T, dt *DirtyTracker) {
				t.Helper()
				assert.True(t, dt.IsKeyDirty("rules"))
			},
		},
		{
			name:  "map unchanged stays clean",
			setup: func(_ *DirtyTracker) {},
			assert: func(t *testing.T, dt *DirtyTracker) {
				t.Helper()
				assert.False(t, dt.IsKeyDirty("worktrees"))
			},
		},
		{
			name: "map changed marks dirty",
			setup: func(dt *DirtyTracker) {
				dt.Set("worktrees", map[string]testWorktree{
					"feat-z": {Branch: "feature/z"},
				})
			},
			assert: func(t *testing.T, dt *DirtyTracker) {
				t.Helper()
				assert.True(t, dt.IsKeyDirty("worktrees"))
			},
		},
		{
			name: "DirtyCount includes non-comparable types",
			setup: func(dt *DirtyTracker) {
				dt.Set("rules", []testRule{{Source: "changed", Files: nil}})
				dt.Set("worktrees", map[string]testWorktree{
					"new": {Branch: "new-branch"},
				})
			},
			assert: func(t *testing.T, dt *DirtyTracker) {
				t.Helper()
				assert.Equal(t, 2, dt.DirtyCount())
			},
		},
		{
			name: "ResetKey restores non-comparable type",
			setup: func(dt *DirtyTracker) {
				dt.Set("rules", []testRule{{Source: "changed", Files: nil}})
				dt.ResetKey("rules")
			},
			assert: func(t *testing.T, dt *DirtyTracker) {
				t.Helper()
				assert.False(t, dt.IsKeyDirty("rules"))
			},
		},
		{
			name: "MarkClean resets baseline for non-comparable types",
			setup: func(dt *DirtyTracker) {
				dt.Set("rules", []testRule{{Source: "new", Files: []string{"a"}}})
				dt.MarkClean()
			},
			assert: func(t *testing.T, dt *DirtyTracker) {
				t.Helper()
				assert.False(t, dt.IsDirty())
				assert.Equal(t, 0, dt.DirtyCount())
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			originals := map[string]any{
				"rules":     rules,
				"worktrees": worktrees,
			}
			dt := NewDirtyTracker(originals)
			tc.setup(dt)
			tc.assert(t, dt)
		})
	}
}

func TestCopyValue_deep_copies_inner_slices(t *testing.T) {
	tests := []struct {
		input  any
		mutate func(copied any)
		assert func(t *testing.T, original, copied any)
		name   string
	}{
		{
			name: "struct slice with inner string slice",
			input: []testRule{
				{Source: "main", Files: []string{".env", ".config"}},
				{Source: "dev", Files: []string{"Makefile"}},
			},
			mutate: func(copied any) {
				rules := copied.([]testRule)
				rules[0].Files[0] = "MUTATED"
				rules[0].Files = append(rules[0].Files, "extra")
				rules[1].Source = "CHANGED"
			},
			assert: func(t *testing.T, original, copied any) {
				t.Helper()
				origRules := original.([]testRule)
				copiedRules := copied.([]testRule)
				assert.Equal(t, ".env", origRules[0].Files[0],
					"mutating copied inner slice must not affect original")
				assert.Len(t, origRules[0].Files, 2,
					"appending to copied inner slice must not affect original length")
				assert.Equal(t, "main", origRules[0].Source,
					"original struct fields must be unchanged")
				assert.Equal(t, "MUTATED", copiedRules[0].Files[0],
					"mutation should be visible on the copy")
			},
		},
		{
			name: "map with struct values containing slices",
			input: map[string]testRule{
				"rule1": {Source: "main", Files: []string{"a", "b"}},
			},
			mutate: func(copied any) {
				m := copied.(map[string]testRule)
				r := m["rule1"]
				r.Files[0] = "MUTATED"
				m["rule1"] = r
			},
			assert: func(t *testing.T, original, copied any) {
				t.Helper()
				origMap := original.(map[string]testRule)
				assert.Equal(t, "a", origMap["rule1"].Files[0],
					"mutating copied map value's inner slice must not affect original")
			},
		},
		{
			name:   "nil slice stays nil",
			input:  []testRule(nil),
			mutate: func(_ any) {},
			assert: func(t *testing.T, original, copied any) {
				t.Helper()
				assert.Nil(t, copied)
			},
		},
		{
			name:   "empty slice stays empty",
			input:  []testRule{},
			mutate: func(_ any) {},
			assert: func(t *testing.T, _, copied any) {
				t.Helper()
				rules := copied.([]testRule)
				assert.Empty(t, rules)
			},
		},
		{
			name:  "string slice independence",
			input: []string{"a", "b", "c"},
			mutate: func(copied any) {
				s := copied.([]string)
				s[0] = "MUTATED"
			},
			assert: func(t *testing.T, original, _ any) {
				t.Helper()
				origSlice := original.([]string)
				assert.Equal(t, "a", origSlice[0],
					"mutating copied string slice must not affect original")
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			original := tc.input
			copied := copyValue(original)
			tc.mutate(copied)
			tc.assert(t, original, copied)
		})
	}
}

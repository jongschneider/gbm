package config

import (
	"reflect"
	"sort"
)

// DirtyTracker compares current field values against the last-saved baseline.
// It is used by the Config TUI to show which fields have been modified and to
// support reset-to-original and save operations.
type DirtyTracker struct {
	original map[string]any // snapshot at load/save time
	current  map[string]any // live values (updated via Set)
}

// NewDirtyTracker creates a tracker seeded with the given original values.
// The provided map is deep-copied so that callers cannot mutate the baseline.
func NewDirtyTracker(originals map[string]any) *DirtyTracker {
	return &DirtyTracker{
		original: copyMap(originals),
		current:  copyMap(originals),
	}
}

// Set updates the current value for a key. If the key did not exist in the
// original snapshot it is treated as a new field (original defaults to nil).
func (d *DirtyTracker) Set(key string, value any) {
	d.current[key] = value
}

// IsDirty reports whether any key differs from the baseline.
func (d *DirtyTracker) IsDirty() bool {
	return d.DirtyCount() > 0
}

// DirtyCount returns the number of keys whose current value differs from the
// original baseline. This drives the "[N modified]" status bar indicator.
func (d *DirtyTracker) DirtyCount() int {
	count := 0
	for key := range d.allKeys() {
		if !valuesEqual(d.original[key], d.current[key]) {
			count++
		}
	}
	return count
}

// DirtyKeys returns a sorted list of keys whose current value differs from the
// original baseline.
func (d *DirtyTracker) DirtyKeys() []string {
	var keys []string
	for key := range d.allKeys() {
		if !valuesEqual(d.original[key], d.current[key]) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}

// IsKeyDirty reports whether a single key differs from the baseline.
func (d *DirtyTracker) IsKeyDirty(key string) bool {
	return !valuesEqual(d.original[key], d.current[key])
}

// ResetKey restores a single key to its original (last-saved) value.
func (d *DirtyTracker) ResetKey(key string) {
	orig, ok := d.original[key]
	if !ok {
		delete(d.current, key)
		return
	}
	d.current[key] = copyValue(orig)
}

// ResetAll restores all keys to their original (last-saved) values.
func (d *DirtyTracker) ResetAll() {
	d.current = copyMap(d.original)
}

// MarkClean makes the current values the new baseline. This is called after
// a successful save so that subsequent edits are compared against the newly
// persisted state.
func (d *DirtyTracker) MarkClean() {
	d.original = copyMap(d.current)
}

// GetOriginal returns the original (last-saved) value for a key.
func (d *DirtyTracker) GetOriginal(key string) any {
	return d.original[key]
}

// --- internal helpers ---.

// allKeys returns the union of keys present in original and current.
func (d *DirtyTracker) allKeys() map[string]struct{} {
	keys := make(map[string]struct{}, len(d.original)+len(d.current))
	for k := range d.original {
		keys[k] = struct{}{}
	}
	for k := range d.current {
		keys[k] = struct{}{}
	}
	return keys
}

// valuesEqual compares two values for dirty-tracking purposes.
// Strings, ints, and bools use ==. String slices use element-wise comparison
// with nil and empty slice treated as equivalent. Non-comparable types (slices
// of structs, maps, etc.) fall back to reflect.DeepEqual.
func valuesEqual(a, b any) bool {
	// Fast path: both nil.
	if a == nil && b == nil {
		return true
	}

	// String slice comparison (handles nil vs []string{}).
	aSlice, aIsSlice := toStringSlice(a)
	bSlice, bIsSlice := toStringSlice(b)
	if aIsSlice || bIsSlice {
		// If one side is a slice and the other is nil, normalise nil to empty.
		if !aIsSlice {
			aSlice = nil
		}
		if !bIsSlice {
			bSlice = nil
		}
		return stringSlicesEqual(aSlice, bSlice)
	}

	// Use reflect.DeepEqual for non-comparable types (slices, maps, structs
	// containing slices/maps) to avoid runtime panics.
	if !isComparable(a) || !isComparable(b) {
		return reflect.DeepEqual(a, b)
	}

	// Scalar comparison (string, int, bool, etc.).
	return a == b
}

// isComparable reports whether a value can be safely compared with ==.
func isComparable(v any) bool {
	if v == nil {
		return true
	}
	return reflect.TypeOf(v).Comparable()
}

// toStringSlice attempts to interpret v as a string slice.
func toStringSlice(v any) ([]string, bool) {
	if v == nil {
		return nil, false
	}
	s, ok := v.([]string)
	return s, ok
}

// stringSlicesEqual compares two string slices, treating nil and empty as equal.
func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// copyMap returns a deep copy of a map[string]any, cloning slice values so
// that mutations to the copy do not affect the original.
func copyMap(m map[string]any) map[string]any {
	if m == nil {
		return make(map[string]any)
	}
	c := make(map[string]any, len(m))
	for k, v := range m {
		c[k] = copyValue(v)
	}
	return c
}

// copyValue returns a deep copy of a value. String slices are cloned;
// scalars are returned as-is (they are immutable). Non-comparable types
// (slices of structs, maps) are deep-copied via reflect to prevent
// aliasing between original and current snapshots.
func copyValue(v any) any {
	if v == nil {
		return nil
	}
	if s, ok := v.([]string); ok {
		cp := make([]string, len(s))
		copy(cp, s)
		return cp
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Slice:
		cp := reflect.MakeSlice(rv.Type(), rv.Len(), rv.Len())
		reflect.Copy(cp, rv)
		return cp.Interface()
	case reflect.Map:
		cp := reflect.MakeMap(rv.Type())
		for _, key := range rv.MapKeys() {
			cp.SetMapIndex(key, rv.MapIndex(key))
		}
		return cp.Interface()
	default:
		return v
	}
}

package config

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// editTestAccessor implements ConfigAccessor for edit lifecycle tests.
type editTestAccessor struct {
	values map[string]any
}

func (a *editTestAccessor) GetValue(key string) any { return a.values[key] }
func (a *editTestAccessor) SetValue(key string, v any) error {
	a.values[key] = v
	return nil
}

// newEditTestModel creates a ConfigModel with an accessor and dirty tracker
// initialized from the given values. Suitable for edit lifecycle tests.
func newEditTestModel(values map[string]any) *ConfigModel {
	accessor := &editTestAccessor{values: values}
	dt := NewDirtyTracker(values)
	m := NewConfigModel(
		WithAccessor(accessor),
		WithDirtyTracker(dt),
	)
	m.width = 80
	m.height = 24
	m.InitSections()
	return m
}

func TestHandleEdit_StringField(t *testing.T) {
	m := newEditTestModel(map[string]any{
		"default_branch": "main",
		"worktrees_dir":  "worktrees",
	})

	// Focus should be on the first field (Default Branch).
	assert.Equal(t, StateBrowsing, m.State())

	// Press 'e' to enter editing.
	result, cmd := m.Update(runeKey('e'))
	updated := result.(*ConfigModel)

	assert.Equal(t, StateEditing, updated.State())
	assert.NotNil(t, cmd, "should return blink command for text input")

	// The active field row should be in editing state.
	fr := updated.activeFieldRow()
	require.NotNil(t, fr)
	assert.Equal(t, FieldEditing, fr.State())
	assert.Equal(t, "default_branch", fr.Meta().Key)
}

func TestHandleEdit_BoolField(t *testing.T) {
	m := newEditTestModel(map[string]any{
		"default_branch": "main",
		"worktrees_dir":  "worktrees",
	})

	// Switch to JIRA tab and navigate to "Reverse" (bool field in Filters group).
	m.activeTab = TabJira
	m.syncFocusedField()

	// Find the Reverse field index.
	var reverseIdx int
	for i, f := range jiraFields {
		if f.Key == "jira.filters.reverse" {
			reverseIdx = i
			break
		}
	}

	// Set up the bool field with a value.
	m.fieldRows[TabJira][reverseIdx].SetValue(false)
	// Navigate focus to the Reverse field.
	section := m.activeSection()
	for _, r := range section.Rows() {
		if r.Kind == RowField && r.FieldIndex == reverseIdx {
			break
		}
		if r.IsFocusable() {
			section.MoveFocusDown()
		}
	}
	// Just manually focus the right row.
	rows := section.Rows()
	for i, r := range rows {
		if r.Kind == RowField && r.FieldIndex == reverseIdx {
			// Set focus index directly through navigation.
			for section.FocusIndex() != i {
				section.MoveFocusDown()
			}
			break
		}
	}
	m.syncFocusedField()

	// Press 'e' -- should toggle bool, not enter editing state.
	result, cmd := m.Update(runeKey('e'))
	updated := result.(*ConfigModel)

	assert.Equal(t, StateBrowsing, updated.State(), "bool toggle should stay in browsing")
	assert.Nil(t, cmd, "bool toggle should not return a command")

	// Value should be toggled to true.
	fr := updated.fieldRows[TabJira][reverseIdx]
	assert.Equal(t, true, fr.Value())
	assert.True(t, fr.IsDirty())
}

func TestHandleEditingKey_Confirm_Success(t *testing.T) {
	m := newEditTestModel(map[string]any{
		"default_branch": "main",
		"worktrees_dir":  "worktrees",
	})

	// Enter editing mode on "Default Branch".
	m.Update(runeKey('e'))
	require.Equal(t, StateEditing, m.State())

	fr := m.activeFieldRow()
	require.NotNil(t, fr)

	// Simulate typing a new value.
	fr.UpdateInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d', 'e', 'v'}})

	// Clear the input and set new value directly for test clarity.
	fr.input.SetValue("develop")
	fr.input.SetCursor(len("develop"))

	// Press enter to confirm.
	result, _ := m.Update(enterKey())
	updated := result.(*ConfigModel)

	assert.Equal(t, StateBrowsing, updated.State())
	assert.Equal(t, "develop", fr.Value())
	assert.True(t, fr.IsDirty(), "field should be dirty after edit")
	assert.Equal(t, 1, updated.dirty.DirtyCount(), "dirty count should be 1")
}

func TestHandleEditingKey_Confirm_ValidationFails(t *testing.T) {
	m := newEditTestModel(map[string]any{
		"default_branch": "main",
		"worktrees_dir":  "worktrees",
	})

	// Enter editing mode on "Default Branch" (has ValidateRequired).
	m.Update(runeKey('e'))
	require.Equal(t, StateEditing, m.State())

	fr := m.activeFieldRow()
	require.NotNil(t, fr)

	// Set input to empty string (will fail validation).
	fr.input.SetValue("")

	// Press enter -- should fail validation and stay in editing.
	result, _ := m.Update(enterKey())
	updated := result.(*ConfigModel)

	assert.Equal(t, StateEditing, updated.State(), "should stay in editing on validation failure")
	assert.NotEmpty(t, fr.EditError(), "should have validation error")
}

func TestHandleEditingKey_Cancel(t *testing.T) {
	m := newEditTestModel(map[string]any{
		"default_branch": "main",
		"worktrees_dir":  "worktrees",
	})

	// Enter editing mode.
	m.Update(runeKey('e'))
	require.Equal(t, StateEditing, m.State())

	fr := m.activeFieldRow()
	require.NotNil(t, fr)

	// Type something different.
	fr.input.SetValue("changed")

	// Press esc to cancel.
	result, _ := m.Update(escKey())
	updated := result.(*ConfigModel)

	assert.Equal(t, StateBrowsing, updated.State())
	assert.Equal(t, FieldBrowsing, fr.State())
	// Value should NOT have changed.
	assert.Equal(t, "main", fr.Value())
}

func TestHandleEditingKey_ForceQuit(t *testing.T) {
	m := newEditTestModel(map[string]any{
		"default_branch": "main",
		"worktrees_dir":  "worktrees",
	})

	// Enter editing mode.
	m.Update(runeKey('e'))
	require.Equal(t, StateEditing, m.State())

	// Press ctrl-c -- should cancel edit, not quit.
	result, cmd := m.Update(ctrlCKey())
	updated := result.(*ConfigModel)

	assert.Equal(t, StateBrowsing, updated.State())
	assert.Nil(t, cmd, "ctrl-c during editing should cancel, not quit")
}

func TestHandleEditingKey_ForwardToInput(t *testing.T) {
	m := newEditTestModel(map[string]any{
		"default_branch": "main",
		"worktrees_dir":  "worktrees",
	})

	// Enter editing mode.
	m.Update(runeKey('e'))
	require.Equal(t, StateEditing, m.State())

	fr := m.activeFieldRow()
	require.NotNil(t, fr)

	// Type a character -- should be forwarded to the text input.
	m.Update(runeKey('x'))

	// The text input should contain the typed character.
	inputVal := fr.input.Value()
	assert.Contains(t, inputVal, "x")
}

func TestEditDirtyTracking(t *testing.T) {
	m := newEditTestModel(map[string]any{
		"default_branch": "main",
		"worktrees_dir":  "worktrees",
	})

	assert.Equal(t, 0, m.dirty.DirtyCount(), "should start clean")

	// Enter editing, change value, confirm.
	m.Update(runeKey('e'))
	fr := m.activeFieldRow()
	require.NotNil(t, fr)
	fr.input.SetValue("develop")
	m.Update(enterKey())

	assert.Equal(t, 1, m.dirty.DirtyCount(), "should have 1 dirty field")
	assert.True(t, m.dirty.IsKeyDirty("default_branch"))
	assert.True(t, fr.IsDirty())
}

func TestEditDirtyMarkerVisible(t *testing.T) {
	m := newEditTestModel(map[string]any{
		"default_branch": "main",
		"worktrees_dir":  "worktrees",
	})

	// Edit and confirm.
	m.Update(runeKey('e'))
	fr := m.activeFieldRow()
	require.NotNil(t, fr)
	fr.input.SetValue("develop")
	m.Update(enterKey())

	// Status bar should show modified count.
	statusBar := m.viewStatusBar()
	assert.Contains(t, statusBar, "[1 modified]")
}

func TestResetFieldSyncsFieldRow(t *testing.T) {
	m := newEditTestModel(map[string]any{
		"default_branch": "main",
		"worktrees_dir":  "worktrees",
	})

	// Edit and confirm a change.
	m.Update(runeKey('e'))
	fr := m.activeFieldRow()
	require.NotNil(t, fr)
	fr.input.SetValue("develop")
	m.Update(enterKey())
	require.Equal(t, "develop", fr.Value())
	require.True(t, fr.IsDirty())

	// Press 'r' to reset.
	m.focusedFieldKey = "default_branch"
	m.Update(runeKey('r'))
	require.Equal(t, StateResetConfirm, m.State())

	// Confirm reset with 'y'.
	m.Update(runeKey('y'))
	assert.Equal(t, StateBrowsing, m.State())

	// Field row should be restored.
	assert.Equal(t, "main", fr.Value())
	assert.False(t, fr.IsDirty())
	assert.Equal(t, 0, m.dirty.DirtyCount())
}

func TestResetAllSyncsFieldRows(t *testing.T) {
	values := map[string]any{
		"default_branch": "main",
		"worktrees_dir":  "worktrees",
	}
	m := newEditTestModel(values)

	// Edit first field.
	m.Update(runeKey('e'))
	fr1 := m.activeFieldRow()
	require.NotNil(t, fr1)
	fr1.input.SetValue("develop")
	m.Update(enterKey())

	// Navigate down to second field and edit it.
	m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m.Update(runeKey('e'))
	fr2 := m.activeFieldRow()
	require.NotNil(t, fr2)
	fr2.input.SetValue("other_dir")
	m.Update(enterKey())

	require.Equal(t, 2, m.dirty.DirtyCount())

	// Press 'R' to reset all.
	m.Update(runeKey('R'))
	require.Equal(t, StateResetAllConfirm, m.State())

	// Confirm with 'y'.
	m.Update(runeKey('y'))
	assert.Equal(t, StateBrowsing, m.State())
	assert.Equal(t, 0, m.dirty.DirtyCount())

	assert.Equal(t, "main", fr1.Value())
	assert.False(t, fr1.IsDirty())
	assert.Equal(t, "worktrees", fr2.Value())
	assert.False(t, fr2.IsDirty())
}

func TestSearchState(t *testing.T) {
	m := newEditTestModel(map[string]any{
		"default_branch": "main",
		"worktrees_dir":  "worktrees",
	})

	// Press '/' to open search.
	result, _ := m.Update(runeKey('/'))
	updated := result.(*ConfigModel)
	assert.Equal(t, StateSearch, updated.State())

	// Section search should be active.
	s := updated.activeSection()
	require.NotNil(t, s)
	assert.True(t, s.IsSearchActive())
}

func TestSearchKey_TypeAndClose(t *testing.T) {
	m := newEditTestModel(map[string]any{
		"default_branch": "main",
		"worktrees_dir":  "worktrees",
	})

	// Switch to JIRA tab for more fields to search.
	m.activeTab = TabJira
	m.syncFocusedField()

	// Open search.
	m.Update(runeKey('/'))
	require.Equal(t, StateSearch, m.State())

	// Type "host".
	m.Update(runeKey('h'))
	m.Update(runeKey('o'))
	m.Update(runeKey('s'))
	m.Update(runeKey('t'))

	s := m.activeSection()
	require.NotNil(t, s)
	assert.Equal(t, "host", s.Search().Query())

	// Close search with esc.
	m.Update(escKey())
	assert.Equal(t, StateBrowsing, m.State())
	assert.False(t, s.IsSearchActive())
}

func TestSearchKey_BackspaceRemovesCharacter(t *testing.T) {
	m := newEditTestModel(map[string]any{
		"default_branch": "main",
		"worktrees_dir":  "worktrees",
	})

	m.activeTab = TabJira
	m.syncFocusedField()

	// Open search and type.
	m.Update(runeKey('/'))
	m.Update(runeKey('a'))
	m.Update(runeKey('b'))

	s := m.activeSection()
	assert.Equal(t, "ab", s.Search().Query())

	// Backspace removes last character.
	m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	assert.Equal(t, "a", s.Search().Query())
}

func TestSearchKey_NavigateFilteredResults(t *testing.T) {
	m := newEditTestModel(map[string]any{
		"default_branch": "main",
		"worktrees_dir":  "worktrees",
	})

	m.activeTab = TabJira
	m.syncFocusedField()

	// Open search.
	m.Update(runeKey('/'))

	// Navigate down/up while in search state.
	m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m.Update(tea.KeyMsg{Type: tea.KeyUp})

	// Should still be in search state.
	assert.Equal(t, StateSearch, m.State())
}

func TestFieldRowsInitializedOnConstruction(t *testing.T) {
	m := newEditTestModel(map[string]any{
		"default_branch": "main",
		"worktrees_dir":  "worktrees",
	})

	// General tab should have 2 field rows (generalFields has 2 fields).
	assert.Len(t, m.fieldRows[TabGeneral], len(generalFields))

	// JIRA tab should have field rows matching jiraFields count.
	assert.Len(t, m.fieldRows[TabJira], len(jiraFields))

	// File Copy tab should have field rows matching fileCopyAutoFields count.
	assert.Len(t, m.fieldRows[TabFileCopy], len(fileCopyAutoFields))

	// Worktrees tab has no fields.
	assert.Empty(t, m.fieldRows[TabWorktrees])
}

func TestFieldRowsPopulatedFromAccessor(t *testing.T) {
	m := newEditTestModel(map[string]any{
		"default_branch": "main",
		"worktrees_dir":  "worktrees",
	})

	fr := m.fieldRows[TabGeneral][0]
	assert.Equal(t, "main", fr.Value())
	assert.Equal(t, "default_branch", fr.Meta().Key)
}

func TestEditWithAccessor_WritesBack(t *testing.T) {
	accessor := &editTestAccessor{values: map[string]any{
		"default_branch": "main",
		"worktrees_dir":  "worktrees",
	}}
	dt := NewDirtyTracker(accessor.values)
	m := NewConfigModel(
		WithAccessor(accessor),
		WithDirtyTracker(dt),
	)
	m.width = 80
	m.height = 24
	m.InitSections()

	// Enter editing, change value, confirm.
	m.Update(runeKey('e'))
	fr := m.activeFieldRow()
	require.NotNil(t, fr)
	fr.input.SetValue("develop")
	m.Update(enterKey())

	// Accessor should have the updated value.
	assert.Equal(t, "develop", accessor.values["default_branch"])
}

package config

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// handleEdit processes the `e` key press on a focused row.
// For bool fields it toggles immediately; for string/int/sensitive fields
// it enters inline editing mode.
func (m *ConfigModel) handleEdit() (tea.Model, tea.Cmd) {
	s := m.activeSection()
	if s == nil {
		return m, nil
	}

	row := s.FocusedRow()
	if row.Kind != RowField {
		// RowEntry editing is handled by overlays (future task).
		return m, nil
	}

	if row.FieldIndex < 0 || row.FieldIndex >= len(m.fieldRows[m.activeTab]) {
		return m, nil
	}

	fr := m.fieldRows[m.activeTab][row.FieldIndex]

	switch fr.Meta().Type {
	case Bool:
		return m.handleBoolToggle(fr, row.FieldIndex)
	case StringList, ObjectList:
		// List/object editing handled by overlays (future task).
		return m, nil
	default:
		// String, Int, SensitiveString: enter inline editing.
		cmd := fr.EnterEditing()
		fr.SetWidth(m.width)
		fr.SetLabelWidth(s.labelWidth())
		m.state = StateEditing
		return m, cmd
	}
}

// handleBoolToggle toggles a boolean field, writes the value back through
// the accessor, and updates dirty tracking and section display.
func (m *ConfigModel) handleBoolToggle(fr *FieldRow, fieldIdx int) (tea.Model, tea.Cmd) {
	newVal, err := fr.ToggleBool()
	if err != nil {
		return m, nil
	}

	fieldKey := fr.Meta().Key
	m.setAccessorValue(fieldKey, newVal)
	m.dirty.Set(fieldKey, newVal)
	fr.SetDirty(m.dirty.IsKeyDirty(fieldKey))
	m.activeSection().SetFieldValue(fieldIdx, formatFieldValue(newVal))

	return m, nil
}

// activeFieldRow returns the FieldRow currently being edited.
// Returns nil if no valid field row is focused.
func (m *ConfigModel) activeFieldRow() *FieldRow {
	s := m.activeSection()
	if s == nil {
		return nil
	}

	row := s.FocusedRow()
	if row.Kind != RowField {
		return nil
	}

	if row.FieldIndex < 0 || row.FieldIndex >= len(m.fieldRows[m.activeTab]) {
		return nil
	}

	return m.fieldRows[m.activeTab][row.FieldIndex]
}

// commitEdit validates the edited value, writes it back through the accessor,
// and updates dirty tracking and section display. Returns true on success.
func (m *ConfigModel) commitEdit() bool {
	fr := m.activeFieldRow()
	if fr == nil {
		return false
	}

	newVal, err := fr.CommitEditing()
	if err != nil {
		// Validation failed -- stay in editing, error shown inline.
		return false
	}

	fieldKey := fr.Meta().Key
	m.setAccessorValue(fieldKey, newVal)
	m.dirty.Set(fieldKey, newVal)
	fr.SetDirty(m.dirty.IsKeyDirty(fieldKey))

	// Update section display.
	s := m.activeSection()
	if s != nil {
		row := s.FocusedRow()
		if row.Kind == RowField && row.FieldIndex >= 0 {
			s.SetFieldValue(row.FieldIndex, formatFieldValue(newVal))
		}
	}

	return true
}

// cancelEdit cancels the current inline edit and returns to browsing.
func (m *ConfigModel) cancelEdit() {
	fr := m.activeFieldRow()
	if fr != nil {
		fr.CancelEditing()
	}
}

// resetFieldRows updates field rows and section display after a single-field
// reset. The dirty tracker has already been updated; this syncs the FieldRow
// value, dirty flag, and section display value.
func (m *ConfigModel) resetFieldRows(fieldKey string) {
	origVal := m.dirty.GetOriginal(fieldKey)
	m.setAccessorValue(fieldKey, origVal)

	// Find and update the matching field row across all tabs.
	for tab := range tabCount {
		for i, fr := range m.fieldRows[tab] {
			if fr.Meta().Key != fieldKey {
				continue
			}
			fr.SetValue(origVal)
			fr.SetDirty(false)
			if m.sections[tab] != nil {
				m.sections[tab].SetFieldValue(i, formatFieldValue(origVal))
			}
			return
		}
	}
}

// resetAllFieldRows updates all field rows and section displays after a
// reset-all operation. The dirty tracker has already been reset; this syncs
// each FieldRow value, dirty flag, and section display value.
func (m *ConfigModel) resetAllFieldRows() {
	for tab := range tabCount {
		for i, fr := range m.fieldRows[tab] {
			fieldKey := fr.Meta().Key
			origVal := m.dirty.GetOriginal(fieldKey)

			fr.SetValue(origVal)
			fr.SetDirty(false)
			m.setAccessorValue(fieldKey, origVal)

			if m.sections[tab] != nil {
				m.sections[tab].SetFieldValue(i, formatFieldValue(origVal))
			}
		}
	}
}

// setAccessorValue writes a value through the accessor if available.
// Errors are silently ignored since accessor writes during TUI interaction
// always operate on valid, already-validated keys.
func (m *ConfigModel) setAccessorValue(fieldKey string, value any) {
	if m.accessor == nil {
		return
	}
	//nolint:errcheck // TUI accessor writes are best-effort; invalid keys
	// would indicate a programming error, not a user error.
	m.accessor.SetValue(fieldKey, value)
}

// handleSearchKey processes key presses in search state.
// Runes are added to the search query, backspace removes, esc closes,
// and up/down navigate the filtered results.
func (m *ConfigModel) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	s := m.activeSection()
	if s == nil {
		m.state = StateBrowsing
		return m, nil
	}

	switch {
	case key.Matches(msg, m.searchKeys.Close):
		s.CloseSearch()
		m.state = StateBrowsing
		m.syncFocusedField()
		return m, nil

	case msg.Type == tea.KeyBackspace:
		s.SearchHandleBackspace()
		m.syncFocusedField()
		return m, nil

	case key.Matches(msg, m.browsingKeys.Down):
		s.MoveFocusDown()
		m.syncFocusedField()
		return m, nil

	case key.Matches(msg, m.browsingKeys.Up):
		s.MoveFocusUp()
		m.syncFocusedField()
		return m, nil

	case msg.Type == tea.KeyRunes:
		for _, r := range msg.Runes {
			s.SearchHandleRune(r)
		}
		m.syncFocusedField()
		return m, nil
	}

	return m, nil
}

package config

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// --- Overlay key routing ---.

// handleOverlayKey routes key messages to the active overlay.
func (m *ConfigModel) handleOverlayKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.activeOverlay { //nolint:exhaustive // overlayNone falls through to browsing
	case overlayList:
		return m.handleListOverlayKey(msg)
	case overlayRule:
		return m.handleRuleOverlayKey(msg)
	case overlayWorktree:
		return m.handleWorktreeOverlayKey(msg)
	}
	// No active overlay -- fall back to browsing.
	m.state = StateBrowsing
	return m, nil
}

// handleListOverlayKey forwards keys to the ListOverlay and handles its result.
func (m *ConfigModel) handleListOverlayKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.listOverlay == nil {
		m.closeOverlay()
		return m, nil
	}

	result, cmd := m.listOverlay.Update(msg)
	if result == nil {
		return m, cmd
	}

	m.handleListOverlayResult(result)
	m.closeOverlay()
	return m, cmd
}

// handleRuleOverlayKey forwards keys to the RuleOverlay and handles its result.
func (m *ConfigModel) handleRuleOverlayKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.ruleOverlay == nil {
		m.closeOverlay()
		return m, nil
	}

	result, cmd := m.ruleOverlay.HandleKey(msg)
	if result == nil {
		return m, cmd
	}

	m.handleRuleOverlayResult(result)
	m.closeOverlay()
	return m, cmd
}

// handleWorktreeOverlayKey forwards keys to the WorktreeOverlay and handles closure.
func (m *ConfigModel) handleWorktreeOverlayKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.worktreeOverlay == nil {
		m.closeOverlay()
		return m, nil
	}

	cmd, closed := m.worktreeOverlay.HandleKey(msg)
	if !closed {
		return m, cmd
	}

	m.handleWorktreeOverlayClose()
	m.closeOverlay()
	return m, cmd
}

// closeOverlay resets all overlay state and returns to browsing.
func (m *ConfigModel) closeOverlay() {
	m.listOverlay = nil
	m.ruleOverlay = nil
	m.worktreeOverlay = nil
	m.activeOverlay = overlayNone
	m.overlayFieldKey = ""
	m.overlayEntryIdx = 0
	m.state = StateBrowsing
}

// --- Open overlays from handleEdit ---.

// openListOverlay opens a ListOverlay for the given StringList field.
func (m *ConfigModel) openListOverlay(fr *FieldRow) (tea.Model, tea.Cmd) {
	currentVal := fr.Value()
	items, _ := currentVal.([]string)
	if items == nil {
		items = []string{}
	}

	title := tabLabels[m.activeTab] + " > " + fr.Meta().Label
	m.listOverlay = NewListOverlay(title, items, m.theme)
	m.listOverlay.SetSize(m.width, m.height)
	m.activeOverlay = overlayList
	m.overlayFieldKey = fr.Meta().Key
	m.state = StateOverlay
	return m, nil
}

// openRuleOverlay opens a RuleOverlay for the file copy rule at entryIdx.
func (m *ConfigModel) openRuleOverlay(entryIdx int) (tea.Model, tea.Cmd) {
	source, files := m.getFileCopyRule(entryIdx)

	m.ruleOverlay = NewRuleOverlay(source, files,
		WithRuleTheme(m.theme), WithRuleWidth(m.overlayWidth()),
		WithRuleWorktreeNames(m.worktreeNames()))
	m.activeOverlay = overlayRule
	m.overlayEntryIdx = entryIdx
	m.state = StateOverlay
	return m, nil
}

// openNewRuleOverlay opens a RuleOverlay for creating a new file copy rule.
func (m *ConfigModel) openNewRuleOverlay() (tea.Model, tea.Cmd) {
	m.ruleOverlay = NewRuleOverlayForNew(
		WithRuleTheme(m.theme), WithRuleWidth(m.overlayWidth()),
		WithRuleWorktreeNames(m.worktreeNames()))
	m.activeOverlay = overlayRule
	m.overlayEntryIdx = -1
	m.state = StateOverlay
	return m, nil
}

// openWorktreeOverlay opens a WorktreeOverlay for the worktree at entryIdx.
func (m *ConfigModel) openWorktreeOverlay(entryIdx int) (tea.Model, tea.Cmd) {
	name, values := m.getWorktreeEntry(entryIdx)
	existingNames := m.worktreeNames()

	m.worktreeOverlay = NewWorktreeOverlay(name, values,
		WithWorktreeTheme(m.theme), WithWorktreeWidth(m.overlayWidth()),
		WithExistingNames(removeString(existingNames, name)),
		WithWorktreeNames(existingNames))
	m.activeOverlay = overlayWorktree
	m.overlayEntryIdx = entryIdx
	m.state = StateOverlay
	return m, textinput.Blink
}

// openNewWorktreeOverlay opens a WorktreeOverlay for creating a new worktree entry.
func (m *ConfigModel) openNewWorktreeOverlay() (tea.Model, tea.Cmd) {
	existingNames := m.worktreeNames()

	m.worktreeOverlay = NewWorktreeOverlayForNew(
		WithWorktreeTheme(m.theme), WithWorktreeWidth(m.overlayWidth()),
		WithExistingNames(existingNames),
		WithWorktreeNames(existingNames))
	m.activeOverlay = overlayWorktree
	m.overlayEntryIdx = -1
	m.state = StateOverlay
	return m, textinput.Blink
}

// overlayWidth returns the width to use for overlay rendering.
func (m *ConfigModel) overlayWidth() int {
	return max(m.width*2/3, 40)
}

// --- Overlay result handlers ---.

// handleListOverlayResult applies the list overlay result to the config.
func (m *ConfigModel) handleListOverlayResult(result *ListOverlayResultMsg) {
	if !result.Committed {
		return
	}

	fieldKey := m.overlayFieldKey
	if fieldKey == "" {
		return
	}

	m.setAccessorValue(fieldKey, result.Items)
	m.dirty.Set(fieldKey, result.Items)

	// Update the FieldRow value and dirty state.
	for _, fr := range m.fieldRows[m.activeTab] {
		if fr.Meta().Key != fieldKey {
			continue
		}
		fr.SetValue(result.Items)
		fr.SetDirty(m.dirty.IsKeyDirty(fieldKey))

		// Update section display.
		s := m.activeSection()
		if s == nil {
			break
		}
		for _, r := range s.Rows() {
			if r.Kind == RowField && r.FieldIndex >= 0 &&
				r.FieldIndex < len(m.fieldRows[m.activeTab]) &&
				m.fieldRows[m.activeTab][r.FieldIndex].Meta().Key == fieldKey {
				s.SetFieldValue(r.FieldIndex, formatFieldValue(result.Items))
				break
			}
		}
		break
	}
}

// handleRuleOverlayResult applies the rule overlay result to the config.
func (m *ConfigModel) handleRuleOverlayResult(result *RuleOverlayResultMsg) {
	if !result.Committed {
		return
	}

	if m.accessor == nil {
		return
	}

	if m.overlayEntryIdx < 0 {
		// New rule: append.
		m.appendFileCopyRule(result.SourceWorktree, result.Files)
	} else {
		// Existing rule: update in place.
		m.updateFileCopyRule(m.overlayEntryIdx, result.SourceWorktree, result.Files)
	}

	m.rebuildFileCopyEntries()
	m.dirty.Set("file_copy.rules", m.accessor.GetValue("file_copy.rules"))
}

// handleWorktreeOverlayClose applies the worktree overlay result to the config.
func (m *ConfigModel) handleWorktreeOverlayClose() {
	if m.worktreeOverlay == nil || !m.worktreeOverlay.IsConfirmed() {
		return
	}

	if m.accessor == nil {
		return
	}

	name := m.worktreeOverlay.Name()
	values := m.worktreeOverlay.Values()

	if m.overlayEntryIdx < 0 {
		// New worktree: add to map.
		m.addWorktreeEntry(name, values)
	} else {
		// Existing worktree: update.
		oldName, _ := m.getWorktreeEntry(m.overlayEntryIdx)
		m.updateWorktreeEntry(oldName, name, values)
	}

	m.rebuildWorktreeEntries()
	m.dirty.Set("worktrees", m.accessor.GetValue("worktrees"))
}

// --- Add / Delete entry handlers ---.

// handleAddEntry opens the appropriate overlay for adding a new entry.
func (m *ConfigModel) handleAddEntry() (tea.Model, tea.Cmd) {
	switch m.activeTab { //nolint:exhaustive // only entry-list tabs support add
	case TabFileCopy:
		return m.openNewRuleOverlay()
	case TabWorktrees:
		return m.openNewWorktreeOverlay()
	}
	return m, nil
}

// handleDeleteEntry initiates deletion of the focused entry row.
func (m *ConfigModel) handleDeleteEntry() (tea.Model, tea.Cmd) {
	s := m.activeSection()
	if s == nil {
		return m, nil
	}

	row := s.FocusedRow()
	if row.Kind != RowEntry || row.EntryIndex < 0 {
		return m, nil
	}

	if m.activeTab != TabFileCopy && m.activeTab != TabWorktrees {
		return m, nil
	}

	m.overlayEntryIdx = row.EntryIndex
	m.state = StateDeleteConfirm
	return m, nil
}

// handleDeleteConfirmKey processes y/n input for entry deletion.
func (m *ConfigModel) handleDeleteConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.confirmKeys.Confirm):
		m.executeDeleteEntry()
		m.state = StateBrowsing
		return m, nil

	case key.Matches(msg, m.confirmKeys.Deny),
		key.Matches(msg, m.confirmKeys.Cancel):
		m.overlayEntryIdx = 0
		m.state = StateBrowsing
		return m, nil
	}
	return m, nil
}

// executeDeleteEntry removes the entry at overlayEntryIdx from the config.
func (m *ConfigModel) executeDeleteEntry() {
	if m.accessor == nil {
		return
	}

	switch m.activeTab { //nolint:exhaustive // only entry-list tabs support delete
	case TabFileCopy:
		m.deleteFileCopyRule(m.overlayEntryIdx)
		m.rebuildFileCopyEntries()
		m.dirty.Set("file_copy.rules", m.accessor.GetValue("file_copy.rules"))

	case TabWorktrees:
		name, _ := m.getWorktreeEntry(m.overlayEntryIdx)
		m.deleteWorktreeEntry(name)
		m.rebuildWorktreeEntries()
		m.dirty.Set("worktrees", m.accessor.GetValue("worktrees"))
	}

	m.overlayEntryIdx = 0
}

// --- Overlay view rendering ---.

// viewOverlay renders the active overlay.
func (m *ConfigModel) viewOverlay() string {
	switch m.activeOverlay { //nolint:exhaustive // overlayNone returns empty
	case overlayList:
		if m.listOverlay != nil {
			return m.listOverlay.View(m.width, m.height)
		}
	case overlayRule:
		if m.ruleOverlay != nil {
			return m.viewRuleOverlayCentered()
		}
	case overlayWorktree:
		if m.worktreeOverlay != nil {
			return m.viewWorktreeOverlayCentered()
		}
	}
	return ""
}

// viewRuleOverlayCentered renders the rule overlay centered on screen.
func (m *ConfigModel) viewRuleOverlayCentered() string {
	m.ruleOverlay.SetWidth(m.overlayWidth())
	box := m.ruleOverlay.View()
	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(box)
}

// viewWorktreeOverlayCentered renders the worktree overlay centered on screen.
func (m *ConfigModel) viewWorktreeOverlayCentered() string {
	m.worktreeOverlay.SetWidth(m.overlayWidth())
	box := m.worktreeOverlay.View()
	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(box)
}

// viewDeleteConfirm renders the entry deletion confirmation overlay.
func (m *ConfigModel) viewDeleteConfirm() string {
	var entryName string
	switch m.activeTab {
	case TabFileCopy:
		source, files := m.getFileCopyRule(m.overlayEntryIdx)
		entryName = source + ": " + strings.Join(files, ", ")
	case TabWorktrees:
		name, _ := m.getWorktreeEntry(m.overlayEntryIdx)
		entryName = name
	default:
		entryName = fmt.Sprintf("entry %d", m.overlayEntryIdx+1)
	}

	if len(entryName) > 40 {
		entryName = entryName[:37] + "..."
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.theme.ErrorAccent)

	bodyStyle := lipgloss.NewStyle().
		Foreground(m.theme.Muted)

	hintStyle := lipgloss.NewStyle().
		Foreground(m.theme.Accent)

	content := titleStyle.Render("Delete Entry") + "\n\n" +
		bodyStyle.Render(fmt.Sprintf("Delete %q?", entryName)) + "\n\n" +
		hintStyle.Render("y") + bodyStyle.Render(" delete  ") +
		hintStyle.Render("n") + bodyStyle.Render("/") +
		hintStyle.Render("esc") + bodyStyle.Render(" cancel")

	innerWidth := max(m.width-4, 30)
	boxStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.ErrorAccent).
		Padding(1, 2).
		Width(innerWidth)

	box := boxStyle.Render(content)

	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(box)
}

// --- Config data accessors for overlays ---.

// getFileCopyRule extracts the source worktree and files from the rule at idx.
func (m *ConfigModel) getFileCopyRule(idx int) (string, []string) {
	if m.accessor == nil {
		return "", nil
	}
	val := m.accessor.GetValue("file_copy.rules")
	if val == nil {
		return "", nil
	}

	rv := reflect.ValueOf(val)
	if rv.Kind() != reflect.Slice || idx < 0 || idx >= rv.Len() {
		return "", nil
	}

	item := rv.Index(idx)
	if item.Kind() == reflect.Ptr {
		item = item.Elem()
	}
	source := reflectStringField(item, "SourceWorktree")
	files := reflectStringSliceField(item, "Files")
	return source, files
}

// appendFileCopyRule adds a new rule to the file_copy.rules slice.
func (m *ConfigModel) appendFileCopyRule(source string, files []string) {
	val := m.accessor.GetValue("file_copy.rules")
	rv := reflect.ValueOf(val)

	if !rv.IsValid() || rv.Kind() != reflect.Slice {
		return
	}

	// Derive the element type from the slice's type — works even on empty slices.
	elemType := rv.Type().Elem()
	newRule := reflect.New(elemType).Elem()
	setReflectStringField(newRule, "SourceWorktree", source)
	setReflectStringSliceField(newRule, "Files", files)

	newSlice := reflect.Append(rv, newRule)
	//nolint:errcheck // TUI accessor writes are best-effort on validated keys
	m.accessor.SetValue("file_copy.rules", newSlice.Interface())
}

// updateFileCopyRule updates the rule at idx with new source and files.
func (m *ConfigModel) updateFileCopyRule(idx int, source string, files []string) {
	val := m.accessor.GetValue("file_copy.rules")
	rv := reflect.ValueOf(val)
	if !rv.IsValid() || rv.Kind() != reflect.Slice || idx < 0 || idx >= rv.Len() {
		return
	}

	item := rv.Index(idx)
	if item.Kind() == reflect.Ptr {
		item = item.Elem()
	}
	setReflectStringField(item, "SourceWorktree", source)
	setReflectStringSliceField(item, "Files", files)

	//nolint:errcheck // TUI accessor writes are best-effort on validated keys
	m.accessor.SetValue("file_copy.rules", rv.Interface())
}

// deleteFileCopyRule removes the rule at idx from the slice.
func (m *ConfigModel) deleteFileCopyRule(idx int) {
	val := m.accessor.GetValue("file_copy.rules")
	rv := reflect.ValueOf(val)
	if !rv.IsValid() || rv.Kind() != reflect.Slice || idx < 0 || idx >= rv.Len() {
		return
	}

	newSlice := reflect.AppendSlice(rv.Slice(0, idx), rv.Slice(idx+1, rv.Len()))
	//nolint:errcheck // TUI accessor writes are best-effort on validated keys
	m.accessor.SetValue("file_copy.rules", newSlice.Interface())
}

// rebuildFileCopyEntries rebuilds the File Copy tab's entry list from the accessor.
func (m *ConfigModel) rebuildFileCopyEntries() {
	if m.accessor == nil {
		return
	}
	entries := formatFileCopyRules(m.accessor.GetValue("file_copy.rules"))
	s := m.sections[TabFileCopy]
	if s != nil {
		s.UpdateEntries(entries)
	}
}

// getWorktreeEntry extracts the name and values for the worktree at idx.
func (m *ConfigModel) getWorktreeEntry(idx int) (string, [3]string) {
	if m.accessor == nil {
		return "", [3]string{}
	}
	val := m.accessor.GetValue("worktrees")
	if val == nil {
		return "", [3]string{}
	}

	rv := reflect.ValueOf(val)
	if rv.Kind() != reflect.Map {
		return "", [3]string{}
	}

	// Sort keys to get stable ordering matching what formatWorktreeEntries produces.
	keys := rv.MapKeys()
	names := make([]string, 0, len(keys))
	for _, k := range keys {
		names = append(names, k.String())
	}
	sort.Strings(names)

	if idx < 0 || idx >= len(names) {
		return "", [3]string{}
	}

	name := names[idx]
	entry := rv.MapIndex(reflect.ValueOf(name))
	if !entry.IsValid() {
		return name, [3]string{}
	}
	if entry.Kind() == reflect.Ptr {
		entry = entry.Elem()
	}

	return name, [3]string{
		reflectStringField(entry, "Branch"),
		reflectStringField(entry, "MergeInto"),
		reflectStringField(entry, "Description"),
	}
}

// worktreeNames returns sorted list of current worktree names.
func (m *ConfigModel) worktreeNames() []string {
	if m.accessor == nil {
		return nil
	}
	val := m.accessor.GetValue("worktrees")
	if val == nil {
		return nil
	}

	rv := reflect.ValueOf(val)
	if rv.Kind() != reflect.Map {
		return nil
	}

	keys := rv.MapKeys()
	names := make([]string, 0, len(keys))
	for _, k := range keys {
		names = append(names, k.String())
	}
	sort.Strings(names)
	return names
}

// addWorktreeEntry adds a new worktree entry to the map.
func (m *ConfigModel) addWorktreeEntry(name string, values [3]string) {
	val := m.accessor.GetValue("worktrees")
	rv := reflect.ValueOf(val)

	if !rv.IsValid() || rv.Kind() != reflect.Map {
		return
	}

	// Derive the value type from the map's type — works even on empty maps.
	entryType := rv.Type().Elem()
	newEntry := reflect.New(entryType).Elem()
	setReflectStringField(newEntry, "Branch", values[0])
	setReflectStringField(newEntry, "MergeInto", values[1])
	setReflectStringField(newEntry, "Description", values[2])

	rv.SetMapIndex(reflect.ValueOf(name), newEntry)
	//nolint:errcheck // TUI accessor writes are best-effort on validated keys
	m.accessor.SetValue("worktrees", rv.Interface())
}

// updateWorktreeEntry updates an existing worktree entry, handling renames.
func (m *ConfigModel) updateWorktreeEntry(oldName, newName string, values [3]string) {
	val := m.accessor.GetValue("worktrees")
	rv := reflect.ValueOf(val)
	if !rv.IsValid() || rv.Kind() != reflect.Map {
		return
	}

	// If renamed, delete old key.
	if oldName != newName {
		rv.SetMapIndex(reflect.ValueOf(oldName), reflect.Value{})
	}

	entry := rv.MapIndex(reflect.ValueOf(oldName))
	if !entry.IsValid() {
		// Old entry gone (renamed); create new.
		m.addWorktreeEntry(newName, values)
		return
	}

	// Update existing entry in place.
	if entry.Kind() == reflect.Ptr {
		entry = entry.Elem()
	}

	// Maps return non-addressable values; we must create a new struct.
	newEntry := reflect.New(entry.Type()).Elem()
	newEntry.Set(entry)
	setReflectStringField(newEntry, "Branch", values[0])
	setReflectStringField(newEntry, "MergeInto", values[1])
	setReflectStringField(newEntry, "Description", values[2])

	rv.SetMapIndex(reflect.ValueOf(newName), newEntry)
	//nolint:errcheck // TUI accessor writes are best-effort on validated keys
	m.accessor.SetValue("worktrees", rv.Interface())
}

// deleteWorktreeEntry removes the worktree with the given name.
func (m *ConfigModel) deleteWorktreeEntry(name string) {
	val := m.accessor.GetValue("worktrees")
	rv := reflect.ValueOf(val)
	if !rv.IsValid() || rv.Kind() != reflect.Map {
		return
	}

	rv.SetMapIndex(reflect.ValueOf(name), reflect.Value{})
	//nolint:errcheck // TUI accessor writes are best-effort on validated keys
	m.accessor.SetValue("worktrees", rv.Interface())
}

// rebuildWorktreeEntries rebuilds the Worktrees tab's entry list from the accessor.
func (m *ConfigModel) rebuildWorktreeEntries() {
	if m.accessor == nil {
		return
	}
	entries := formatWorktreeEntries(m.accessor.GetValue("worktrees"))
	s := m.sections[TabWorktrees]
	if s != nil {
		s.UpdateEntries(entries)
	}
}

// --- Reflect helpers ---.

// setReflectStringField sets a string field on a struct by name.
func setReflectStringField(v reflect.Value, name, value string) {
	if v.Kind() != reflect.Struct {
		return
	}
	f := v.FieldByName(name)
	if !f.IsValid() || f.Kind() != reflect.String || !f.CanSet() {
		return
	}
	f.SetString(value)
}

// setReflectStringSliceField sets a []string field on a struct by name.
func setReflectStringSliceField(v reflect.Value, name string, value []string) {
	if v.Kind() != reflect.Struct {
		return
	}
	f := v.FieldByName(name)
	if !f.IsValid() || f.Kind() != reflect.Slice || !f.CanSet() {
		return
	}
	f.Set(reflect.ValueOf(copyStrings(value)))
}

// removeString returns a copy of ss with the first occurrence of s removed.
func removeString(ss []string, s string) []string {
	result := make([]string, 0, len(ss))
	for _, v := range ss {
		if v != s {
			result = append(result, v)
		}
	}
	return result
}

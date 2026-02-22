package config

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Sections returns the sections array for external inspection (e.g., tests).
func (m *ConfigModel) Sections() [tabCount]*SectionModel {
	return m.sections
}

// activeSection returns the SectionModel for the currently active tab.
func (m *ConfigModel) activeSection() *SectionModel {
	return m.sections[m.activeTab]
}

// syncFocusedField updates focusedFieldType and focusedFieldKey from the
// active section's currently focused row.
func (m *ConfigModel) syncFocusedField() {
	s := m.activeSection()
	if s == nil {
		return
	}
	row := s.FocusedRow()
	if row.Kind == RowField && row.FieldIndex >= 0 && row.FieldIndex < len(s.fields) {
		f := s.fields[row.FieldIndex]
		m.focusedFieldType = f.Type
		m.focusedFieldKey = f.Key
		m.focusedFieldDesc = f.Description
	} else if row.Kind == RowEntry {
		m.focusedFieldType = ObjectList
		m.focusedFieldKey = ""
		m.focusedFieldDesc = ""
	} else {
		m.focusedFieldType = String
		m.focusedFieldKey = ""
		m.focusedFieldDesc = ""
	}
}

// initEmptySections creates section models with field metadata but without
// populating values from the accessor. This ensures the model always has
// renderable sections, even before InitSections() is called.
func (m *ConfigModel) initEmptySections() {
	vpHeight := 20
	w := 80
	if m.height > 0 {
		vpHeight = max(m.height-5, 1)
	}
	if m.width > 0 {
		w = m.width
	}

	m.sections[TabGeneral] = m.buildSection(generalFields, vpHeight, w, nil)
	m.sections[TabJira] = m.buildSection(jiraFields, vpHeight, w, nil)
	m.sections[TabFileCopy] = m.buildSection(fileCopyAutoFields, vpHeight, w, &entryListConfig{
		label:    "Rules",
		entries:  nil,
		emptyMsg: "(no rules configured — press a to add)",
	})
	m.sections[TabWorktrees] = m.buildSection(nil, vpHeight, w, &entryListConfig{
		label:    "Worktrees",
		entries:  nil,
		emptyMsg: "(no worktrees configured — press a to add)",
	})

	// Build field rows for each tab.
	m.buildFieldRows(TabGeneral, generalFields)
	m.buildFieldRows(TabJira, jiraFields)
	m.buildFieldRows(TabFileCopy, fileCopyAutoFields)
	m.buildFieldRows(TabWorktrees, nil)

	m.syncFocusedField()
}

// InitSections creates and populates a SectionModel for each tab using
// field metadata from sections.go and values from the accessor.
func (m *ConfigModel) InitSections() {
	contentHeight := max(m.height-5, 1)
	if contentHeight <= 0 {
		contentHeight = 20
	}
	w := m.width
	if w <= 0 {
		w = 80
	}

	m.sections[TabGeneral] = m.buildSection(generalFields, contentHeight, w, nil)
	m.sections[TabJira] = m.buildSection(jiraFields, contentHeight, w, nil)
	m.sections[TabFileCopy] = m.buildFileCopySection(contentHeight, w)
	m.sections[TabWorktrees] = m.buildWorktreesSection(contentHeight, w)

	// Build field rows for each tab, populating from the accessor.
	m.buildFieldRows(TabGeneral, generalFields)
	m.buildFieldRows(TabJira, jiraFields)
	m.buildFieldRows(TabFileCopy, fileCopyAutoFields)
	m.buildFieldRows(TabWorktrees, nil)

	// Sync focused field from the initial section.
	m.syncFocusedField()
}

// buildSection creates a SectionModel for a set of fields, populating values
// from the accessor.
func (m *ConfigModel) buildSection(
	fields []FieldMeta, vpHeight, width int,
	entryOpt *entryListConfig,
) *SectionModel {
	opts := []SectionOption{
		WithSectionTheme(m.theme),
		WithViewportHeight(vpHeight),
		WithWidth(width),
	}
	if entryOpt != nil {
		opts = append(opts, WithEntryList(entryOpt.label, entryOpt.entries, entryOpt.emptyMsg))
	}

	sm := NewSectionModel(fields, opts...)

	if m.accessor != nil {
		for i, f := range fields {
			val := m.accessor.GetValue(f.Key)
			sm.SetFieldValue(i, formatFieldValue(val))
		}
	}

	return sm
}

// entryListConfig holds the parameters for creating an entry list section.
type entryListConfig struct {
	label    string
	emptyMsg string
	entries  []string
}

// buildFileCopySection creates the File Copy tab section with auto-copy fields
// and a rules entry list.
func (m *ConfigModel) buildFileCopySection(vpHeight, width int) *SectionModel {
	var entries []string
	if m.accessor != nil {
		entries = formatFileCopyRules(m.accessor.GetValue("file_copy.rules"))
	}
	return m.buildSection(fileCopyAutoFields, vpHeight, width, &entryListConfig{
		label:    "Rules",
		entries:  entries,
		emptyMsg: "(no rules configured — press a to add)",
	})
}

// buildWorktreesSection creates the Worktrees tab section with an entry list
// of worktree entries.
func (m *ConfigModel) buildWorktreesSection(vpHeight, width int) *SectionModel {
	var entries []string
	if m.accessor != nil {
		entries = formatWorktreeEntries(m.accessor.GetValue("worktrees"))
	}
	// Worktrees tab has no config fields, just the entry list.
	return m.buildSection(nil, vpHeight, width, &entryListConfig{
		label:    "Worktrees",
		entries:  entries,
		emptyMsg: "(no worktrees configured — press a to add)",
	})
}

// buildFieldRows creates FieldRow instances for the given tab's fields and
// populates their values from the accessor. The field rows are stored in
// m.fieldRows[tab] parallel to the section's field metadata.
func (m *ConfigModel) buildFieldRows(tab SectionTab, fields []FieldMeta) {
	rows := make([]*FieldRow, len(fields))
	for i, f := range fields {
		fr := NewFieldRow(f, m.theme)
		if m.accessor != nil {
			val := m.accessor.GetValue(f.Key)
			fr.SetValue(val)
		}
		// Sync dirty flag from the tracker.
		fr.SetDirty(m.dirty.IsKeyDirty(f.Key))
		m.wireDynamicSuggestions(fr)
		rows[i] = fr
	}
	m.fieldRows[tab] = rows
}

// wireDynamicSuggestions installs closure-based suggestions on fields that
// benefit from async git data. The closures capture m and read the cache at
// call time, so data arriving after buildFieldRows is still picked up.
func (m *ConfigModel) wireDynamicSuggestions(fr *FieldRow) {
	switch fr.Meta().Key {
	case "default_branch":
		fr.SetSuggestions(func() []string {
			if len(m.gitBranches) > 0 {
				return m.gitBranches
			}
			return []string{"main", "master", "develop", "development"}
		})
	case "file_copy.auto.source_worktree":
		fr.SetSuggestions(func() []string {
			return append([]string{"{default}"}, m.worktreeNamesForSuggestions()...)
		})
	}
}

// --- Value formatting helpers ---.

// formatFieldValue converts a typed value from the accessor to a display string.
func formatFieldValue(value any) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	case bool:
		if v {
			return "yes"
		}
		return "no"
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case []string:
		return strings.Join(v, ", ")
	default:
		return fmt.Sprint(v)
	}
}

// formatFileCopyRules converts file copy rules from the accessor to entry
// list summary strings. The value is expected to be a slice of structs with
// SourceWorktree and Files fields ([]service.FileCopyRule), but since we
// cannot import that type here, we use reflect to extract fields.
func formatFileCopyRules(value any) []string {
	if value == nil {
		return nil
	}
	rv := reflect.ValueOf(value)
	if rv.Kind() != reflect.Slice {
		return nil
	}
	n := rv.Len()
	if n == 0 {
		return nil
	}
	entries := make([]string, 0, n)
	for i := range n {
		item := rv.Index(i)
		if item.Kind() == reflect.Ptr {
			item = item.Elem()
		}
		source := reflectStringField(item, "SourceWorktree")
		files := reflectStringSliceField(item, "Files")
		if source == "" {
			source = "?"
		}
		summary := source + ": " + strings.Join(files, ", ")
		entries = append(entries, summary)
	}
	return entries
}

// formatWorktreeEntries converts the worktrees map from the accessor to entry
// list summary strings. The value is expected to be map[string]WorktreeConfig,
// but we use reflect since we cannot import the concrete type.
func formatWorktreeEntries(value any) []string {
	if value == nil {
		return nil
	}
	rv := reflect.ValueOf(value)
	if rv.Kind() != reflect.Map {
		return nil
	}
	keys := rv.MapKeys()
	if len(keys) == 0 {
		return nil
	}
	entries := make([]string, 0, len(keys))
	for _, k := range keys {
		entries = append(entries, k.String())
	}
	sort.Strings(entries)
	return entries
}

// reflectStringField extracts a string field from a struct by name.
func reflectStringField(v reflect.Value, name string) string {
	if v.Kind() != reflect.Struct {
		return ""
	}
	f := v.FieldByName(name)
	if !f.IsValid() || f.Kind() != reflect.String {
		return ""
	}
	return f.String()
}

// reflectStringSliceField extracts a []string field from a struct by name.
func reflectStringSliceField(v reflect.Value, name string) []string {
	if v.Kind() != reflect.Struct {
		return nil
	}
	f := v.FieldByName(name)
	if !f.IsValid() || f.Kind() != reflect.Slice {
		return nil
	}
	result := make([]string, f.Len())
	for i := range f.Len() {
		result[i] = f.Index(i).String()
	}
	return result
}

// --- Async git data fetch ---.

// handleGitBranchesMsg stores fetched branch names in the cache.
func (m *ConfigModel) handleGitBranchesMsg(msg gitBranchesMsg) (tea.Model, tea.Cmd) {
	if msg.err == nil && len(msg.branches) > 0 {
		m.gitBranches = msg.branches
	}
	return m, nil
}

// handleGitWorktreesMsg stores fetched worktree names in the cache.
func (m *ConfigModel) handleGitWorktreesMsg(msg gitWorktreesMsg) (tea.Model, tea.Cmd) {
	if msg.err == nil && len(msg.names) > 0 {
		m.gitWorktreeNames = msg.names
	}
	return m, nil
}

// fetchGitBranches returns a tea.Cmd that fetches branch names asynchronously.
func (m *ConfigModel) fetchGitBranches() tea.Cmd {
	p := m.gitProvider
	return func() tea.Msg {
		branches, err := p.ListBranches()
		return gitBranchesMsg{branches: branches, err: err}
	}
}

// fetchGitWorktrees returns a tea.Cmd that fetches worktree names asynchronously.
func (m *ConfigModel) fetchGitWorktrees() tea.Cmd {
	p := m.gitProvider
	return func() tea.Msg {
		names, err := p.ListWorktreeNames()
		return gitWorktreesMsg{names: names, err: err}
	}
}

package config

import (
	"fmt"
	"gbm/pkg/tui"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// RowKind distinguishes the different row types rendered inside a section.
type RowKind int

const (
	// RowField is a focusable config field (label + value).
	RowField RowKind = iota
	// RowGroupHeader is a visual-only separator between groups of fields.
	RowGroupHeader
	// RowEntry is a focusable summary row for an entry list item
	// (file copy rule or worktree entry).
	RowEntry
	// RowEmpty is a non-focusable placeholder shown when an entry list is empty.
	RowEmpty
)

// Row represents a single renderable row in a SectionModel.
// Field rows and entry rows are focusable; group headers and empty
// placeholders are not.
type Row struct {
	// Label is the display text. For fields it is the field label, for group
	// headers it is the group name, for entries it is the summary line, and
	// for empty rows it is the placeholder message.
	Label string

	// Value is the display value for field rows (e.g., "main", "yes", "3").
	// Empty for non-field rows.
	Value string

	// Group is the group name this row belongs to. Used for {/} group jumping.
	Group string

	// FieldIndex maps a focusable row back to its index in the SectionModel's
	// fields slice. -1 for non-field rows.
	FieldIndex int

	// EntryIndex maps an entry row back to its index in the entries slice.
	// -1 for non-entry rows.
	EntryIndex int

	// Kind identifies the row type.
	Kind RowKind
}

// IsFocusable reports whether this row can receive focus.
func (r Row) IsFocusable() bool {
	return r.Kind == RowField || r.Kind == RowEntry
}

// SectionModel is the scrollable field list for a single tab in the Config TUI.
// It manages layout rows (fields + group headers + entry list), focus tracking,
// viewport scrolling, and renders the visible portion of the section.
//
// When a search filter is active, the section operates on a filtered subset
// of rows. Navigation, rendering, and position indicators all reflect the
// filtered view. The full row list is preserved for restoration when search
// is closed.
type SectionModel struct {
	theme          *tui.Theme
	search         *SearchFilter
	emptyState     *EmptyState
	entryLabel     string
	emptyMessage   string
	fields         []FieldMeta
	filteredRows   []Row
	entries        []string
	rows           []Row
	focusIndex     int
	scrollOffset   int
	viewportHeight int
	width          int
	hasEntryList   bool
}

// NewSectionModel creates a SectionModel for the given fields.
// Options can configure entry lists, themes, and viewport size.
func NewSectionModel(fields []FieldMeta, opts ...SectionOption) *SectionModel {
	s := &SectionModel{
		theme:          tui.DefaultTheme(),
		fields:         fields,
		viewportHeight: 20,
	}
	for _, opt := range opts {
		opt(s)
	}
	s.search = NewSearchFilter(s.theme)
	s.buildRows()
	s.focusFirst()
	return s
}

// SectionOption configures a SectionModel during construction.
type SectionOption func(*SectionModel)

// WithSectionTheme sets the theme.
func WithSectionTheme(theme *tui.Theme) SectionOption {
	return func(s *SectionModel) {
		if theme != nil {
			s.theme = theme
		}
	}
}

// WithEntryList configures an entry list section appended after the fields.
// label is the group header (e.g., "Rules"); if empty, no header is rendered.
// entries are the summary lines, and emptyMsg is shown when entries is empty.
func WithEntryList(label string, entries []string, emptyMsg string) SectionOption {
	return func(s *SectionModel) {
		s.entryLabel = label
		s.entries = entries
		s.emptyMessage = emptyMsg
		s.hasEntryList = true
	}
}

// WithEmptyState configures the section's empty state for "not configured" sections.
func WithEmptyState(es *EmptyState) SectionOption {
	return func(s *SectionModel) {
		s.emptyState = es
	}
}

// WithViewportHeight sets the viewport height (number of visible rows).
func WithViewportHeight(h int) SectionOption {
	return func(s *SectionModel) {
		if h > 0 {
			s.viewportHeight = h
		}
	}
}

// WithWidth sets the rendering width.
func WithWidth(w int) SectionOption {
	return func(s *SectionModel) {
		if w > 0 {
			s.width = w
		}
	}
}

// buildRows constructs the flattened row list from fields and entries.
// Group headers are inserted when consecutive fields have different Group values.
func (s *SectionModel) buildRows() {
	s.rows = nil
	currentGroup := ""

	for i, f := range s.fields {
		if f.Group != "" && f.Group != currentGroup {
			s.rows = append(s.rows, Row{
				Label:      f.Group,
				Group:      f.Group,
				Kind:       RowGroupHeader,
				FieldIndex: -1,
				EntryIndex: -1,
			})
			currentGroup = f.Group
		}

		s.rows = append(s.rows, Row{
			Label:      f.Label,
			Value:      "",
			Group:      f.Group,
			Kind:       RowField,
			FieldIndex: i,
			EntryIndex: -1,
		})
	}

	if !s.hasEntryList {
		return
	}

	// Append entry list group header if label is non-empty.
	if s.entryLabel != "" {
		s.rows = append(s.rows, Row{
			Label:      s.entryLabel,
			Group:      s.entryLabel,
			Kind:       RowGroupHeader,
			FieldIndex: -1,
			EntryIndex: -1,
		})
	}

	entryGroup := s.entryLabel

	if len(s.entries) == 0 {
		s.rows = append(s.rows, Row{
			Label:      s.emptyMessage,
			Group:      entryGroup,
			Kind:       RowEmpty,
			FieldIndex: -1,
			EntryIndex: -1,
		})
	} else {
		for i, entry := range s.entries {
			s.rows = append(s.rows, Row{
				Label:      entry,
				Group:      entryGroup,
				Kind:       RowEntry,
				FieldIndex: -1,
				EntryIndex: i,
			})
		}
	}
}

// visibleRows returns the row list currently in effect. When search is active
// and has a non-empty query, this returns the filtered subset; otherwise
// it returns the full row list.
func (s *SectionModel) visibleRows() []Row {
	if s.search != nil && s.search.IsActive() && s.search.Query() != "" {
		return s.filteredRows
	}
	return s.rows
}

// applyFilter rebuilds the filtered row list from the current search query,
// resets focus to the first matching focusable row, and resets scroll.
func (s *SectionModel) applyFilter() {
	if s.search == nil {
		return
	}
	s.filteredRows = s.search.FilterRows(s.rows)
	s.scrollOffset = 0
	s.focusIndex = 0
	rows := s.visibleRows()
	for i, r := range rows {
		if r.IsFocusable() {
			s.focusIndex = i
			return
		}
	}
}

// focusFirst sets focusIndex to the first focusable row.
// If no focusable row exists, focusIndex is set to 0.
func (s *SectionModel) focusFirst() {
	rows := s.visibleRows()
	for i, r := range rows {
		if r.IsFocusable() {
			s.focusIndex = i
			return
		}
	}
	s.focusIndex = 0
}

// FocusIndex returns the current focus index.
func (s *SectionModel) FocusIndex() int {
	return s.focusIndex
}

// ScrollOffset returns the current scroll offset.
func (s *SectionModel) ScrollOffset() int {
	return s.scrollOffset
}

// Rows returns the currently visible row list.
// When search is active, this returns filtered rows; otherwise the full list.
func (s *SectionModel) Rows() []Row {
	return s.visibleRows()
}

// AllRows returns the full unfiltered row list regardless of search state.
func (s *SectionModel) AllRows() []Row {
	return s.rows
}

// FocusedRow returns the currently focused row.
// Returns a zero-value Row if the section is empty.
func (s *SectionModel) FocusedRow() Row {
	rows := s.visibleRows()
	if len(rows) == 0 {
		return Row{FieldIndex: -1, EntryIndex: -1}
	}
	if s.focusIndex >= len(rows) {
		return Row{FieldIndex: -1, EntryIndex: -1}
	}
	return rows[s.focusIndex]
}

// FieldCount returns the number of focusable rows (fields + entries)
// in the currently visible row list.
func (s *SectionModel) FieldCount() int {
	count := 0
	for _, r := range s.visibleRows() {
		if r.IsFocusable() {
			count++
		}
	}
	return count
}

// FocusPosition returns the 1-based position of the focused field among
// all focusable rows in the visible list. Returns 0 if no focusable rows exist.
func (s *SectionModel) FocusPosition() int {
	pos := 0
	for i, r := range s.visibleRows() {
		if r.IsFocusable() {
			pos++
		}
		if i == s.focusIndex {
			return pos
		}
	}
	return 0
}

// --- Navigation ---.

// MoveFocusDown moves focus to the next focusable row, wrapping to the first
// if at the end. Updates scroll offset to keep the focused row visible.
func (s *SectionModel) MoveFocusDown() {
	rows := s.visibleRows()
	if len(rows) == 0 {
		return
	}

	start := s.focusIndex
	for i := 1; i < len(rows); i++ {
		idx := (start + i) % len(rows)
		if rows[idx].IsFocusable() {
			s.focusIndex = idx
			s.ensureVisible()
			return
		}
	}
}

// MoveFocusUp moves focus to the previous focusable row, wrapping to the last
// if at the beginning. Updates scroll offset to keep the focused row visible.
func (s *SectionModel) MoveFocusUp() {
	rows := s.visibleRows()
	if len(rows) == 0 {
		return
	}

	start := s.focusIndex
	n := len(rows)
	for i := 1; i < n; i++ {
		idx := (start - i + n) % n
		if rows[idx].IsFocusable() {
			s.focusIndex = idx
			s.ensureVisible()
			return
		}
	}
}

// JumpToFirst moves focus to the first focusable row and scrolls to the top.
func (s *SectionModel) JumpToFirst() {
	for i, r := range s.visibleRows() {
		if r.IsFocusable() {
			s.focusIndex = i
			s.scrollOffset = 0
			return
		}
	}
}

// JumpToLast moves focus to the last focusable row.
func (s *SectionModel) JumpToLast() {
	rows := s.visibleRows()
	for i := len(rows) - 1; i >= 0; i-- {
		if rows[i].IsFocusable() {
			s.focusIndex = i
			s.ensureVisible()
			return
		}
	}
}

// JumpToNextGroup moves focus to the first focusable row in the next group.
// If already in the last group, wraps to the first group.
func (s *SectionModel) JumpToNextGroup() {
	rows := s.visibleRows()
	if len(rows) == 0 {
		return
	}

	currentGroup := rows[s.focusIndex].Group
	n := len(rows)

	for i := s.focusIndex + 1; i < s.focusIndex+n; i++ {
		idx := i % n
		r := rows[idx]
		if r.Group != currentGroup {
			targetGroup := r.Group
			for j := idx; j < idx+n; j++ {
				jIdx := j % n
				if rows[jIdx].Group == targetGroup && rows[jIdx].IsFocusable() {
					s.focusIndex = jIdx
					s.ensureVisible()
					return
				}
				if rows[jIdx].Group != targetGroup && jIdx != idx {
					break
				}
			}
			currentGroup = targetGroup
		}
	}
}

// JumpToPrevGroup moves focus to the first focusable row in the previous group.
// If already in the first group, wraps to the last group.
func (s *SectionModel) JumpToPrevGroup() {
	rows := s.visibleRows()
	if len(rows) == 0 {
		return
	}

	currentGroup := rows[s.focusIndex].Group
	n := len(rows)

	for i := s.focusIndex - 1 + n; i > s.focusIndex; i-- {
		idx := i % n
		r := rows[idx]
		if r.Group != currentGroup {
			targetGroup := r.Group

			// Find the start of this group by scanning backward.
			groupStart := idx
			for {
				prev := (groupStart - 1 + n) % n
				if rows[prev].Group != targetGroup {
					break
				}
				groupStart = prev
				if groupStart == idx {
					break
				}
			}

			// Find the first focusable row in the target group.
			for j := groupStart; ; j = (j + 1) % n {
				if rows[j].Group == targetGroup && rows[j].IsFocusable() {
					s.focusIndex = j
					s.ensureVisible()
					return
				}
				if rows[j].Group != targetGroup {
					break
				}
			}
			currentGroup = targetGroup
		}
	}
}

// ensureVisible adjusts scrollOffset so that focusIndex is within the viewport.
func (s *SectionModel) ensureVisible() {
	if s.viewportHeight <= 0 {
		return
	}

	if s.focusIndex < s.scrollOffset {
		s.scrollOffset = s.focusIndex
	}

	if s.focusIndex >= s.scrollOffset+s.viewportHeight {
		s.scrollOffset = s.focusIndex - s.viewportHeight + 1
	}

	maxOffset := max(len(s.visibleRows())-s.viewportHeight, 0)
	if s.scrollOffset > maxOffset {
		s.scrollOffset = maxOffset
	}
	if s.scrollOffset < 0 {
		s.scrollOffset = 0
	}
}

// SetViewportHeight updates the viewport height and re-clamps the scroll offset.
func (s *SectionModel) SetViewportHeight(h int) {
	if h > 0 {
		s.viewportHeight = h
		s.ensureVisible()
	}
}

// SetWidth updates the rendering width.
func (s *SectionModel) SetWidth(w int) {
	if w > 0 {
		s.width = w
	}
}

// SetFieldValue sets the display value for a field row. fieldIdx is the index
// into the fields slice (not the rows slice). Updates both the full row list
// and the filtered row list if search is active.
func (s *SectionModel) SetFieldValue(fieldIdx int, value string) {
	for i, r := range s.rows {
		if r.Kind == RowField && r.FieldIndex == fieldIdx {
			s.rows[i].Value = value
			break
		}
	}
	// Also update in filtered rows to keep them in sync.
	for i, r := range s.filteredRows {
		if r.Kind == RowField && r.FieldIndex == fieldIdx {
			s.filteredRows[i].Value = value
			break
		}
	}
}

// UpdateEntries rebuilds the entry list portion of the rows with new entries.
func (s *SectionModel) UpdateEntries(entries []string) {
	s.entries = entries
	s.buildRows()
	if s.search != nil && s.search.IsActive() {
		s.applyFilter()
	}
	if s.focusIndex >= len(s.visibleRows()) {
		s.focusFirst()
	}
	s.ensureVisible()
}

// --- Search ---.

// Search returns the section's search filter.
func (s *SectionModel) Search() *SearchFilter {
	return s.search
}

// OpenSearch activates the search bar and resets focus.
func (s *SectionModel) OpenSearch() {
	s.search.Open()
	s.applyFilter()
}

// CloseSearch deactivates the search bar, restores the full row list,
// and resets focus to the first focusable row.
func (s *SectionModel) CloseSearch() {
	s.search.Close()
	s.filteredRows = nil
	s.scrollOffset = 0
	s.focusFirst()
}

// SearchHandleRune appends a rune to the search query and re-applies the filter.
func (s *SectionModel) SearchHandleRune(r rune) {
	s.search.HandleRune(r)
	s.applyFilter()
}

// SearchHandleBackspace removes the last character from the query and
// re-applies the filter.
func (s *SectionModel) SearchHandleBackspace() {
	s.search.HandleBackspace()
	s.applyFilter()
}

// IsSearchActive reports whether the search filter is currently open.
func (s *SectionModel) IsSearchActive() bool {
	return s.search != nil && s.search.IsActive()
}

// --- Rendering ---.

// View renders the visible portion of the section within the viewport.
// When the section is empty (not configured), it renders a placeholder.
// When search is active, the search bar is rendered at the top and the
// viewport is reduced by one line to accommodate it.
func (s *SectionModel) View() string {
	if s.IsEmpty() {
		return s.ViewEmpty()
	}

	rows := s.visibleRows()

	searchActive := s.search != nil && s.search.IsActive()
	searchBar := ""
	vpHeight := s.viewportHeight

	if searchActive {
		searchBar = s.search.View(s.effectiveWidth())
		vpHeight = max(vpHeight-1, 1)
	}

	if len(rows) == 0 {
		if searchBar != "" {
			// Show search bar even with no results.
			padding := make([]string, vpHeight)
			for i := range padding {
				padding[i] = ""
			}
			return searchBar + "\n" + strings.Join(padding, "\n")
		}
		return ""
	}

	end := min(s.scrollOffset+vpHeight, len(rows))

	var lines []string
	for i := s.scrollOffset; i < end; i++ {
		lines = append(lines, s.renderVisibleRow(rows, i))
	}

	for len(lines) < vpHeight {
		lines = append(lines, "")
	}

	if fc := s.FieldCount(); fc > 0 && len(lines) > 0 {
		indicator := s.renderPositionIndicator()
		lastIdx := len(lines) - 1
		lines[lastIdx] = s.overlayRight(lines[lastIdx], indicator)
	}

	if searchBar != "" {
		return searchBar + "\n" + strings.Join(lines, "\n")
	}
	return strings.Join(lines, "\n")
}

// RenderRow renders a single row from the given row list by index.
// This is used by external callers (e.g., viewContentEditing) to render
// individual rows when substituting the editing view for a specific field.
func (s *SectionModel) RenderRow(rows []Row, idx int) string {
	return s.renderVisibleRow(rows, idx)
}

// renderVisibleRow renders a single row from the given row list by index.
func (s *SectionModel) renderVisibleRow(rows []Row, idx int) string {
	r := rows[idx]
	focused := idx == s.focusIndex

	switch r.Kind {
	case RowGroupHeader:
		return s.renderGroupHeader(r)
	case RowField:
		return s.renderFieldRow(r, focused)
	case RowEntry:
		return s.renderEntryRow(r, focused)
	case RowEmpty:
		return s.renderEmptyRow(r)
	default:
		return ""
	}
}

// renderGroupHeader renders a group separator line: "  -- Name ----...".
func (s *SectionModel) renderGroupHeader(r Row) string {
	style := lipgloss.NewStyle().Foreground(s.theme.Muted)

	prefix := "  -- "
	suffix := " "
	w := s.effectiveWidth()
	used := len(prefix) + len(r.Label) + len(suffix)
	dashes := max(w-used, 4)

	return style.Render(prefix + r.Label + suffix + strings.Repeat("-", dashes))
}

// renderFieldRow renders a focusable field row with optional focus indicator.
func (s *SectionModel) renderFieldRow(r Row, focused bool) string {
	lw := s.labelWidth()

	paddedLabel := r.Label
	if len(paddedLabel) < lw {
		paddedLabel += strings.Repeat(" ", lw-len(paddedLabel))
	}

	value := r.Value
	if value == "" {
		value = "--"
	}

	if focused {
		cursorStyle := lipgloss.NewStyle().
			Foreground(s.theme.Cursor).Bold(true)
		labelStyle := lipgloss.NewStyle().
			Foreground(s.theme.Accent).Bold(true)

		return cursorStyle.Render("> ") + "  " +
			labelStyle.Render(paddedLabel) + "  " + value
	}

	return "    " + paddedLabel + "  " + value
}

// renderEntryRow renders a focusable entry list summary row.
func (s *SectionModel) renderEntryRow(r Row, focused bool) string {
	label := fmt.Sprintf("%d. %s", r.EntryIndex+1, r.Label)

	if focused {
		cursorStyle := lipgloss.NewStyle().
			Foreground(s.theme.Cursor).Bold(true)
		entryStyle := lipgloss.NewStyle().
			Foreground(s.theme.Accent).Bold(true)
		return "  " + cursorStyle.Render(">") + " " + entryStyle.Render(label)
	}

	return "    " + label
}

// renderEmptyRow renders the empty-state placeholder for an entry list.
func (s *SectionModel) renderEmptyRow(r Row) string {
	style := lipgloss.NewStyle().
		Foreground(s.theme.Muted).Italic(true)
	return "    " + style.Render(r.Label)
}

// renderPositionIndicator returns the position indicator string (e.g., "4/24").
func (s *SectionModel) renderPositionIndicator() string {
	pos := s.FocusPosition()
	total := s.FieldCount()
	if total == 0 {
		return ""
	}

	style := lipgloss.NewStyle().Foreground(s.theme.Muted)
	return style.Render(fmt.Sprintf("%d/%d", pos, total))
}

// overlayRight places text at the right edge of a line, overwriting any
// existing content at that position.
func (s *SectionModel) overlayRight(line, overlay string) string {
	w := s.effectiveWidth()
	oLen := lipgloss.Width(overlay)
	if oLen >= w {
		return overlay
	}

	lineW := lipgloss.Width(line)
	if lineW < w {
		line += strings.Repeat(" ", w-lineW)
	}

	padW := w - oLen
	return lipgloss.NewStyle().Width(padW).Render(
		lipgloss.NewStyle().Width(padW).Render(line[:min(len(line), padW)]),
	) + overlay
}

// effectiveWidth returns the usable rendering width.
func (s *SectionModel) effectiveWidth() int {
	if s.width > 0 {
		return s.width
	}
	return 72
}

// labelWidth returns the column width for field labels, calculated from the
// longest label in the fields slice.
func (s *SectionModel) labelWidth() int {
	maxLen := 0
	for _, f := range s.fields {
		if len(f.Label) > maxLen {
			maxLen = len(f.Label)
		}
	}
	if maxLen < 10 {
		maxLen = 10
	}
	return maxLen
}

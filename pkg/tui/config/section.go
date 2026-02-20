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
type SectionModel struct {
	theme        *tui.Theme
	entryLabel   string
	emptyMessage string
	rows         []Row
	fields       []FieldMeta
	entries      []string

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

// focusFirst sets focusIndex to the first focusable row.
// If no focusable row exists, focusIndex is set to 0.
func (s *SectionModel) focusFirst() {
	for i, r := range s.rows {
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

// Rows returns the flattened row list.
func (s *SectionModel) Rows() []Row {
	return s.rows
}

// FocusedRow returns the currently focused row.
// Returns a zero-value Row if the section is empty.
func (s *SectionModel) FocusedRow() Row {
	if len(s.rows) == 0 {
		return Row{FieldIndex: -1, EntryIndex: -1}
	}
	return s.rows[s.focusIndex]
}

// FieldCount returns the number of focusable rows (fields + entries).
func (s *SectionModel) FieldCount() int {
	count := 0
	for _, r := range s.rows {
		if r.IsFocusable() {
			count++
		}
	}
	return count
}

// FocusPosition returns the 1-based position of the focused field among
// all focusable rows. Returns 0 if no focusable rows exist.
func (s *SectionModel) FocusPosition() int {
	pos := 0
	for i, r := range s.rows {
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
	if len(s.rows) == 0 {
		return
	}

	start := s.focusIndex
	for i := 1; i < len(s.rows); i++ {
		idx := (start + i) % len(s.rows)
		if s.rows[idx].IsFocusable() {
			s.focusIndex = idx
			s.ensureVisible()
			return
		}
	}
}

// MoveFocusUp moves focus to the previous focusable row, wrapping to the last
// if at the beginning. Updates scroll offset to keep the focused row visible.
func (s *SectionModel) MoveFocusUp() {
	if len(s.rows) == 0 {
		return
	}

	start := s.focusIndex
	n := len(s.rows)
	for i := 1; i < n; i++ {
		idx := (start - i + n) % n
		if s.rows[idx].IsFocusable() {
			s.focusIndex = idx
			s.ensureVisible()
			return
		}
	}
}

// JumpToFirst moves focus to the first focusable row and scrolls to the top.
func (s *SectionModel) JumpToFirst() {
	for i, r := range s.rows {
		if r.IsFocusable() {
			s.focusIndex = i
			s.scrollOffset = 0
			return
		}
	}
}

// JumpToLast moves focus to the last focusable row.
func (s *SectionModel) JumpToLast() {
	for i := len(s.rows) - 1; i >= 0; i-- {
		if s.rows[i].IsFocusable() {
			s.focusIndex = i
			s.ensureVisible()
			return
		}
	}
}

// JumpToNextGroup moves focus to the first focusable row in the next group.
// If already in the last group, wraps to the first group.
func (s *SectionModel) JumpToNextGroup() {
	if len(s.rows) == 0 {
		return
	}

	currentGroup := s.rows[s.focusIndex].Group
	n := len(s.rows)

	for i := s.focusIndex + 1; i < s.focusIndex+n; i++ {
		idx := i % n
		r := s.rows[idx]
		if r.Group != currentGroup {
			targetGroup := r.Group
			for j := idx; j < idx+n; j++ {
				jIdx := j % n
				if s.rows[jIdx].Group == targetGroup && s.rows[jIdx].IsFocusable() {
					s.focusIndex = jIdx
					s.ensureVisible()
					return
				}
				if s.rows[jIdx].Group != targetGroup && jIdx != idx {
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
	if len(s.rows) == 0 {
		return
	}

	currentGroup := s.rows[s.focusIndex].Group
	n := len(s.rows)

	for i := s.focusIndex - 1 + n; i > s.focusIndex; i-- {
		idx := i % n
		r := s.rows[idx]
		if r.Group != currentGroup {
			targetGroup := r.Group

			// Find the start of this group by scanning backward.
			groupStart := idx
			for {
				prev := (groupStart - 1 + n) % n
				if s.rows[prev].Group != targetGroup {
					break
				}
				groupStart = prev
				if groupStart == idx {
					break
				}
			}

			// Find the first focusable row in the target group.
			for j := groupStart; ; j = (j + 1) % n {
				if s.rows[j].Group == targetGroup && s.rows[j].IsFocusable() {
					s.focusIndex = j
					s.ensureVisible()
					return
				}
				if s.rows[j].Group != targetGroup {
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

	maxOffset := max(len(s.rows)-s.viewportHeight, 0)
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
// into the fields slice (not the rows slice).
func (s *SectionModel) SetFieldValue(fieldIdx int, value string) {
	for i, r := range s.rows {
		if r.Kind == RowField && r.FieldIndex == fieldIdx {
			s.rows[i].Value = value
			return
		}
	}
}

// UpdateEntries rebuilds the entry list portion of the rows with new entries.
func (s *SectionModel) UpdateEntries(entries []string) {
	s.entries = entries
	s.buildRows()
	if s.focusIndex >= len(s.rows) {
		s.focusFirst()
	}
	s.ensureVisible()
}

// --- Rendering ---.

// View renders the visible portion of the section within the viewport.
func (s *SectionModel) View() string {
	if len(s.rows) == 0 {
		return ""
	}

	end := min(s.scrollOffset+s.viewportHeight, len(s.rows))

	var lines []string
	for i := s.scrollOffset; i < end; i++ {
		lines = append(lines, s.renderRow(i))
	}

	for len(lines) < s.viewportHeight {
		lines = append(lines, "")
	}

	if fc := s.FieldCount(); fc > 0 && len(lines) > 0 {
		indicator := s.renderPositionIndicator()
		lastIdx := len(lines) - 1
		lines[lastIdx] = s.overlayRight(lines[lastIdx], indicator)
	}

	return strings.Join(lines, "\n")
}

// renderRow renders a single row by index.
func (s *SectionModel) renderRow(idx int) string {
	r := s.rows[idx]
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

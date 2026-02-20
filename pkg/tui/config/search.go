package config

import (
	"gbm/pkg/tui"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// SearchFilter provides incremental field search/filtering for a SectionModel.
// When active, it filters the visible rows to only those whose label contains
// the search query (case-insensitive substring match). Group headers are shown
// only when they have at least one matching child field. Entry rows and empty
// rows are always hidden during search.
type SearchFilter struct {
	theme  *tui.Theme
	query  string
	active bool
}

// NewSearchFilter creates a new inactive search filter.
func NewSearchFilter(theme *tui.Theme) *SearchFilter {
	if theme == nil {
		theme = tui.DefaultTheme()
	}
	return &SearchFilter{theme: theme}
}

// IsActive reports whether the search filter is currently open.
func (sf *SearchFilter) IsActive() bool {
	return sf.active
}

// Query returns the current search query.
func (sf *SearchFilter) Query() string {
	return sf.query
}

// Open activates the search bar with an empty query.
func (sf *SearchFilter) Open() {
	sf.active = true
	sf.query = ""
}

// Close deactivates the search bar and clears the query.
func (sf *SearchFilter) Close() {
	sf.active = false
	sf.query = ""
}

// HandleRune appends a character to the search query.
func (sf *SearchFilter) HandleRune(r rune) {
	sf.query += string(r)
}

// HandleBackspace removes the last character from the search query.
// If the query is already empty, this is a no-op.
func (sf *SearchFilter) HandleBackspace() {
	if sf.query != "" {
		sf.query = sf.query[:len(sf.query)-1]
	}
}

// FilterRows returns the subset of rows that match the current search query.
// If the query is empty, all rows are returned (unfiltered).
//
// Matching rules:
//   - Field rows match if their Label contains the query (case-insensitive).
//   - Group headers are included only when at least one field in the same
//     group matches.
//   - Entry rows and empty rows are excluded during active search.
//   - When the query is empty, all rows are returned unchanged.
func (sf *SearchFilter) FilterRows(rows []Row) []Row {
	if !sf.active || sf.query == "" {
		return rows
	}

	q := strings.ToLower(sf.query)

	// First pass: determine which groups have at least one matching field.
	matchingGroups := make(map[string]bool)
	for _, r := range rows {
		if r.Kind == RowField && strings.Contains(strings.ToLower(r.Label), q) {
			matchingGroups[r.Group] = true
		}
	}

	// Second pass: build filtered row list.
	var filtered []Row
	for _, r := range rows {
		switch r.Kind {
		case RowField:
			if strings.Contains(strings.ToLower(r.Label), q) {
				filtered = append(filtered, r)
			}
		case RowGroupHeader:
			if matchingGroups[r.Group] {
				filtered = append(filtered, r)
			}
		case RowEntry, RowEmpty:
			// Excluded during search.
		}
	}

	return filtered
}

// View renders the search bar. Returns an empty string when the filter is
// not active. The search bar is a single line: "/ <query>_" where _ is the
// cursor indicator.
func (sf *SearchFilter) View(width int) string {
	if !sf.active {
		return ""
	}

	promptStyle := lipgloss.NewStyle().
		Foreground(sf.theme.Accent).
		Bold(true)

	queryStyle := lipgloss.NewStyle().
		Foreground(sf.theme.InputFg)

	cursorStyle := lipgloss.NewStyle().
		Foreground(sf.theme.Cursor).
		Bold(true)

	prompt := promptStyle.Render("/ ")
	query := queryStyle.Render(sf.query)
	cursor := cursorStyle.Render("_")

	line := prompt + query + cursor

	// Pad to width if needed.
	lineWidth := lipgloss.Width(line)
	if lineWidth < width {
		line += strings.Repeat(" ", width-lineWidth)
	}

	return line
}

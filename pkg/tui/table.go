package tui

import (
	"strings"

	"gbm/pkg/tui/async"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	fzf "github.com/koki-develop/go-fzf"
)

// Table is a reusable table component wrapping bubbles/table with theme support.
// It applies consistent styling from the Theme and supports async cell rendering.
type Table struct {
	model     table.Model
	ctx       *Context
	theme     *Theme
	columns   []table.Column
	rows      []table.Row
	asyncRows map[int]*AsyncRow // Row index -> AsyncRow for tracking async cells
	height    int
	focused   bool

	// Filter support
	filterEnabled bool            // Whether "/" filter mode is available
	filterActive  bool            // Whether currently in filter mode
	filterInput   textinput.Model // Text input for filter query
	allRows       []table.Row     // All rows (unfiltered source of truth)
	rowMapping    []int           // Maps filtered index -> original index in allRows
	cycling       bool            // Whether up/down navigation wraps around
}

// Column represents a table column definition.
type Column struct {
	Title string
	Width int
}

// AsyncRow represents a table row with support for async cell loading.
// It holds static string values and async cells that may be loading.
type AsyncRow struct {
	staticCells map[int]string              // Column index -> static cell value
	asyncCells  map[int]*async.Cell[string] // Column index -> async cell
	tickCount   uint64                      // For syncing spinner animation
}

// NewAsyncRow creates a new AsyncRow with initial static cells.
func NewAsyncRow(cells ...string) *AsyncRow {
	staticCells := make(map[int]string)
	for i, cell := range cells {
		staticCells[i] = cell
	}
	return &AsyncRow{
		staticCells: staticCells,
		asyncCells:  make(map[int]*async.Cell[string]),
	}
}

// WithAsyncCell adds or replaces an async cell at the given column index.
func (ar *AsyncRow) WithAsyncCell(colIdx int, cell *async.Cell[string]) *AsyncRow {
	ar.asyncCells[colIdx] = cell
	// Remove from static cells if present
	delete(ar.staticCells, colIdx)
	return ar
}

// GetCell returns the display value for a cell at the given column index.
// For async cells, returns the current View() (spinner or value).
// For static cells, returns the string value.
func (ar *AsyncRow) GetCell(colIdx int) string {
	if asyncCell, ok := ar.asyncCells[colIdx]; ok {
		return asyncCell.View()
	}
	if staticVal, ok := ar.staticCells[colIdx]; ok {
		return staticVal
	}
	return ""
}

// ToTableRow converts the AsyncRow to a bubbles/table.Row for rendering.
func (ar *AsyncRow) ToTableRow(numCols int) table.Row {
	row := make(table.Row, numCols)
	for i := 0; i < numCols; i++ {
		row[i] = ar.GetCell(i)
	}
	return row
}

// Tick advances the spinner animation for all async cells in this row.
func (ar *AsyncRow) Tick() {
	ar.tickCount++
	for _, cell := range ar.asyncCells {
		cell.Tick()
	}
}

// IsLoading returns true if any async cell in this row is still loading.
func (ar *AsyncRow) IsLoading() bool {
	for _, cell := range ar.asyncCells {
		if cell.IsLoading() {
			return true
		}
	}
	return false
}

// NewTable creates a new Table with the given context for theming.
func NewTable(ctx *Context) *Table {
	return &Table{
		ctx:       ctx,
		theme:     ctx.Theme,
		asyncRows: make(map[int]*AsyncRow),
		focused:   true,
		height:    10,
	}
}

// WithColumns sets the table columns.
func (t *Table) WithColumns(columns []Column) *Table {
	t.columns = make([]table.Column, len(columns))
	for i, c := range columns {
		t.columns[i] = table.Column{Title: c.Title, Width: c.Width}
	}
	return t
}

// WithRows sets the table rows.
func (t *Table) WithRows(rows []table.Row) *Table {
	t.rows = rows
	return t
}

// WithHeight sets the visible height of the table.
func (t *Table) WithHeight(height int) *Table {
	t.height = height
	return t
}

// WithFocused sets whether the table is focused.
func (t *Table) WithFocused(focused bool) *Table {
	t.focused = focused
	return t
}

// WithFilterable enables "/" filter mode for the table.
// When enabled, pressing "/" shows a filter input that filters rows by matching
// any column content. Press Enter/Esc to exit filter mode.
func (t *Table) WithFilterable(enabled bool) *Table {
	t.filterEnabled = enabled
	return t
}

// WithCycling enables wrap-around navigation (up on first row goes to last, etc).
func (t *Table) WithCycling(enabled bool) *Table {
	t.cycling = enabled
	return t
}

// applyTheme applies the theme styles to the table.
func (t *Table) applyTheme() {
	styles := table.DefaultStyles()
	styles.Header = t.theme.Table.Header
	styles.Selected = t.theme.Table.Selected
	styles.Cell = t.theme.Table.Cell
	t.model.SetStyles(styles)
}

// Build finalizes the table configuration and applies theme.
// Call this after setting all column/row options.
func (t *Table) Build() *Table {
	// Store original rows for filtering
	if t.filterEnabled {
		t.allRows = make([]table.Row, len(t.rows))
		copy(t.allRows, t.rows)
		t.rowMapping = make([]int, len(t.rows))
		for i := range t.rows {
			t.rowMapping[i] = i
		}

		// Initialize filter text input
		ti := textinput.New()
		ti.Prompt = "" // No prompt, we render "/" ourselves
		ti.Placeholder = "type to filter..."
		ti.CharLimit = 100
		ti.Width = 40
		t.filterInput = ti
	}

	opts := []table.Option{
		table.WithColumns(t.columns),
		table.WithRows(t.rows),
		table.WithHeight(t.height),
		table.WithFocused(t.focused),
	}
	t.model = table.New(opts...)
	t.applyTheme()
	return t
}

// Init initializes the table (required by tea.Model interface).
func (t *Table) Init() tea.Cmd {
	return nil
}

// SetAsyncCell sets an async cell at the given row and column indices.
// Creates or updates the AsyncRow if needed.
func (t *Table) SetAsyncCell(rowIdx int, colIdx int, cell *async.Cell[string]) {
	// Ensure AsyncRow exists
	if _, ok := t.asyncRows[rowIdx]; !ok {
		// Create new AsyncRow and populate with existing static cells from table row
		asyncRow := NewAsyncRow()
		if rowIdx < len(t.rows) {
			// Copy all existing cells from the table row as static cells
			for i, cellVal := range t.rows[rowIdx] {
				asyncRow.staticCells[i] = cellVal
			}
		}
		t.asyncRows[rowIdx] = asyncRow
	}
	t.asyncRows[rowIdx].WithAsyncCell(colIdx, cell)

	// Update the table row to reflect async cell
	t.updateTableRow(rowIdx)
}

// updateTableRow refreshes a specific table row with current async cell values.
func (t *Table) updateTableRow(rowIdx int) {
	if rowIdx >= len(t.rows) {
		return
	}

	asyncRow, ok := t.asyncRows[rowIdx]
	if !ok {
		return
	}

	newRow := asyncRow.ToTableRow(len(t.columns))
	t.rows[rowIdx] = newRow

	// Also update allRows if filtering is enabled to keep them in sync
	if t.filterEnabled && rowIdx < len(t.allRows) {
		t.allRows[rowIdx] = newRow
	}

	// Only update model rows if not actively filtering
	// (when filtering, applyFilter manages what's shown)
	if !t.filterActive {
		t.model.SetRows(t.rows)
	} else {
		// Re-apply filter to update the displayed filtered rows
		t.applyFilter()
	}
}

// Update handles input and state changes (required by tea.Model interface).
func (t *Table) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// Handle key messages for filter mode and cycling
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		// Filter mode handling
		if t.filterEnabled {
			if t.filterActive {
				cmd = t.handleFilterInput(keyMsg)
				if cmd != nil || t.filterActive {
					return t, cmd
				}
				// Filter was deactivated, fall through to normal handling
			} else if keyMsg.String() == "/" {
				t.filterActive = true
				t.filterInput.Focus()
				return t, textinput.Blink
			}
		}

		// Cycling navigation
		if t.cycling && !t.filterActive {
			switch keyMsg.String() {
			case "up", "k":
				t.moveCursor(-1)
				return t, nil
			case "down", "j":
				t.moveCursor(1)
				return t, nil
			}
		}
	}

	t.model, cmd = t.model.Update(msg)

	// Tick all async rows to update spinner animations
	for _, asyncRow := range t.asyncRows {
		asyncRow.Tick()
	}

	// Refresh all async rows' table display
	for rowIdx := range t.asyncRows {
		t.updateTableRow(rowIdx)
	}

	return t, cmd
}

// handleFilterInput processes key input when filter is active.
// Returns a command if the input was handled, nil otherwise.
func (t *Table) handleFilterInput(keyMsg tea.KeyMsg) tea.Cmd {
	switch keyMsg.String() {
	case "enter":
		// Select current row, clear filter, exit filter mode
		// Remember the original index of selected row before clearing
		originalIdx := t.OriginalIndex()
		t.filterActive = false
		t.filterInput.Blur()
		t.clearFilter()
		// Set cursor to the originally selected row
		if originalIdx >= 0 && originalIdx < len(t.allRows) {
			t.model.SetCursor(originalIdx)
		}
		return nil

	case "esc":
		// Cancel filter, clear and exit
		t.filterActive = false
		t.filterInput.Blur()
		t.clearFilter()
		return nil

	case "up", "k":
		// Navigate up in filtered results
		t.moveCursor(-1)
		return nil

	case "down", "j":
		// Navigate down in filtered results
		t.moveCursor(1)
		return nil

	case "backspace":
		// If input is empty, clear filter and exit
		if t.filterInput.Value() == "" {
			t.filterActive = false
			t.filterInput.Blur()
			t.clearFilter()
			return nil
		}
		// Otherwise, let textinput handle backspace
		fallthrough

	default:
		// Update text input
		var cmd tea.Cmd
		t.filterInput, cmd = t.filterInput.Update(keyMsg)
		t.applyFilter()
		return cmd
	}
}

// tableRowItems implements fzf.Items for fuzzy searching table rows.
type tableRowItems struct {
	rows []table.Row
}

func (tri tableRowItems) ItemString(i int) string {
	return strings.Join(tri.rows[i], " ")
}

func (tri tableRowItems) Len() int {
	return len(tri.rows)
}

// applyFilter filters rows based on current filter input using fuzzy matching.
func (t *Table) applyFilter() {
	query := strings.TrimSpace(t.filterInput.Value())

	if query == "" {
		t.clearFilter()
		return
	}

	// Use fuzzy search to match rows
	items := tableRowItems{rows: t.allRows}
	matches := fzf.Search(items, query)

	// Build filtered rows and mapping from matches
	filtered := make([]table.Row, len(matches))
	mapping := make([]int, len(matches))
	for i, match := range matches {
		filtered[i] = t.allRows[match.Index]
		mapping[i] = match.Index
	}

	prevRowCount := len(t.model.Rows())
	t.rowMapping = mapping
	t.model.SetRows(filtered)

	// Reset cursor to top if out of bounds or if we went from 0 to some rows
	if len(filtered) > 0 && (t.model.Cursor() >= len(filtered) || prevRowCount == 0) {
		t.model.SetCursor(0)
	}
}

// clearFilter resets to show all rows.
func (t *Table) clearFilter() {
	t.filterInput.SetValue("")
	t.rowMapping = make([]int, len(t.allRows))
	for i := range t.allRows {
		t.rowMapping[i] = i
	}
	t.model.SetRows(t.allRows)
}

// moveCursor moves the cursor with wrap-around support.
func (t *Table) moveCursor(delta int) {
	rows := t.model.Rows()
	if len(rows) == 0 {
		return
	}

	cursor := t.model.Cursor()
	newCursor := cursor + delta

	if newCursor < 0 {
		newCursor = len(rows) - 1
	} else if newCursor >= len(rows) {
		newCursor = 0
	}

	t.model.SetCursor(newCursor)
}

// View renders the table (required by tea.Model interface).
func (t *Table) View() string {
	output := t.theme.Table.Base.Render(t.model.View())

	// Show filter input if active
	if t.filterEnabled && t.filterActive {
		filterStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("212")).
			Bold(true)
		output += "\n" + filterStyle.Render("/") + t.filterInput.View()
	} else if t.filterEnabled && t.filterInput.Value() != "" {
		// Show filter indicator when filtered but not actively editing
		filterStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))
		output += "\n" + filterStyle.Render("Filter: "+t.filterInput.Value()+" (/ to edit, backspace to clear)")
	}

	return output
}

// SetRows updates the table rows dynamically.
// When filtering is enabled, this updates the source rows and reapplies the filter.
func (t *Table) SetRows(rows []table.Row) {
	if t.filterEnabled {
		t.allRows = make([]table.Row, len(rows))
		copy(t.allRows, rows)
		t.applyFilter()
	} else {
		t.model.SetRows(rows)
	}
}

// Cursor returns the current cursor position (selected row index).
func (t *Table) Cursor() int {
	return t.model.Cursor()
}

// SelectedRow returns the currently selected row.
func (t *Table) SelectedRow() table.Row {
	cursor := t.model.Cursor()
	rows := t.model.Rows()
	if cursor >= 0 && cursor < len(rows) {
		return rows[cursor]
	}
	return nil
}

// SetCursor sets the cursor position.
func (t *Table) SetCursor(pos int) {
	t.model.SetCursor(pos)
}

// SetHeight sets the visible height.
func (t *Table) SetHeight(height int) {
	t.model.SetHeight(height)
}

// SetColumns updates the table columns dynamically for responsive resizing.
func (t *Table) SetColumns(columns []Column) {
	t.columns = make([]table.Column, len(columns))
	for i, c := range columns {
		t.columns[i] = table.Column{Title: c.Title, Width: c.Width}
	}
	t.model.SetColumns(t.columns)
}

// Rows returns the current table rows (filtered if filter is active).
func (t *Table) Rows() []table.Row {
	return t.model.Rows()
}

// AllRows returns all rows (unfiltered). Returns nil if filtering is not enabled.
func (t *Table) AllRows() []table.Row {
	return t.allRows
}

// OriginalIndex returns the original row index for the current cursor position.
// When filtering is enabled, this maps the filtered index back to the source index.
// Returns -1 if cursor is out of bounds.
func (t *Table) OriginalIndex() int {
	cursor := t.model.Cursor()
	if !t.filterEnabled || len(t.rowMapping) == 0 {
		return cursor
	}
	if cursor < 0 || cursor >= len(t.rowMapping) {
		return -1
	}
	return t.rowMapping[cursor]
}

// IsFilterActive returns true if the filter input is currently focused.
func (t *Table) IsFilterActive() bool {
	return t.filterActive
}

// FilterQuery returns the current filter query string.
func (t *Table) FilterQuery() string {
	if !t.filterEnabled {
		return ""
	}
	return t.filterInput.Value()
}

// ClearFilter clears the current filter and shows all rows.
func (t *Table) ClearFilter() {
	if t.filterEnabled {
		t.clearFilter()
	}
}

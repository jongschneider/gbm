package tui

import (
	"gbm/pkg/tui/async"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
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

	// If there's an async row, use its rendered cells
	if asyncRow, ok := t.asyncRows[rowIdx]; ok {
		t.rows[rowIdx] = asyncRow.ToTableRow(len(t.columns))
		t.model.SetRows(t.rows)
	}
}

// Update handles input and state changes (required by tea.Model interface).
func (t *Table) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
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

// View renders the table (required by tea.Model interface).
func (t *Table) View() string {
	return t.theme.Table.Base.Render(t.model.View())
}

// SetRows updates the table rows dynamically.
func (t *Table) SetRows(rows []table.Row) {
	t.model.SetRows(rows)
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

// Rows returns the current table rows.
func (t *Table) Rows() []table.Row {
	return t.model.Rows()
}

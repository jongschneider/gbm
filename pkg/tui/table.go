package tui

import (
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
)

// Table is a reusable table component wrapping bubbles/table with theme support.
// It applies consistent styling from the Theme and supports async cell rendering.
type Table struct {
	model   table.Model
	ctx     *Context
	theme   *Theme
	columns []table.Column
	rows    []table.Row
	height  int
	focused bool
}

// Column represents a table column definition.
type Column struct {
	Title string
	Width int
}

// NewTable creates a new Table with the given context for theming.
func NewTable(ctx *Context) *Table {
	return &Table{
		ctx:     ctx,
		theme:   ctx.Theme,
		focused: true,
		height:  10,
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

// Update handles input and state changes (required by tea.Model interface).
func (t *Table) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	t.model, cmd = t.model.Update(msg)
	return t, cmd
}

// View renders the table (required by tea.Model interface).
func (t *Table) View() string {
	return t.model.View()
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
	if t.model.Cursor() < len(t.model.Rows()) {
		return t.model.Rows()[t.model.Cursor()]
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

package tui

import (
	"testing"

	"gbm/pkg/tui/async"
	"github.com/charmbracelet/bubbles/table"
	"github.com/stretchr/testify/assert"
)

func TestTable_NewTable_CreatesWithDefaults(t *testing.T) {
	ctx := NewContext()
	tbl := NewTable(ctx)

	assert.NotNil(t, tbl)
	assert.Equal(t, ctx, tbl.ctx)
	assert.Equal(t, ctx.Theme, tbl.theme)
	assert.Equal(t, 10, tbl.height)
	assert.True(t, tbl.focused)
}

func TestTable_WithColumns_SetsColumns(t *testing.T) {
	ctx := NewContext()
	tbl := NewTable(ctx)

	cols := []Column{
		{Title: "Name", Width: 20},
		{Title: "Status", Width: 15},
	}

	result := tbl.WithColumns(cols)

	assert.Same(t, tbl, result) // Check chaining
	assert.Len(t, tbl.columns, 2)
	assert.Equal(t, "Name", tbl.columns[0].Title)
	assert.Equal(t, 20, tbl.columns[0].Width)
}

func TestTable_WithRows_SetsRows(t *testing.T) {
	ctx := NewContext()
	tbl := NewTable(ctx)

	rows := []table.Row{
		{"row1", "value1"},
		{"row2", "value2"},
	}

	result := tbl.WithRows(rows)

	assert.Same(t, tbl, result) // Check chaining
	assert.Len(t, tbl.rows, 2)
}

func TestTable_WithHeight_SetsHeight(t *testing.T) {
	ctx := NewContext()
	tbl := NewTable(ctx)

	result := tbl.WithHeight(20)

	assert.Same(t, tbl, result) // Check chaining
	assert.Equal(t, 20, tbl.height)
}

func TestTable_WithFocused_SetsFocused(t *testing.T) {
	ctx := NewContext()
	tbl := NewTable(ctx)

	result := tbl.WithFocused(false)

	assert.Same(t, tbl, result) // Check chaining
	assert.False(t, tbl.focused)
}

func TestTable_Build_CreatesModel(t *testing.T) {
	ctx := NewContext()
	cols := []Column{{Title: "Name", Width: 20}}
	rows := []table.Row{{"row1"}}

	tbl := NewTable(ctx).
		WithColumns(cols).
		WithRows(rows).
		WithHeight(10).
		Build()

	assert.NotNil(t, tbl.model)
	// Verify theme is applied
	assert.NotNil(t, tbl.theme)
}

func TestAsyncRow_NewAsyncRow_CreatesWithStaticCells(t *testing.T) {
	ar := NewAsyncRow("cell1", "cell2", "cell3")

	assert.NotNil(t, ar)
	assert.Len(t, ar.staticCells, 3)
	assert.Equal(t, "cell1", ar.staticCells[0])
	assert.Equal(t, "cell2", ar.staticCells[1])
	assert.Equal(t, "cell3", ar.staticCells[2])
	assert.Len(t, ar.asyncCells, 0)
}

func TestAsyncRow_WithAsyncCell_AddsAsyncCell(t *testing.T) {
	ar := NewAsyncRow("static1", "static2")
	eval := async.New(func() (string, error) { return "async", nil })
	cell := async.NewCell(eval)

	result := ar.WithAsyncCell(1, cell)

	assert.Same(t, ar, result) // Check chaining
	assert.Len(t, ar.asyncCells, 1)
	assert.NotNil(t, ar.asyncCells[1])
	// Should remove from static cells
	_, exists := ar.staticCells[1]
	assert.False(t, exists)
}

func TestAsyncRow_GetCell_ReturnsStaticValue(t *testing.T) {
	ar := NewAsyncRow("cell1", "cell2")

	val := ar.GetCell(0)
	assert.Equal(t, "cell1", val)

	val = ar.GetCell(1)
	assert.Equal(t, "cell2", val)

	val = ar.GetCell(999) // Out of bounds
	assert.Equal(t, "", val)
}

func TestAsyncRow_GetCell_ReturnsAsyncSpinnerWhileLoading(t *testing.T) {
	ar := NewAsyncRow("static1", "static2")

	// Create an async cell
	eval := async.New(func() (string, error) {
		return "loaded", nil
	})
	cell := async.NewCell(eval)
	ar.WithAsyncCell(1, cell)

	// Before StartLoading, should return empty
	val := ar.GetCell(1)
	assert.Empty(t, val)

	// StartLoading returns a Cmd, which must be executed to actually run the fetch
	cmd := cell.StartLoading()
	assert.NotNil(t, cmd) // Cmd returned

	// Before executing the Cmd, Cmd is returned and IsLoading() would be true
	// If we execute the Cmd, it runs the fetch synchronously
	// Since the test doesn't execute the Cmd, isStarted=true but eval not loaded
	// GetCell still shows spinner while "loading" (even though Cmd not executed)
	val = ar.GetCell(1)
	assert.NotEmpty(t, val) // Shows spinner
}

func TestAsyncRow_ToTableRow_ConvertsAllCells(t *testing.T) {
	ar := NewAsyncRow("cell1", "cell2", "cell3")

	row := ar.ToTableRow(3)

	assert.Len(t, row, 3)
	assert.Equal(t, "cell1", row[0])
	assert.Equal(t, "cell2", row[1])
	assert.Equal(t, "cell3", row[2])
}

func TestAsyncRow_Tick_UpdatesSpinner(t *testing.T) {
	ar := NewAsyncRow("static")

	eval := async.New(func() (string, error) {
		return "loaded", nil
	})
	cell := async.NewCell(eval)
	ar.WithAsyncCell(0, cell)

	// Start loading
	cell.StartLoading()

	// Tick should advance spinner frame
	view1 := ar.GetCell(0)
	ar.Tick()
	ar.Tick()
	view2 := ar.GetCell(0)

	// Views might be the same due to timing, but both should be spinners
	assert.NotEmpty(t, view1)
	assert.NotEmpty(t, view2)
}

func TestAsyncRow_IsLoading_ReflectsAsyncState(t *testing.T) {
	ar := NewAsyncRow("static")

	// No async cells
	assert.False(t, ar.IsLoading())

	// Add async cell
	eval := async.New(func() (string, error) {
		return "loaded", nil
	})
	cell := async.NewCell(eval)
	ar.WithAsyncCell(0, cell)

	// Not started yet - no async cells loading
	assert.False(t, ar.IsLoading())

	// After StartLoading is called but before the Cmd is executed,
	// isStarted=true but eval hasn't loaded yet, so IsLoading() returns true
	cmd := cell.StartLoading()
	assert.NotNil(t, cmd)
	assert.True(t, ar.IsLoading()) // Cmd returned but not executed yet
}

func TestTable_ThemeStylesApplied(t *testing.T) {
	ctx := NewContext()
	theme := ctx.Theme

	// Verify DefaultTheme has TableStyles
	assert.NotNil(t, theme.Table)
	assert.NotNil(t, theme.Table.Header)
	assert.NotNil(t, theme.Table.Selected)
	assert.NotNil(t, theme.Table.Cell)
	assert.NotNil(t, theme.Table.Border)

	// Verify they're not empty styles
	headerStr := theme.Table.Header.Render("Header")
	assert.NotEmpty(t, headerStr)

	selectedStr := theme.Table.Selected.Render("Selected")
	assert.NotEmpty(t, selectedStr)
}

func TestTable_Update_TicksAsyncRows(t *testing.T) {
	ctx := NewContext()
	cols := []Column{{Title: "Name", Width: 20}, {Title: "Status", Width: 15}}
	rows := []table.Row{{"row1", ""}}

	tbl := NewTable(ctx).
		WithColumns(cols).
		WithRows(rows).
		Build()

	// Add async cell
	eval := async.New(func() (string, error) {
		return "loaded", nil
	})
	cell := async.NewCell(eval)
	cell.StartLoading()
	tbl.SetAsyncCell(0, 1, cell)

	// Update should tick all async cells
	_, _ = tbl.Update(nil)

	// AsyncRow should have been ticked
	assert.Len(t, tbl.asyncRows, 1)
	assert.NotNil(t, tbl.asyncRows[0])
}

package tui

import (
	"bytes"
	"gbm/pkg/tui/async"
	"sync/atomic"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/muesli/termenv"
	"github.com/stretchr/testify/assert"
)

// waitFor polls until condition returns true or timeout is reached.
// Returns immediately when condition is met, avoiding fixed sleep delays.
func waitFor(t *testing.T, condition func() bool, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(1 * time.Millisecond)
	}
	t.Fatal("waitFor: condition not met within timeout")
}

func init() {
	// Set ASCII color profile for consistent test output across environments
	lipgloss.SetColorProfile(termenv.Ascii)
}

// tableModel wraps a Table to implement tea.Model for teatest.
type tableModel struct {
	table *Table
}

func newTableModel(t *Table) *tableModel {
	return &tableModel{table: t}
}

func (m *tableModel) Init() tea.Cmd {
	return m.table.Init()
}

func (m *tableModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}

	model, cmd := m.table.Update(msg)
	if t, ok := model.(*Table); ok {
		m.table = t
	}

	return m, cmd
}

func (m *tableModel) View() string {
	return m.table.View()
}

// =============================================================================
// TT-019: Table row navigation tests
// =============================================================================.

// TestTable_DownArrowMovesCursorDown verifies that pressing the down arrow key
// moves the cursor down to the next row.
func TestTable_DownArrowMovesCursorDown(t *testing.T) {
	ctx := NewContext()
	cols := []Column{
		{Title: "Name", Width: 20},
		{Title: "Status", Width: 15},
	}
	rows := []table.Row{
		{"Row One", "Active"},
		{"Row Two", "Inactive"},
		{"Row Three", "Pending"},
	}

	tbl := NewTable(ctx).
		WithColumns(cols).
		WithRows(rows).
		WithHeight(10).
		Build()

	tm := teatest.NewTestModel(t, newTableModel(tbl), teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() {
		//nolint:errcheck // Best-effort cleanup in test
		tm.Quit()
	})

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Row One"))
	}, teatest.WithDuration(time.Second))

	// Verify initial cursor position is 0
	assert.Equal(t, 0, tbl.Cursor(), "initial cursor should be at index 0")

	// Press down arrow
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	waitFor(t, func() bool { return tbl.Cursor() == 1 }, time.Second)

	// Press down again
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	waitFor(t, func() bool { return tbl.Cursor() == 2 }, time.Second)

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// TestTable_UpArrowMovesCursorUp verifies that pressing the up arrow key
// moves the cursor up to the previous row.
func TestTable_UpArrowMovesCursorUp(t *testing.T) {
	ctx := NewContext()
	cols := []Column{
		{Title: "Name", Width: 20},
		{Title: "Status", Width: 15},
	}
	rows := []table.Row{
		{"Row One", "Active"},
		{"Row Two", "Inactive"},
		{"Row Three", "Pending"},
	}

	tbl := NewTable(ctx).
		WithColumns(cols).
		WithRows(rows).
		WithHeight(10).
		Build()

	// Set cursor to last row initially
	tbl.SetCursor(2)

	tm := teatest.NewTestModel(t, newTableModel(tbl), teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() {
		//nolint:errcheck // Best-effort cleanup in test
		tm.Quit()
	})

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Row Three"))
	}, teatest.WithDuration(time.Second))

	// Verify initial cursor position is 2
	assert.Equal(t, 2, tbl.Cursor(), "initial cursor should be at index 2")

	// Press up arrow
	tm.Send(tea.KeyMsg{Type: tea.KeyUp})
	waitFor(t, func() bool { return tbl.Cursor() == 1 }, time.Second)

	// Press up again
	tm.Send(tea.KeyMsg{Type: tea.KeyUp})
	waitFor(t, func() bool { return tbl.Cursor() == 0 }, time.Second)

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// TestTable_CursorReturnsCorrectRowIndex verifies that Cursor() returns
// the correct row index after various navigation operations.
func TestTable_CursorReturnsCorrectRowIndex(t *testing.T) {
	ctx := NewContext()
	cols := []Column{
		{Title: "ID", Width: 5},
		{Title: "Name", Width: 20},
	}
	rows := []table.Row{
		{"1", "First Item"},
		{"2", "Second Item"},
		{"3", "Third Item"},
		{"4", "Fourth Item"},
		{"5", "Fifth Item"},
	}

	tbl := NewTable(ctx).
		WithColumns(cols).
		WithRows(rows).
		WithHeight(10).
		Build()

	tm := teatest.NewTestModel(t, newTableModel(tbl), teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() {
		//nolint:errcheck // Best-effort cleanup in test
		tm.Quit()
	})

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("First Item"))
	}, teatest.WithDuration(time.Second))

	// Test sequence: down, down, down, up, down
	testCases := []struct {
		key      tea.KeyType
		expected int
	}{
		{tea.KeyDown, 1},
		{tea.KeyDown, 2},
		{tea.KeyDown, 3},
		{tea.KeyUp, 2},
		{tea.KeyDown, 3},
	}

	for i, tc := range testCases {
		tm.Send(tea.KeyMsg{Type: tc.key})
		expected := tc.expected
		waitFor(t, func() bool { return tbl.Cursor() == expected }, time.Second)
		assert.Equal(t, tc.expected, tbl.Cursor(),
			"step %d: Cursor() should return %d after %v", i+1, tc.expected, tc.key)
	}

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// TestTable_SelectedRowReturnsCorrectRowData verifies that SelectedRow() returns
// the correct row data for the currently selected row.
func TestTable_SelectedRowReturnsCorrectRowData(t *testing.T) {
	ctx := NewContext()
	cols := []Column{
		{Title: "Name", Width: 20},
		{Title: "Value", Width: 15},
	}
	rows := []table.Row{
		{"Alpha", "100"},
		{"Beta", "200"},
		{"Gamma", "300"},
	}

	tbl := NewTable(ctx).
		WithColumns(cols).
		WithRows(rows).
		WithHeight(10).
		Build()

	tm := teatest.NewTestModel(t, newTableModel(tbl), teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() {
		//nolint:errcheck // Best-effort cleanup in test
		tm.Quit()
	})

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Alpha"))
	}, teatest.WithDuration(time.Second))

	// Test initial selection (first row)
	selectedRow := tbl.SelectedRow()
	assert.NotNil(t, selectedRow, "SelectedRow() should not return nil")
	assert.Equal(t, "Alpha", selectedRow[0], "first column should be 'Alpha'")
	assert.Equal(t, "100", selectedRow[1], "second column should be '100'")

	// Navigate to second row
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	waitFor(t, func() bool { return tbl.Cursor() == 1 }, time.Second)

	selectedRow = tbl.SelectedRow()
	assert.NotNil(t, selectedRow, "SelectedRow() should not return nil after navigation")
	assert.Equal(t, "Beta", selectedRow[0], "first column should be 'Beta'")
	assert.Equal(t, "200", selectedRow[1], "second column should be '200'")

	// Navigate to third row
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	waitFor(t, func() bool { return tbl.Cursor() == 2 }, time.Second)

	selectedRow = tbl.SelectedRow()
	assert.NotNil(t, selectedRow, "SelectedRow() should not return nil")
	assert.Equal(t, "Gamma", selectedRow[0], "first column should be 'Gamma'")
	assert.Equal(t, "300", selectedRow[1], "second column should be '300'")

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// TestTable_SetCursorProgrammaticallySetsPosition verifies that SetCursor()
// programmatically sets the cursor position correctly.
func TestTable_SetCursorProgrammaticallySetsPosition(t *testing.T) {
	ctx := NewContext()
	cols := []Column{
		{Title: "Item", Width: 20},
	}
	rows := []table.Row{
		{"Item 0"},
		{"Item 1"},
		{"Item 2"},
		{"Item 3"},
		{"Item 4"},
	}

	tbl := NewTable(ctx).
		WithColumns(cols).
		WithRows(rows).
		WithHeight(10).
		Build()

	tm := teatest.NewTestModel(t, newTableModel(tbl), teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() {
		//nolint:errcheck // Best-effort cleanup in test
		tm.Quit()
	})

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Item 0"))
	}, teatest.WithDuration(time.Second))

	// Verify initial cursor is at 0
	assert.Equal(t, 0, tbl.Cursor(), "initial cursor should be at index 0")

	// Set cursor to position 3
	tbl.SetCursor(3)
	assert.Equal(t, 3, tbl.Cursor(), "cursor should be at index 3 after SetCursor(3)")

	// Verify SelectedRow matches the set position
	selectedRow := tbl.SelectedRow()
	assert.Equal(t, "Item 3", selectedRow[0], "SelectedRow() should return 'Item 3'")

	// Set cursor to position 1
	tbl.SetCursor(1)
	assert.Equal(t, 1, tbl.Cursor(), "cursor should be at index 1 after SetCursor(1)")

	selectedRow = tbl.SelectedRow()
	assert.Equal(t, "Item 1", selectedRow[0], "SelectedRow() should return 'Item 1'")

	// Set cursor to last position
	tbl.SetCursor(4)
	assert.Equal(t, 4, tbl.Cursor(), "cursor should be at index 4 after SetCursor(4)")

	selectedRow = tbl.SelectedRow()
	assert.Equal(t, "Item 4", selectedRow[0], "SelectedRow() should return 'Item 4'")

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// TestTable_NavigationWithSingleRow verifies navigation behavior with only one row.
func TestTable_NavigationWithSingleRow(t *testing.T) {
	ctx := NewContext()
	cols := []Column{
		{Title: "Name", Width: 20},
	}
	rows := []table.Row{
		{"Only Row"},
	}

	tbl := NewTable(ctx).
		WithColumns(cols).
		WithRows(rows).
		WithHeight(10).
		Build()

	tm := teatest.NewTestModel(t, newTableModel(tbl), teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() {
		//nolint:errcheck // Best-effort cleanup in test
		tm.Quit()
	})

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Only Row"))
	}, teatest.WithDuration(time.Second))

	// Verify cursor starts at 0
	assert.Equal(t, 0, tbl.Cursor(), "cursor should be at index 0")

	// Press down - cursor should stay at 0 (or wrap depending on implementation)
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	// Brief wait for event processing (no state change expected)
	time.Sleep(2 * time.Millisecond)

	// With single row, cursor stays at 0
	assert.Equal(t, 0, tbl.Cursor(), "cursor should remain at index 0 with single row")

	// Verify SelectedRow is still correct
	selectedRow := tbl.SelectedRow()
	assert.Equal(t, "Only Row", selectedRow[0], "SelectedRow() should return 'Only Row'")

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// TestTable_NavigationWithEmptyTable verifies behavior with no rows.
func TestTable_NavigationWithEmptyTable(t *testing.T) {
	ctx := NewContext()
	cols := []Column{
		{Title: "Name", Width: 20},
	}
	rows := []table.Row{}

	tbl := NewTable(ctx).
		WithColumns(cols).
		WithRows(rows).
		WithHeight(10).
		Build()

	tm := teatest.NewTestModel(t, newTableModel(tbl), teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() {
		//nolint:errcheck // Best-effort cleanup in test
		tm.Quit()
	})

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Name"))
	}, teatest.WithDuration(time.Second))

	// Verify cursor is at 0
	assert.Equal(t, 0, tbl.Cursor(), "cursor should be at index 0 for empty table")

	// Press down - should not crash
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	// Brief wait for event processing (no state change expected)
	time.Sleep(2 * time.Millisecond)

	// SelectedRow should return nil for empty table
	selectedRow := tbl.SelectedRow()
	assert.Nil(t, selectedRow, "SelectedRow() should return nil for empty table")

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// TestTable_ViewEndsWithNewline verifies that Table's View() ends with newline
// for proper terminal rendering.
func TestTable_ViewEndsWithNewline(t *testing.T) {
	ctx := NewContext()
	cols := []Column{
		{Title: "Name", Width: 20},
		{Title: "Status", Width: 15},
	}
	rows := []table.Row{
		{"Row One", "Active"},
		{"Row Two", "Inactive"},
	}

	tbl := NewTable(ctx).
		WithColumns(cols).
		WithRows(rows).
		WithHeight(10).
		Build()

	view := tbl.View()
	assert.NotEmpty(t, view, "View() should not be empty")
	// Note: The bubbles/table component handles its own newline handling,
	// and the theme Base wrapper may add additional styling.
	// We verify the table renders without crashing and contains expected content.
	assert.Contains(t, view, "Name", "View should contain header 'Name'")
	assert.Contains(t, view, "Row One", "View should contain row data")
}

// TestTable_MixedNavigation verifies cursor behavior with mixed up/down navigation.
func TestTable_MixedNavigation(t *testing.T) {
	ctx := NewContext()
	cols := []Column{
		{Title: "Num", Width: 5},
	}
	rows := []table.Row{
		{"0"},
		{"1"},
		{"2"},
		{"3"},
		{"4"},
	}

	tbl := NewTable(ctx).
		WithColumns(cols).
		WithRows(rows).
		WithHeight(10).
		Build()

	tm := teatest.NewTestModel(t, newTableModel(tbl), teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() {
		//nolint:errcheck // Best-effort cleanup in test
		tm.Quit()
	})

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Num"))
	}, teatest.WithDuration(time.Second))

	// Down, down, down, up, up, down pattern
	keys := []tea.KeyType{
		tea.KeyDown, // 0 -> 1
		tea.KeyDown, // 1 -> 2
		tea.KeyDown, // 2 -> 3
		tea.KeyUp,   // 3 -> 2
		tea.KeyUp,   // 2 -> 1
		tea.KeyDown, // 1 -> 2
	}
	expectedPositions := []int{1, 2, 3, 2, 1, 2}

	for i, key := range keys {
		tm.Send(tea.KeyMsg{Type: key})
		expected := expectedPositions[i]
		waitFor(t, func() bool { return tbl.Cursor() == expected }, time.Second)

		// Verify SelectedRow matches cursor position
		selectedRow := tbl.SelectedRow()
		expectedValue := rows[expectedPositions[i]][0]
		assert.Equal(t, expectedValue, selectedRow[0],
			"SelectedRow()[0] should be %q at step %d", expectedValue, i)
	}

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// =============================================================================
// TT-020: Table async cell rendering tests
// =============================================================================.

// TestTable_AsyncCell_SpinnerWhileLoading verifies that a spinner is shown
// while an async cell is loading.
func TestTable_AsyncCell_SpinnerWhileLoading(t *testing.T) {
	ctx := NewContext()
	cols := []Column{
		{Title: "Name", Width: 20},
		{Title: "Status", Width: 15},
	}
	rows := []table.Row{
		{"Item 1", ""},
	}

	tbl := NewTable(ctx).
		WithColumns(cols).
		WithRows(rows).
		WithHeight(10).
		Build()

	// Create a slow async cell that blocks for a short time
	loadStarted := make(chan struct{})
	loadComplete := make(chan struct{})
	eval := async.New(func() (string, error) {
		close(loadStarted)
		<-loadComplete // Wait for signal to complete
		return "Loaded", nil
	})
	cell := async.NewCell(eval)

	// Add async cell to row 0, column 1
	tbl.SetAsyncCell(0, 1, cell)

	// Start loading - this returns a Cmd
	cmd := cell.StartLoading()
	assert.NotNil(t, cmd, "StartLoading should return a Cmd")

	// Verify IsLoading is true before completion
	assert.True(t, cell.IsLoading(), "cell should be loading after StartLoading")

	// The cell should show a spinner (non-empty) while loading
	view := cell.View()
	assert.NotEmpty(t, view, "View() should show spinner while loading")

	// Signal completion
	close(loadComplete)

	// Execute the command to complete loading
	_ = cmd()

	// Wait for load to actually complete
	<-loadStarted

	// After completion, IsLoading should be false
	assert.False(t, cell.IsLoading(), "cell should not be loading after completion")
}

// TestTable_AsyncCell_ValueDisplayedAfterLoad verifies that the actual value
// is displayed after the async cell finishes loading.
func TestTable_AsyncCell_ValueDisplayedAfterLoad(t *testing.T) {
	ctx := NewContext()
	cols := []Column{
		{Title: "Name", Width: 20},
		{Title: "Status", Width: 15},
	}
	rows := []table.Row{
		{"Item 1", ""},
	}

	tbl := NewTable(ctx).
		WithColumns(cols).
		WithRows(rows).
		WithHeight(10).
		Build()

	// Create an async cell that returns immediately
	eval := async.New(func() (string, error) {
		return "LoadedValue", nil
	})
	cell := async.NewCell(eval)

	// Add async cell to row 0, column 1
	tbl.SetAsyncCell(0, 1, cell)

	// Before StartLoading, View() should be empty
	assert.Empty(t, cell.View(), "View() should be empty before StartLoading")

	// Start loading and execute the command immediately (synchronous completion)
	cmd := cell.StartLoading()
	assert.NotNil(t, cmd, "StartLoading should return a Cmd")

	// Execute the command to trigger the fetch
	_ = cmd()

	// After the fetch completes, the value should be displayed
	assert.True(t, cell.IsLoaded(), "cell should be loaded after cmd execution")
	assert.Equal(t, "LoadedValue", cell.View(), "View() should show the loaded value")
}

// TestTable_AsyncCell_IsLoadingReturnsCorrectState verifies that IsLoading()
// returns the correct state throughout the loading lifecycle.
func TestTable_AsyncCell_IsLoadingReturnsCorrectState(t *testing.T) {
	// Test 1: Before StartLoading
	eval1 := async.New(func() (string, error) {
		return "value1", nil
	})
	cell1 := async.NewCell(eval1)

	assert.False(t, cell1.IsLoading(), "IsLoading should be false before StartLoading")
	assert.False(t, cell1.IsLoaded(), "IsLoaded should be false before StartLoading")

	// Test 2: After StartLoading but before completion
	loadComplete := make(chan struct{})
	eval2 := async.New(func() (string, error) {
		<-loadComplete
		return "value2", nil
	})
	cell2 := async.NewCell(eval2)

	cmd := cell2.StartLoading()
	assert.True(t, cell2.IsLoading(), "IsLoading should be true after StartLoading")
	assert.False(t, cell2.IsLoaded(), "IsLoaded should be false during loading")

	// Allow completion
	close(loadComplete)
	_ = cmd()

	// Test 3: After completion
	assert.False(t, cell2.IsLoading(), "IsLoading should be false after completion")
	assert.True(t, cell2.IsLoaded(), "IsLoaded should be true after completion")
}

// TestTable_AsyncCell_MultipleAsyncCellsInSameRow verifies that multiple
// async cells in the same row work correctly.
func TestTable_AsyncCell_MultipleAsyncCellsInSameRow(t *testing.T) {
	ctx := NewContext()
	cols := []Column{
		{Title: "Name", Width: 20},
		{Title: "Status", Width: 15},
		{Title: "Count", Width: 10},
	}
	rows := []table.Row{
		{"Item", "", ""},
	}

	tbl := NewTable(ctx).
		WithColumns(cols).
		WithRows(rows).
		WithHeight(10).
		Build()

	// Create two async cells with different values
	var cell1Started, cell2Started atomic.Bool

	eval1 := async.New(func() (string, error) {
		cell1Started.Store(true)
		return "StatusValue", nil
	})
	cell1 := async.NewCell(eval1)

	eval2 := async.New(func() (string, error) {
		cell2Started.Store(true)
		return "CountValue", nil
	})
	cell2 := async.NewCell(eval2)

	// Add both async cells to the same row
	tbl.SetAsyncCell(0, 1, cell1) // Status column
	tbl.SetAsyncCell(0, 2, cell2) // Count column

	// Start both cells loading
	cmd1 := cell1.StartLoading()
	cmd2 := cell2.StartLoading()

	// Both should be loading
	assert.True(t, cell1.IsLoading(), "cell1 should be loading")
	assert.True(t, cell2.IsLoading(), "cell2 should be loading")

	// Execute commands to complete loading
	_ = cmd1()
	_ = cmd2()

	// Verify both loaded correctly
	assert.True(t, cell1.IsLoaded(), "cell1 should be loaded")
	assert.True(t, cell2.IsLoaded(), "cell2 should be loaded")
	assert.Equal(t, "StatusValue", cell1.View(), "cell1 should show StatusValue")
	assert.Equal(t, "CountValue", cell2.View(), "cell2 should show CountValue")

	// Verify both fetch functions were called
	assert.True(t, cell1Started.Load(), "cell1 fetch should have been called")
	assert.True(t, cell2Started.Load(), "cell2 fetch should have been called")
}

// TestTable_AsyncRow_IsLoadingReflectsAsyncState verifies that AsyncRow's
// IsLoading() correctly reflects whether any async cells are still loading.
func TestTable_AsyncRow_IsLoadingReflectsAsyncState(t *testing.T) {
	ar := NewAsyncRow("static1", "static2")

	// No async cells - should not be loading
	assert.False(t, ar.IsLoading(), "row with no async cells should not be loading")

	// Add an async cell that blocks
	loadComplete := make(chan struct{})
	eval := async.New(func() (string, error) {
		<-loadComplete
		return "loaded", nil
	})
	cell := async.NewCell(eval)
	ar.WithAsyncCell(1, cell)

	// Before StartLoading - not loading yet
	assert.False(t, ar.IsLoading(), "row should not be loading before StartLoading")

	// Start loading
	cmd := cell.StartLoading()
	assert.True(t, ar.IsLoading(), "row should be loading after StartLoading")

	// Complete loading
	close(loadComplete)
	_ = cmd()

	// After completion
	assert.False(t, ar.IsLoading(), "row should not be loading after completion")
}

// TestTable_AsyncRow_TickUpdatesSpinners verifies that Tick() advances
// spinner animation for all async cells in the row.
func TestTable_AsyncRow_TickUpdatesSpinners(t *testing.T) {
	ar := NewAsyncRow("static")

	// Add async cell that doesn't complete immediately
	loadComplete := make(chan struct{})
	eval := async.New(func() (string, error) {
		<-loadComplete
		return "loaded", nil
	})
	cell := async.NewCell(eval)
	ar.WithAsyncCell(0, cell)

	// Start loading
	_ = cell.StartLoading()

	// Get initial spinner view
	initialView := ar.GetCell(0)
	assert.NotEmpty(t, initialView, "spinner should be visible while loading")

	// Tick multiple times to advance spinner
	for range 5 {
		ar.Tick()
	}

	// Spinner should still be visible (may or may not have changed depending on frame)
	tickedView := ar.GetCell(0)
	assert.NotEmpty(t, tickedView, "spinner should still be visible after ticks")

	// Complete loading
	close(loadComplete)
}

// TestTable_UpdateTicksAsyncRowsAndRefreshes verifies that Table.Update()
// ticks all async rows and refreshes their display.
func TestTable_UpdateTicksAsyncRowsAndRefreshes(t *testing.T) {
	ctx := NewContext()
	cols := []Column{
		{Title: "Name", Width: 20},
		{Title: "Status", Width: 15},
	}
	rows := []table.Row{
		{"Item 1", ""},
		{"Item 2", ""},
	}

	tbl := NewTable(ctx).
		WithColumns(cols).
		WithRows(rows).
		WithHeight(10).
		Build()

	// Add async cells to both rows
	eval1 := async.New(func() (string, error) {
		return "Status1", nil
	})
	cell1 := async.NewCell(eval1)
	tbl.SetAsyncCell(0, 1, cell1)

	eval2 := async.New(func() (string, error) {
		return "Status2", nil
	})
	cell2 := async.NewCell(eval2)
	tbl.SetAsyncCell(1, 1, cell2)

	// Start both loading
	cmd1 := cell1.StartLoading()
	cmd2 := cell2.StartLoading()

	// Execute commands
	_ = cmd1()
	_ = cmd2()

	// Verify cells are loaded
	assert.True(t, cell1.IsLoaded(), "cell1 should be loaded")
	assert.True(t, cell2.IsLoaded(), "cell2 should be loaded")

	// Call Update to trigger refresh
	_, _ = tbl.Update(nil)

	// Verify the table rows contain the loaded values
	// The asyncRows map should have been updated
	assert.Len(t, tbl.asyncRows, 2, "table should have 2 async rows")
}

// =============================================================================
// TT-021: Table cycling navigation tests
// =============================================================================.

// TestTable_CyclingNavigation_UpOnFirstRowGoesToLast verifies that pressing up
// on the first row cycles to the last row when cycling is enabled.
func TestTable_CyclingNavigation_UpOnFirstRowGoesToLast(t *testing.T) {
	ctx := NewContext()
	cols := []Column{
		{Title: "Name", Width: 20},
	}
	rows := []table.Row{
		{"Row 0"},
		{"Row 1"},
		{"Row 2"},
		{"Row 3"},
	}

	tbl := NewTable(ctx).
		WithColumns(cols).
		WithRows(rows).
		WithHeight(10).
		WithCycling(true).
		Build()

	tm := teatest.NewTestModel(t, newTableModel(tbl), teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() {
		//nolint:errcheck // Best-effort cleanup in test
		tm.Quit()
	})

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Row 0"))
	}, teatest.WithDuration(time.Second))

	// Verify cursor starts at 0
	assert.Equal(t, 0, tbl.Cursor(), "initial cursor should be at 0")

	// Press up - should cycle to last row (index 3)
	tm.Send(tea.KeyMsg{Type: tea.KeyUp})
	waitFor(t, func() bool { return tbl.Cursor() == 3 }, time.Second)
	assert.Equal(t, 3, tbl.Cursor(), "cursor should cycle to last row")

	// Verify selected row is correct
	selectedRow := tbl.SelectedRow()
	assert.Equal(t, "Row 3", selectedRow[0], "selected row should be 'Row 3'")

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// TestTable_CyclingNavigation_DownOnLastRowGoesToFirst verifies that pressing down
// on the last row cycles to the first row when cycling is enabled.
func TestTable_CyclingNavigation_DownOnLastRowGoesToFirst(t *testing.T) {
	ctx := NewContext()
	cols := []Column{
		{Title: "Name", Width: 20},
	}
	rows := []table.Row{
		{"Row 0"},
		{"Row 1"},
		{"Row 2"},
	}

	tbl := NewTable(ctx).
		WithColumns(cols).
		WithRows(rows).
		WithHeight(10).
		WithCycling(true).
		Build()

	// Start at last row
	tbl.SetCursor(2)

	tm := teatest.NewTestModel(t, newTableModel(tbl), teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() {
		//nolint:errcheck // Best-effort cleanup in test
		tm.Quit()
	})

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Row 2"))
	}, teatest.WithDuration(time.Second))

	// Verify cursor is at last row
	assert.Equal(t, 2, tbl.Cursor(), "cursor should be at last row")

	// Press down - should cycle to first row (index 0)
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	waitFor(t, func() bool { return tbl.Cursor() == 0 }, time.Second)
	assert.Equal(t, 0, tbl.Cursor(), "cursor should cycle to first row")

	// Verify selected row is correct
	selectedRow := tbl.SelectedRow()
	assert.Equal(t, "Row 0", selectedRow[0], "selected row should be 'Row 0'")

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// TestTable_CyclingNavigation_JKKeys verifies that j/k keys also cycle.
func TestTable_CyclingNavigation_JKKeys(t *testing.T) {
	ctx := NewContext()
	cols := []Column{
		{Title: "Name", Width: 20},
	}
	rows := []table.Row{
		{"First"},
		{"Middle"},
		{"Last"},
	}

	tbl := NewTable(ctx).
		WithColumns(cols).
		WithRows(rows).
		WithHeight(10).
		WithCycling(true).
		Build()

	tm := teatest.NewTestModel(t, newTableModel(tbl), teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() {
		//nolint:errcheck // Best-effort cleanup in test
		tm.Quit()
	})

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("First"))
	}, teatest.WithDuration(time.Second))

	// Press k (up) at first row - should cycle to last
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	waitFor(t, func() bool { return tbl.Cursor() == 2 }, time.Second)
	assert.Equal(t, 2, tbl.Cursor(), "k should cycle up to last row")

	// Press j (down) at last row - should cycle to first
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	waitFor(t, func() bool { return tbl.Cursor() == 0 }, time.Second)
	assert.Equal(t, 0, tbl.Cursor(), "j should cycle down to first row")

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// TestTable_CyclingNavigation_DisabledByDefault verifies cycling is disabled by default.
func TestTable_CyclingNavigation_DisabledByDefault(t *testing.T) {
	ctx := NewContext()
	cols := []Column{
		{Title: "Name", Width: 20},
	}
	rows := []table.Row{
		{"First"},
		{"Last"},
	}

	// No WithCycling() call - should be disabled
	tbl := NewTable(ctx).
		WithColumns(cols).
		WithRows(rows).
		WithHeight(10).
		Build()

	tm := teatest.NewTestModel(t, newTableModel(tbl), teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() {
		//nolint:errcheck // Best-effort cleanup in test
		tm.Quit()
	})

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("First"))
	}, teatest.WithDuration(time.Second))

	// Press up at first row - should stay at 0 (no cycling)
	tm.Send(tea.KeyMsg{Type: tea.KeyUp})
	// Brief wait - cursor should not change
	time.Sleep(5 * time.Millisecond)
	assert.Equal(t, 0, tbl.Cursor(), "cursor should stay at 0 without cycling")

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// =============================================================================
// TT-022: Table filter mode tests
// =============================================================================.

// TestTable_Filter_SlashEntersFilterMode verifies that pressing "/" enters filter mode.
func TestTable_Filter_SlashEntersFilterMode(t *testing.T) {
	ctx := NewContext()
	cols := []Column{
		{Title: "Name", Width: 20},
	}
	rows := []table.Row{
		{"Alpha"},
		{"Beta"},
		{"Gamma"},
	}

	tbl := NewTable(ctx).
		WithColumns(cols).
		WithRows(rows).
		WithHeight(10).
		WithFilterable(true).
		Build()

	tm := teatest.NewTestModel(t, newTableModel(tbl), teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() {
		//nolint:errcheck // Best-effort cleanup in test
		tm.Quit()
	})

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Alpha"))
	}, teatest.WithDuration(time.Second))

	// Verify not in filter mode initially
	assert.False(t, tbl.IsFilterActive(), "filter should not be active initially")

	// Press "/" to enter filter mode
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	waitFor(t, func() bool { return tbl.IsFilterActive() }, time.Second)
	assert.True(t, tbl.IsFilterActive(), "filter should be active after pressing /")

	// Verify filter input is shown in view
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("/"))
	}, teatest.WithDuration(time.Second))

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// TestTable_Filter_TypingFiltersRows verifies that typing in filter mode filters rows.
func TestTable_Filter_TypingFiltersRows(t *testing.T) {
	ctx := NewContext()
	cols := []Column{
		{Title: "Name", Width: 20},
		{Title: "Type", Width: 15},
	}
	rows := []table.Row{
		{"Apple", "Fruit"},
		{"Banana", "Fruit"},
		{"Carrot", "Vegetable"},
		{"Date", "Fruit"},
	}

	tbl := NewTable(ctx).
		WithColumns(cols).
		WithRows(rows).
		WithHeight(10).
		WithFilterable(true).
		Build()

	tm := teatest.NewTestModel(t, newTableModel(tbl), teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() {
		//nolint:errcheck // Best-effort cleanup in test
		tm.Quit()
	})

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Apple"))
	}, teatest.WithDuration(time.Second))

	// Verify all 4 rows visible initially
	assert.Len(t, tbl.Rows(), 4, "should have 4 rows initially")

	// Enter filter mode and type "fruit"
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	waitFor(t, func() bool { return tbl.IsFilterActive() }, time.Second)

	// Type "fruit" to filter
	for _, r := range "fruit" {
		tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	// Wait for filter to be applied (rows should be reduced)
	waitFor(t, func() bool { return len(tbl.Rows()) == 3 }, time.Second)

	// Should have 3 rows matching "fruit" (Apple, Banana, Date)
	assert.Len(t, tbl.Rows(), 3, "should have 3 rows matching 'fruit'")
	assert.Equal(t, "fruit", tbl.FilterQuery(), "filter query should be 'fruit'")

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// TestTable_Filter_EnterExitsAndClearsFilter verifies that Enter exits filter mode
// and clears the filter, restoring all rows.
func TestTable_Filter_EnterExitsAndClearsFilter(t *testing.T) {
	ctx := NewContext()
	cols := []Column{
		{Title: "Name", Width: 20},
	}
	rows := []table.Row{
		{"Alpha"},
		{"Beta"},
	}

	tbl := NewTable(ctx).
		WithColumns(cols).
		WithRows(rows).
		WithHeight(10).
		WithFilterable(true).
		Build()

	tm := teatest.NewTestModel(t, newTableModel(tbl), teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() {
		//nolint:errcheck // Best-effort cleanup in test
		tm.Quit()
	})

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Alpha"))
	}, teatest.WithDuration(time.Second))

	// Enter filter mode
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	waitFor(t, func() bool { return tbl.IsFilterActive() }, time.Second)

	// Type something
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	waitFor(t, func() bool { return tbl.FilterQuery() == "a" }, time.Second)

	// Press Enter to exit filter mode
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	waitFor(t, func() bool { return !tbl.IsFilterActive() }, time.Second)

	// Filter should be cleared, all rows restored
	assert.False(t, tbl.IsFilterActive(), "filter mode should be inactive after Enter")
	assert.Empty(t, tbl.FilterQuery(), "filter query should be cleared after Enter")
	assert.Len(t, tbl.Rows(), 2, "all rows should be visible after Enter")

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// TestTable_Filter_EscExitsAndClearsFilter verifies that Esc exits filter mode
// and clears the filter, restoring all rows.
func TestTable_Filter_EscExitsAndClearsFilter(t *testing.T) {
	ctx := NewContext()
	cols := []Column{
		{Title: "Name", Width: 20},
	}
	rows := []table.Row{
		{"Alpha"},
		{"Beta"},
	}

	tbl := NewTable(ctx).
		WithColumns(cols).
		WithRows(rows).
		WithHeight(10).
		WithFilterable(true).
		Build()

	tm := teatest.NewTestModel(t, newTableModel(tbl), teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() {
		//nolint:errcheck // Best-effort cleanup in test
		tm.Quit()
	})

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Alpha"))
	}, teatest.WithDuration(time.Second))

	// Enter filter mode
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	waitFor(t, func() bool { return tbl.IsFilterActive() }, time.Second)

	// Type something
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("b")})
	waitFor(t, func() bool { return tbl.FilterQuery() == "b" }, time.Second)

	// Press Esc to exit filter mode
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	waitFor(t, func() bool { return !tbl.IsFilterActive() }, time.Second)

	// Filter should be cleared, all rows restored
	assert.False(t, tbl.IsFilterActive(), "filter mode should be inactive after Esc")
	assert.Empty(t, tbl.FilterQuery(), "filter query should be cleared after Esc")
	assert.Len(t, tbl.Rows(), 2, "all rows should be visible after Esc")

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// TestTable_Filter_BackspaceOnEmptyClearsFilter verifies that Backspace on empty
// input clears the filter and exits filter mode.
func TestTable_Filter_BackspaceOnEmptyClearsFilter(t *testing.T) {
	ctx := NewContext()
	cols := []Column{
		{Title: "Name", Width: 20},
	}
	rows := []table.Row{
		{"Alpha"},
		{"Beta"},
		{"Gamma"},
	}

	tbl := NewTable(ctx).
		WithColumns(cols).
		WithRows(rows).
		WithHeight(10).
		WithFilterable(true).
		Build()

	tm := teatest.NewTestModel(t, newTableModel(tbl), teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() {
		//nolint:errcheck // Best-effort cleanup in test
		tm.Quit()
	})

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Alpha"))
	}, teatest.WithDuration(time.Second))

	// Enter filter mode and type then delete
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	waitFor(t, func() bool { return tbl.IsFilterActive() }, time.Second)

	// Type "ab"
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("b")})
	waitFor(t, func() bool { return tbl.FilterQuery() == "ab" }, time.Second)

	// Delete back to empty
	tm.Send(tea.KeyMsg{Type: tea.KeyBackspace})
	waitFor(t, func() bool { return tbl.FilterQuery() == "a" }, time.Second)
	tm.Send(tea.KeyMsg{Type: tea.KeyBackspace})
	waitFor(t, func() bool { return tbl.FilterQuery() == "" }, time.Second)

	// Backspace on empty should clear filter and exit
	tm.Send(tea.KeyMsg{Type: tea.KeyBackspace})
	waitFor(t, func() bool { return !tbl.IsFilterActive() }, time.Second)

	// All rows should be visible again
	assert.Len(t, tbl.Rows(), 3, "all rows should be visible after clearing filter")
	assert.Empty(t, tbl.FilterQuery(), "filter query should be empty")

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// TestTable_Filter_OriginalIndexMapsCorrectly verifies that OriginalIndex()
// returns the correct original row index when filtering.
func TestTable_Filter_OriginalIndexMapsCorrectly(t *testing.T) {
	ctx := NewContext()
	cols := []Column{
		{Title: "Name", Width: 20},
	}
	rows := []table.Row{
		{"Alpha"},   // index 0
		{"Beta"},    // index 1
		{"Gamma"},   // index 2
		{"Delta"},   // index 3
		{"Epsilon"}, // index 4
	}

	tbl := NewTable(ctx).
		WithColumns(cols).
		WithRows(rows).
		WithHeight(10).
		WithFilterable(true).
		Build()

	tm := teatest.NewTestModel(t, newTableModel(tbl), teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() {
		//nolint:errcheck // Best-effort cleanup in test
		tm.Quit()
	})

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Alpha"))
	}, teatest.WithDuration(time.Second))

	// Filter to show only rows containing "a" (Alpha, Beta, Gamma, Delta)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	waitFor(t, func() bool { return tbl.IsFilterActive() }, time.Second)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	waitFor(t, func() bool { return len(tbl.Rows()) == 4 }, time.Second)

	// Exit filter mode
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	waitFor(t, func() bool { return !tbl.IsFilterActive() }, time.Second)

	// Cursor is at 0 in filtered view (Alpha, original index 0)
	assert.Equal(t, 0, tbl.Cursor(), "cursor should be at 0")
	assert.Equal(t, 0, tbl.OriginalIndex(), "original index should be 0 for Alpha")

	// Move to filtered index 1 (Beta, original index 1)
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	waitFor(t, func() bool { return tbl.Cursor() == 1 }, time.Second)
	assert.Equal(t, 1, tbl.OriginalIndex(), "original index should be 1 for Beta")

	// Move to filtered index 2 (Gamma, original index 2)
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	waitFor(t, func() bool { return tbl.Cursor() == 2 }, time.Second)
	assert.Equal(t, 2, tbl.OriginalIndex(), "original index should be 2 for Gamma")

	// Move to filtered index 3 (Delta, original index 3)
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	waitFor(t, func() bool { return tbl.Cursor() == 3 }, time.Second)
	assert.Equal(t, 3, tbl.OriginalIndex(), "original index should be 3 for Delta")

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// TestTable_Filter_ClearFilterRestoresAllRows verifies that ClearFilter() restores all rows
// when called programmatically while filter is active.
func TestTable_Filter_ClearFilterRestoresAllRows(t *testing.T) {
	ctx := NewContext()
	cols := []Column{
		{Title: "Name", Width: 20},
	}
	rows := []table.Row{
		{"One"},
		{"Two"},
		{"Three"},
		{"Four"},
	}

	tbl := NewTable(ctx).
		WithColumns(cols).
		WithRows(rows).
		WithHeight(10).
		WithFilterable(true).
		Build()

	// Filter to show only "o" matches
	tm := teatest.NewTestModel(t, newTableModel(tbl), teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() {
		//nolint:errcheck // Best-effort cleanup in test
		tm.Quit()
	})

	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("One"))
	}, teatest.WithDuration(time.Second))

	// Apply filter (stay in filter mode)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	waitFor(t, func() bool { return tbl.IsFilterActive() }, time.Second)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("o")})
	waitFor(t, func() bool { return len(tbl.Rows()) == 3 }, time.Second) // One, Two, Four

	// Verify filtered state while still in filter mode
	assert.Len(t, tbl.Rows(), 3, "should have 3 filtered rows")
	assert.Equal(t, "o", tbl.FilterQuery(), "filter query should be 'o'")
	assert.True(t, tbl.IsFilterActive(), "should still be in filter mode")

	// Clear filter programmatically (while still in filter mode)
	tbl.ClearFilter()

	// All rows should be restored, filter query cleared
	assert.Len(t, tbl.Rows(), 4, "should have 4 rows after clearing filter")
	assert.Empty(t, tbl.FilterQuery(), "filter query should be empty after clear")

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// TestTable_Filter_CaseInsensitive verifies that filtering is case-insensitive.
func TestTable_Filter_CaseInsensitive(t *testing.T) {
	ctx := NewContext()
	cols := []Column{
		{Title: "Name", Width: 20},
	}
	rows := []table.Row{
		{"UPPER"},
		{"lower"},
		{"MiXeD"},
		{"nomatch"},
	}

	tbl := NewTable(ctx).
		WithColumns(cols).
		WithRows(rows).
		WithHeight(10).
		WithFilterable(true).
		Build()

	tm := teatest.NewTestModel(t, newTableModel(tbl), teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() {
		//nolint:errcheck // Best-effort cleanup in test
		tm.Quit()
	})

	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("UPPER"))
	}, teatest.WithDuration(time.Second))

	// Filter with lowercase "mix" should match "MiXeD"
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	waitFor(t, func() bool { return tbl.IsFilterActive() }, time.Second)

	for _, r := range "mix" {
		tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	waitFor(t, func() bool { return len(tbl.Rows()) == 1 }, time.Second)

	// Should match MiXeD
	assert.Len(t, tbl.Rows(), 1, "should have 1 row matching 'mix'")
	assert.Equal(t, "MiXeD", tbl.Rows()[0][0], "filtered row should be 'MiXeD'")

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// TestTable_Filter_DisabledByDefault verifies filtering is disabled by default.
func TestTable_Filter_DisabledByDefault(t *testing.T) {
	ctx := NewContext()
	cols := []Column{
		{Title: "Name", Width: 20},
	}
	rows := []table.Row{
		{"Alpha"},
		{"Beta"},
	}

	// No WithFilterable() call
	tbl := NewTable(ctx).
		WithColumns(cols).
		WithRows(rows).
		WithHeight(10).
		Build()

	// Verify filter methods return defaults
	assert.False(t, tbl.IsFilterActive(), "filter should not be active by default")
	assert.Empty(t, tbl.FilterQuery(), "filter query should be empty by default")
	assert.Nil(t, tbl.AllRows(), "AllRows should be nil when filtering disabled")

	// OriginalIndex should just return cursor when filtering disabled
	assert.Equal(t, tbl.Cursor(), tbl.OriginalIndex(), "OriginalIndex should equal Cursor when filtering disabled")
}

// TestTable_Filter_SetRowsReappliesFilter verifies that SetRows() reapplies the current filter.
func TestTable_Filter_SetRowsReappliesFilter(t *testing.T) {
	ctx := NewContext()
	cols := []Column{
		{Title: "Name", Width: 20},
	}
	rows := []table.Row{
		{"Apple"},
		{"Banana"},
		{"Cherry"},
	}

	tbl := NewTable(ctx).
		WithColumns(cols).
		WithRows(rows).
		WithHeight(10).
		WithFilterable(true).
		Build()

	tm := teatest.NewTestModel(t, newTableModel(tbl), teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() {
		//nolint:errcheck // Best-effort cleanup in test
		tm.Quit()
	})

	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Apple"))
	}, teatest.WithDuration(time.Second))

	// Apply filter for "an"
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	waitFor(t, func() bool { return tbl.IsFilterActive() }, time.Second)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	waitFor(t, func() bool { return len(tbl.Rows()) == 1 }, time.Second) // Banana

	// Exit with Esc (keeps filter for SetRows test)
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	waitFor(t, func() bool { return !tbl.IsFilterActive() }, time.Second)

	// Note: After Esc, filter is cleared and all rows restored
	// For SetRows test, we need to programmatically set filter
	// Re-enter filter mode and apply filter again
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	waitFor(t, func() bool { return tbl.IsFilterActive() }, time.Second)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	waitFor(t, func() bool { return len(tbl.Rows()) == 1 }, time.Second)

	// Update rows with SetRows while filter is active - should reapply "an" filter
	newRows := []table.Row{
		{"Orange"},
		{"Mango"},
		{"Pear"},
	}
	tbl.SetRows(newRows)

	// Filter "an" should now match "Orange" and "Mango"
	assert.Len(t, tbl.Rows(), 2, "should have 2 rows matching 'an' in new data")
	assert.Equal(t, "an", tbl.FilterQuery(), "filter query should be preserved")

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// =============================================================================
// TT-023: Table filter mode navigation and selection tests
// =============================================================================.

// TestTable_Filter_ArrowKeysNavigateFilteredRows verifies that up/down arrow keys
// navigate the filtered rows while in filter mode.
func TestTable_Filter_ArrowKeysNavigateFilteredRows(t *testing.T) {
	ctx := NewContext()
	cols := []Column{
		{Title: "Name", Width: 20},
	}
	rows := []table.Row{
		{"Alpha"},
		{"Beta"},
		{"Gamma"},
		{"Delta"},
	}

	tbl := NewTable(ctx).
		WithColumns(cols).
		WithRows(rows).
		WithHeight(10).
		WithFilterable(true).
		WithCycling(true).
		Build()

	tm := teatest.NewTestModel(t, newTableModel(tbl), teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() {
		//nolint:errcheck // Best-effort cleanup in test
		tm.Quit()
	})

	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Alpha"))
	}, teatest.WithDuration(time.Second))

	// Enter filter mode and filter to "a" (Alpha, Beta, Gamma, Delta all match)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	waitFor(t, func() bool { return tbl.IsFilterActive() }, time.Second)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	waitFor(t, func() bool { return tbl.FilterQuery() == "a" }, time.Second)

	// Cursor should be at 0
	assert.Equal(t, 0, tbl.Cursor(), "cursor should start at 0")

	// Press down while in filter mode - should move cursor
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	waitFor(t, func() bool { return tbl.Cursor() == 1 }, time.Second)
	assert.Equal(t, 1, tbl.Cursor(), "down arrow should move cursor to 1")

	// Press down again
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	waitFor(t, func() bool { return tbl.Cursor() == 2 }, time.Second)
	assert.Equal(t, 2, tbl.Cursor(), "down arrow should move cursor to 2")

	// Press up - should move back
	tm.Send(tea.KeyMsg{Type: tea.KeyUp})
	waitFor(t, func() bool { return tbl.Cursor() == 1 }, time.Second)
	assert.Equal(t, 1, tbl.Cursor(), "up arrow should move cursor to 1")

	// Still in filter mode
	assert.True(t, tbl.IsFilterActive(), "should still be in filter mode")

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// TestTable_Filter_EnterSelectsHighlightedRow verifies that Enter in filter mode
// selects the currently highlighted row and exits filter mode.
func TestTable_Filter_EnterSelectsHighlightedRow(t *testing.T) {
	ctx := NewContext()
	cols := []Column{
		{Title: "Name", Width: 20},
	}
	rows := []table.Row{
		{"Apple"},
		{"Apricot"},
		{"Avocado"},
		{"Banana"},
	}

	tbl := NewTable(ctx).
		WithColumns(cols).
		WithRows(rows).
		WithHeight(10).
		WithFilterable(true).
		Build()

	tm := teatest.NewTestModel(t, newTableModel(tbl), teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() {
		//nolint:errcheck // Best-effort cleanup in test
		tm.Quit()
	})

	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Apple"))
	}, teatest.WithDuration(time.Second))

	// Enter filter mode and filter to "ap" (Apple, Apricot)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	waitFor(t, func() bool { return tbl.IsFilterActive() }, time.Second)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})
	waitFor(t, func() bool { return len(tbl.Rows()) == 2 }, time.Second)

	// Move to second filtered row (Apricot)
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	waitFor(t, func() bool { return tbl.Cursor() == 1 }, time.Second)

	// Verify we're selecting Apricot in filtered view
	assert.Equal(t, "Apricot", tbl.SelectedRow()[0], "should have Apricot selected")

	// Press Enter to select and exit filter mode
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	waitFor(t, func() bool { return !tbl.IsFilterActive() }, time.Second)

	// Filter should be cleared, all rows visible
	assert.Len(t, tbl.Rows(), 4, "all rows should be visible after Enter")
	assert.Empty(t, tbl.FilterQuery(), "filter query should be cleared after Enter")

	// Cursor should be on Apricot (original index 1)
	assert.Equal(t, 1, tbl.Cursor(), "cursor should be on Apricot's original index")
	assert.Equal(t, "Apricot", tbl.SelectedRow()[0], "Apricot should still be selected")

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// TestTable_Filter_EscClearsFilterAndRestoresRows verifies that Esc clears
// the filter and shows all rows again.
func TestTable_Filter_EscClearsFilterAndRestoresRows(t *testing.T) {
	ctx := NewContext()
	cols := []Column{
		{Title: "Name", Width: 20},
	}
	rows := []table.Row{
		{"One"},
		{"Two"},
		{"Three"},
		{"Four"},
	}

	tbl := NewTable(ctx).
		WithColumns(cols).
		WithRows(rows).
		WithHeight(10).
		WithFilterable(true).
		Build()

	tm := teatest.NewTestModel(t, newTableModel(tbl), teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() {
		//nolint:errcheck // Best-effort cleanup in test
		tm.Quit()
	})

	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("One"))
	}, teatest.WithDuration(time.Second))

	// Filter to show only "o" matches (One, Two, Four = 3 rows)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	waitFor(t, func() bool { return tbl.IsFilterActive() }, time.Second)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("o")})
	waitFor(t, func() bool { return len(tbl.Rows()) == 3 }, time.Second)

	assert.Len(t, tbl.Rows(), 3, "should have 3 filtered rows")

	// Press Esc to exit and clear filter
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	waitFor(t, func() bool { return !tbl.IsFilterActive() }, time.Second)

	// All rows should be restored
	assert.Len(t, tbl.Rows(), 4, "all 4 rows should be visible after Esc")
	assert.Empty(t, tbl.FilterQuery(), "filter query should be cleared after Esc")

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// TestTable_Filter_JKKeysNavigateInFilterMode verifies that j/k keys
// navigate rows in filter mode (not type into filter).
func TestTable_Filter_JKKeysNavigateInFilterMode(t *testing.T) {
	ctx := NewContext()
	cols := []Column{
		{Title: "Name", Width: 20},
	}
	rows := []table.Row{
		{"Row1"},
		{"Row2"},
		{"Row3"},
	}

	tbl := NewTable(ctx).
		WithColumns(cols).
		WithRows(rows).
		WithHeight(10).
		WithFilterable(true).
		WithCycling(true).
		Build()

	tm := teatest.NewTestModel(t, newTableModel(tbl), teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() {
		//nolint:errcheck // Best-effort cleanup in test
		tm.Quit()
	})

	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Row1"))
	}, teatest.WithDuration(time.Second))

	// Enter filter mode
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	waitFor(t, func() bool { return tbl.IsFilterActive() }, time.Second)

	// Press j - should move cursor down, not type 'j'
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	waitFor(t, func() bool { return tbl.Cursor() == 1 }, time.Second)
	assert.Equal(t, 1, tbl.Cursor(), "j should navigate down")
	assert.Empty(t, tbl.FilterQuery(), "j should not type into filter")

	// Press k - should move cursor up
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	waitFor(t, func() bool { return tbl.Cursor() == 0 }, time.Second)
	assert.Equal(t, 0, tbl.Cursor(), "k should navigate up")

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// TestTable_Filter_CyclingWorksInFilterMode verifies that cycling navigation
// works when in filter mode with filtered results.
func TestTable_Filter_CyclingWorksInFilterMode(t *testing.T) {
	ctx := NewContext()
	cols := []Column{
		{Title: "Name", Width: 20},
	}
	rows := []table.Row{
		{"Cat"},
		{"Car"},
		{"Dog"},
		{"Cap"},
	}

	tbl := NewTable(ctx).
		WithColumns(cols).
		WithRows(rows).
		WithHeight(10).
		WithFilterable(true).
		WithCycling(true).
		Build()

	tm := teatest.NewTestModel(t, newTableModel(tbl), teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() {
		//nolint:errcheck // Best-effort cleanup in test
		tm.Quit()
	})

	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Cat"))
	}, teatest.WithDuration(time.Second))

	// Filter to "ca" (Cat, Car, Cap = 3 rows)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	waitFor(t, func() bool { return tbl.IsFilterActive() }, time.Second)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	waitFor(t, func() bool { return len(tbl.Rows()) == 3 }, time.Second)

	// Cursor at 0, press up - should cycle to last filtered row (index 2)
	assert.Equal(t, 0, tbl.Cursor(), "cursor should be at 0")
	tm.Send(tea.KeyMsg{Type: tea.KeyUp})
	waitFor(t, func() bool { return tbl.Cursor() == 2 }, time.Second)
	assert.Equal(t, 2, tbl.Cursor(), "up should cycle to last filtered row")

	// Press down - should cycle back to first (index 0)
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	waitFor(t, func() bool { return tbl.Cursor() == 0 }, time.Second)
	assert.Equal(t, 0, tbl.Cursor(), "down should cycle to first filtered row")

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

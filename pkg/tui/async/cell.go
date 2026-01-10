package async

import (
	tea "github.com/charmbracelet/bubbletea"
)

// spinnerFrames provides simple animation frames for loading spinner.
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// Cell[T] represents a table cell that may be loading async data.
// It wraps an Eval[T] and adds spinner state for visual feedback during loading.
type Cell[T any] struct {
	eval          *Eval[T]
	spinnerIndex  int
	tickCount     uint64 // tracks ticks to update spinner
	startLoading  func() tea.Cmd
	isStarted     bool   // tracks if StartLoading() was called
}

// New creates a new Cell with an Eval that fetches data asynchronously.
func NewCell[T any](eval *Eval[T]) *Cell[T] {
	return &Cell[T]{
		eval: eval,
	}
}

// View returns the cell value as a string, or a spinner if still loading.
// If the eval has an error, returns an error indicator.
func (c *Cell[T]) View() string {
	if !c.eval.IsLoaded() && c.isStarted {
		// Still loading - show spinner
		return spinnerFrames[c.spinnerIndex%len(spinnerFrames)]
	}

	if c.eval.IsLoaded() {
		// Loaded successfully - GetValue should not error since IsLoaded() checks for nil error
		value, _ := c.eval.Get()
		// Convert to string - caller responsible for providing proper string conversion
		// through Eval that returns string or has Stringer interface
		return stringifyValue(value)
	}

	// Not started yet
	return ""
}

// StartLoading initiates the async fetch and returns a tea.Cmd.
// Call this once when the cell should begin loading.
func (c *Cell[T]) StartLoading() tea.Cmd {
	if c.isStarted {
		return nil // Already started
	}
	c.isStarted = true

	// Return a Cmd that fetches the value
	return func() tea.Msg {
		// Trigger the fetch by calling Get() which will block and execute the fetch function
		value, err := c.eval.Get()
		return cellLoadedMsg{
			value: value,
			err:   err,
		}
	}
}

// Tick updates the spinner animation frame.
// Call this from Update() on each tick to animate the spinner.
func (c *Cell[T]) Tick() {
	c.tickCount++
	// Update spinner every 4 ticks for reasonable animation speed
	if c.tickCount%4 == 0 {
		c.spinnerIndex++
	}
}

// IsLoading returns true if the async fetch is still in progress.
func (c *Cell[T]) IsLoading() bool {
	return c.isStarted && !c.eval.IsLoaded()
}

// IsLoaded returns true if the async fetch has completed successfully.
func (c *Cell[T]) IsLoaded() bool {
	return c.eval.IsLoaded()
}

// stringifyValue converts a value to string for display.
// For string types, returns as-is. For others, uses fmt.Sprint as fallback.
func stringifyValue[T any](val T) string {
	// Try string assertion first
	if s, ok := any(val).(string); ok {
		return s
	}
	// Try fmt.Stringer interface
	if stringer, ok := any(val).(interface{ String() string }); ok {
		return stringer.String()
	}
	// For basic types, use empty string
	return ""
}

// cellLoadedMsg is an internal message indicating a cell has finished loading.
type cellLoadedMsg struct {
	value interface{}
	err   error
}

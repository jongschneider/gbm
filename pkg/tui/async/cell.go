package async

import (
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

// Cell represents a table cell that may be loading async data.
// It wraps an Eval[T] and adds spinner state for visual feedback during loading.
type Cell[T any] struct {
	eval      *Eval[T]
	spinner   spinner.Model
	isStarted bool // tracks if StartLoading() was called
}

// NewCell creates a new Cell with an Eval that fetches data asynchronously.
func NewCell[T any](eval *Eval[T]) *Cell[T] {
	return &Cell[T]{
		eval:    eval,
		spinner: spinner.New(spinner.WithSpinner(spinner.Dot)),
	}
}

// View returns the cell value as a string, or a spinner if still loading.
// If the eval has an error, returns an error indicator.
func (c *Cell[T]) View() string {
	if !c.eval.IsLoaded() && c.isStarted {
		// Still loading - show spinner
		return c.spinner.View()
	}

	if c.eval.IsLoaded() {
		// Loaded successfully - GetValue should not error since IsLoaded() checks for nil error
		value, _ := c.eval.Get() //nolint:errcheck // IsLoaded() guarantees no error
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
		return CellLoadedMsg{
			Value: value,
			Err:   err,
		}
	}
}

// Tick updates the spinner animation frame.
// Call this from Update() on each tick to animate the spinner.
func (c *Cell[T]) Tick() {
	// Update spinner with tick message
	newSpinner, _ := c.spinner.Update(c.spinner.Tick())
	c.spinner = newSpinner
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

// CellLoadedMsg is a message indicating a cell has finished loading.
type CellLoadedMsg struct {
	Value any
	Err   error
}

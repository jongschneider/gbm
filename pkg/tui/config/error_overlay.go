package config

import (
	"fmt"
	"gbm/pkg/tui"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ErrorOverlay renders a navigable list of validation errors as a modal
// overlay. Pressing enter on an error triggers a jump to the offending
// field (switching tab and scrolling), and esc closes the overlay.
type ErrorOverlay struct {
	theme  *tui.Theme
	errors []ValidationError
	keys   ErrorsOverlayKeyMap
	cursor int
}

// NewErrorOverlay creates an ErrorOverlay with the given errors.
func NewErrorOverlay(errs []ValidationError, theme *tui.Theme) *ErrorOverlay {
	if theme == nil {
		theme = tui.DefaultTheme()
	}
	return &ErrorOverlay{
		theme:  theme,
		errors: errs,
		keys:   NewErrorsOverlayKeys(),
	}
}

// SetErrors replaces the current error list and resets the cursor.
func (o *ErrorOverlay) SetErrors(errs []ValidationError) {
	o.errors = errs
	o.cursor = 0
}

// Errors returns the current list of validation errors.
func (o *ErrorOverlay) Errors() []ValidationError {
	return o.errors
}

// Cursor returns the currently selected error index.
func (o *ErrorOverlay) Cursor() int {
	return o.cursor
}

// HasErrors reports whether there are any validation errors.
func (o *ErrorOverlay) HasErrors() bool {
	return len(o.errors) > 0
}

// SelectedError returns the currently selected validation error.
// Returns a zero-value ValidationError if the list is empty.
func (o *ErrorOverlay) SelectedError() ValidationError {
	if len(o.errors) == 0 {
		return ValidationError{FieldIndex: -1}
	}
	if o.cursor >= len(o.errors) {
		o.cursor = len(o.errors) - 1
	}
	return o.errors[o.cursor]
}

// ErrorOverlayAction indicates what the caller should do after a key press:
//   - ErrorActionNone: no action needed (navigation handled internally)
//   - ErrorActionClose: close the overlay (esc)
//   - ErrorActionJump: jump to the selected error's field (enter)
type ErrorOverlayAction int

const (
	// ErrorActionNone means the overlay handled the key internally.
	ErrorActionNone ErrorOverlayAction = iota
	// ErrorActionClose means the overlay should be closed.
	ErrorActionClose
	// ErrorActionJump means the caller should jump to the selected error's field.
	ErrorActionJump
)

// HandleKey processes a raw tea.KeyMsg string and returns the appropriate action.
func (o *ErrorOverlay) HandleKey(msg interface{ String() string }) ErrorOverlayAction {
	s := msg.String()
	switch s {
	case "esc":
		return ErrorActionClose
	case "enter":
		if len(o.errors) > 0 {
			return ErrorActionJump
		}
		return ErrorActionClose
	case "up", "k":
		o.moveUp()
		return ErrorActionNone
	case "down", "j":
		o.moveDown()
		return ErrorActionNone
	}
	return ErrorActionNone
}

// moveUp moves the cursor up, wrapping to the last item.
func (o *ErrorOverlay) moveUp() {
	if len(o.errors) == 0 {
		return
	}
	o.cursor = (o.cursor - 1 + len(o.errors)) % len(o.errors)
}

// moveDown moves the cursor down, wrapping to the first item.
func (o *ErrorOverlay) moveDown() {
	if len(o.errors) == 0 {
		return
	}
	o.cursor = (o.cursor + 1) % len(o.errors)
}

// View renders the error overlay as a centered, bordered modal.
func (o *ErrorOverlay) View(width, height int) string {
	innerWidth := max(width-4, 20)
	innerHeight := max(height-4, 1)

	lines := o.renderLines(innerWidth)

	// Clamp to viewport.
	if len(lines) > innerHeight {
		// Ensure the selected error is visible.
		scrollOffset := 0
		selectedLine := o.cursor + 1 // +1 for header
		if selectedLine >= scrollOffset+innerHeight {
			scrollOffset = selectedLine - innerHeight + 1
		}
		if scrollOffset > len(lines)-innerHeight {
			scrollOffset = len(lines) - innerHeight
		}
		if scrollOffset < 0 {
			scrollOffset = 0
		}
		lines = lines[scrollOffset : scrollOffset+innerHeight]
	}

	// Pad to fill the viewport.
	for len(lines) < innerHeight {
		lines = append(lines, "")
	}

	content := strings.Join(lines, "\n")

	// Title and close hint.
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(o.theme.ErrorAccent)

	closeHint := lipgloss.NewStyle().
		Foreground(o.theme.Muted).
		Render("esc to close . enter to jump")

	header := titleStyle.Render(fmt.Sprintf("Validation Errors (%d)", len(o.errors))) +
		"  " + closeHint

	boxStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(o.theme.ErrorAccent).
		Padding(0, 1).
		Width(innerWidth + 2)

	box := boxStyle.Render(header + "\n" + content)

	centered := lipgloss.NewStyle().
		Width(width).
		Height(height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(box)

	return centered
}

// renderLines produces the list of error lines with cursor indicator.
func (o *ErrorOverlay) renderLines(width int) []string {
	if len(o.errors) == 0 {
		noErrStyle := lipgloss.NewStyle().
			Foreground(o.theme.SuccessAccent)
		return []string{noErrStyle.Render("No validation errors.")}
	}

	cursorStyle := lipgloss.NewStyle().
		Foreground(o.theme.Cursor).
		Bold(true)

	selectedStyle := lipgloss.NewStyle().
		Foreground(o.theme.ErrorAccent).
		Bold(true)

	normalStyle := lipgloss.NewStyle().
		Foreground(o.theme.Muted)

	tabStyle := lipgloss.NewStyle().
		Foreground(o.theme.Accent)

	var lines []string
	for i, e := range o.errors {
		tabName := tabLabels[e.Tab]
		prefix := "  "
		var line string
		if i == o.cursor {
			prefix = cursorStyle.Render("> ")
			line = prefix +
				tabStyle.Render("["+tabName+"]") + " " +
				selectedStyle.Render(e.FieldLabel+": "+e.Message)
		} else {
			line = prefix +
				tabStyle.Render("["+tabName+"]") + " " +
				normalStyle.Render(e.FieldLabel+": "+e.Message)
		}

		// Truncate if too wide.
		if lipgloss.Width(line) > width {
			raw := stripAnsi(line)
			if len(raw) > width-3 {
				raw = raw[:width-3] + "..."
			}
			line = raw
		}

		lines = append(lines, line)
	}

	return lines
}

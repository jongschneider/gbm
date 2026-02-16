// Package fields provides concrete Field implementations for wizard forms.
package fields

import (
	"gbm/pkg/tui"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ValidationOverlay displays a list of validation errors with dismissal instructions.
// It handles Escape and Enter keys to dismiss.
type ValidationOverlay struct {
	theme  *tui.Theme
	title  string
	errors []string
	width  int
	height int
}

// NewValidationOverlay creates a new validation error overlay.
func NewValidationOverlay(errors []string) *ValidationOverlay {
	return &ValidationOverlay{
		theme:  tui.DefaultTheme(),
		errors: errors,
		title:  "Validation Errors",
	}
}

// WithTheme sets the theme for the overlay.
func (v *ValidationOverlay) WithTheme(theme *tui.Theme) *ValidationOverlay {
	if theme != nil {
		v.theme = theme
	}
	return v
}

// WithTitle sets a custom title for the overlay.
func (v *ValidationOverlay) WithTitle(title string) *ValidationOverlay {
	v.title = title
	return v
}

// WithWidth sets the width for the overlay.
func (v *ValidationOverlay) WithWidth(width int) *ValidationOverlay {
	v.width = width
	return v
}

// WithHeight sets the height for the overlay.
func (v *ValidationOverlay) WithHeight(height int) *ValidationOverlay {
	v.height = height
	return v
}

// Init implements tea.Model.
func (v *ValidationOverlay) Init() tea.Cmd {
	return nil
}

// ValidationOverlayDismissedMsg is sent when the overlay is dismissed.
type ValidationOverlayDismissedMsg struct{}

// Update implements tea.Model.
func (v *ValidationOverlay) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "enter":
			return v, func() tea.Msg {
				return ValidationOverlayDismissedMsg{}
			}
		}
	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
	}
	return v, nil
}

// View implements tea.Model.
func (v *ValidationOverlay) View() string {
	var b strings.Builder

	// Title style - error color
	titleStyle := v.theme.Focused.Error.Bold(true)

	// Error item style
	errorStyle := v.theme.Focused.Error

	// Description style for help text
	helpStyle := v.theme.Blurred.Description

	// Border style for the box
	boxWidth := 60
	if v.width > 0 && v.width < boxWidth+4 {
		boxWidth = v.width - 4
	}

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(v.theme.ErrorAccent).
		Padding(1, 2).
		Width(boxWidth)

	// Build content
	var content strings.Builder
	content.WriteString(titleStyle.Render(v.title))
	content.WriteString("\n\n")

	// List errors with bullet points
	for _, err := range v.errors {
		content.WriteString(errorStyle.Render("• " + err))
		content.WriteString("\n")
	}

	content.WriteString("\n")
	content.WriteString(helpStyle.Render("Press Escape or Enter to dismiss"))

	b.WriteString(boxStyle.Render(content.String()))
	return b.String()
}

// Errors returns the list of error messages.
func (v *ValidationOverlay) Errors() []string {
	return v.errors
}

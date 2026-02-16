package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// validationOverlay displays a list of validation errors with dismissal instructions.
// It handles Escape and Enter keys to dismiss. This is a tui-package-local overlay
// to avoid import cycles with tui/fields.
type validationOverlay struct {
	theme  *Theme
	title  string
	errors []string
	width  int
}

// newValidationOverlay creates a new validation error overlay.
func newValidationOverlay(errs []string, theme *Theme, width int) *validationOverlay {
	if theme == nil {
		theme = DefaultTheme()
	}
	return &validationOverlay{
		theme:  theme,
		title:  "Validation Errors",
		errors: errs,
		width:  width,
	}
}

// ValidationOverlayDismissedMsg is sent when the validation overlay is dismissed.
type ValidationOverlayDismissedMsg struct{}

// Update processes key messages for the validation overlay.
func (v *validationOverlay) Update(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc", "enter":
		return func() tea.Msg {
			return ValidationOverlayDismissedMsg{}
		}
	}
	return nil
}

// View renders the validation error overlay.
func (v *validationOverlay) View() string {
	var b strings.Builder

	titleStyle := v.theme.Focused.Error.Bold(true)
	errorStyle := v.theme.Focused.Error
	helpStyle := v.theme.Blurred.Description

	boxWidth := 60
	if v.width > 0 && v.width < boxWidth+4 {
		boxWidth = v.width - 4
	}

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(v.theme.ErrorAccent).
		Padding(1, 2).
		Width(boxWidth)

	var content strings.Builder
	content.WriteString(titleStyle.Render(v.title))
	content.WriteString("\n\n")

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
func (v *validationOverlay) Errors() []string {
	return v.errors
}

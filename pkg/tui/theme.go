package tui

import "github.com/charmbracelet/lipgloss"

// FieldStyles contains lipgloss styles for rendering a field in a particular state.
type FieldStyles struct {
	Title       lipgloss.Style
	Description lipgloss.Style
	Input       lipgloss.Style
	Error       lipgloss.Style
}

// Theme contains visual styles for TUI components.
type Theme struct {
	Focused FieldStyles
	Blurred FieldStyles
}

// DefaultTheme returns a Theme with sensible default styles that work
// on both light and dark terminal backgrounds.
func DefaultTheme() *Theme {
	return &Theme{
		Focused: FieldStyles{
			// Bright, bold styles for focused fields
			Title:       lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86")).Underline(true),
			Description: lipgloss.NewStyle().Foreground(lipgloss.Color("246")).Italic(true),
			Input:       lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Background(lipgloss.Color("238")).Bold(true),
			Error:       lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true),
		},
		Blurred: FieldStyles{
			// Muted styles for blurred fields
			Title:       lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
			Description: lipgloss.NewStyle().Foreground(lipgloss.Color("235")),
			Input:       lipgloss.NewStyle().Foreground(lipgloss.Color("243")),
			Error:       lipgloss.NewStyle().Foreground(lipgloss.Color("124")),
		},
	}
}

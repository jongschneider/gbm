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
			Title:       lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212")),
			Description: lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
			Input:       lipgloss.NewStyle().Foreground(lipgloss.Color("255")),
			Error:       lipgloss.NewStyle().Foreground(lipgloss.Color("196")),
		},
		Blurred: FieldStyles{
			Title:       lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
			Description: lipgloss.NewStyle().Foreground(lipgloss.Color("238")),
			Input:       lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
			Error:       lipgloss.NewStyle().Foreground(lipgloss.Color("196")),
		},
	}
}

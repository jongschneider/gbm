package tui

import "github.com/charmbracelet/lipgloss"

// FieldStyles contains lipgloss styles for rendering a field in a particular state.
type FieldStyles struct {
	Title       lipgloss.Style
	Description lipgloss.Style
	Input       lipgloss.Style
	Error       lipgloss.Style
}

// TableStyles contains lipgloss styles for rendering table components.
type TableStyles struct {
	Header   lipgloss.Style
	Selected lipgloss.Style
	Cell     lipgloss.Style
	Border   lipgloss.Style
	Base     lipgloss.Style // Base style with borders for entire table
}

// Theme contains visual styles for TUI components.
type Theme struct {
	Focused FieldStyles
	Blurred FieldStyles
	Table   TableStyles
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
		Table: TableStyles{
			// Header style: gray border bottom, not bold (matching DefaultTableStyles)
			Header: lipgloss.NewStyle().
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("240")).
				BorderBottom(true).
				Bold(false),
			// Selected row: light yellow on purple (matching DefaultTableStyles)
			Selected: lipgloss.NewStyle().
				Foreground(lipgloss.Color("229")).
				Background(lipgloss.Color("57")).
				Bold(false),
			// Normal cell: default styling
			Cell: lipgloss.NewStyle(),
			// Border style: subtle gray
			Border: lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
			// Base style: border around entire table
			Base: lipgloss.NewStyle().
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("240")),
		},
	}
}

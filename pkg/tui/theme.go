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

	// Adaptive colors for use by overlays and other components
	Accent        lipgloss.AdaptiveColor // Primary accent (e.g., titles)
	Muted         lipgloss.AdaptiveColor // Muted text (e.g., descriptions)
	Highlight     lipgloss.AdaptiveColor // Highlighted elements (e.g., keys in help)
	Border        lipgloss.AdaptiveColor // Border color
	ErrorAccent   lipgloss.AdaptiveColor // Error color
	SuccessAccent lipgloss.AdaptiveColor // Success/confirm color
	SelectedFg    lipgloss.AdaptiveColor // Selected row foreground
	SelectedBg    lipgloss.AdaptiveColor // Selected row background
	InputBg       lipgloss.AdaptiveColor // Input field background when focused
	InputFg       lipgloss.AdaptiveColor // Input field foreground when focused
	BlurredMuted  lipgloss.AdaptiveColor // Very muted text for blurred elements
	Cursor        lipgloss.AdaptiveColor // Cursor indicator color
}

// DefaultTheme returns a Theme with sensible default styles that work
// on both light and dark terminal backgrounds using adaptive colors.
func DefaultTheme() *Theme {
	// Define adaptive colors that work on both light and dark backgrounds
	accent := lipgloss.AdaptiveColor{Light: "#0077b6", Dark: "#5fd7af"}       // Cyan/teal
	muted := lipgloss.AdaptiveColor{Light: "#6c757d", Dark: "#a8a8a8"}        // Gray
	highlight := lipgloss.AdaptiveColor{Light: "#b5651d", Dark: "#ffd75f"}    // Gold/amber
	border := lipgloss.AdaptiveColor{Light: "#adb5bd", Dark: "#585858"}       // Gray border
	errorColor := lipgloss.AdaptiveColor{Light: "#dc3545", Dark: "#ff5f5f"}   // Red
	successColor := lipgloss.AdaptiveColor{Light: "#198754", Dark: "#5faf5f"} // Green
	selectedFg := lipgloss.AdaptiveColor{Light: "#ffffff", Dark: "#ffffaf"}
	selectedBg := lipgloss.AdaptiveColor{Light: "#6f42c1", Dark: "#5f5faf"}
	inputBg := lipgloss.AdaptiveColor{Light: "#e9ecef", Dark: "#3a3a3a"}
	inputFg := lipgloss.AdaptiveColor{Light: "#212529", Dark: "#ffffff"}
	blurredMuted := lipgloss.AdaptiveColor{Light: "#868e96", Dark: "#6c6c6c"}
	cursor := lipgloss.AdaptiveColor{Light: "#d63384", Dark: "#ff87d7"} // Magenta/pink

	return &Theme{
		Accent:        accent,
		Muted:         muted,
		Highlight:     highlight,
		Border:        border,
		ErrorAccent:   errorColor,
		SuccessAccent: successColor,
		SelectedFg:    selectedFg,
		SelectedBg:    selectedBg,
		InputBg:       inputBg,
		InputFg:       inputFg,
		BlurredMuted:  blurredMuted,
		Cursor:        cursor,

		Focused: FieldStyles{
			// Bright, bold styles for focused fields
			Title:       lipgloss.NewStyle().Bold(true).Foreground(accent).Underline(true),
			Description: lipgloss.NewStyle().Foreground(muted).Italic(true),
			Input:       lipgloss.NewStyle().Foreground(inputFg).Background(inputBg).Bold(true),
			Error:       lipgloss.NewStyle().Foreground(errorColor).Bold(true),
		},
		Blurred: FieldStyles{
			// Muted styles for blurred fields
			Title:       lipgloss.NewStyle().Foreground(blurredMuted),
			Description: lipgloss.NewStyle().Foreground(blurredMuted),
			Input:       lipgloss.NewStyle().Foreground(muted),
			Error:       lipgloss.NewStyle().Foreground(errorColor),
		},
		Table: TableStyles{
			// Header style: gray border bottom, not bold
			Header: lipgloss.NewStyle().
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(border).
				BorderBottom(true).
				Bold(false),
			// Selected row: adaptive colors for both light and dark backgrounds
			Selected: lipgloss.NewStyle().
				Foreground(selectedFg).
				Background(selectedBg).
				Bold(false),
			// Normal cell: default styling
			Cell: lipgloss.NewStyle(),
			// Border style: subtle gray
			Border: lipgloss.NewStyle().Foreground(border),
			// Base style: border around entire table
			Base: lipgloss.NewStyle().
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(border),
		},
	}
}

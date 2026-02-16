package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Shortcut represents a keyboard shortcut with its key and description.
type Shortcut struct {
	Key         string
	Description string
}

// ShortcutGroup groups related shortcuts under a common heading.
type ShortcutGroup struct {
	Name      string
	Shortcuts []Shortcut
}

// HelpOverlay displays keyboard shortcuts organized by context.
// It handles Escape, '?', and Enter keys to dismiss.
type HelpOverlay struct {
	theme  *Theme
	groups []ShortcutGroup
	width  int
	height int
}

// NewHelpOverlay creates a new help overlay with default shortcuts.
func NewHelpOverlay() *HelpOverlay {
	return &HelpOverlay{
		theme:  DefaultTheme(),
		groups: defaultShortcuts(),
	}
}

// defaultShortcuts returns the standard keyboard shortcuts for the config TUI.
func defaultShortcuts() []ShortcutGroup {
	return []ShortcutGroup{
		{
			Name: "Sidebar",
			Shortcuts: []Shortcut{
				{Key: "j/k/↑/↓", Description: "Navigate sections"},
				{Key: "l/→/Enter", Description: "Focus content"},
				{Key: "r", Description: "Reset from file"},
				{Key: "q", Description: "Quit (save prompt if dirty)"},
				{Key: "?", Description: "This help screen"},
			},
		},
		{
			Name: "Content Pane",
			Shortcuts: []Shortcut{
				{Key: "Esc", Description: "Return to sidebar"},
				{Key: "Tab", Description: "Next field"},
				{Key: "Shift+Tab", Description: "Previous field"},
				{Key: "Space/h/l", Description: "Toggle Yes/No (confirm fields)"},
			},
		},
		{
			Name: "Table Forms (FileCopy, Worktrees)",
			Shortcuts: []Shortcut{
				{Key: "j/k/↑/↓", Description: "Navigate rows"},
				{Key: "a", Description: "Add new entry"},
				{Key: "e", Description: "Edit selected entry"},
				{Key: "d", Description: "Delete selected entry"},
			},
		},
		{
			Name: "Global",
			Shortcuts: []Shortcut{
				{Key: "Ctrl+S", Description: "Save all sections"},
				{Key: "Ctrl+C", Description: "Force quit"},
			},
		},
		{
			Name: "Modals",
			Shortcuts: []Shortcut{
				{Key: "y/Y", Description: "Confirm"},
				{Key: "n/N", Description: "Cancel"},
				{Key: "Esc", Description: "Close modal"},
				{Key: "Enter", Description: "Confirm/dismiss"},
			},
		},
	}
}

// WithTheme sets the theme for the overlay.
func (h *HelpOverlay) WithTheme(theme *Theme) *HelpOverlay {
	if theme != nil {
		h.theme = theme
	}
	return h
}

// WithWidth sets the width for the overlay.
func (h *HelpOverlay) WithWidth(width int) *HelpOverlay {
	h.width = width
	return h
}

// WithHeight sets the height for the overlay.
func (h *HelpOverlay) WithHeight(height int) *HelpOverlay {
	h.height = height
	return h
}

// WithGroups sets custom shortcut groups.
func (h *HelpOverlay) WithGroups(groups []ShortcutGroup) *HelpOverlay {
	h.groups = groups
	return h
}

// Init implements tea.Model.
func (h *HelpOverlay) Init() tea.Cmd {
	return nil
}

// HelpOverlayDismissedMsg is sent when the overlay is dismissed.
type HelpOverlayDismissedMsg struct{}

// Update implements tea.Model.
func (h *HelpOverlay) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "?", "enter":
			return h, func() tea.Msg {
				return HelpOverlayDismissedMsg{}
			}
		}
	case tea.WindowSizeMsg:
		h.width = msg.Width
		h.height = msg.Height
	}
	return h, nil
}

// View implements tea.Model.
func (h *HelpOverlay) View() string {
	// Title style - focused cyan
	titleStyle := h.theme.Focused.Title

	// Group header style - use theme's accent color for adaptive coloring
	groupStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(h.theme.Accent).
		MarginTop(1)

	// Key style - use theme's highlight color for adaptive coloring
	keyStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(h.theme.Highlight).
		Width(12)

	// Description style - muted
	descStyle := h.theme.Blurred.Description

	// Help text style
	helpTextStyle := h.theme.Blurred.Description.Italic(true)

	// Calculate box width
	boxWidth := 50
	if h.width > 0 && h.width < boxWidth+4 {
		boxWidth = h.width - 4
	}

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(h.theme.Border).
		Padding(1, 2).
		Width(boxWidth)

	// Build content
	var content strings.Builder
	content.WriteString(titleStyle.Render("Help - Keyboard Shortcuts"))
	content.WriteString("\n")

	for _, group := range h.groups {
		content.WriteString(groupStyle.Render(group.Name))
		content.WriteString("\n")

		for _, shortcut := range group.Shortcuts {
			line := keyStyle.Render(shortcut.Key) + descStyle.Render(shortcut.Description)
			content.WriteString(line)
			content.WriteString("\n")
		}
	}

	content.WriteString("\n")
	content.WriteString(helpTextStyle.Render("Press Escape, '?', or Enter to close"))

	return boxStyle.Render(content.String())
}

// Groups returns the shortcut groups.
func (h *HelpOverlay) Groups() []ShortcutGroup {
	return h.groups
}

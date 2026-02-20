package config

import (
	"fmt"
	"gbm/pkg/tui"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// helpEntry represents a single key-description pair in the help overlay.
type helpEntry struct {
	key  string
	desc string
}

// helpSection represents a titled group of key bindings in the help overlay.
type helpSection struct {
	title   string
	entries []helpEntry
}

// HelpOverlay renders a scrollable full-screen overlay listing all keybindings.
// It is opened with ? from browsing state and closed with ? or esc.
type HelpOverlay struct {
	theme    *tui.Theme
	sections []helpSection
	scroll   int // first visible line (0-based)
}

// NewHelpOverlay creates a new HelpOverlay with the standard keybinding reference.
func NewHelpOverlay(theme *tui.Theme) *HelpOverlay {
	return &HelpOverlay{
		theme:    theme,
		sections: buildHelpSections(),
	}
}

// HandleKey processes a raw tea.KeyMsg and returns true if the overlay should close.
func (h *HelpOverlay) HandleKey(msg interface{ String() string }, viewportHeight int) bool {
	s := msg.String()
	switch s {
	case "?", "esc":
		return true
	case "up", "k":
		if h.scroll > 0 {
			h.scroll--
		}
		return false
	case "down", "j":
		h.scrollDown(viewportHeight)
		return false
	}
	return false
}

// scrollDown advances the scroll position by one line, clamped to content bounds.
func (h *HelpOverlay) scrollDown(viewportHeight int) {
	totalLines := h.totalLines()
	maxScroll := max(totalLines-viewportHeight, 0)
	if h.scroll < maxScroll {
		h.scroll++
	}
}

// ResetScroll resets the scroll position to the top.
func (h *HelpOverlay) ResetScroll() {
	h.scroll = 0
}

// Scroll returns the current scroll offset.
func (h *HelpOverlay) Scroll() int {
	return h.scroll
}

// totalLines returns the total number of rendered lines across all sections.
func (h *HelpOverlay) totalLines() int {
	count := 0
	for i, sec := range h.sections {
		count++ // section title
		count++ // blank line after title (underline separator)
		count += len(sec.entries)
		if i < len(h.sections)-1 {
			count++ // blank line between sections
		}
	}
	return count
}

// View renders the help overlay content for the given viewport dimensions.
// The overlay is rendered as a bordered box with a title, containing
// the keybinding reference. Content scrolls if it exceeds the viewport.
func (h *HelpOverlay) View(width, height int) string {
	// Reserve space for the overlay border and title.
	// Top border (1) + title line (1) + bottom border (1) = 3 lines of chrome.
	innerWidth := max(
		// 2 chars padding on each side
		width-4, 20)
	innerHeight := max(
		// border + title chrome
		height-4, 1)

	lines := h.renderLines(innerWidth)

	// Apply scrolling.
	if h.scroll > len(lines) {
		h.scroll = max(len(lines)-1, 0)
	}
	visible := lines[h.scroll:]
	if len(visible) > innerHeight {
		visible = visible[:innerHeight]
	}

	// Pad to fill the viewport so the border extends fully.
	for len(visible) < innerHeight {
		visible = append(visible, "")
	}

	content := strings.Join(visible, "\n")

	// Build scroll indicator.
	scrollInfo := ""
	totalLines := len(lines)
	if totalLines > innerHeight {
		scrollInfo = fmt.Sprintf(" [%d/%d] ", h.scroll+1, totalLines)
	}

	// Render in a bordered box.
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(h.theme.Accent)

	title := titleStyle.Render("Keybinding Reference")

	boxStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(h.theme.Border).
		Padding(0, 1).
		Width(innerWidth + 2) // +2 for padding

	footer := ""
	if scrollInfo != "" {
		footerStyle := lipgloss.NewStyle().
			Foreground(h.theme.Muted)
		footer = footerStyle.Render(scrollInfo)
	}

	closeHint := lipgloss.NewStyle().
		Foreground(h.theme.Muted).
		Render("? or esc to close")

	header := title + "  " + closeHint
	if footer != "" {
		header += "  " + footer
	}

	box := boxStyle.Render(header + "\n" + content)

	// Center the box in the viewport.
	centered := lipgloss.NewStyle().
		Width(width).
		Height(height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(box)

	return centered
}

// renderLines produces the full content as a slice of styled lines.
func (h *HelpOverlay) renderLines(width int) []string {
	sectionStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(h.theme.Accent)

	keyStyle := lipgloss.NewStyle().
		Foreground(h.theme.Highlight).
		Width(20).
		Align(lipgloss.Left)

	descStyle := lipgloss.NewStyle().
		Foreground(h.theme.Muted)

	separatorStyle := lipgloss.NewStyle().
		Foreground(h.theme.Border)

	var lines []string
	for i, sec := range h.sections {
		// Section title with underline.
		lines = append(lines, sectionStyle.Render(sec.title))
		sepLen := min(len(sec.title)+4, width)
		lines = append(lines, separatorStyle.Render(strings.Repeat("\u2500", sepLen)))

		for _, entry := range sec.entries {
			line := keyStyle.Render(entry.key) + descStyle.Render(entry.desc)
			lines = append(lines, line)
		}

		// Blank line between sections (but not after the last one).
		if i < len(h.sections)-1 {
			lines = append(lines, "")
		}
	}
	return lines
}

// buildHelpSections constructs the help reference from the defined keybindings.
// Sections: Primary Keys, Vim Shortcuts, Overlay Keys.
func buildHelpSections() []helpSection {
	return []helpSection{
		{
			title: "Primary Keys",
			entries: []helpEntry{
				{key: "tab / shift-tab", desc: "next / prev section tab"},
				{key: "up / down", desc: "navigate fields"},
				{key: "e", desc: "edit field (toggle bool, open list)"},
				{key: "enter", desc: "save & quit"},
				{key: "s", desc: "save config"},
				{key: "r", desc: "reset field to saved value"},
				{key: "R", desc: "reset all fields"},
				{key: "/", desc: "open field search"},
				{key: "?", desc: "toggle help overlay"},
				{key: "a", desc: "add entry (rules / worktrees)"},
				{key: "d", desc: "delete entry"},
				{key: "q", desc: "quit (with dirty guard)"},
				{key: "ctrl-c", desc: "quit (cancels edit first)"},
			},
		},
		{
			title: "Vim Shortcuts",
			entries: []helpEntry{
				{key: "j / k", desc: "down / up (same as arrow keys)"},
				{key: "g", desc: "jump to first field"},
				{key: "G", desc: "jump to last field"},
				{key: "{ / }", desc: "jump to prev / next group"},
			},
		},
		{
			title: "Editing",
			entries: []helpEntry{
				{key: "enter", desc: "confirm edit"},
				{key: "esc", desc: "cancel edit"},
				{key: "ctrl-c", desc: "cancel edit"},
				{key: "ctrl-z", desc: "undo (Bubble Tea native)"},
			},
		},
		{
			title: "Search",
			entries: []helpEntry{
				{key: "(type)", desc: "filter fields by label"},
				{key: "esc", desc: "clear and close search"},
			},
		},
		{
			title: "List Overlay",
			entries: []helpEntry{
				{key: "up / down", desc: "select item"},
				{key: "a", desc: "add new item"},
				{key: "d", desc: "delete selected item"},
				{key: "enter", desc: "confirm changes"},
				{key: "esc", desc: "discard changes"},
			},
		},
		{
			title: "Editor Overlay",
			entries: []helpEntry{
				{key: "up / down", desc: "navigate fields"},
				{key: "e", desc: "edit focused field"},
				{key: "r", desc: "rename entry"},
				{key: "enter", desc: "confirm and close"},
				{key: "esc", desc: "cancel and discard"},
			},
		},
		{
			title: "Errors Overlay",
			entries: []helpEntry{
				{key: "up / down", desc: "select error"},
				{key: "enter", desc: "jump to error field"},
				{key: "esc", desc: "close overlay"},
			},
		},
	}
}

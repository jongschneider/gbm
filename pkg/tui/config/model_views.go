package config

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
)

// viewTooSmall renders the "terminal too small" message.
func (m *ConfigModel) viewTooSmall() string {
	msg := fmt.Sprintf(
		"Terminal too small (%dx%d). Minimum: %dx%d.",
		m.width, m.height, MinTermWidth, MinTermHeight,
	)
	style := lipgloss.NewStyle().
		Foreground(m.theme.ErrorAccent).
		Bold(true).
		Width(m.width).
		Align(lipgloss.Center)

	// Center vertically by padding with newlines.
	padding := ""
	if m.height > 1 {
		topPad := m.height / 2
		padding = strings.Repeat("\n", topPad)
	}
	return padding + style.Render(msg)
}

// viewTabBar renders the tab bar with active tab highlighting and error badges.
func (m *ConfigModel) viewTabBar() string {
	activeStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.theme.Accent).
		Padding(0, 1)

	inactiveStyle := lipgloss.NewStyle().
		Foreground(m.theme.Muted).
		Padding(0, 1)

	var tabs []string
	for i := range tabCount {
		label := tabLabels[i]

		// Add validation error badge.
		if m.tabBadges[i] {
			label += " (!)"
		}

		if SectionTab(i) == m.activeTab {
			label = "[" + label + "]"
			tabs = append(tabs, activeStyle.Render(label))
		} else {
			tabs = append(tabs, inactiveStyle.Render(label))
		}
	}

	bar := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)

	// Add a separator line under the tab bar.
	separator := lipgloss.NewStyle().
		Foreground(m.theme.Border).
		Width(m.width).
		Render(strings.Repeat("\u2500", m.width))

	return bar + "\n" + separator
}

// viewContent renders the main content area for the active tab.
// Delegates to the active section's View() method, falling back to
// blank lines when the section is not initialized.
// When editing, the focused field's row is replaced with the FieldRow's
// inline editing view (text input + description + error).
func (m *ConfigModel) viewContent() string {
	contentHeight := max(m.height-5, 1)
	section := m.activeSection()
	if section == nil {
		// Fallback for uninitialized state.
		return strings.Repeat("\n", contentHeight)
	}
	section.SetViewportHeight(contentHeight)
	section.SetWidth(m.width)

	// During editing, render the FieldRow's editing view for the focused row.
	if m.state == StateEditing {
		fr := m.activeFieldRow()
		if fr != nil {
			return m.viewContentEditing(section, fr, contentHeight)
		}
	}

	return section.View()
}

// viewContentEditing renders the section with the focused field replaced by
// the FieldRow's editing view (inline text input with description and error).
func (m *ConfigModel) viewContentEditing(
	section *SectionModel, fr *FieldRow, vpHeight int,
) string {
	rows := section.Rows()
	if len(rows) == 0 {
		return section.View()
	}

	focusIdx := section.FocusIndex()
	scrollOff := section.ScrollOffset()

	end := min(scrollOff+vpHeight, len(rows))

	var lines []string
	for i := scrollOff; i < end; i++ {
		if i == focusIdx {
			// Replace the focused field's line with the FieldRow editing view.
			lines = append(lines, fr.View())
		} else {
			lines = append(lines, section.RenderRow(rows, i))
		}
	}

	for len(lines) < vpHeight {
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}

// viewStatusBar renders the status bar with dirty count, file status,
// context-sensitive keybindings, and flash messages.
func (m *ConfigModel) viewStatusBar() string {
	separator := lipgloss.NewStyle().
		Foreground(m.theme.Border).
		Width(m.width).
		Render(strings.Repeat("\u2500", m.width))

	var parts []string

	// Dirty count or new file indicator.
	if m.isNewFile {
		newStyle := lipgloss.NewStyle().
			Foreground(m.theme.SuccessAccent).
			Bold(true)
		parts = append(parts, newStyle.Render("[new file]"))
	} else if count := m.dirty.DirtyCount(); count > 0 {
		dirtyStyle := lipgloss.NewStyle().
			Foreground(m.theme.Highlight).
			Bold(true)
		parts = append(parts, dirtyStyle.Render(fmt.Sprintf("[%d modified]", count)))
	}

	// Flash message takes priority over keybinding hints.
	if m.flashMessage != "" {
		flashStyle := lipgloss.NewStyle().
			Foreground(m.theme.SuccessAccent)
		parts = append(parts, flashStyle.Render(m.flashMessage))
	} else {
		// Context-sensitive keybinding hints.
		parts = append(parts, m.statusKeyHints())
	}

	// Description line: always reserved to avoid layout jank.
	descLine := ""
	if m.state == StateBrowsing && m.focusedFieldDesc != "" {
		descLine = lipgloss.NewStyle().
			Foreground(m.theme.Muted).
			Italic(true).
			Width(m.width).
			Render(m.focusedFieldDesc)
	}

	content := lipgloss.NewStyle().
		Width(m.width).
		Render(strings.Join(parts, "  "))

	return separator + "\n" + descLine + "\n" + content
}

// statusKeyHints returns the context-sensitive keybinding hints for the
// current state, showing 3-4 of the most relevant keys.
func (m *ConfigModel) statusKeyHints() string {
	hintStyle := lipgloss.NewStyle().Foreground(m.theme.Muted)
	keyStyle := lipgloss.NewStyle().Foreground(m.theme.Highlight)

	formatHint := func(b key.Binding) string {
		keys := b.Help().Key
		desc := b.Help().Desc
		return keyStyle.Render(keys) + " " + hintStyle.Render(desc)
	}

	sep := hintStyle.Render(" . ")

	switch m.state {
	case StateEditing:
		return formatHint(m.editingKeys.Confirm) + sep + formatHint(m.editingKeys.Cancel)

	default: // StateBrowsing
		// Build the edit verb based on focused field type.
		editVerb := EditHelpVerb(fieldTypeToString(m.focusedFieldType))
		editKey := keyStyle.Render("e") + " " + hintStyle.Render(editVerb)

		return strings.Join([]string{
			formatHint(m.browsingKeys.NextTab) +
				"/" + keyStyle.Render("S-tab") + " " + hintStyle.Render("section"),
			formatHint(m.browsingKeys.Up),
			editKey,
			formatHint(m.browsingKeys.Help),
			formatHint(m.browsingKeys.SaveQuit),
		}, sep)
	}
}

// viewOverwriteConfirm renders the overwrite confirmation overlay.
func (m *ConfigModel) viewOverwriteConfirm() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.theme.Highlight)

	bodyStyle := lipgloss.NewStyle().
		Foreground(m.theme.Muted)

	hintStyle := lipgloss.NewStyle().
		Foreground(m.theme.Accent)

	content := titleStyle.Render("File changed externally") + "\n\n" +
		bodyStyle.Render("The config file has been modified outside the editor.\nOverwrite with your changes?") + "\n\n" +
		hintStyle.Render("y") + bodyStyle.Render(" overwrite  ") +
		hintStyle.Render("n") + bodyStyle.Render("/") +
		hintStyle.Render("esc") + bodyStyle.Render(" cancel")

	innerWidth := max(m.width-4, 30)
	boxStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Highlight).
		Padding(1, 2).
		Width(innerWidth)

	box := boxStyle.Render(content)

	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(box)
}

// viewWriteError renders the write error overlay.
func (m *ConfigModel) viewWriteError() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.theme.ErrorAccent)

	bodyStyle := lipgloss.NewStyle().
		Foreground(m.theme.Muted)

	hintStyle := lipgloss.NewStyle().
		Foreground(m.theme.Accent)

	content := titleStyle.Render("Write Error") + "\n\n" +
		bodyStyle.Render(m.writeErrorMsg) + "\n\n" +
		hintStyle.Render("esc") + bodyStyle.Render(" close")

	innerWidth := max(m.width-4, 30)
	boxStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.ErrorAccent).
		Padding(1, 2).
		Width(innerWidth)

	box := boxStyle.Render(content)

	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(box)
}

// fieldTypeToString converts a FieldType to the string format expected by
// EditHelpVerb. This bridges the typed enum and the string-based verb lookup.
func fieldTypeToString(ft FieldType) string {
	switch ft {
	case String:
		return "string"
	case SensitiveString:
		return "sensitive_string"
	case Int:
		return "int"
	case Bool:
		return "bool"
	case StringList:
		return "string_list"
	case ObjectList:
		return "object_list"
	default:
		return "string"
	}
}

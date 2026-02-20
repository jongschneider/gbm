package config

import (
	"fmt"
	"gbm/pkg/tui"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// listOverlayState represents the current sub-state of the list overlay.
type listOverlayState int

const (
	// listBrowsing is the default state: navigating items with up/down.
	listBrowsing listOverlayState = iota
	// listAdding is active when the text input for a new item is shown.
	listAdding
	// listConfirmDelete is active when a delete confirmation prompt is shown.
	listConfirmDelete
	// listConfirmDiscard is active when a discard-changes confirmation is shown.
	listConfirmDiscard
)

// ListOverlayResultMsg is sent by the ListOverlay when the user commits
// or discards changes. The parent model uses this to update the field value.
type ListOverlayResultMsg struct {
	// Items contains the final list of strings after edits.
	// Nil if the user discarded changes.
	Items []string
	// Committed is true when the user pressed enter to confirm.
	Committed bool
}

// ListOverlay renders a centered modal for editing a string list field.
// It supports adding new items, deleting existing items with confirmation,
// and committing or discarding the full set of changes.
type ListOverlay struct {
	theme    *tui.Theme
	title    string
	keys     ListOverlayKeyMap
	items    []string
	original []string
	input    textinput.Model
	cursor   int
	width    int
	height   int
	state    listOverlayState
}

// NewListOverlay creates a ListOverlay for editing the given items.
// The title is displayed as a breadcrumb path (e.g., "JIRA > Filters > Status").
func NewListOverlay(title string, items []string, theme *tui.Theme) *ListOverlay {
	if theme == nil {
		theme = tui.DefaultTheme()
	}

	// Deep copy items so mutations don't affect the caller.
	copied := make([]string, len(items))
	copy(copied, items)

	origCopy := make([]string, len(items))
	copy(origCopy, items)

	ti := textinput.New()
	ti.Prompt = "> "
	ti.CharLimit = 256
	ti.Width = 30

	return &ListOverlay{
		theme:    theme,
		keys:     NewListOverlayKeys(),
		input:    ti,
		title:    title,
		items:    copied,
		original: origCopy,
		state:    listBrowsing,
	}
}

// SetSize updates the available viewport dimensions for rendering.
func (l *ListOverlay) SetSize(width, height int) {
	l.width = width
	l.height = height
}

// State returns the current sub-state of the overlay.
func (l *ListOverlay) State() listOverlayState {
	return l.state
}

// Cursor returns the current cursor position.
func (l *ListOverlay) Cursor() int {
	return l.cursor
}

// Items returns a copy of the current items list.
func (l *ListOverlay) Items() []string {
	cp := make([]string, len(l.items))
	copy(cp, l.items)
	return cp
}

// HasChanges reports whether the current items differ from the original.
func (l *ListOverlay) HasChanges() bool {
	if len(l.items) != len(l.original) {
		return true
	}
	for i := range l.items {
		if l.items[i] != l.original[i] {
			return true
		}
	}
	return false
}

// Update processes a tea.Msg and returns a result message if the overlay
// should close, or nil if it remains open. The returned tea.Cmd should be
// forwarded by the parent model.
func (l *ListOverlay) Update(msg tea.Msg) (*ListOverlayResultMsg, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return nil, nil
	}

	switch l.state {
	case listBrowsing:
		return l.handleBrowsing(keyMsg)
	case listAdding:
		return l.handleAdding(keyMsg)
	case listConfirmDelete:
		return l.handleConfirmDelete(keyMsg)
	case listConfirmDiscard:
		return l.handleConfirmDiscard(keyMsg)
	}

	return nil, nil
}

// handleBrowsing processes keys in the default browsing state.
func (l *ListOverlay) handleBrowsing(msg tea.KeyMsg) (*ListOverlayResultMsg, tea.Cmd) {
	switch {
	case key.Matches(msg, l.keys.Up):
		l.moveCursorUp()
		return nil, nil

	case key.Matches(msg, l.keys.Down):
		l.moveCursorDown()
		return nil, nil

	case key.Matches(msg, l.keys.Add):
		l.state = listAdding
		l.input.SetValue("")
		l.input.Focus()
		return nil, textinput.Blink

	case key.Matches(msg, l.keys.Delete):
		if len(l.items) == 0 {
			return nil, nil
		}
		l.state = listConfirmDelete
		return nil, nil

	case key.Matches(msg, l.keys.Confirm):
		result := &ListOverlayResultMsg{
			Items:     l.Items(),
			Committed: true,
		}
		return result, nil

	case key.Matches(msg, l.keys.Cancel):
		if !l.HasChanges() {
			result := &ListOverlayResultMsg{Committed: false}
			return result, nil
		}
		l.state = listConfirmDiscard
		return nil, nil
	}

	return nil, nil
}

// handleAdding processes keys while the add-item text input is active.
func (l *ListOverlay) handleAdding(msg tea.KeyMsg) (*ListOverlayResultMsg, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		value := strings.TrimSpace(l.input.Value())
		if value != "" {
			l.items = append(l.items, value)
			l.cursor = len(l.items) - 1
		}
		l.state = listBrowsing
		l.input.Blur()
		return nil, nil

	case tea.KeyEsc:
		l.state = listBrowsing
		l.input.Blur()
		return nil, nil

	default:
		// Forward to text input.
		var cmd tea.Cmd
		l.input, cmd = l.input.Update(msg)
		return nil, cmd
	}
}

// handleConfirmDelete processes keys during the delete confirmation prompt.
func (l *ListOverlay) handleConfirmDelete(msg tea.KeyMsg) (*ListOverlayResultMsg, tea.Cmd) {
	switch msg.String() {
	case "y":
		if l.cursor >= 0 && l.cursor < len(l.items) {
			l.items = append(l.items[:l.cursor], l.items[l.cursor+1:]...)
			if l.cursor >= len(l.items) && len(l.items) > 0 {
				l.cursor = len(l.items) - 1
			}
		}
		l.state = listBrowsing
		return nil, nil

	case "n", "esc":
		l.state = listBrowsing
		return nil, nil
	}

	return nil, nil
}

// handleConfirmDiscard processes keys during the discard confirmation prompt.
func (l *ListOverlay) handleConfirmDiscard(msg tea.KeyMsg) (*ListOverlayResultMsg, tea.Cmd) {
	switch msg.String() {
	case "y":
		result := &ListOverlayResultMsg{Committed: false}
		return result, nil

	case "n", "esc":
		l.state = listBrowsing
		return nil, nil
	}

	return nil, nil
}

// moveCursorUp moves the cursor up, wrapping to the last item.
func (l *ListOverlay) moveCursorUp() {
	if len(l.items) == 0 {
		return
	}
	l.cursor = (l.cursor - 1 + len(l.items)) % len(l.items)
}

// moveCursorDown moves the cursor down, wrapping to the first item.
func (l *ListOverlay) moveCursorDown() {
	if len(l.items) == 0 {
		return
	}
	l.cursor = (l.cursor + 1) % len(l.items)
}

// View renders the list overlay as a centered modal over a dimmed background.
func (l *ListOverlay) View(width, height int) string {
	// Calculate overlay dimensions.
	overlayWidth := min(max(width*2/3, 30), width-4)
	innerWidth := max(overlayWidth-4, 20) // 2 padding + 2 border

	// Build the overlay content.
	var content strings.Builder

	// Render items as a numbered list with selection cursor.
	if len(l.items) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(l.theme.Muted).Italic(true)
		content.WriteString(emptyStyle.Render("  (empty list)"))
		content.WriteString("\n")
	} else {
		for i, item := range l.items {
			line := l.renderItem(i, item, innerWidth)
			content.WriteString(line)
			content.WriteString("\n")
		}
	}

	// Add sub-state specific content.
	switch l.state { //nolint:exhaustive // listBrowsing has no extra content
	case listAdding:
		content.WriteString("\n")
		content.WriteString(l.input.View())
		content.WriteString("\n")

	case listConfirmDelete:
		if l.cursor >= 0 && l.cursor < len(l.items) {
			content.WriteString("\n")
			promptStyle := lipgloss.NewStyle().
				Foreground(l.theme.ErrorAccent).Bold(true)
			itemName := l.items[l.cursor]
			if len(itemName) > 20 {
				itemName = itemName[:17] + "..."
			}
			content.WriteString(promptStyle.Render(
				fmt.Sprintf("  Delete %q? y/n", itemName)))
			content.WriteString("\n")
		}

	case listConfirmDiscard:
		content.WriteString("\n")
		promptStyle := lipgloss.NewStyle().
			Foreground(l.theme.ErrorAccent).Bold(true)
		content.WriteString(promptStyle.Render(
			"  Discard changes? y/n"))
		content.WriteString("\n")
	}

	// Build the footer hint line.
	footer := l.renderFooter()

	// Assemble the title.
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(l.theme.Accent)

	// Build the bordered box.
	boxStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(l.theme.Border).
		Padding(0, 1).
		Width(innerWidth + 2) // +2 for padding

	body := titleStyle.Render(l.title) + "\n\n" +
		content.String() + "\n" + footer

	box := boxStyle.Render(body)

	// Center the box over a dimmed background.
	centered := lipgloss.NewStyle().
		Width(width).
		Height(height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(box)

	return centered
}

// renderItem renders a single list item with cursor and number.
func (l *ListOverlay) renderItem(idx int, item string, maxWidth int) string {
	num := fmt.Sprintf("%d. ", idx+1)
	prefix := "    "
	if idx == l.cursor && l.state != listAdding {
		prefix = "  > "
	}

	// Truncate item text if it would overflow.
	availWidth := max(maxWidth-len(prefix)-len(num), 5)
	display := item
	if len(display) > availWidth {
		display = display[:availWidth-3] + "..."
	}

	line := prefix + num + display

	if idx == l.cursor && l.state != listAdding {
		cursorStyle := lipgloss.NewStyle().
			Foreground(l.theme.Cursor).Bold(true)
		itemStyle := lipgloss.NewStyle().
			Foreground(l.theme.Accent).Bold(true)
		return cursorStyle.Render(prefix[:3]) +
			cursorStyle.Render(string(prefix[3])) +
			itemStyle.Render(num+display)
	}

	return line
}

// renderFooter returns the keybinding hint line for the current state.
func (l *ListOverlay) renderFooter() string {
	hintStyle := lipgloss.NewStyle().Foreground(l.theme.Muted)
	keyStyle := lipgloss.NewStyle().Foreground(l.theme.Highlight)
	sep := hintStyle.Render(" . ")

	formatHint := func(b key.Binding) string {
		return keyStyle.Render(b.Help().Key) + " " + hintStyle.Render(b.Help().Desc)
	}

	switch l.state {
	case listAdding:
		return keyStyle.Render("enter") + " " + hintStyle.Render("confirm") +
			sep + keyStyle.Render("esc") + " " + hintStyle.Render("cancel")

	case listConfirmDelete, listConfirmDiscard:
		return keyStyle.Render("y") + " " + hintStyle.Render("yes") +
			sep + keyStyle.Render("n") + " " + hintStyle.Render("no")

	default:
		return formatHint(l.keys.Add) + sep +
			formatHint(l.keys.Delete) + sep +
			formatHint(l.keys.Confirm) + sep +
			formatHint(l.keys.Cancel)
	}
}

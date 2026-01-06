// Package fields provides concrete Field implementations for wizard forms.
package fields

import (
	"strings"

	"gbm/pkg/tui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Confirm is a field that displays Yes/No options with an optional summary.
type Confirm struct {
	key       string
	title     string
	summary   string
	selected  bool // true = Yes, false = No
	value     bool
	complete  bool
	cancelled bool
	focused   bool
	theme     *tui.Theme
	width     int
	height    int
}

// NewConfirm creates a new Confirm field with the given title/question.
func NewConfirm(key, title string) *Confirm {
	return &Confirm{
		key:      key,
		title:    title,
		selected: true, // Default to Yes
		theme:    tui.DefaultTheme(),
	}
}

// WithSummary adds additional context text displayed before the Yes/No buttons.
func (c *Confirm) WithSummary(summary string) *Confirm {
	c.summary = summary
	return c
}

// Init implements Field.Init.
func (c *Confirm) Init() tea.Cmd {
	return nil
}

// Update implements Field.Update.
func (c *Confirm) Update(msg tea.Msg) (tui.Field, tea.Cmd) {
	if !c.focused {
		return c, nil
	}

	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return c, nil
	}

	switch keyMsg.String() {
	// Navigation: left/right arrows and h/l (vim-style)
	case "left", "h":
		c.selected = false // Select No
	case "right", "l":
		c.selected = true // Select Yes

	// Toggle with tab
	case "tab":
		c.selected = !c.selected

	// Shortcut keys: y for Yes, n for No (immediate submit)
	case "y", "Y":
		c.value = true
		c.complete = true
		return c, func() tea.Msg { return tui.NextStepMsg{} }

	case "n", "N":
		c.value = false
		c.complete = true
		c.cancelled = true
		return c, func() tea.Msg { return tui.CancelMsg{} }

	// Confirm selection with Enter
	case "enter":
		c.value = c.selected
		c.complete = true
		if c.selected {
			// Yes selected - proceed
			return c, func() tea.Msg { return tui.NextStepMsg{} }
		}
		// No selected - cancel
		c.cancelled = true
		return c, func() tea.Msg { return tui.CancelMsg{} }
	}

	return c, nil
}

// View implements Field.View.
func (c *Confirm) View() string {
	var b strings.Builder

	// Get styles based on focus state
	styles := c.getStyles()

	// Render title
	b.WriteString(styles.Title.Render(c.title))
	b.WriteString("\n")

	// Render summary if present
	if c.summary != "" {
		b.WriteString("\n")
		b.WriteString(styles.Description.Render(c.summary))
		b.WriteString("\n")
	}

	b.WriteString("\n")

	// Render Yes/No buttons
	yesStyle := lipgloss.NewStyle().
		Padding(0, 2).
		MarginRight(2)
	noStyle := lipgloss.NewStyle().
		Padding(0, 2)

	// Highlight selected button
	if c.focused {
		if c.selected {
			yesStyle = yesStyle.
				Bold(true).
				Foreground(lipgloss.Color("255")).
				Background(lipgloss.Color("62"))
			noStyle = noStyle.
				Foreground(lipgloss.Color("240"))
		} else {
			yesStyle = yesStyle.
				Foreground(lipgloss.Color("240"))
			noStyle = noStyle.
				Bold(true).
				Foreground(lipgloss.Color("255")).
				Background(lipgloss.Color("196"))
		}
	} else {
		// Blurred state - both buttons muted
		yesStyle = yesStyle.Foreground(lipgloss.Color("240"))
		noStyle = noStyle.Foreground(lipgloss.Color("240"))
	}

	yesBtn := yesStyle.Render("Yes")
	noBtn := noStyle.Render("No")

	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Center, yesBtn, noBtn))

	return b.String()
}

// Focus implements Field.Focus.
func (c *Confirm) Focus() tea.Cmd {
	c.focused = true
	return nil
}

// Blur implements Field.Blur.
func (c *Confirm) Blur() tea.Cmd {
	c.focused = false
	return nil
}

// IsComplete implements Field.IsComplete.
func (c *Confirm) IsComplete() bool {
	return c.complete
}

// IsCancelled implements Field.IsCancelled.
func (c *Confirm) IsCancelled() bool {
	return c.cancelled
}

// Error implements Field.Error.
func (c *Confirm) Error() error {
	return nil
}

// Skip implements Field.Skip.
func (c *Confirm) Skip() bool {
	return false
}

// WithTheme implements Field.WithTheme.
func (c *Confirm) WithTheme(theme *tui.Theme) tui.Field {
	c.theme = theme
	return c
}

// WithWidth implements Field.WithWidth.
func (c *Confirm) WithWidth(width int) tui.Field {
	c.width = width
	return c
}

// WithHeight implements Field.WithHeight.
func (c *Confirm) WithHeight(height int) tui.Field {
	c.height = height
	return c
}

// GetKey implements Field.GetKey.
func (c *Confirm) GetKey() string {
	return c.key
}

// GetValue implements Field.GetValue.
func (c *Confirm) GetValue() any {
	return c.value
}

// getStyles returns the appropriate styles based on focus state.
func (c *Confirm) getStyles() tui.FieldStyles {
	if c.theme == nil {
		c.theme = tui.DefaultTheme()
	}
	if c.focused {
		return c.theme.Focused
	}
	return c.theme.Blurred
}

// Ensure Confirm implements Field interface at compile time.
var _ tui.Field = (*Confirm)(nil)

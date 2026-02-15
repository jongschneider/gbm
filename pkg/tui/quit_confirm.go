package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// quitConfirm is a minimal yes/no confirmation field used by ConfigModel
// for the quit-with-unsaved-changes dialog. It lives in the tui package
// to avoid an import cycle with tui/fields.
type quitConfirm struct {
	theme    *Theme
	title    string
	selected bool // true = Yes, false = No
	complete bool
	value    bool
	focused  bool
}

// newQuitConfirm creates a new quit confirmation field.
func newQuitConfirm(theme *Theme) Field {
	if theme == nil {
		theme = DefaultTheme()
	}
	return &quitConfirm{
		theme:    theme,
		title:    "Discard unsaved changes?",
		selected: true, // Default to Yes
	}
}

func (c *quitConfirm) Init() tea.Cmd { return nil }

func (c *quitConfirm) Update(msg tea.Msg) (Field, tea.Cmd) {
	if !c.focused {
		return c, nil
	}

	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return c, nil
	}

	switch keyMsg.String() {
	case "left", "h":
		c.selected = true
	case "right", "l":
		c.selected = false
	case "tab", " ":
		c.selected = !c.selected
	case "y", "Y":
		c.value = true
		c.complete = true
		return c, nil
	case "n", "N":
		c.value = false
		c.complete = true
		return c, nil
	case "enter":
		c.value = c.selected
		c.complete = true
		return c, nil
	}

	return c, nil
}

func (c *quitConfirm) View() string {
	var b strings.Builder

	styles := c.theme.Focused
	if !c.focused {
		styles = c.theme.Blurred
	}

	b.WriteString(styles.Title.Render(c.title))
	b.WriteString("\n\n")

	yesStyle := lipgloss.NewStyle().Padding(0, 2).MarginRight(2)
	noStyle := lipgloss.NewStyle().Padding(0, 2)

	if c.focused {
		if c.selected {
			yesStyle = yesStyle.Bold(true).
				Foreground(c.theme.InputFg).
				Background(c.theme.SuccessAccent)
			noStyle = noStyle.Foreground(c.theme.BlurredMuted)
		} else {
			yesStyle = yesStyle.Foreground(c.theme.BlurredMuted)
			noStyle = noStyle.Bold(true).
				Foreground(c.theme.InputFg).
				Background(c.theme.ErrorAccent)
		}
	} else {
		yesStyle = yesStyle.Foreground(c.theme.BlurredMuted)
		noStyle = noStyle.Foreground(c.theme.BlurredMuted)
	}

	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Center,
		yesStyle.Render("Yes"),
		noStyle.Render("No"),
	))
	b.WriteString("\n")

	return b.String()
}

func (c *quitConfirm) Focus() tea.Cmd {
	c.focused = true
	return nil
}

func (c *quitConfirm) Blur() tea.Cmd {
	c.focused = false
	return nil
}

func (c *quitConfirm) IsComplete() bool  { return c.complete }
func (c *quitConfirm) IsCancelled() bool { return c.complete && !c.value }
func (c *quitConfirm) Error() error      { return nil }
func (c *quitConfirm) Skip() bool        { return false }
func (c *quitConfirm) GetKey() string    { return "quit_confirm" }
func (c *quitConfirm) GetValue() any     { return c.value }

func (c *quitConfirm) WithTheme(theme *Theme) Field {
	c.theme = theme
	return c
}

func (c *quitConfirm) WithWidth(_ int) Field  { return c }
func (c *quitConfirm) WithHeight(_ int) Field { return c }

// Ensure quitConfirm implements Field at compile time.
var _ Field = (*quitConfirm)(nil)

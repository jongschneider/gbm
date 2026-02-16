package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SaveConfirmContext indicates what triggered the save confirmation dialog.
type SaveConfirmContext int

const (
	// SaveContextSave means the dialog was triggered by Ctrl+S.
	SaveContextSave SaveConfirmContext = iota
	// SaveContextQuit means the dialog was triggered by q while dirty.
	SaveContextQuit
)

// saveConfirm is a minimal yes/no confirmation field used by ConfigModel
// for the save-configuration dialog. It lives in the tui package
// to avoid an import cycle with tui/fields.
type saveConfirm struct {
	theme    *Theme
	title    string
	selected bool // true = Yes, false = No
	complete bool
	value    bool
	focused  bool
}

// newSaveConfirm creates a new save confirmation field.
func newSaveConfirm(theme *Theme) *saveConfirm {
	if theme == nil {
		theme = DefaultTheme()
	}
	return &saveConfirm{
		theme:    theme,
		title:    "Save configuration?",
		selected: true, // Default to Yes
	}
}

// WithTitle sets the title of the save confirmation dialog.
func (c *saveConfirm) WithTitle(title string) *saveConfirm {
	c.title = title
	return c
}

func (c *saveConfirm) Init() tea.Cmd { return nil }

func (c *saveConfirm) Update(msg tea.Msg) (Field, tea.Cmd) {
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

func (c *saveConfirm) View() string {
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

func (c *saveConfirm) Focus() tea.Cmd {
	c.focused = true
	return nil
}

func (c *saveConfirm) Blur() tea.Cmd {
	c.focused = false
	return nil
}

func (c *saveConfirm) IsComplete() bool  { return c.complete }
func (c *saveConfirm) IsCancelled() bool { return c.complete && !c.value }
func (c *saveConfirm) Error() error      { return nil }
func (c *saveConfirm) Skip() bool        { return false }
func (c *saveConfirm) GetKey() string    { return "save_confirm" }
func (c *saveConfirm) GetValue() any     { return c.value }

func (c *saveConfirm) WithTheme(theme *Theme) Field {
	c.theme = theme
	return c
}

func (c *saveConfirm) WithWidth(_ int) Field  { return c }
func (c *saveConfirm) WithHeight(_ int) Field { return c }

// Ensure saveConfirm implements Field at compile time.
var _ Field = (*saveConfirm)(nil)

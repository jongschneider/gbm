// Package fields provides concrete Field implementations for wizard forms.
package fields

import (
	"fmt"
	"strings"

	"gbm/pkg/tui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Option represents a selectable item in the Selector.
type Option struct {
	Label string
	Value string
}

// Selector is a field that displays a list of options for the user to select.
type Selector struct {
	key         string
	title       string
	options     []Option
	cursor      int
	selected    string
	complete    bool
	cancelled   bool
	focused     bool
	theme       *tui.Theme
	width       int
	height      int
	cursorStyle lipgloss.Style
}

// NewSelector creates a new Selector with the given title and options.
func NewSelector(key, title string, options []Option) *Selector {
	return &Selector{
		key:         key,
		title:       title,
		options:     options,
		cursor:      0,
		theme:       tui.DefaultTheme(),
		cursorStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("212")),
	}
}

// Init implements Field.Init.
func (s *Selector) Init() tea.Cmd {
	return nil
}

// Update implements Field.Update.
func (s *Selector) Update(msg tea.Msg) (tui.Field, tea.Cmd) {
	if !s.focused {
		return s, nil
	}

	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return s, nil
	}

	switch keyMsg.String() {
	// Navigation: up arrow and k
	case "up", "k":
		s.cursor--
		if s.cursor < 0 {
			s.cursor = len(s.options) - 1 // Wrap to bottom
		}

	// Navigation: down arrow and j
	case "down", "j":
		s.cursor++
		if s.cursor >= len(s.options) {
			s.cursor = 0 // Wrap to top
		}

	// Confirm selection
	case "enter":
		if len(s.options) > 0 {
			s.selected = s.options[s.cursor].Value
			s.complete = true
			return s, func() tea.Msg { return tui.NextStepMsg{} }
		}
	}

	return s, nil
}

// View implements Field.View.
func (s *Selector) View() string {
	var b strings.Builder

	// Get styles based on focus state
	styles := s.getStyles()

	// Render title
	b.WriteString(styles.Title.Render(s.title))
	b.WriteString("\n\n")

	// Render options
	for i, opt := range s.options {
		cursor := "  " // No cursor for non-selected items
		if i == s.cursor {
			cursor = s.cursorStyle.Render("▸ ") // Highlighted cursor
		}

		line := fmt.Sprintf("%s%s", cursor, opt.Label)

		// Apply input style to highlighted option
		if i == s.cursor && s.focused {
			line = styles.Input.Render(line)
		} else if i == s.cursor && !s.focused {
			// Blurred but still highlighted - use a muted version of input style
			dimmedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
			line = dimmedStyle.Render(line)
		}

		b.WriteString(line)
		if i < len(s.options)-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

// Focus implements Field.Focus.
func (s *Selector) Focus() tea.Cmd {
	s.focused = true
	return nil
}

// Blur implements Field.Blur.
func (s *Selector) Blur() tea.Cmd {
	s.focused = false
	return nil
}

// IsComplete implements Field.IsComplete.
func (s *Selector) IsComplete() bool {
	return s.complete
}

// IsCancelled implements Field.IsCancelled.
func (s *Selector) IsCancelled() bool {
	return s.cancelled
}

// Error implements Field.Error.
func (s *Selector) Error() error {
	return nil
}

// Skip implements Field.Skip.
func (s *Selector) Skip() bool {
	return false
}

// WithTheme implements Field.WithTheme.
func (s *Selector) WithTheme(theme *tui.Theme) tui.Field {
	s.theme = theme
	return s
}

// WithWidth implements Field.WithWidth.
func (s *Selector) WithWidth(width int) tui.Field {
	s.width = width
	return s
}

// WithHeight implements Field.WithHeight.
func (s *Selector) WithHeight(height int) tui.Field {
	s.height = height
	return s
}

// GetKey implements Field.GetKey.
func (s *Selector) GetKey() string {
	return s.key
}

// GetValue implements Field.GetValue.
func (s *Selector) GetValue() any {
	return s.selected
}

// getStyles returns the appropriate styles based on focus state.
func (s *Selector) getStyles() tui.FieldStyles {
	if s.theme == nil {
		s.theme = tui.DefaultTheme()
	}
	if s.focused {
		return s.theme.Focused
	}
	return s.theme.Blurred
}

// Ensure Selector implements Field interface at compile time.
var _ tui.Field = (*Selector)(nil)

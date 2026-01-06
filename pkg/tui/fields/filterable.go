// Package fields provides concrete Field implementations for wizard forms.
package fields

import (
	"fmt"
	"strings"

	"gbm/pkg/tui"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Filterable is a field that displays a text input above a filterable list of options.
// Users can type to filter the list or enter a custom value.
type Filterable struct {
	key          string
	title        string
	description  string
	options      []Option
	filtered     []Option
	cursor       int
	textInput    textinput.Model
	selected     string
	complete     bool
	cancelled    bool
	focused      bool
	theme        *tui.Theme
	width        int
	height       int
	cursorStyle  lipgloss.Style
	noMatchStyle lipgloss.Style
}

// NewFilterable creates a new Filterable field with the given title, description, and options.
func NewFilterable(key, title, description string, options []Option) *Filterable {
	ti := textinput.New()
	ti.Placeholder = "Type to filter or enter custom value..."
	ti.CharLimit = 200
	ti.Width = 60

	// Copy options to filtered list initially
	filtered := make([]Option, len(options))
	copy(filtered, options)

	return &Filterable{
		key:          key,
		title:        title,
		description:  description,
		options:      options,
		filtered:     filtered,
		cursor:       0,
		textInput:    ti,
		theme:        tui.DefaultTheme(),
		cursorStyle:  lipgloss.NewStyle().Foreground(lipgloss.Color("212")),
		noMatchStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Italic(true),
	}
}

// Init implements Field.Init.
func (f *Filterable) Init() tea.Cmd {
	return textinput.Blink
}

// Update implements Field.Update.
func (f *Filterable) Update(msg tea.Msg) (tui.Field, tea.Cmd) {
	if !f.focused {
		return f, nil
	}

	var cmd tea.Cmd

	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		// Pass non-key messages to text input (for blink, etc.)
		f.textInput, cmd = f.textInput.Update(msg)
		return f, cmd
	}

	switch keyMsg.String() {
	// Navigation: up arrow and k
	case "up", "k":
		if len(f.filtered) > 0 {
			f.cursor--
			if f.cursor < 0 {
				f.cursor = len(f.filtered) - 1 // Wrap to bottom
			}
		}
		return f, nil

	// Navigation: down arrow and j
	case "down", "j":
		if len(f.filtered) > 0 {
			f.cursor++
			if f.cursor >= len(f.filtered) {
				f.cursor = 0 // Wrap to top
			}
		}
		return f, nil

	// Confirm selection
	case "enter":
		// If list has items and one is selected, use that
		if len(f.filtered) > 0 && f.cursor >= 0 && f.cursor < len(f.filtered) {
			f.selected = f.filtered[f.cursor].Value
		} else {
			// Use trimmed text input as custom value
			f.selected = strings.TrimSpace(f.textInput.Value())
		}
		f.complete = true
		return f, func() tea.Msg { return tui.NextStepMsg{} }

	default:
		// Update text input
		f.textInput, cmd = f.textInput.Update(msg)

		// Filter the list based on text input
		f.filterOptions()

		return f, cmd
	}
}

// filterOptions filters the options list based on the current text input value.
func (f *Filterable) filterOptions() {
	query := strings.ToLower(strings.TrimSpace(f.textInput.Value()))

	if query == "" {
		// No filter, show all options
		f.filtered = make([]Option, len(f.options))
		copy(f.filtered, f.options)
	} else {
		// Filter options that match the query (case-insensitive)
		f.filtered = []Option{}
		for _, opt := range f.options {
			if strings.Contains(strings.ToLower(opt.Label), query) ||
				strings.Contains(strings.ToLower(opt.Value), query) {
				f.filtered = append(f.filtered, opt)
			}
		}
	}

	// Reset cursor to valid position
	if f.cursor >= len(f.filtered) {
		if len(f.filtered) > 0 {
			f.cursor = 0
		} else {
			f.cursor = -1
		}
	}
}

// View implements Field.View.
func (f *Filterable) View() string {
	var b strings.Builder

	// Get styles based on focus state
	styles := f.getStyles()

	// Render title
	b.WriteString(styles.Title.Render(f.title))
	b.WriteString("\n")

	// Render description if present
	if f.description != "" {
		b.WriteString(styles.Description.Render(f.description))
		b.WriteString("\n")
	}

	b.WriteString("\n")

	// Render text input
	b.WriteString(f.textInput.View())
	b.WriteString("\n\n")

	// Render filtered options or "No matches" message
	if len(f.filtered) == 0 {
		inputValue := strings.TrimSpace(f.textInput.Value())
		if inputValue != "" {
			b.WriteString(f.noMatchStyle.Render(fmt.Sprintf("No matches. Press Enter to use: %q", inputValue)))
		} else {
			b.WriteString(f.noMatchStyle.Render("No matches"))
		}
	} else {
		// Render options
		for i, opt := range f.filtered {
			cursor := "  " // No cursor
			if i == f.cursor {
				cursor = f.cursorStyle.Render("▸ ")
			}

			line := fmt.Sprintf("%s%s", cursor, opt.Label)
			if i == f.cursor && f.focused {
				line = styles.Input.Render(line)
			}
			b.WriteString(line)
			if i < len(f.filtered)-1 {
				b.WriteString("\n")
			}
		}
	}

	return b.String()
}

// Focus implements Field.Focus.
func (f *Filterable) Focus() tea.Cmd {
	f.focused = true
	f.textInput.Focus()
	return textinput.Blink
}

// Blur implements Field.Blur.
func (f *Filterable) Blur() tea.Cmd {
	f.focused = false
	f.textInput.Blur()
	return nil
}

// IsComplete implements Field.IsComplete.
func (f *Filterable) IsComplete() bool {
	return f.complete
}

// IsCancelled implements Field.IsCancelled.
func (f *Filterable) IsCancelled() bool {
	return f.cancelled
}

// Error implements Field.Error.
func (f *Filterable) Error() error {
	return nil
}

// Skip implements Field.Skip.
func (f *Filterable) Skip() bool {
	return false
}

// WithTheme implements Field.WithTheme.
func (f *Filterable) WithTheme(theme *tui.Theme) tui.Field {
	f.theme = theme
	return f
}

// WithWidth implements Field.WithWidth.
func (f *Filterable) WithWidth(width int) tui.Field {
	f.width = width
	f.textInput.Width = width - 4
	return f
}

// WithHeight implements Field.WithHeight.
func (f *Filterable) WithHeight(height int) tui.Field {
	f.height = height
	return f
}

// GetKey implements Field.GetKey.
func (f *Filterable) GetKey() string {
	return f.key
}

// GetValue implements Field.GetValue.
func (f *Filterable) GetValue() any {
	return f.selected
}

// getStyles returns the appropriate styles based on focus state.
func (f *Filterable) getStyles() tui.FieldStyles {
	if f.theme == nil {
		f.theme = tui.DefaultTheme()
	}
	if f.focused {
		return f.theme.Focused
	}
	return f.theme.Blurred
}

// Ensure Filterable implements Field interface at compile time.
var _ tui.Field = (*Filterable)(nil)

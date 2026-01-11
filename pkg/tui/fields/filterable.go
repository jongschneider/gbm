// Package fields provides concrete Field implementations for wizard forms.
package fields

import (
	"fmt"
	"strings"

	"gbm/pkg/tui"
	"gbm/pkg/tui/async"

	"github.com/charmbracelet/bubbles/spinner"
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

	// Async options support
	optionsFetch func() ([]Option, error)
	isLoading    bool
	loadErr      error
	spinner      spinner.Model
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
		spinner:      spinner.New(spinner.WithSpinner(spinner.Dot)),
	}
}

// WithOptionsFuncAsync configures Filterable to load options asynchronously using FetchCmd.
// The provided function will be called to fetch options without blocking the event loop.
func (f *Filterable) WithOptionsFuncAsync(fetch func() ([]Option, error)) *Filterable {
	f.optionsFetch = fetch
	return f
}

// Init implements Field.Init.
// If async options are configured, returns a FetchCmd to load them without blocking.
func (f *Filterable) Init() tea.Cmd {
	if f.optionsFetch != nil {
		f.isLoading = true
		return async.FetchCmd(f.optionsFetch)
	}
	return textinput.Blink
}

// Update implements Field.Update.
func (f *Filterable) Update(msg tea.Msg) (tui.Field, tea.Cmd) {
	// Handle FetchMsg for async option loading
	if fetchMsg, ok := msg.(async.FetchMsg[[]Option]); ok {
		f.isLoading = false
		if fetchMsg.Err != nil {
			f.loadErr = fetchMsg.Err
			return f, nil
		}
		f.options = fetchMsg.Value
		f.filterOptions()
		return f, nil
	}

	// Handle spinner tick for visual feedback during loading
	if _, ok := msg.(spinner.TickMsg); ok {
		f.spinner, _ = f.spinner.Update(msg)
		if f.isLoading {
			return f, f.spinner.Tick
		}
		return f, nil
	}

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

	// Block all input except cancel/quit while loading options
	if f.isLoading {
		// Only allow cancel (Ctrl+C) or quit (q) keys while loading
		if keyMsg.String() != "ctrl+c" && keyMsg.String() != "q" {
			// Ignore all other input while loading
			return f, nil
		}
		// Fall through to let ctrl+c and q be handled normally (likely by wizard)
	}

	switch keyMsg.String() {
	// Navigation: up arrow, k, and ctrl+k
	case KeyUp, "k", KeyCtrlUp:
		if len(f.filtered) > 0 {
			f.cursor--
			if f.cursor < 0 {
				f.cursor = len(f.filtered) - 1 // Wrap to bottom
			}
		}
		return f, nil

	// Navigation: down arrow, j, and ctrl+j
	case KeyDown, "j", KeyCtrlDown:
		if len(f.filtered) > 0 {
			f.cursor++
			if f.cursor >= len(f.filtered) {
				f.cursor = 0 // Wrap to top
			}
		}
		return f, nil

	// Confirm selection
	case KeyEnter:
		// Require options to be loaded before allowing submission
		if f.isLoading {
			// Still loading, don't allow submission
			return f, nil
		}
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

	// Render text input with focus styling
	inputView := f.textInput.View()
	if f.focused {
		inputView = styles.Input.Render(inputView)
	}
	b.WriteString(inputView)
	b.WriteString("\n\n")

	// If loading options asynchronously, show spinner
	if f.isLoading {
		b.WriteString(fmt.Sprintf("%s Loading options...", f.spinner.View()))
		return b.String()
	}

	// Show any async error
	if f.loadErr != nil {
		b.WriteString(styles.Error.Render(fmt.Sprintf("Error loading options: %v", f.loadErr)))
		return b.String()
	}

	// Render filtered options or "No matches" message
	if len(f.filtered) == 0 {
		inputValue := strings.TrimSpace(f.textInput.Value())
		if inputValue != "" {
			b.WriteString(styles.Error.Render(fmt.Sprintf("No matches. Press Enter to use: %q", inputValue)))
		} else {
			b.WriteString(styles.Description.Render("No matches"))
		}
	} else {
		// Render options
		for i, opt := range f.filtered {
			cursor := "  " // No cursor for non-selected items
			if i == f.cursor {
				cursor = f.cursorStyle.Render("▸ ") // Highlighted cursor
			}

			line := fmt.Sprintf("%s%s", cursor, opt.Label)

			// Apply input style to highlighted option
			if i == f.cursor && f.focused {
				line = styles.Input.Render(line)
			} else if i == f.cursor && !f.focused {
				// Blurred but still highlighted - use a muted version of input style
				dimmedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
				line = dimmedStyle.Render(line)
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

	// If currently loading options, show spinner
	if f.isLoading {
		return tea.Batch(
			textinput.Blink,
			f.spinner.Tick,
		)
	}

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

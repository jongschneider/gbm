// Package fields provides concrete Field implementations for wizard forms.
package fields

import (
	"fmt"
	"gbm/pkg/tui"
	"gbm/pkg/tui/async"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Filterable is a field that displays a text input above a filterable list of options.
// Users can type to filter the list or enter a custom value.
type Filterable struct {
	cursorStyle    lipgloss.Style
	noMatchStyle   lipgloss.Style
	loadErr        error
	theme          *tui.Theme
	optionsFetch   func() ([]Option, error)
	title          string
	description    string
	key            string
	selected       string
	options        []Option
	filtered       []Option
	textInput      textinput.Model
	spinner        spinner.Model
	cursor         int
	height         int
	width          int
	viewportOffset int
	focused        bool
	cancelled      bool
	isLoading      bool
	complete       bool
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
		return f.handleFetchMsg(fetchMsg)
	}

	// Handle spinner tick for visual feedback during loading
	if _, ok := msg.(spinner.TickMsg); ok {
		return f.handleSpinnerTick(msg)
	}

	if !f.focused {
		return f, nil
	}

	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		// Pass non-key messages to text input (for blink, etc.)
		var cmd tea.Cmd
		f.textInput, cmd = f.textInput.Update(msg)
		return f, cmd
	}

	return f.handleKeyMsg(keyMsg)
}

// handleFetchMsg processes async option loading results.
func (f *Filterable) handleFetchMsg(fetchMsg async.FetchMsg[[]Option]) (tui.Field, tea.Cmd) {
	f.isLoading = false
	if fetchMsg.Err != nil {
		f.loadErr = fetchMsg.Err
		return f, nil
	}
	f.options = fetchMsg.Value
	f.filterOptions()
	return f, nil
}

// handleSpinnerTick processes spinner tick messages.
func (f *Filterable) handleSpinnerTick(msg tea.Msg) (tui.Field, tea.Cmd) {
	f.spinner, _ = f.spinner.Update(msg)
	if f.isLoading {
		return f, f.spinner.Tick
	}
	return f, nil
}

// handleKeyMsg processes keyboard messages.
func (f *Filterable) handleKeyMsg(keyMsg tea.KeyMsg) (tui.Field, tea.Cmd) {
	// Block all input except cancel/quit while loading options
	if f.isLoading && keyMsg.String() != "ctrl+c" && keyMsg.String() != "q" {
		return f, nil
	}

	switch keyMsg.String() {
	case KeyUp, KeyCtrlUp:
		f.navigateUp()
		return f, nil

	case KeyDown, KeyCtrlDown:
		f.navigateDown()
		return f, nil

	case KeyEnter:
		return f.handleEnter()

	default:
		return f.handleTextInput(keyMsg)
	}
}

// navigateUp moves the cursor up in the filtered list.
func (f *Filterable) navigateUp() {
	if len(f.filtered) == 0 {
		return
	}
	f.cursor--
	if f.cursor < 0 {
		f.cursor = len(f.filtered) - 1
		f.adjustViewport()
	} else if f.cursor < f.viewportOffset {
		f.viewportOffset = f.cursor
	}
}

// navigateDown moves the cursor down in the filtered list.
func (f *Filterable) navigateDown() {
	if len(f.filtered) == 0 {
		return
	}
	f.cursor++
	if f.cursor >= len(f.filtered) {
		f.cursor = 0
		f.viewportOffset = 0
	} else {
		f.adjustViewport()
	}
}

// handleEnter processes the enter key to confirm selection.
func (f *Filterable) handleEnter() (tui.Field, tea.Cmd) {
	if f.isLoading {
		return f, nil
	}
	if len(f.filtered) > 0 && f.cursor >= 0 && f.cursor < len(f.filtered) {
		f.selected = f.filtered[f.cursor].Value
	} else {
		f.selected = strings.TrimSpace(f.textInput.Value())
	}
	f.complete = true
	return f, func() tea.Msg { return tui.NextStepMsg{} }
}

// handleTextInput processes text input for filtering.
func (f *Filterable) handleTextInput(keyMsg tea.KeyMsg) (tui.Field, tea.Cmd) {
	var cmd tea.Cmd
	f.textInput, cmd = f.textInput.Update(keyMsg)
	f.filterOptions()
	return f, cmd
}

// filterOptions filters the options list based on the current text input value.
func (f *Filterable) filterOptions() {
	rawValue := f.textInput.Value()
	query := strings.ToLower(strings.TrimSpace(rawValue))

	if query == "" {
		// No filter, show all options
		f.filtered = make([]Option, len(f.options))
		copy(f.filtered, f.options)
	} else {
		// Filter options that match the query (case-insensitive)
		f.filtered = []Option{}
		for _, opt := range f.options {
			labelLower := strings.ToLower(opt.Label)
			valueLower := strings.ToLower(opt.Value)
			if strings.Contains(labelLower, query) || strings.Contains(valueLower, query) {
				f.filtered = append(f.filtered, opt)
			}
		}
	}

	// Reset cursor and viewport to valid positions when filter changes
	if len(f.filtered) > 0 {
		if f.cursor < 0 || f.cursor >= len(f.filtered) {
			f.cursor = 0
		}
	} else {
		f.cursor = -1
	}
	f.viewportOffset = 0
}

// visibleItemCount returns how many list items can fit in the available height.
// Reserves space for header (title, description, input field).
func (f *Filterable) visibleItemCount() int {
	if f.height <= 0 {
		return 10 // Default fallback
	}
	// Header takes: title (1) + description (1) + blank (1) + input (1) + blank (1) = 5 lines
	headerLines := 5
	if f.description == "" {
		headerLines = 4
	}
	available := f.height - headerLines
	if available < 1 {
		return 1
	}
	return available
}

// adjustViewport ensures the cursor is visible within the viewport.
func (f *Filterable) adjustViewport() {
	visible := f.visibleItemCount()

	// If cursor is above viewport, scroll up
	if f.cursor < f.viewportOffset {
		f.viewportOffset = f.cursor
	}

	// If cursor is below viewport, scroll down
	if f.cursor >= f.viewportOffset+visible {
		f.viewportOffset = f.cursor - visible + 1
	}

	// Ensure viewport doesn't go negative
	if f.viewportOffset < 0 {
		f.viewportOffset = 0
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
		b.WriteString(f.spinner.View() + " Loading options...\n")
		return b.String()
	}

	// Show any async error
	if f.loadErr != nil {
		b.WriteString(styles.Error.Render(fmt.Sprintf("Error loading options: %v", f.loadErr)))
		b.WriteString("\n")
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
		b.WriteString("\n")
	} else {
		// Calculate visible range
		visible := f.visibleItemCount()
		start := f.viewportOffset
		end := min(start+visible, len(f.filtered))

		// Render only visible options
		for i := start; i < end; i++ {
			opt := f.filtered[i]

			var line string
			if i == f.cursor {
				// Selected item with cursor
				line = "▸ " + opt.Label
				if f.focused {
					line = styles.Input.Render(line)
				} else {
					// Blurred but still highlighted - use a muted version
					dimmedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
					line = dimmedStyle.Render(line)
				}
			} else {
				// Non-selected item - plain text with indent
				line = "  " + opt.Label
			}

			b.WriteString(line)
			b.WriteString("\n")
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

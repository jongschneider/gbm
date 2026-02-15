// Package fields provides concrete Field implementations for wizard forms.
package fields

import (
	"gbm/pkg/tui"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// ValidatorFunc is a function that validates text input and returns an error if invalid.
type ValidatorFunc func(string) error

// TextInput is a field that allows the user to enter or edit text.
type TextInput struct {
	err          error
	theme        *tui.Theme
	validator    ValidatorFunc
	title        string
	description  string
	key          string
	defaultValue string
	value        string
	textInput    textinput.Model
	width        int
	height       int
	focused      bool
	cancelled    bool
	complete     bool
	masked       bool
	hasDefault   bool // Track if WithDefault was called
}

// NewTextInput creates a new TextInput field with the given title and description.
func NewTextInput(key, title, description string) *TextInput {
	ti := textinput.New()
	ti.Placeholder = "Enter value..."
	ti.CharLimit = 200
	ti.Width = 60

	return &TextInput{
		key:         key,
		title:       title,
		description: description,
		textInput:   ti,
		theme:       tui.DefaultTheme(),
	}
}

// WithDefault sets the default value that is pre-filled when the field is focused.
func (t *TextInput) WithDefault(value string) tui.Field {
	t.defaultValue = value
	t.hasDefault = true
	// Also set it immediately so GetValue() can return it before Focus is called
	t.textInput.SetValue(value)
	t.textInput.SetCursor(len(value))
	return t
}

// WithValidator sets a validation function to run when the user presses Enter.
func (t *TextInput) WithValidator(validator ValidatorFunc) *TextInput {
	t.validator = validator
	return t
}

// WithPlaceholder sets a custom placeholder text.
func (t *TextInput) WithPlaceholder(placeholder string) *TextInput {
	t.textInput.Placeholder = placeholder
	return t
}

// SetMasked sets whether the input value should be masked (displayed as asterisks).
func (t *TextInput) SetMasked(masked bool) *TextInput {
	t.masked = masked
	if masked {
		t.textInput.EchoMode = textinput.EchoPassword
	} else {
		t.textInput.EchoMode = textinput.EchoNormal
	}
	return t
}

// Init implements Field.Init.
func (t *TextInput) Init() tea.Cmd {
	return textinput.Blink
}

// Update implements Field.Update.
func (t *TextInput) Update(msg tea.Msg) (tui.Field, tea.Cmd) {
	if !t.focused {
		return t, nil
	}

	var cmd tea.Cmd

	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		// Pass non-key messages to text input (for blink, cursor updates, etc.)
		t.textInput, cmd = t.textInput.Update(msg)
		return t, cmd
	}

	switch keyMsg.String() {
	case KeyEnter:
		// Get trimmed value
		value := strings.TrimSpace(t.textInput.Value())

		// Run validation if configured
		if t.validator != nil {
			err := t.validator(value)
			if err != nil {
				t.err = err
				return t, nil
			}
		}

		// Validation passed (or no validator)
		t.err = nil
		t.value = value
		t.complete = true
		return t, func() tea.Msg { return tui.NextStepMsg{} }

	default:
		// Clear error when user starts typing
		if t.err != nil {
			t.err = nil
		}

		// Pass to text input for character handling, cursor movement, etc.
		t.textInput, cmd = t.textInput.Update(msg)
		return t, cmd
	}
}

// View implements Field.View.
func (t *TextInput) View() string {
	var b strings.Builder

	// Get styles based on focus state
	styles := t.getStyles()

	// Render title
	b.WriteString(styles.Title.Render(t.title))
	b.WriteString("\n")

	// Render description if present, wrapping to the field width
	if t.description != "" {
		descStyle := styles.Description
		if t.width > 0 {
			descStyle = descStyle.Width(t.width - 2)
		}
		b.WriteString(descStyle.Render(t.description))
		b.WriteString("\n")
	}

	b.WriteString("\n")

	// Render text input with focus styling
	inputView := t.textInput.View()
	if t.focused {
		inputView = styles.Input.Render(inputView)
	}
	b.WriteString(inputView)

	// Render error if present
	if t.err != nil {
		b.WriteString("\n")
		b.WriteString(styles.Error.Render(t.err.Error()))
	}

	b.WriteString("\n")
	return b.String()
}

// Focus implements Field.Focus.
func (t *TextInput) Focus() tea.Cmd {
	t.focused = true
	t.textInput.Focus()

	// Position cursor at end of current text so the user can continue editing.
	// Note: default values are already populated via WithDefault() at construction
	// time -- we must NOT overwrite the current value here or user edits are lost.
	t.textInput.SetCursor(len(t.textInput.Value()))

	return textinput.Blink
}

// Blur implements Field.Blur.
func (t *TextInput) Blur() tea.Cmd {
	t.focused = false
	t.textInput.Blur()
	return nil
}

// IsComplete implements Field.IsComplete.
func (t *TextInput) IsComplete() bool {
	return t.complete
}

// IsCancelled implements Field.IsCancelled.
func (t *TextInput) IsCancelled() bool {
	return t.cancelled
}

// Error implements Field.Error.
func (t *TextInput) Error() error {
	return t.err
}

// Skip implements Field.Skip.
func (t *TextInput) Skip() bool {
	return false
}

// WithTheme implements Field.WithTheme.
func (t *TextInput) WithTheme(theme *tui.Theme) tui.Field {
	t.theme = theme
	return t
}

// WithWidth implements Field.WithWidth.
func (t *TextInput) WithWidth(width int) tui.Field {
	t.width = width
	t.textInput.Width = width - 4
	return t
}

// WithHeight implements Field.WithHeight.
func (t *TextInput) WithHeight(height int) tui.Field {
	t.height = height
	return t
}

// GetKey implements Field.GetKey.
func (t *TextInput) GetKey() string {
	return t.key
}

// RunValidator runs the validator on the current value and returns the error.
// This allows forms to validate fields without requiring the user to press Enter.
// Returns nil if no validator is set or if validation passes.
func (t *TextInput) RunValidator() error {
	if t.validator == nil {
		return nil
	}
	value := strings.TrimSpace(t.textInput.Value())
	err := t.validator(value)
	if err != nil {
		t.err = err
	}
	return err
}

// SetError sets the error state on the field.
// This is used to highlight fields that failed validation.
func (t *TextInput) SetError(err error) {
	t.err = err
}

// GetValue implements Field.GetValue.
func (t *TextInput) GetValue() any {
	// If WithDefault was called, return the current textInput value (which is pre-populated)
	// Otherwise return the stored value (which is only set when confirmed)
	if t.hasDefault && t.textInput.Value() != "" {
		return t.textInput.Value()
	}
	return t.value
}

// getStyles returns the appropriate styles based on focus state.
func (t *TextInput) getStyles() tui.FieldStyles {
	if t.theme == nil {
		t.theme = tui.DefaultTheme()
	}
	if t.focused {
		return t.theme.Focused
	}
	return t.theme.Blurred
}

// Ensure TextInput implements Field interface at compile time.
var _ tui.Field = (*TextInput)(nil)

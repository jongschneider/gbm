package config

import (
	"fmt"
	"gbm/pkg/tui"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// FieldRowState represents whether a field row is in browsing or editing mode.
type FieldRowState int

const (
	// FieldBrowsing is the default state: displaying the field value.
	FieldBrowsing FieldRowState = iota
	// FieldEditing is active when the inline text input is open.
	FieldEditing
)

// FieldRow renders a single config field in browsing or editing mode.
// In browsing mode it displays the label and formatted value with optional
// focus indicator, dirty marker, and type-specific styling. In editing mode
// it shows an inline text input with description hint and validation error.
type FieldRow struct {
	value      any
	theme      *tui.Theme
	editErr    string
	origText   string
	meta       FieldMeta
	input      textinput.Model
	state      FieldRowState
	labelWidth int
	width      int
	focused    bool
	dirty      bool
	sensitive  bool
}

// NewFieldRow creates a FieldRow for the given field metadata.
func NewFieldRow(meta FieldMeta, theme *tui.Theme) *FieldRow {
	if theme == nil {
		theme = tui.DefaultTheme()
	}

	ti := textinput.New()
	ti.CharLimit = 256
	ti.Prompt = ""

	return &FieldRow{
		theme:     theme,
		meta:      meta,
		input:     ti,
		sensitive: meta.Type == SensitiveString,
	}
}

// --- State accessors ---.

// State returns the current FieldRowState.
func (f *FieldRow) State() FieldRowState {
	return f.state
}

// Meta returns the field metadata.
func (f *FieldRow) Meta() FieldMeta {
	return f.meta
}

// Value returns the current typed value.
func (f *FieldRow) Value() any {
	return f.value
}

// IsDirty reports whether the field has been modified.
func (f *FieldRow) IsDirty() bool {
	return f.dirty
}

// IsFocused reports whether the field currently has focus.
func (f *FieldRow) IsFocused() bool {
	return f.focused
}

// --- Mutators ---.

// SetValue updates the display value.
func (f *FieldRow) SetValue(v any) {
	f.value = v
}

// SetDirty marks or clears the dirty flag.
func (f *FieldRow) SetDirty(dirty bool) {
	f.dirty = dirty
}

// SetFocused marks or clears focus on this row.
func (f *FieldRow) SetFocused(focused bool) {
	f.focused = focused
}

// SetLabelWidth sets the column width for the label. This is typically the
// longest label among sibling fields so that values align vertically.
func (f *FieldRow) SetLabelWidth(w int) {
	f.labelWidth = w
}

// SetWidth sets the total rendering width available.
func (f *FieldRow) SetWidth(w int) {
	f.width = w
	// Reserve space for cursor(2) + padding(2) + label + gap(2) + some value
	inputWidth := max(w-f.labelWidth-8, 10)
	f.input.Width = inputWidth
}

// --- Edit lifecycle ---.

// EnterEditing transitions from browsing to editing state.
// For bool fields this is a no-op; use ToggleBool instead.
// Returns a tea.Cmd for the text input cursor blink.
func (f *FieldRow) EnterEditing() tea.Cmd {
	if f.meta.Type == Bool || f.meta.Type == ObjectList {
		return nil
	}

	f.state = FieldEditing
	f.editErr = ""

	// Pre-fill the input with the current display value.
	displayVal := f.formatValueForInput()
	f.origText = displayVal
	f.input.SetValue(displayVal)
	f.input.SetCursor(len(displayVal))
	f.input.Focus()

	if f.sensitive {
		f.input.EchoMode = textinput.EchoNormal
	}

	return textinput.Blink
}

// CancelEditing discards changes and returns to browsing state.
func (f *FieldRow) CancelEditing() {
	f.state = FieldBrowsing
	f.editErr = ""
	f.input.Blur()
}

// CommitEditing validates and commits the edited value.
// Returns the coerced value and nil error on success.
// On validation failure, the error message is stored and the field stays in editing state.
func (f *FieldRow) CommitEditing() (any, error) {
	raw := strings.TrimSpace(f.input.Value())

	// Coerce the raw string to the target type.
	coerced, err := CoerceValue(f.meta.Type, raw)
	if err != nil {
		f.editErr = err.Error()
		return nil, err
	}

	// Run field-level validation if defined.
	if f.meta.Validate != nil {
		if err := f.meta.Validate(coerced); err != nil {
			f.editErr = err.Error()
			return nil, err
		}
	}

	// Success: exit editing.
	f.editErr = ""
	f.state = FieldBrowsing
	f.input.Blur()
	f.value = coerced

	return coerced, nil
}

// ToggleBool inverts a boolean value. This is called directly on `e` for
// bool fields instead of entering editing state.
func (f *FieldRow) ToggleBool() (any, error) {
	if f.meta.Type != Bool {
		return nil, fmt.Errorf("cannot toggle non-bool field %q", f.meta.Key)
	}

	current, _ := f.value.(bool)
	newVal := !current
	f.value = newVal

	return newVal, nil
}

// UpdateInput forwards a tea.Msg to the embedded text input during editing.
// Returns any tea.Cmd produced by the text input.
func (f *FieldRow) UpdateInput(msg tea.Msg) tea.Cmd {
	if f.state != FieldEditing {
		return nil
	}

	// Clear error on new input.
	if _, ok := msg.(tea.KeyMsg); ok {
		f.editErr = ""
	}

	var cmd tea.Cmd
	f.input, cmd = f.input.Update(msg)
	return cmd
}

// --- Rendering ---.

// View renders the field row as a single line (browsing) or multi-line (editing).
func (f *FieldRow) View() string {
	if f.state == FieldEditing {
		return f.viewEditing()
	}
	return f.viewBrowsing()
}

// viewBrowsing renders the browsing-state row:
//
//	focused:   "> * Label      value"
//	unfocused: "    Label      value"
func (f *FieldRow) viewBrowsing() string {
	lw := f.effectiveLabelWidth()
	ew := f.effectiveWidth()

	// Build prefix: cursor + dirty marker.
	prefix := f.browsingPrefix()

	// Pad label to lw.
	label := f.meta.Label
	if len(label) < lw {
		label += strings.Repeat(" ", lw-len(label))
	}

	// Format the value with type-specific styling.
	value := f.formatDisplayValue()

	// Truncate value if it would exceed the available width.
	// Available = total - prefix(4) - label - gap(2).
	availValue := max(ew-4-lw-2, 3)
	value = f.truncate(value, availValue)

	// Apply styles.
	if f.focused {
		cursorStyle := lipgloss.NewStyle().
			Foreground(f.theme.Cursor).Bold(true)
		labelStyle := lipgloss.NewStyle().
			Foreground(f.theme.Accent).Bold(true)

		// prefix already contains cursor ">" and dirty marker
		return cursorStyle.Render(prefix[0:1]) +
			f.styleDirtyMarker(prefix) + "  " +
			labelStyle.Render(label) + "  " + value
	}

	return prefix + label + "  " + value
}

// browsingPrefix returns the 4-character prefix for a browsing row.
// Position 0: '>' if focused, ' ' otherwise
// Position 1: ' '
// Position 2: '*' if dirty, ' ' otherwise
// Position 3: ' '.
func (f *FieldRow) browsingPrefix() string {
	var b [4]byte
	if f.focused {
		b[0] = '>'
	} else {
		b[0] = ' '
	}
	b[1] = ' '
	if f.dirty {
		b[2] = '*'
	} else {
		b[2] = ' '
	}
	b[3] = ' '
	return string(b[:])
}

// styleDirtyMarker returns the styled portion of the prefix after the cursor char.
// This handles the dirty marker with highlight color.
func (f *FieldRow) styleDirtyMarker(prefix string) string {
	if len(prefix) < 4 {
		return ""
	}
	// Characters 1-3: " * " or "   "
	rest := prefix[1:4]
	if f.dirty {
		dirtyStyle := lipgloss.NewStyle().
			Foreground(f.theme.Highlight).Bold(true)
		return string(rest[0]) + dirtyStyle.Render(string(rest[1])) + string(rest[2])
	}
	return rest
}

// formatDisplayValue returns the styled display string for the current value.
func (f *FieldRow) formatDisplayValue() string {
	if f.value == nil {
		return f.emptyPlaceholder()
	}

	switch f.meta.Type {
	case Bool:
		return f.formatBool()
	case SensitiveString:
		return f.formatSensitive()
	case StringList:
		return f.formatStringList()
	case Int:
		return f.formatInt()
	default: // String, ObjectList
		return f.formatString()
	}
}

// formatBool renders a boolean with bold/dim and green/red coloring.
func (f *FieldRow) formatBool() string {
	b, ok := f.value.(bool)
	if !ok {
		return f.emptyPlaceholder()
	}

	if b {
		style := lipgloss.NewStyle().
			Foreground(f.theme.SuccessAccent).Bold(true)
		return style.Render("yes")
	}

	style := lipgloss.NewStyle().
		Foreground(f.theme.ErrorAccent).Faint(true)
	return style.Render("no")
}

// formatSensitive renders a sensitive string as masked when unfocused,
// or as the actual value when focused.
func (f *FieldRow) formatSensitive() string {
	s, ok := f.value.(string)
	if !ok || s == "" {
		return f.emptyPlaceholder()
	}

	if f.focused {
		return s
	}
	return strings.Repeat("*", min(len(s), 8))
}

// formatStringList renders a string slice as a comma-separated preview.
func (f *FieldRow) formatStringList() string {
	sl, ok := f.value.([]string)
	if !ok || len(sl) == 0 {
		return f.emptyPlaceholder()
	}
	return strings.Join(sl, ", ")
}

// formatInt renders an integer value.
func (f *FieldRow) formatInt() string {
	switch v := f.value.(type) {
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	default:
		return f.emptyPlaceholder()
	}
}

// formatString renders a plain string value.
func (f *FieldRow) formatString() string {
	s, ok := f.value.(string)
	if !ok || s == "" {
		return f.emptyPlaceholder()
	}
	return s
}

// emptyPlaceholder returns the styled "--" placeholder for empty values.
func (f *FieldRow) emptyPlaceholder() string {
	style := lipgloss.NewStyle().Foreground(f.theme.Muted)
	return style.Render("--")
}

// truncate limits a display string to maxWidth, appending "..." if truncated.
// It uses lipgloss.Width for accurate ANSI-aware width measurement.
func (f *FieldRow) truncate(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}

	w := lipgloss.Width(s)
	if w <= maxWidth {
		return s
	}

	if maxWidth <= 3 {
		return s[:maxWidth]
	}

	// Strip ANSI, truncate the raw text, then we lose styling. This is
	// acceptable since truncation only happens for very long values.
	raw := stripAnsi(s)
	if len(raw) <= maxWidth-3 {
		return raw + "..."
	}
	return raw[:maxWidth-3] + "..."
}

// viewEditing renders the editing-state row with inline text input.
func (f *FieldRow) viewEditing() string {
	lw := f.effectiveLabelWidth()

	// Label portion (same alignment as browsing).
	label := f.meta.Label
	if len(label) < lw {
		label += strings.Repeat(" ", lw-len(label))
	}

	cursorStyle := lipgloss.NewStyle().
		Foreground(f.theme.Cursor).Bold(true)
	labelStyle := lipgloss.NewStyle().
		Foreground(f.theme.Accent).Bold(true)

	// First line: cursor + label + input
	line := cursorStyle.Render(">") + "   " +
		labelStyle.Render(label) + "  " + f.input.View()

	// Description hint (if present).
	if f.meta.Description != "" {
		descStyle := lipgloss.NewStyle().
			Foreground(f.theme.Muted).Italic(true)
		line += "\n" + strings.Repeat(" ", 4+lw+2) +
			descStyle.Render(f.meta.Description)
	}

	// Validation error (if present).
	if f.editErr != "" {
		errStyle := lipgloss.NewStyle().
			Foreground(f.theme.ErrorAccent).Bold(true)
		line += "\n" + strings.Repeat(" ", 4+lw+2) +
			errStyle.Render(f.editErr)
	}

	return line
}

// --- Helpers ---.

// formatValueForInput converts the current value to a string suitable for
// pre-filling the text input.
func (f *FieldRow) formatValueForInput() string {
	if f.value == nil {
		return ""
	}

	switch f.meta.Type {
	case Int:
		switch v := f.value.(type) {
		case int:
			return strconv.Itoa(v)
		case int64:
			return strconv.FormatInt(v, 10)
		default:
			return ""
		}
	case StringList:
		if sl, ok := f.value.([]string); ok {
			return strings.Join(sl, ", ")
		}
		return ""
	default:
		if s, ok := f.value.(string); ok {
			return s
		}
		return fmt.Sprint(f.value)
	}
}

// effectiveLabelWidth returns the label column width, falling back to a minimum.
func (f *FieldRow) effectiveLabelWidth() int {
	if f.labelWidth > 0 {
		return f.labelWidth
	}
	lw := max(len(f.meta.Label), 10)
	return lw
}

// effectiveWidth returns the total rendering width.
func (f *FieldRow) effectiveWidth() int {
	if f.width > 0 {
		return f.width
	}
	return 72
}

// stripAnsi removes ANSI escape sequences from a string for truncation purposes.
func stripAnsi(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	inEsc := false
	for _, r := range s {
		if r == '\033' {
			inEsc = true
			continue
		}
		if inEsc {
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
				inEsc = false
			}
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

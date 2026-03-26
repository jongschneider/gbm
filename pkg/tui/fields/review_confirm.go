package fields

import (
	"fmt"
	"gbm/pkg/tui"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ReviewAttribute defines a single attribute displayed on the review screen.
type ReviewAttribute struct {
	Validator func(string) error // optional validation on edit confirm
	Label     string
	Key       string // WorkflowState field key (worktree_name, branch_name, base_branch, or custom)
	Editable  bool
}

// ReviewConfirm is a confirmation field that displays all worktree attributes
// in a summary view and allows inline editing of editable attributes.
type ReviewConfirm struct {
	state      *tui.WorkflowState
	theme      *tui.Theme
	key        string
	title      string
	wtDir      string
	attrs      []ReviewAttribute
	attrErrors []error // per-attribute validation errors (len == len(attrs))
	editInput  textinput.Model
	cursor     int
	width      int
	height     int
	editing    bool
	focused    bool
	complete   bool
	cancelled  bool
	value      bool
}

// NewReviewConfirm creates a ReviewConfirm field that displays the given attributes
// and reads/writes values through the provided WorkflowState pointer.
func NewReviewConfirm(key, title string, state *tui.WorkflowState, attrs []ReviewAttribute, wtDir string) *ReviewConfirm {
	ti := textinput.New()
	ti.CharLimit = 200
	ti.Width = 60

	return &ReviewConfirm{
		key:        key,
		title:      title,
		state:      state,
		attrs:      attrs,
		attrErrors: make([]error, len(attrs)),
		wtDir:      wtDir,
		cursor:     len(attrs), // default to Create button
		editInput:  ti,
		theme:      tui.DefaultTheme(),
	}
}

// Init implements Field.Init.
func (r *ReviewConfirm) Init() tea.Cmd {
	return nil
}

// Update implements Field.Update.
func (r *ReviewConfirm) Update(msg tea.Msg) (tui.Field, tea.Cmd) {
	if !r.focused {
		return r, nil
	}

	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		if r.editing {
			var cmd tea.Cmd
			r.editInput, cmd = r.editInput.Update(msg)
			return r, cmd
		}
		return r, nil
	}

	if r.editing {
		return r.updateEditing(keyMsg)
	}
	return r.updateNavigating(keyMsg)
}

// updateNavigating handles key events in navigation mode.
func (r *ReviewConfirm) updateNavigating(keyMsg tea.KeyMsg) (tui.Field, tea.Cmd) {
	totalItems := len(r.attrs) + 2 // attrs + Create + Cancel

	switch keyMsg.String() {
	case KeyUp, "k":
		if r.cursor > 0 {
			r.cursor--
		}
	case KeyDown, "j":
		if r.cursor < totalItems-1 {
			r.cursor++
		}
	case "tab":
		r.cursor = (r.cursor + 1) % totalItems

	case KeyEnter:
		return r.handleEnter()

	case "y", "Y":
		if r.hasValidationErrors() {
			return r, nil // block confirm while errors exist
		}
		r.value = true
		r.complete = true
		return r, func() tea.Msg { return tui.NextStepMsg{} }

	case "n", "N":
		r.value = false
		r.complete = true
		r.cancelled = true
		return r, func() tea.Msg { return tui.CancelMsg{} }
	}

	return r, nil
}

// handleEnter processes Enter based on current cursor position.
func (r *ReviewConfirm) handleEnter() (tui.Field, tea.Cmd) {
	createIdx := len(r.attrs)
	cancelIdx := len(r.attrs) + 1

	switch {
	case r.cursor < len(r.attrs):
		// Attribute row — start editing if editable
		attr := r.attrs[r.cursor]
		if !attr.Editable {
			return r, nil
		}
		r.editing = true
		currentVal := r.getAttrValue(attr.Key)
		r.editInput.SetValue(currentVal)
		r.editInput.SetCursor(len(currentVal))
		r.editInput.Focus()
		return r, textinput.Blink

	case r.cursor == createIdx:
		// Create button — blocked while validation errors exist
		if r.hasValidationErrors() {
			return r, nil
		}
		r.value = true
		r.complete = true
		return r, func() tea.Msg { return tui.NextStepMsg{} }

	case r.cursor == cancelIdx:
		// Cancel button
		r.value = false
		r.complete = true
		r.cancelled = true
		return r, func() tea.Msg { return tui.CancelMsg{} }
	}

	return r, nil
}

// updateEditing handles key events while editing an attribute inline.
func (r *ReviewConfirm) updateEditing(keyMsg tea.KeyMsg) (tui.Field, tea.Cmd) {
	idx := r.cursor

	switch keyMsg.Type { //nolint:exhaustive // Only handling relevant keys
	case tea.KeyEnter:
		newVal := strings.TrimSpace(r.editInput.Value())
		if newVal == "" {
			return r, nil
		}
		// Run validator if present
		attr := r.attrs[idx]
		if attr.Validator != nil {
			if err := attr.Validator(newVal); err != nil {
				r.attrErrors[idx] = err
				return r, nil
			}
		}
		// Validation passed — apply edit and clear error
		r.attrErrors[idx] = nil
		r.setAttrValue(attr.Key, newVal)
		r.editing = false
		r.editInput.Blur()
		return r, nil

	case tea.KeyEsc:
		// Cancel edit — restore original error state by re-validating
		r.editing = false
		r.editInput.Blur()
		attr := r.attrs[idx]
		if attr.Validator != nil {
			r.attrErrors[idx] = attr.Validator(r.getAttrValue(attr.Key))
		}
		return r, nil
	}

	// Clear error when user types
	if r.attrErrors[idx] != nil {
		r.attrErrors[idx] = nil
	}

	var cmd tea.Cmd
	r.editInput, cmd = r.editInput.Update(keyMsg)
	return r, cmd
}

// View implements Field.View.
func (r *ReviewConfirm) View() string {
	var b strings.Builder

	styles := r.getStyles()

	// Title
	b.WriteString(styles.Title.Render(r.title))
	b.WriteString("\n\n")

	// Compute label width for alignment
	labelWidth := 0
	for _, attr := range r.attrs {
		if len(attr.Label) > labelWidth {
			labelWidth = len(attr.Label)
		}
	}
	// Add label for the computed path row
	pathLabel := "Path"
	if len(pathLabel) > labelWidth {
		labelWidth = len(pathLabel)
	}

	// Render attribute rows
	for i, attr := range r.attrs {
		val := r.getAttrValue(attr.Key)
		r.renderRow(&b, i, attr.Label, val, attr.Editable, labelWidth)
	}

	// Render computed path row (read-only, not a selectable item)
	pathVal := r.computePath()
	if pathVal != "" {
		pathStyle := lipgloss.NewStyle().Foreground(r.theme.Muted)
		label := fmt.Sprintf("  %-*s", labelWidth, pathLabel)
		b.WriteString(pathStyle.Render(label + "  " + pathVal))
		b.WriteString("\n")
	}

	b.WriteString("\n")

	// Render Create/Cancel buttons
	r.renderButtons(&b)

	// Help text
	b.WriteString("\n")
	helpStyle := lipgloss.NewStyle().Foreground(r.theme.Muted).Italic(true)
	if r.editing {
		b.WriteString(helpStyle.Render("enter confirm • esc cancel"))
	} else {
		b.WriteString(helpStyle.Render("↑/↓ navigate • enter edit/select • y create • n cancel"))
	}
	b.WriteString("\n")

	return b.String()
}

// renderRow renders a single attribute row.
func (r *ReviewConfirm) renderRow(b *strings.Builder, idx int, label, value string, editable bool, labelWidth int) {
	isSelected := r.cursor == idx && r.focused
	isEditing := isSelected && r.editing

	// Cursor indicator
	cursor := "  "
	if isSelected {
		cursor = lipgloss.NewStyle().Foreground(r.theme.Cursor).Render("> ")
	}

	// Label
	paddedLabel := fmt.Sprintf("%-*s", labelWidth, label)

	attrErr := r.attrErrors[idx]

	if isEditing {
		// Show inline text input
		labelStyle := lipgloss.NewStyle().Bold(true).Foreground(r.theme.Accent)
		b.WriteString(cursor)
		b.WriteString(labelStyle.Render(paddedLabel))
		b.WriteString("  ")
		b.WriteString(r.editInput.View())
		b.WriteString("\n")
		// Show validation error if present
		if attrErr != nil {
			errStyle := lipgloss.NewStyle().Foreground(r.theme.ErrorAccent).Bold(true)
			b.WriteString("  ")
			b.WriteString(strings.Repeat(" ", labelWidth))
			b.WriteString("  ")
			b.WriteString(errStyle.Render(attrErr.Error()))
			b.WriteString("\n")
		}
		return
	}

	// Normal display
	var labelStyle, valueStyle lipgloss.Style
	if isSelected {
		labelStyle = lipgloss.NewStyle().Bold(true).Foreground(r.theme.SelectedFg).Background(r.theme.SelectedBg)
		valueStyle = lipgloss.NewStyle().Foreground(r.theme.SelectedFg).Background(r.theme.SelectedBg)
	} else if attrErr != nil {
		labelStyle = lipgloss.NewStyle().Bold(true).Foreground(r.theme.ErrorAccent)
		valueStyle = lipgloss.NewStyle().Foreground(r.theme.ErrorAccent)
	} else {
		labelStyle = lipgloss.NewStyle().Bold(true).Foreground(r.theme.Accent)
		valueStyle = lipgloss.NewStyle()
	}

	suffix := ""
	if editable && r.focused && !isSelected {
		suffix = lipgloss.NewStyle().Foreground(r.theme.BlurredMuted).Render("  [enter to edit]")
	}

	b.WriteString(cursor)
	b.WriteString(labelStyle.Render(paddedLabel))
	b.WriteString("  ")
	b.WriteString(valueStyle.Render(value))
	b.WriteString(suffix)
	b.WriteString("\n")
	// Show validation error below the row in navigation mode
	if attrErr != nil && !isEditing {
		errStyle := lipgloss.NewStyle().Foreground(r.theme.ErrorAccent).Bold(true)
		b.WriteString("  ")
		b.WriteString(strings.Repeat(" ", labelWidth))
		b.WriteString("  ")
		b.WriteString(errStyle.Render(attrErr.Error()))
		b.WriteString("\n")
	}
}

// renderButtons renders the Create/Cancel buttons.
func (r *ReviewConfirm) renderButtons(b *strings.Builder) {
	createIdx := len(r.attrs)
	cancelIdx := len(r.attrs) + 1

	createStyle := lipgloss.NewStyle().Padding(0, 2).MarginRight(2)
	cancelStyle := lipgloss.NewStyle().Padding(0, 2)

	hasErrors := r.hasValidationErrors()

	if r.focused {
		switch r.cursor {
		case createIdx:
			if hasErrors {
				// Dim Create button when validation errors exist
				createStyle = createStyle.Foreground(r.theme.BlurredMuted).Strikethrough(true)
			} else {
				createStyle = createStyle.Bold(true).
					Foreground(r.theme.InputFg).
					Background(r.theme.SuccessAccent)
			}
			cancelStyle = cancelStyle.Foreground(r.theme.BlurredMuted)
		case cancelIdx:
			createStyle = createStyle.Foreground(r.theme.BlurredMuted)
			cancelStyle = cancelStyle.Bold(true).
				Foreground(r.theme.InputFg).
				Background(r.theme.ErrorAccent)
		default:
			createStyle = createStyle.Foreground(r.theme.BlurredMuted)
			cancelStyle = cancelStyle.Foreground(r.theme.BlurredMuted)
		}
	} else {
		createStyle = createStyle.Foreground(r.theme.BlurredMuted)
		cancelStyle = cancelStyle.Foreground(r.theme.BlurredMuted)
	}

	createBtn := createStyle.Render("Create")
	cancelBtn := cancelStyle.Render("Cancel")

	b.WriteString("  ")
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Center, createBtn, cancelBtn))
	b.WriteString("\n")
}

// computePath returns the computed worktree path for display.
func (r *ReviewConfirm) computePath() string {
	if r.wtDir == "" {
		return ""
	}
	name := r.getAttrValue(tui.FieldKeyWorktreeName)
	if name == "" {
		return ""
	}
	return filepath.Join(r.wtDir, name)
}

// getAttrValue reads an attribute value from the WorkflowState.
func (r *ReviewConfirm) getAttrValue(key string) string {
	switch key {
	case tui.FieldKeyWorktreeName:
		return r.state.WorktreeName
	case tui.FieldKeyBranchName:
		return r.state.BranchName
	case tui.FieldKeyBaseBranch:
		return r.state.BaseBranch
	default:
		if v := r.state.GetField(key); v != nil {
			if s, ok := v.(string); ok {
				return s
			}
		}
		return ""
	}
}

// setAttrValue writes an attribute value back to the WorkflowState.
func (r *ReviewConfirm) setAttrValue(key, value string) {
	switch key {
	case tui.FieldKeyWorktreeName:
		r.state.WorktreeName = value
	case tui.FieldKeyBranchName:
		r.state.BranchName = value
	case tui.FieldKeyBaseBranch:
		r.state.BaseBranch = value
	default:
		r.state.SetField(key, value)
	}
}

// validateAll runs all attribute validators against current state values.
// Sets per-attribute errors and returns true if any errors exist.
func (r *ReviewConfirm) validateAll() bool {
	hasErrors := false
	for i, attr := range r.attrs {
		if attr.Validator == nil {
			r.attrErrors[i] = nil
			continue
		}
		val := r.getAttrValue(attr.Key)
		r.attrErrors[i] = attr.Validator(val)
		if r.attrErrors[i] != nil {
			hasErrors = true
		}
	}
	return hasErrors
}

// hasValidationErrors returns true if any attribute has a validation error.
func (r *ReviewConfirm) hasValidationErrors() bool {
	for _, err := range r.attrErrors {
		if err != nil {
			return true
		}
	}
	return false
}

// firstErrorIndex returns the index of the first attribute with a validation error, or -1.
func (r *ReviewConfirm) firstErrorIndex() int {
	for i, err := range r.attrErrors {
		if err != nil {
			return i
		}
	}
	return -1
}

// Focus implements Field.Focus.
func (r *ReviewConfirm) Focus() tea.Cmd {
	r.focused = true
	// Validate all attributes on entry — surface pre-existing issues
	if r.validateAll() {
		// Move cursor to first errored attribute so the user sees it immediately
		if idx := r.firstErrorIndex(); idx >= 0 {
			r.cursor = idx
		}
	}
	return nil
}

// Blur implements Field.Blur.
func (r *ReviewConfirm) Blur() tea.Cmd {
	r.focused = false
	return nil
}

// IsComplete implements Field.IsComplete.
func (r *ReviewConfirm) IsComplete() bool {
	return r.complete
}

// IsCancelled implements Field.IsCancelled.
func (r *ReviewConfirm) IsCancelled() bool {
	return r.cancelled
}

// Error implements Field.Error.
func (r *ReviewConfirm) Error() error {
	return nil
}

// Skip implements Field.Skip.
func (r *ReviewConfirm) Skip() bool {
	return false
}

// WithTheme implements Field.WithTheme.
func (r *ReviewConfirm) WithTheme(theme *tui.Theme) tui.Field {
	r.theme = theme
	return r
}

// WithWidth implements Field.WithWidth.
func (r *ReviewConfirm) WithWidth(width int) tui.Field {
	r.width = width
	r.editInput.Width = width - 20
	return r
}

// WithHeight implements Field.WithHeight.
func (r *ReviewConfirm) WithHeight(height int) tui.Field {
	r.height = height
	return r
}

// GetKey implements Field.GetKey.
func (r *ReviewConfirm) GetKey() string {
	return r.key
}

// GetValue implements Field.GetValue.
func (r *ReviewConfirm) GetValue() any {
	return r.value
}

// getStyles returns the appropriate styles based on focus state.
func (r *ReviewConfirm) getStyles() tui.FieldStyles {
	if r.theme == nil {
		r.theme = tui.DefaultTheme()
	}
	if r.focused {
		return r.theme.Focused
	}
	return r.theme.Blurred
}

var _ tui.Field = (*ReviewConfirm)(nil)

// Package config provides TUI components for configuration management.
package config

import (
	"errors"
	"gbm/pkg/tui"
	"gbm/pkg/tui/fields"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// BasicsFormConfig holds configuration for the Basics form.
type BasicsFormConfig struct {
	OnSave        func(data map[string]string) error
	Theme         *tui.Theme
	DefaultBranch string
	WorktreesDir  string
}

// BasicsForm renders a form for editing basic config settings (default_branch, worktrees_dir).
type BasicsForm struct {
	theme                *tui.Theme
	onSave               func(data map[string]string) error
	discardField         tui.Field
	defaultBranchField   tui.Field
	worktreesDirField    tui.Field
	validationOverlay    *fields.ValidationOverlay
	helpOverlay          *tui.HelpOverlay
	width                int
	height               int
	focusedFieldIdx      int
	submitted            bool
	cancelled            bool
	showConfirmDiscard   bool
	showValidationErrors bool
	showHelp             bool
	insertMode           bool // vim-style insert mode for text inputs
}

// NewBasicsForm creates a new Basics configuration form.
func NewBasicsForm(config BasicsFormConfig) *BasicsForm {
	if config.Theme == nil {
		config.Theme = tui.DefaultTheme()
	}

	// Validator for branch names: alphanumeric, -, /
	branchValidator := func(value string) error {
		if value == "" {
			return errors.New("default_branch is required")
		}
		// Allow alphanumeric, hyphens, forward slashes, underscores
		for _, r := range value {
			isAlphaNum := (r >= 'a' && r <= 'z') ||
				(r >= 'A' && r <= 'Z') ||
				(r >= '0' && r <= '9')
			isSpecial := r == '-' || r == '/' || r == '_'
			if !isAlphaNum && !isSpecial {
				return errors.New("branch name contains invalid characters (use alphanumeric, -, /, _)")
			}
		}
		return nil
	}

	// Validator for directory paths: non-empty
	dirValidator := func(value string) error {
		if value == "" {
			return errors.New("worktrees_dir is required")
		}
		return nil
	}

	defaultBranchFieldPtr := fields.NewTextInput(
		"default_branch",
		"Default Branch",
		"Default branch for new worktrees (e.g., main, develop)",
	)
	defaultBranchFieldPtr.WithValidator(branchValidator)
	defaultBranchField := defaultBranchFieldPtr.
		WithDefault(config.DefaultBranch).
		WithTheme(config.Theme)

	worktreesDirFieldPtr := fields.NewTextInput(
		"worktrees_dir",
		"Worktrees Directory",
		"Directory where worktrees are created (e.g., ./worktrees)",
	)
	worktreesDirFieldPtr.WithValidator(dirValidator)
	worktreesDirField := worktreesDirFieldPtr.
		WithDefault(config.WorktreesDir).
		WithTheme(config.Theme)

	// Create discard confirmation field
	discardField := fields.NewConfirm(
		"discard_confirm",
		"Discard unsaved changes?",
	).WithTheme(config.Theme)

	return &BasicsForm{
		theme:              config.Theme,
		onSave:             config.OnSave,
		discardField:       discardField,
		defaultBranchField: defaultBranchField,
		worktreesDirField:  worktreesDirField,
		focusedFieldIdx:    0,
	}
}

// Init implements tea.Model.
func (f *BasicsForm) Init() tea.Cmd {
	return tea.Batch(
		f.focusedField().Init(),
		f.focusedField().Focus(),
	)
}

// Update implements tea.Model.
func (f *BasicsForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle help overlay
	if f.showHelp {
		return f.handleHelpOverlay(msg)
	}

	// Handle validation error overlay
	if f.showValidationErrors {
		return f.handleValidationOverlay(msg)
	}

	// Handle discard confirmation modal
	if f.showConfirmDiscard {
		return f.handleDiscardConfirmation(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return f.handleKeyMsg(msg)
	case tea.WindowSizeMsg:
		return f.handleWindowSize(msg)
	}

	// Delegate to focused field
	newField, cmd := f.focusedField().Update(msg)
	f.updateFocusedField(newField)
	return f, cmd
}

// handleHelpOverlay processes input while showing the help overlay.
func (f *BasicsForm) handleHelpOverlay(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return f, nil
	}

	switch keyMsg.String() {
	case "esc", "?", "enter":
		f.showHelp = false
		return f, f.focusedField().Focus()
	}
	return f, nil
}

// handleValidationOverlay processes input while showing the validation error overlay.
func (f *BasicsForm) handleValidationOverlay(msg tea.Msg) (tea.Model, tea.Cmd) {
	_, cmd := f.validationOverlay.Update(msg)

	// Check for dismissal message
	if _, ok := msg.(tea.KeyMsg); ok {
		keyMsg := msg.(tea.KeyMsg)
		if keyMsg.String() == "esc" || keyMsg.String() == "b" || keyMsg.String() == "enter" {
			f.showValidationErrors = false
			return f, f.focusedField().Focus()
		}
	}

	return f, cmd
}

// handleDiscardConfirmation processes input while showing the discard confirmation modal.
func (f *BasicsForm) handleDiscardConfirmation(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return f, nil
	}

	newField, cmd := f.discardField.Update(keyMsg)
	f.discardField = newField

	// Check if user answered the confirmation
	confirmVal, ok := f.discardField.GetValue().(bool)
	if ok {
		// Check the type of confirmation
		if keyMsg.String() == "enter" || keyMsg.String() == "y" || confirmVal {
			// User confirmed discard
			f.showConfirmDiscard = false
			f.cancelled = true
			return f, func() tea.Msg {
				return tui.BackBoundaryMsg{}
			}
		} else if keyMsg.String() == "n" || !confirmVal {
			// User cancelled discard
			f.showConfirmDiscard = false
			return f, f.focusedField().Focus()
		}
	}
	return f, cmd
}

// handleKeyMsg processes keyboard input for form navigation and actions.
func (f *BasicsForm) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Insert mode: pass all keys to the field except Esc
	if f.insertMode {
		if msg.Type == tea.KeyEsc {
			f.insertMode = false
			return f, nil
		}
		// Pass key to the focused field
		field, cmd := f.focusedField().Update(msg)
		f.updateFocusedField(field)
		return f, cmd
	}

	// Normal mode (vim-style navigation)

	// Handle vim-style keys in normal mode
	if msg.Type == tea.KeyRunes && len(msg.Runes) == 1 {
		switch msg.Runes[0] {
		case 'j':
			return f.nextField()
		case 'k':
			return f.prevField()
		case 'i':
			f.insertMode = true
			return f, nil
		}
	}

	switch msg.Type { //nolint:exhaustive // Only handling relevant keys
	case tea.KeyTab:
		return f.nextField()

	case tea.KeyShiftTab:
		return f.prevField()

	case tea.KeyEnter:
		if f.focusedFieldIdx == 1 {
			f.submitted = true
			return f, func() tea.Msg {
				return tui.NextStepMsg{}
			}
		}
		return f.nextField()

	case tea.KeyRunes:
		return f.handleRuneKey(msg)
	}

	return f, nil
}

// nextField moves focus to the next field.
func (f *BasicsForm) nextField() (tea.Model, tea.Cmd) {
	f.focusedField().Blur()
	f.focusedFieldIdx = (f.focusedFieldIdx + 1) % 2
	return f, f.focusedField().Focus()
}

// prevField moves focus to the previous field.
func (f *BasicsForm) prevField() (tea.Model, tea.Cmd) {
	f.focusedField().Blur()
	f.focusedFieldIdx = (f.focusedFieldIdx - 1 + 2) % 2
	return f, f.focusedField().Focus()
}

// updateFocusedField updates the focused field after Update.
func (f *BasicsForm) updateFocusedField(field tui.Field) {
	if f.focusedFieldIdx == 0 {
		f.defaultBranchField = field
	} else {
		f.worktreesDirField = field
	}
}

// handleRuneKey processes character input (s for save, q for quit, ? for help).
func (f *BasicsForm) handleRuneKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if len(msg.Runes) == 0 {
		return f, nil
	}

	switch msg.Runes[0] {
	case '?':
		f.showHelp = true
		f.helpOverlay = tui.NewHelpOverlay().
			WithTheme(f.theme).
			WithWidth(f.width).
			WithHeight(f.height)
		return f, nil

	case 's':
		// Validate all fields before saving
		errs := f.Validate()
		if len(errs) > 0 {
			f.showValidationErrors = true
			f.validationOverlay = fields.NewValidationOverlay(errs).
				WithTheme(f.theme).
				WithWidth(f.width).
				WithHeight(f.height)
			return f, nil
		}

		if f.onSave != nil {
			err := f.onSave(f.GetValue())
			if err != nil {
				// Show save error as validation error
				f.showValidationErrors = true
				f.validationOverlay = fields.NewValidationOverlay([]string{err.Error()}).
					WithTheme(f.theme).
					WithTitle("Save Error").
					WithWidth(f.width).
					WithHeight(f.height)
				return f, nil
			}
		}
		return f, func() tea.Msg {
			return tui.BackBoundaryMsg{}
		}

	case 'q':
		f.showConfirmDiscard = true
		return f, f.discardField.Focus()
	}

	return f, nil
}

// Validate runs validators on all fields and returns a list of error messages.
// It also sets the error state on fields that fail validation so they are highlighted.
func (f *BasicsForm) Validate() []string {
	var errs []string

	// Validate default branch field
	if textField, ok := f.defaultBranchField.(*fields.TextInput); ok {
		err := textField.RunValidator()
		if err != nil {
			errs = append(errs, "Default Branch: "+err.Error())
		}
	}

	// Validate worktrees dir field
	if textField, ok := f.worktreesDirField.(*fields.TextInput); ok {
		err := textField.RunValidator()
		if err != nil {
			errs = append(errs, "Worktrees Directory: "+err.Error())
		}
	}

	return errs
}

// handleWindowSize updates dimensions on terminal resize.
func (f *BasicsForm) handleWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	f.width = msg.Width
	f.height = msg.Height
	f.defaultBranchField = f.defaultBranchField.WithWidth(msg.Width).WithHeight(msg.Height)
	f.worktreesDirField = f.worktreesDirField.WithWidth(msg.Width).WithHeight(msg.Height)
	f.discardField = f.discardField.WithWidth(msg.Width).WithHeight(msg.Height)
	if f.validationOverlay != nil {
		f.validationOverlay = f.validationOverlay.WithWidth(msg.Width).WithHeight(msg.Height)
	}

	return f, nil
}

// View implements tea.Model.
func (f *BasicsForm) View() string {
	if f.showHelp && f.helpOverlay != nil {
		return f.helpOverlay.View()
	}

	if f.showValidationErrors && f.validationOverlay != nil {
		return f.validationOverlay.View()
	}

	if f.showConfirmDiscard {
		return f.discardField.View()
	}

	lines := []string{
		f.theme.Focused.Title.Render("Basic Settings"),
		"",
		f.defaultBranchField.View(),
		"",
		f.worktreesDirField.View(),
		"",
		f.theme.Blurred.Description.Render("Tab=next field, s=save, ?=help, q=quit"),
	}

	return strings.Join(lines, "\n")
}

// GetValue returns the form data as a map.
func (f *BasicsForm) GetValue() map[string]string {
	defaultBranchVal, _ := f.defaultBranchField.GetValue().(string)
	worktreesDirVal, _ := f.worktreesDirField.GetValue().(string)

	return map[string]string{
		"default_branch": defaultBranchVal,
		"worktrees_dir":  worktreesDirVal,
	}
}

// IsComplete returns whether the form has been submitted.
func (f *BasicsForm) IsComplete() bool {
	return f.submitted
}

// IsCancelled returns whether the form was cancelled.
func (f *BasicsForm) IsCancelled() bool {
	return f.cancelled
}

// ShowConfirmDiscard returns whether the discard confirmation is shown.
func (f *BasicsForm) ShowConfirmDiscard() bool {
	return f.showConfirmDiscard
}

// Focus gives the form keyboard focus and focuses the first field.
func (f *BasicsForm) Focus() tea.Cmd {
	return f.focusedField().Focus()
}

// Blur removes keyboard focus from the form and all its fields.
func (f *BasicsForm) Blur() tea.Cmd {
	f.insertMode = false
	f.focusedField().Blur()
	return nil
}

// focusedField returns the currently focused field.
func (f *BasicsForm) focusedField() tui.Field {
	if f.focusedFieldIdx == 0 {
		return f.defaultBranchField
	}

	return f.worktreesDirField
}

// FocusedYOffset returns the line number where the focused field starts.
// This implements the tui.FocusReporter interface for auto-scrolling support.
func (f *BasicsForm) FocusedYOffset() int {
	// Count lines helper
	countLines := func(s string) int {
		return strings.Count(s, "\n") + 1
	}

	// Title + empty line = 2 lines
	lineCount := 2

	// Field 0: defaultBranchField
	if f.focusedFieldIdx == 0 {
		return lineCount
	}
	lineCount += countLines(f.defaultBranchField.View()) + 1 // field + empty line

	// Field 1: worktreesDirField
	return lineCount
}

// InInsertMode reports whether the form is in insert mode.
func (f *BasicsForm) InInsertMode() bool { return f.insertMode }

// Ensure BasicsForm implements tui.FocusReporter.
var _ tui.FocusReporter = (*BasicsForm)(nil)

// Ensure BasicsForm implements tui.InsertModeReporter.
var _ tui.InsertModeReporter = (*BasicsForm)(nil)

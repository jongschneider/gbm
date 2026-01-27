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
	DefaultBranch string
	WorktreesDir  string
	OnSave        func(data map[string]string) error
	Theme         *tui.Theme
}

// BasicsForm renders a form for editing basic config settings (default_branch, worktrees_dir).
type BasicsForm struct {
	theme              *tui.Theme
	onSave             func(data map[string]string) error
	discardField       tui.Field
	defaultBranchField tui.Field
	worktreesDirField  tui.Field
	width              int
	height             int
	focusedFieldIdx    int
	submitted          bool
	cancelled          bool
	showConfirmDiscard bool
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
	switch msg.Type {
	case tea.KeyTab:
		f.focusedField().Blur()
		f.focusedFieldIdx = (f.focusedFieldIdx + 1) % 2
		return f, f.focusedField().Focus()

	case tea.KeyShiftTab:
		f.focusedField().Blur()
		f.focusedFieldIdx = (f.focusedFieldIdx - 1 + 2) % 2
		return f, f.focusedField().Focus()

	case tea.KeyEnter:
		if f.focusedFieldIdx == 1 {
			f.submitted = true
			return f, func() tea.Msg {
				return tui.NextStepMsg{}
			}
		}
		f.focusedField().Blur()
		f.focusedFieldIdx = (f.focusedFieldIdx + 1) % 2
		return f, f.focusedField().Focus()

	case tea.KeyEsc:
		f.cancelled = true
		return f, func() tea.Msg {
			return tui.BackBoundaryMsg{}
		}

	case tea.KeyRunes:
		return f.handleRuneKey(msg)
	}

	return f, nil
}

// handleRuneKey processes character input (s for save, q for quit).
func (f *BasicsForm) handleRuneKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if len(msg.Runes) == 0 {
		return f, nil
	}

	switch msg.Runes[0] {
	case 's':
		if f.onSave != nil {
			err := f.onSave(f.GetValue())
			if err != nil {
				// In a full implementation, show error overlay
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

// handleWindowSize updates dimensions on terminal resize.
func (f *BasicsForm) handleWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	f.width = msg.Width
	f.height = msg.Height
	f.defaultBranchField = f.defaultBranchField.WithWidth(msg.Width).WithHeight(msg.Height)
	f.worktreesDirField = f.worktreesDirField.WithWidth(msg.Width).WithHeight(msg.Height)
	f.discardField = f.discardField.WithWidth(msg.Width).WithHeight(msg.Height)

	return f, nil
}

// View implements tea.Model.
func (f *BasicsForm) View() string {
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
		f.theme.Blurred.Description.Render("Tab=next field, s=save, q=quit"),
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

// focusedField returns the currently focused field.
func (f *BasicsForm) focusedField() tui.Field {
	if f.focusedFieldIdx == 0 {
		return f.defaultBranchField
	}

	return f.worktreesDirField
}

// updateFocusedField updates the focused field after Update.
func (f *BasicsForm) updateFocusedField(field tui.Field) {
	if f.focusedFieldIdx == 0 {
		f.defaultBranchField = field
	} else {
		f.worktreesDirField = field
	}
}

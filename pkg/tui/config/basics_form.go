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
	theme              *tui.Theme
	onSave             func(data map[string]string) error
	defaultBranchField tui.Field
	worktreesDirField  tui.Field
	width              int
	height             int
	focusedFieldIdx    int
	submitted          bool
	cancelled          bool
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

	return &BasicsForm{
		theme:              config.Theme,
		onSave:             config.OnSave,
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

// handleKeyMsg processes keyboard input for form navigation and actions.
func (f *BasicsForm) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type { //nolint:exhaustive // Only handling relevant keys
	case tea.KeyEsc:
		return f, func() tea.Msg {
			return tui.BackBoundaryMsg{}
		}

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
	}

	// Pass all other keys to the focused field
	field, cmd := f.focusedField().Update(msg)
	f.updateFocusedField(field)
	return f, cmd
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

	return f, nil
}

// View implements tea.Model.
func (f *BasicsForm) View() string {
	lines := []string{
		f.theme.Focused.Title.Render("Basic Settings"),
		"",
		f.defaultBranchField.View(),
		"",
		f.worktreesDirField.View(),
		"",
		f.theme.Blurred.Description.Render("Tab/Shift+Tab=navigate  Esc=back"),
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

// Focus gives the form keyboard focus and focuses the first field.
func (f *BasicsForm) Focus() tea.Cmd {
	return f.focusedField().Focus()
}

// Blur removes keyboard focus from the form and all its fields.
func (f *BasicsForm) Blur() tea.Cmd {
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

// Ensure BasicsForm implements tui.FocusReporter.
var _ tui.FocusReporter = (*BasicsForm)(nil)

// Package config provides TUI components for configuration management.
package config

import (
	"fmt"
	"strings"

	"gbm/pkg/tui"
	"gbm/pkg/tui/fields"
	tea "github.com/charmbracelet/bubbletea"
)

// BasicsFormConfig holds configuration for the Basics form
type BasicsFormConfig struct {
	DefaultBranch string
	WorktreesDir  string
	Theme         *tui.Theme
}

// BasicsForm renders a form for editing basic config settings (default_branch, worktrees_dir)
type BasicsForm struct {
	defaultBranchField tui.Field
	worktreesDirField  tui.Field
	focusedFieldIdx    int
	width              int
	height             int
	theme              *tui.Theme
	submitted          bool
	cancelled          bool
}

// NewBasicsForm creates a new Basics configuration form
func NewBasicsForm(config BasicsFormConfig) *BasicsForm {
	if config.Theme == nil {
		config.Theme = tui.DefaultTheme()
	}

	// Validator for branch names: alphanumeric, -, /
	branchValidator := func(value string) error {
		if value == "" {
			return fmt.Errorf("default_branch is required")
		}
		// Allow alphanumeric, hyphens, forward slashes, underscores
		for _, r := range value {
			isAlphaNum := (r >= 'a' && r <= 'z') ||
				(r >= 'A' && r <= 'Z') ||
				(r >= '0' && r <= '9')
			isSpecial := r == '-' || r == '/' || r == '_'
			if !isAlphaNum && !isSpecial {
				return fmt.Errorf("branch name contains invalid characters (use alphanumeric, -, /, _)")
			}
		}
		return nil
	}

	// Validator for directory paths: non-empty
	dirValidator := func(value string) error {
		if value == "" {
			return fmt.Errorf("worktrees_dir is required")
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
		defaultBranchField: defaultBranchField,
		worktreesDirField:  worktreesDirField,
		focusedFieldIdx:    0,
		theme:              config.Theme,
	}
}

// Init implements tea.Model
func (f *BasicsForm) Init() tea.Cmd {
	return tea.Batch(
		f.focusedField().Init(),
		f.focusedField().Focus(),
	)
}

// Update implements tea.Model
func (f *BasicsForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyTab:
			// Move to next field
			f.focusedField().Blur()
			f.focusedFieldIdx = (f.focusedFieldIdx + 1) % 2
			return f, f.focusedField().Focus()

		case tea.KeyShiftTab:
			// Move to previous field
			f.focusedField().Blur()
			f.focusedFieldIdx = (f.focusedFieldIdx - 1 + 2) % 2
			return f, f.focusedField().Focus()

		case tea.KeyEnter:
			// Submit form (if on last field)
			if f.focusedFieldIdx == 1 {
				f.submitted = true
				return f, func() tea.Msg {
					return tui.NextStepMsg{}
				}
			}
			// Otherwise move to next field
			f.focusedField().Blur()
			f.focusedFieldIdx = (f.focusedFieldIdx + 1) % 2
			return f, f.focusedField().Focus()

		case tea.KeyEsc:
			f.cancelled = true
			return f, func() tea.Msg {
				return tui.BackBoundaryMsg{}
			}
		}

	case tea.WindowSizeMsg:
		f.width = msg.Width
		f.height = msg.Height
		f.defaultBranchField = f.defaultBranchField.WithWidth(msg.Width).WithHeight(msg.Height)
		f.worktreesDirField = f.worktreesDirField.WithWidth(msg.Width).WithHeight(msg.Height)
	}

	// Delegate to focused field
	newField, cmd := f.focusedField().Update(msg)
	f.updateFocusedField(newField)
	return f, cmd
}

// View implements tea.Model
func (f *BasicsForm) View() string {
	var lines []string
	lines = append(lines, f.theme.Focused.Title.Render("Basic Settings"))
	lines = append(lines, "")
	lines = append(lines, f.defaultBranchField.View())
	lines = append(lines, "")
	lines = append(lines, f.worktreesDirField.View())
	lines = append(lines, "")
	lines = append(lines, f.theme.Blurred.Description.Render("Tab to move between fields, Enter to confirm, Esc to cancel"))
	return strings.Join(lines, "\n")
}

// GetValue returns the form data as a map
func (f *BasicsForm) GetValue() map[string]string {
	return map[string]string{
		"default_branch": f.defaultBranchField.GetValue().(string),
		"worktrees_dir":  f.worktreesDirField.GetValue().(string),
	}
}

// IsComplete returns whether the form has been submitted
func (f *BasicsForm) IsComplete() bool {
	return f.submitted
}

// IsCancelled returns whether the form was cancelled
func (f *BasicsForm) IsCancelled() bool {
	return f.cancelled
}

// focusedField returns the currently focused field
func (f *BasicsForm) focusedField() tui.Field {
	if f.focusedFieldIdx == 0 {
		return f.defaultBranchField
	}
	return f.worktreesDirField
}

// updateFocusedField updates the focused field after Update
func (f *BasicsForm) updateFocusedField(field tui.Field) {
	if f.focusedFieldIdx == 0 {
		f.defaultBranchField = field
	} else {
		f.worktreesDirField = field
	}
}

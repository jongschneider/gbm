// Package config provides TUI components for configuration management.
package config

import (
	"errors"
	"gbm/pkg/tui"
	"gbm/pkg/tui/fields"
	"net/url"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// JiraFormConfig holds configuration for the JIRA form.
type JiraFormConfig struct {
	Enabled  bool
	Host     string
	Username string
	APIToken string
	OnSave   func(data map[string]interface{}) error
	Theme    *tui.Theme
}

// JiraForm renders a form for editing JIRA configuration with enable/disable toggle.
// When disabled, all subsection fields are hidden.
type JiraForm struct {
	theme              *tui.Theme
	onSave             func(data map[string]interface{}) error
	enableField        tui.Field
	discardField       tui.Field
	serverHostField    tui.Field
	serverUserField    tui.Field
	serverTokenField   tui.Field
	width              int
	height             int
	focusedFieldIdx    int
	submitted          bool
	cancelled          bool
	showConfirmDiscard bool
	enabled            bool
}

// NewJiraForm creates a new JIRA configuration form.
func NewJiraForm(config JiraFormConfig) *JiraForm {
	if config.Theme == nil {
		config.Theme = tui.DefaultTheme()
	}

	// Enable/disable toggle
	enableFieldPtr := fields.NewConfirm("jira_enabled", "Enable JIRA Integration?")
	enableField := enableFieldPtr.WithTheme(config.Theme)

	// URL validator for JIRA host
	hostValidator := func(value string) error {
		if value == "" {
			return errors.New("JIRA host is required")
		}
		// Check if it looks like a URL
		if !strings.HasPrefix(value, "http://") && !strings.HasPrefix(value, "https://") {
			return errors.New("JIRA host must start with http:// or https://")
		}
		if _, err := url.Parse(value); err != nil {
			return errors.New("invalid URL format")
		}
		return nil
	}

	// Username validator
	userValidator := func(value string) error {
		if value == "" {
			return errors.New("username is required")
		}
		return nil
	}

	// Token validator
	tokenValidator := func(value string) error {
		if value == "" {
			return errors.New("API token is required")
		}
		return nil
	}

	serverHostFieldPtr := fields.NewTextInput(
		"jira_host",
		"JIRA Host",
		"JIRA server URL (e.g., https://jira.company.com)",
	)
	serverHostFieldPtr.WithValidator(hostValidator)
	serverHostField := serverHostFieldPtr.
		WithDefault(config.Host).
		WithTheme(config.Theme)

	serverUserFieldPtr := fields.NewTextInput(
		"jira_username",
		"Username",
		"JIRA username or email",
	)
	serverUserFieldPtr.WithValidator(userValidator)
	serverUserField := serverUserFieldPtr.
		WithDefault(config.Username).
		WithTheme(config.Theme)

	serverTokenFieldPtr := fields.NewTextInput(
		"jira_api_token",
		"API Token",
		"JIRA API token (masked input)",
	)
	serverTokenFieldPtr.WithValidator(tokenValidator)
	serverTokenFieldPtr.SetMasked(true) // Mask the token display
	serverTokenField := serverTokenFieldPtr.
		WithDefault(config.APIToken).
		WithTheme(config.Theme)

	form := &JiraForm{
		theme:            config.Theme,
		onSave:           config.OnSave,
		enableField:      enableField,
		enabled:          config.Enabled,
		serverHostField:  serverHostField,
		serverUserField:  serverUserField,
		serverTokenField: serverTokenField,
		focusedFieldIdx:  0,
	}

	return form
}

// Init implements tea.Model.Init.
func (f *JiraForm) Init() tea.Cmd {
	return f.enableField.Focus()
}

// Update implements tea.Model.Update.
func (f *JiraForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle discard confirmation
	if f.showConfirmDiscard {
		newField, cmd := f.discardField.Update(msg)
		f.discardField = newField

		confirm, ok := f.discardField.(*fields.Confirm)
		if ok && confirm.IsComplete() {
			if confirm.GetValue().(bool) {
				// User confirmed discard
				f.cancelled = true
				return f, func() tea.Msg {
					return tui.BackBoundaryMsg{}
				}
			}
			// User said no - return to editing
			f.showConfirmDiscard = false
			return f, f.focusedField().Focus()
		}
		return f, cmd
	}

	// Handle window resize
	if sizeMsg, ok := msg.(tea.WindowSizeMsg); ok {
		return f.handleWindowSize(sizeMsg)
	}

	// Handle key messages
	keyMsg, ok := msg.(tea.KeyMsg)
	if ok {
		return f.handleKeyMsg(keyMsg)
	}

	// Pass to focused field
	field, cmd := f.focusedField().Update(msg)
	f.updateFocusedField(field)

	// Check if enable field was completed
	if f.focusedFieldIdx == 0 {
		confirm, ok := field.(*fields.Confirm)
		if ok && confirm.IsComplete() {
			newEnabled := confirm.GetValue().(bool)
			if newEnabled != f.enabled {
				f.enabled = newEnabled
			}
			// Move to next field or back
			if newEnabled {
				f.focusedFieldIdx = 1
				return f, f.focusedField().Focus()
			} else {
				// Can only toggle, no more fields
				return f, nil
			}
		}
	}

	return f, cmd
}

// View implements tea.Model.View.
func (f *JiraForm) View() string {
	if f.showConfirmDiscard {
		return f.discardField.View()
	}

	lines := []string{
		f.theme.Focused.Title.Render("JIRA Configuration"),
		"",
	}

	// Always show enable field
	lines = append(lines, f.enableField.View())
	lines = append(lines, "")

	// Show server fields if enabled
	if f.enabled {
		lines = append(lines, f.theme.Blurred.Description.Render("Server Configuration"))
		lines = append(lines, "")
		lines = append(lines, f.serverHostField.View())
		lines = append(lines, "")
		lines = append(lines, f.serverUserField.View())
		lines = append(lines, "")
		lines = append(lines, f.serverTokenField.View())
		lines = append(lines, "")
	}

	// Help text
	lines = append(lines, f.theme.Blurred.Description.Render("Tab=next, Shift+Tab=prev, s=save, q=quit"))

	return strings.Join(lines, "\n")
}

// handleKeyMsg processes keyboard input.
func (f *JiraForm) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyTab:
		// Navigate to next field based on enabled state
		f.focusedField().Blur()
		if f.enabled {
			// Can navigate through all fields
			maxFields := 4 // enable + 3 server fields
			f.focusedFieldIdx = (f.focusedFieldIdx + 1) % maxFields
		} else {
			// Only one field
			f.focusedFieldIdx = 0
		}
		return f, f.focusedField().Focus()

	case tea.KeyShiftTab:
		f.focusedField().Blur()
		if f.enabled {
			maxFields := 4
			f.focusedFieldIdx = (f.focusedFieldIdx - 1 + maxFields) % maxFields
		} else {
			f.focusedFieldIdx = 0
		}
		return f, f.focusedField().Focus()

	case tea.KeyEsc:
		f.cancelled = true
		return f, func() tea.Msg {
			return tui.BackBoundaryMsg{}
		}

	case tea.KeyRunes:
		if len(msg.Runes) == 0 {
			return f, nil
		}
		switch msg.Runes[0] {
		case 's':
			// Save
			if f.onSave != nil {
				data := f.GetValue()
				if err := f.onSave(data); err != nil {
					// TODO: Show error overlay
					return f, nil
				}
			}
			f.submitted = true
			return f, func() tea.Msg {
				return tui.BackBoundaryMsg{}
			}

		case 'q':
			// Show discard confirmation
			f.showConfirmDiscard = true
			discardConfirm := fields.NewConfirm("discard", "Discard unsaved changes?")
			f.discardField = discardConfirm.WithTheme(f.theme)
			return f, f.discardField.Focus()
		}
	}

	return f, nil
}

// handleWindowSize updates dimensions on terminal resize.
func (f *JiraForm) handleWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	f.width = msg.Width
	f.height = msg.Height
	f.enableField = f.enableField.WithWidth(msg.Width).WithHeight(msg.Height)
	f.serverHostField = f.serverHostField.WithWidth(msg.Width).WithHeight(msg.Height)
	f.serverUserField = f.serverUserField.WithWidth(msg.Width).WithHeight(msg.Height)
	f.serverTokenField = f.serverTokenField.WithWidth(msg.Width).WithHeight(msg.Height)
	if f.discardField != nil {
		f.discardField = f.discardField.WithWidth(msg.Width).WithHeight(msg.Height)
	}
	return f, nil
}

// focusedField returns the currently focused field.
func (f *JiraForm) focusedField() tui.Field {
	switch f.focusedFieldIdx {
	case 0:
		return f.enableField
	case 1:
		return f.serverHostField
	case 2:
		return f.serverUserField
	case 3:
		return f.serverTokenField
	default:
		return f.enableField
	}
}

// updateFocusedField updates the focused field after Update.
func (f *JiraForm) updateFocusedField(field tui.Field) {
	switch f.focusedFieldIdx {
	case 0:
		f.enableField = field
	case 1:
		f.serverHostField = field
	case 2:
		f.serverUserField = field
	case 3:
		f.serverTokenField = field
	}
}

// GetValue returns the form data as a map.
func (f *JiraForm) GetValue() map[string]interface{} {
	data := make(map[string]interface{})
	data["jira_enabled"] = f.enabled

	if f.enabled {
		hostVal, _ := f.serverHostField.GetValue().(string)
		userVal, _ := f.serverUserField.GetValue().(string)
		tokenVal, _ := f.serverTokenField.GetValue().(string)

		data["jira_host"] = hostVal
		data["jira_username"] = userVal
		data["jira_api_token"] = tokenVal
	}

	return data
}

// IsComplete returns whether the form has been submitted.
func (f *JiraForm) IsComplete() bool {
	return f.submitted
}

// IsCancelled returns whether the form was cancelled.
func (f *JiraForm) IsCancelled() bool {
	return f.cancelled
}

// Enabled returns whether JIRA is enabled.
func (f *JiraForm) Enabled() bool {
	return f.enabled
}

// Ensure JiraForm implements tea.Model.
var _ tea.Model = (*JiraForm)(nil)

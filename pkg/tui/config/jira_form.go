// Package config provides TUI components for configuration management.
package config

import (
	"errors"
	"gbm/pkg/tui"
	"gbm/pkg/tui/fields"
	"net/url"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// JiraFormConfig holds configuration for the JIRA form.
type JiraFormConfig struct {
	Theme                      *tui.Theme
	OnSave                     func(data map[string]any) error
	MarkdownFilenamePattern    string
	Host                       string
	Username                   string
	APIToken                   string
	FiltersPriority            string
	FiltersType                string
	AttachmentsDir             string
	FiltersStatus              []string
	AttachmentsMaxSize         int
	MarkdownIncludeAttachments bool
	MarkdownUseRelativeLinks   bool
	MarkdownIncludeComments    bool
	Enabled                    bool
	AttachmentsEnabled         bool
}

// JiraForm renders a form for editing JIRA configuration with enable/disable toggle.
// When disabled, all subsection fields are hidden.
type JiraForm struct {
	attachmentsDirField             tui.Field
	filtersTypeField                tui.Field
	enableField                     tui.Field
	discardField                    tui.Field
	serverHostField                 tui.Field
	serverUserField                 tui.Field
	serverTokenField                tui.Field
	filtersStatusField              tui.Field
	filtersPriorityField            tui.Field
	markdownIncludeAttachmentsField tui.Field
	attachmentsEnabledField         tui.Field
	attachmentsMaxSizeField         tui.Field
	markdownFilenamePatternField    tui.Field
	markdownUseRelativeLinksField   tui.Field
	markdownIncludeCommentsField    tui.Field
	validationOverlay               *fields.ValidationOverlay
	onSave                          func(data map[string]any) error
	theme                           *tui.Theme
	activeSection                   string
	focusedFieldIdx                 int
	width                           int
	height                          int
	sectionIdx                      int
	submitted                       bool
	cancelled                       bool
	enabled                         bool
	showConfirmDiscard              bool
	showValidationErrors            bool
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

	// Filters fields
	filtersStatusFieldPtr := fields.NewTextInput(
		"jira_filters_status",
		"Status Filter",
		"Filter by status (comma-separated, e.g., 'In Dev,Open')",
	)
	filtersStatusField := filtersStatusFieldPtr.
		WithDefault(strings.Join(config.FiltersStatus, ",")).
		WithTheme(config.Theme)

	filtersPriorityFieldPtr := fields.NewTextInput(
		"jira_filters_priority",
		"Priority Filter",
		"Filter by priority (e.g., 'High', 'Medium')",
	)
	filtersPriorityField := filtersPriorityFieldPtr.
		WithDefault(config.FiltersPriority).
		WithTheme(config.Theme)

	filtersTypeFieldPtr := fields.NewTextInput(
		"jira_filters_type",
		"Type Filter",
		"Filter by type (e.g., 'Bug', 'Task')",
	)
	filtersTypeField := filtersTypeFieldPtr.
		WithDefault(config.FiltersType).
		WithTheme(config.Theme)

	// Attachments fields
	attachmentsEnabledFieldPtr := fields.NewConfirm(
		"jira_attachments_enabled",
		"Enable Attachment Downloads?",
	)
	attachmentsEnabledFieldPtr.SetValue(config.AttachmentsEnabled)
	attachmentsEnabledField := attachmentsEnabledFieldPtr.WithTheme(config.Theme)

	attachmentsMaxSizeFieldPtr := fields.NewTextInput(
		"jira_attachments_max_size",
		"Max Size (MB)",
		"Maximum attachment size in MB",
	)
	attachmentsMaxSizeField := attachmentsMaxSizeFieldPtr.
		WithDefault(strconv.Itoa(config.AttachmentsMaxSize)).
		WithTheme(config.Theme)

	attachmentsDirFieldPtr := fields.NewTextInput(
		"jira_attachments_dir",
		"Attachments Directory",
		"Directory to store attachments (relative path)",
	)
	attachmentsDirField := attachmentsDirFieldPtr.
		WithDefault(config.AttachmentsDir).
		WithTheme(config.Theme)

	// Markdown fields
	markdownIncludeCommentsFieldPtr := fields.NewConfirm(
		"jira_markdown_include_comments",
		"Include Comments in Markdown?",
	)
	markdownIncludeCommentsFieldPtr.SetValue(config.MarkdownIncludeComments)
	markdownIncludeCommentsField := markdownIncludeCommentsFieldPtr.WithTheme(config.Theme)

	markdownIncludeAttachmentsFieldPtr := fields.NewConfirm(
		"jira_markdown_include_attachments",
		"Include Attachments in Markdown?",
	)
	markdownIncludeAttachmentsFieldPtr.SetValue(config.MarkdownIncludeAttachments)
	markdownIncludeAttachmentsField := markdownIncludeAttachmentsFieldPtr.WithTheme(config.Theme)

	markdownUseRelativeLinksFieldPtr := fields.NewConfirm(
		"jira_markdown_use_relative_links",
		"Use Relative Links?",
	)
	markdownUseRelativeLinksFieldPtr.SetValue(config.MarkdownUseRelativeLinks)
	markdownUseRelativeLinksField := markdownUseRelativeLinksFieldPtr.WithTheme(config.Theme)

	markdownFilenamePatternFieldPtr := fields.NewTextInput(
		"jira_markdown_filename_pattern",
		"Filename Pattern",
		"Output filename pattern (e.g., '{key}.md')",
	)
	markdownFilenamePatternField := markdownFilenamePatternFieldPtr.
		WithDefault(config.MarkdownFilenamePattern).
		WithTheme(config.Theme)

	form := &JiraForm{
		theme:                           config.Theme,
		onSave:                          config.OnSave,
		enableField:                     enableField,
		enabled:                         config.Enabled,
		serverHostField:                 serverHostField,
		serverUserField:                 serverUserField,
		serverTokenField:                serverTokenField,
		filtersStatusField:              filtersStatusField,
		filtersPriorityField:            filtersPriorityField,
		filtersTypeField:                filtersTypeField,
		attachmentsEnabledField:         attachmentsEnabledField,
		attachmentsMaxSizeField:         attachmentsMaxSizeField,
		attachmentsDirField:             attachmentsDirField,
		markdownIncludeCommentsField:    markdownIncludeCommentsField,
		markdownIncludeAttachmentsField: markdownIncludeAttachmentsField,
		markdownUseRelativeLinksField:   markdownUseRelativeLinksField,
		markdownFilenamePatternField:    markdownFilenamePatternField,
		focusedFieldIdx:                 0,
		activeSection:                   "server",
	}

	return form
}

// Init implements tea.Model.Init.
func (f *JiraForm) Init() tea.Cmd {
	return f.enableField.Focus()
}

// Update implements tea.Model.Update.
func (f *JiraForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle validation error overlay
	if f.showValidationErrors {
		return f.handleValidationOverlay(msg)
	}

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

// handleValidationOverlay processes input while showing the validation error overlay.
func (f *JiraForm) handleValidationOverlay(msg tea.Msg) (tea.Model, tea.Cmd) {
	_, cmd := f.validationOverlay.Update(msg)

	// Check for dismissal message
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.String() == "esc" || keyMsg.String() == "b" || keyMsg.String() == "enter" {
			f.showValidationErrors = false
			return f, f.focusedField().Focus()
		}
	}

	return f, cmd
}

// View implements tea.Model.View.
func (f *JiraForm) View() string {
	if f.showValidationErrors && f.validationOverlay != nil {
		return f.validationOverlay.View()
	}

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

	// Show subsections if enabled
	if f.enabled {
		// Server subsection
		lines = append(lines, f.theme.Blurred.Description.Render("▸ Server Configuration"))
		lines = append(lines, "")
		lines = append(lines, f.serverHostField.View())
		lines = append(lines, "")
		lines = append(lines, f.serverUserField.View())
		lines = append(lines, "")
		lines = append(lines, f.serverTokenField.View())
		lines = append(lines, "")

		// Filters subsection
		lines = append(lines, f.theme.Blurred.Description.Render("▸ Filters"))
		lines = append(lines, "")
		lines = append(lines, f.filtersStatusField.View())
		lines = append(lines, "")
		lines = append(lines, f.filtersPriorityField.View())
		lines = append(lines, "")
		lines = append(lines, f.filtersTypeField.View())
		lines = append(lines, "")

		// Attachments subsection
		lines = append(lines, f.theme.Blurred.Description.Render("▸ Attachments"))
		lines = append(lines, "")
		lines = append(lines, f.attachmentsEnabledField.View())
		lines = append(lines, "")
		lines = append(lines, f.attachmentsMaxSizeField.View())
		lines = append(lines, "")
		lines = append(lines, f.attachmentsDirField.View())
		lines = append(lines, "")

		// Markdown subsection
		lines = append(lines, f.theme.Blurred.Description.Render("▸ Markdown"))
		lines = append(lines, "")
		lines = append(lines, f.markdownIncludeCommentsField.View())
		lines = append(lines, "")
		lines = append(lines, f.markdownIncludeAttachmentsField.View())
		lines = append(lines, "")
		lines = append(lines, f.markdownUseRelativeLinksField.View())
		lines = append(lines, "")
		lines = append(lines, f.markdownFilenamePatternField.View())
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
			// Can navigate through all fields (1-13 when enabled, always include enable field 0)
			maxFields := 14 // enable + 3 server + 3 filters + 3 attachments + 4 markdown
			f.focusedFieldIdx = (f.focusedFieldIdx + 1) % maxFields
		} else {
			// Only enable field
			f.focusedFieldIdx = 0
		}
		return f, f.focusedField().Focus()

	case tea.KeyShiftTab:
		f.focusedField().Blur()
		if f.enabled {
			maxFields := 14
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
			// Validate all fields before saving (only when enabled)
			if f.enabled {
				errs := f.Validate()
				if len(errs) > 0 {
					f.showValidationErrors = true
					f.validationOverlay = fields.NewValidationOverlay(errs).
						WithTheme(f.theme).
						WithWidth(f.width).
						WithHeight(f.height)
					return f, nil
				}
			}

			// Save
			if f.onSave != nil {
				data := f.GetValue()
				err := f.onSave(data)
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
	// Server
	f.serverHostField = f.serverHostField.WithWidth(msg.Width).WithHeight(msg.Height)
	f.serverUserField = f.serverUserField.WithWidth(msg.Width).WithHeight(msg.Height)
	f.serverTokenField = f.serverTokenField.WithWidth(msg.Width).WithHeight(msg.Height)
	// Filters
	f.filtersStatusField = f.filtersStatusField.WithWidth(msg.Width).WithHeight(msg.Height)
	f.filtersPriorityField = f.filtersPriorityField.WithWidth(msg.Width).WithHeight(msg.Height)
	f.filtersTypeField = f.filtersTypeField.WithWidth(msg.Width).WithHeight(msg.Height)
	// Attachments
	f.attachmentsEnabledField = f.attachmentsEnabledField.WithWidth(msg.Width).WithHeight(msg.Height)
	f.attachmentsMaxSizeField = f.attachmentsMaxSizeField.WithWidth(msg.Width).WithHeight(msg.Height)
	f.attachmentsDirField = f.attachmentsDirField.WithWidth(msg.Width).WithHeight(msg.Height)
	// Markdown
	f.markdownIncludeCommentsField = f.markdownIncludeCommentsField.WithWidth(msg.Width).WithHeight(msg.Height)
	f.markdownIncludeAttachmentsField = f.markdownIncludeAttachmentsField.WithWidth(msg.Width).WithHeight(msg.Height)
	f.markdownUseRelativeLinksField = f.markdownUseRelativeLinksField.WithWidth(msg.Width).WithHeight(msg.Height)
	f.markdownFilenamePatternField = f.markdownFilenamePatternField.WithWidth(msg.Width).WithHeight(msg.Height)
	if f.discardField != nil {
		f.discardField = f.discardField.WithWidth(msg.Width).WithHeight(msg.Height)
	}
	if f.validationOverlay != nil {
		f.validationOverlay = f.validationOverlay.WithWidth(msg.Width).WithHeight(msg.Height)
	}
	return f, nil
}

// Validate runs validators on all fields and returns a list of error messages.
// It also sets the error state on fields that fail validation so they are highlighted.
// Only validates required server fields when JIRA is enabled.
func (f *JiraForm) Validate() []string {
	var errs []string

	// Only validate server fields if JIRA is enabled
	if !f.enabled {
		return errs
	}

	// Validate server host field
	if textField, ok := f.serverHostField.(*fields.TextInput); ok {
		err := textField.RunValidator()
		if err != nil {
			errs = append(errs, "JIRA Host: "+err.Error())
		}
	}

	// Validate username field
	if textField, ok := f.serverUserField.(*fields.TextInput); ok {
		err := textField.RunValidator()
		if err != nil {
			errs = append(errs, "Username: "+err.Error())
		}
	}

	// Validate API token field
	if textField, ok := f.serverTokenField.(*fields.TextInput); ok {
		err := textField.RunValidator()
		if err != nil {
			errs = append(errs, "API Token: "+err.Error())
		}
	}

	return errs
}

// focusedField returns the currently focused field.
func (f *JiraForm) focusedField() tui.Field {
	if !f.enabled && f.focusedFieldIdx != 0 {
		// If disabled, only enable field is available
		return f.enableField
	}

	switch f.focusedFieldIdx {
	case 0:
		return f.enableField
	// Server fields (1-3)
	case 1:
		return f.serverHostField
	case 2:
		return f.serverUserField
	case 3:
		return f.serverTokenField
	// Filters fields (4-6)
	case 4:
		return f.filtersStatusField
	case 5:
		return f.filtersPriorityField
	case 6:
		return f.filtersTypeField
	// Attachments fields (7-9)
	case 7:
		return f.attachmentsEnabledField
	case 8:
		return f.attachmentsMaxSizeField
	case 9:
		return f.attachmentsDirField
	// Markdown fields (10-13)
	case 10:
		return f.markdownIncludeCommentsField
	case 11:
		return f.markdownIncludeAttachmentsField
	case 12:
		return f.markdownUseRelativeLinksField
	case 13:
		return f.markdownFilenamePatternField
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
	case 4:
		f.filtersStatusField = field
	case 5:
		f.filtersPriorityField = field
	case 6:
		f.filtersTypeField = field
	case 7:
		f.attachmentsEnabledField = field
	case 8:
		f.attachmentsMaxSizeField = field
	case 9:
		f.attachmentsDirField = field
	case 10:
		f.markdownIncludeCommentsField = field
	case 11:
		f.markdownIncludeAttachmentsField = field
	case 12:
		f.markdownUseRelativeLinksField = field
	case 13:
		f.markdownFilenamePatternField = field
	}
}

// GetValue returns the form data as a map.
func (f *JiraForm) GetValue() map[string]any {
	data := make(map[string]any)
	data["jira_enabled"] = f.enabled

	if f.enabled {
		// Server
		hostVal, _ := f.serverHostField.GetValue().(string)
		userVal, _ := f.serverUserField.GetValue().(string)
		tokenVal, _ := f.serverTokenField.GetValue().(string)
		data["jira_host"] = hostVal
		data["jira_username"] = userVal
		data["jira_api_token"] = tokenVal

		// Filters
		statusVal, _ := f.filtersStatusField.GetValue().(string)
		priorityVal, _ := f.filtersPriorityField.GetValue().(string)
		typeVal, _ := f.filtersTypeField.GetValue().(string)
		data["jira_filters_status"] = statusVal
		data["jira_filters_priority"] = priorityVal
		data["jira_filters_type"] = typeVal

		// Attachments
		attachEnabledVal, _ := f.attachmentsEnabledField.GetValue().(bool)
		attachMaxVal, _ := f.attachmentsMaxSizeField.GetValue().(string)
		attachDirVal, _ := f.attachmentsDirField.GetValue().(string)
		data["jira_attachments_enabled"] = attachEnabledVal
		data["jira_attachments_max_size"] = attachMaxVal
		data["jira_attachments_dir"] = attachDirVal

		// Markdown
		mdCommentsVal, _ := f.markdownIncludeCommentsField.GetValue().(bool)
		mdAttachVal, _ := f.markdownIncludeAttachmentsField.GetValue().(bool)
		mdRelativeLinkVal, _ := f.markdownUseRelativeLinksField.GetValue().(bool)
		mdPatternVal, _ := f.markdownFilenamePatternField.GetValue().(string)
		data["jira_markdown_include_comments"] = mdCommentsVal
		data["jira_markdown_include_attachments"] = mdAttachVal
		data["jira_markdown_use_relative_links"] = mdRelativeLinkVal
		data["jira_markdown_filename_pattern"] = mdPatternVal
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

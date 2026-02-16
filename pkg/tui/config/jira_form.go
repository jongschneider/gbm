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
	helpOverlay                     *tui.HelpOverlay
	onSave                          func(data map[string]any) error
	theme                           *tui.Theme
	activeSection                   string
	focusedFieldIdx                 int
	width                           int
	height                          int
	submitted                       bool
	cancelled                       bool
	enabled                         bool
	showValidationErrors            bool
	showHelp                        bool
	insertMode                      bool // vim-style insert mode for text inputs
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
	// Handle help overlay
	if f.showHelp {
		return f.handleHelpOverlay(msg)
	}

	// Handle validation error overlay
	if f.showValidationErrors {
		return f.handleValidationOverlay(msg)
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

// handleHelpOverlay processes input while showing the help overlay.
func (f *JiraForm) handleHelpOverlay(msg tea.Msg) (tea.Model, tea.Cmd) {
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
	if f.showHelp && f.helpOverlay != nil {
		return f.helpOverlay.View()
	}

	if f.showValidationErrors && f.validationOverlay != nil {
		return f.validationOverlay.View()
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
	lines = append(lines, f.theme.Blurred.Description.Render("Tab=next, Shift+Tab=prev, s=save, ?=help"))

	return strings.Join(lines, "\n")
}

// handleKeyMsg processes keyboard input.
func (f *JiraForm) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Insert mode: pass all keys to the field except Esc
	if f.insertMode {
		return f.handleInsertMode(msg)
	}

	// Normal mode (vim-style navigation)
	if model, cmd, handled := f.handleNormalModeKeys(msg); handled {
		return model, cmd
	}

	return f.handleNavigationKeys(msg)
}

// handleInsertMode processes keys when in insert mode.
func (f *JiraForm) handleInsertMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.Type == tea.KeyEsc {
		f.insertMode = false
		return f, nil
	}
	// Pass key to the focused field
	field, cmd := f.focusedField().Update(msg)
	f.updateFocusedField(field)
	return f, cmd
}

// handleNormalModeKeys handles vim-style keys and special commands in normal mode.
// Returns (model, cmd, true) if key was handled, or (nil, nil, false) to continue processing.
func (f *JiraForm) handleNormalModeKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	_, isTextInput := f.focusedField().(*fields.TextInput)

	// When a Confirm field is focused, delegate toggle/navigation keys to it.
	// Confirm fields handle: space, enter, h, l, left, right, y, n, Y, N, tab.
	if _, ok := f.focusedField().(*fields.Confirm); ok {
		if m, cmd, handled := f.handleConfirmFieldKey(msg); handled {
			return m, cmd, true
		}
	}

	// Handle vim-style keys and commands
	if msg.Type == tea.KeyRunes && len(msg.Runes) == 1 {
		switch msg.Runes[0] {
		case 'j':
			m, cmd := f.nextField()
			return m, cmd, true
		case 'k':
			m, cmd := f.prevField()
			return m, cmd, true
		case 'i':
			if isTextInput {
				f.insertMode = true
				return f, nil, true
			}
		case '?':
			f.showHelp = true
			f.helpOverlay = tui.NewHelpOverlay().
				WithTheme(f.theme).
				WithWidth(f.width).
				WithHeight(f.height)
			return f, nil, true
		case 's':
			return f.handleSave()
		}
	}

	return nil, nil, false
}

// isConfirmKey reports whether the key message is one that a Confirm field handles
// in a form context. Tab is excluded because it is used for field navigation.
func isConfirmKey(msg tea.KeyMsg) bool {
	switch msg.String() {
	case " ", "enter", "h", "l", "left", "right", "y", "Y", "n", "N":
		return true
	}
	return false
}

// handleConfirmFieldKey delegates a key to the focused Confirm field and processes
// any resulting completion. Returns (model, cmd, true) if the key was handled.
func (f *JiraForm) handleConfirmFieldKey(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if !isConfirmKey(msg) {
		return nil, nil, false
	}

	field, cmd := f.focusedField().Update(msg)
	f.updateFocusedField(field)

	confirm, ok := field.(*fields.Confirm)
	if !ok {
		return f, cmd, true
	}

	// If the Confirm field completed (enter, y, n), process it.
	if !confirm.IsComplete() {
		return f, cmd, true
	}

	if f.focusedFieldIdx == 0 {
		// Enable field: update the enabled state.
		newEnabled := confirm.GetValue().(bool)
		if newEnabled != f.enabled {
			f.enabled = newEnabled
		}
		confirm.ResetCompletion()
		if newEnabled {
			m, cmd := f.nextField()
			return m, cmd, true
		}
		// Disabled: stay on enable field, ignore the cmd.
		return f, nil, true
	}

	// Non-enable Confirm field: reset and advance to next field.
	confirm.ResetCompletion()
	m, nextCmd := f.nextField()
	return m, nextCmd, true
}

// handleNavigationKeys handles Tab and Shift+Tab keys.
func (f *JiraForm) handleNavigationKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type { //nolint:exhaustive // Only handling relevant keys
	case tea.KeyTab:
		return f.nextField()
	case tea.KeyShiftTab:
		return f.prevField()
	}
	return f, nil
}

// handleSave validates and saves the form.
func (f *JiraForm) handleSave() (tea.Model, tea.Cmd, bool) {
	// Validate all fields before saving (only when enabled)
	if f.enabled {
		errs := f.Validate()
		if len(errs) > 0 {
			f.showValidationErrors = true
			f.validationOverlay = fields.NewValidationOverlay(errs).
				WithTheme(f.theme).
				WithWidth(f.width).
				WithHeight(f.height)
			return f, nil, true
		}
	}

	// Save
	if f.onSave != nil {
		data := f.GetValue()
		err := f.onSave(data)
		if err != nil {
			f.showValidationErrors = true
			f.validationOverlay = fields.NewValidationOverlay([]string{err.Error()}).
				WithTheme(f.theme).
				WithTitle("Save Error").
				WithWidth(f.width).
				WithHeight(f.height)
			return f, nil, true
		}
	}
	f.submitted = true
	return f, func() tea.Msg {
		return tui.FormFlushCompleteMsg{}
	}, true
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

// Focus gives the form keyboard focus and focuses the first field.
func (f *JiraForm) Focus() tea.Cmd {
	return f.focusedField().Focus()
}

// Blur removes keyboard focus from the form and all its fields.
func (f *JiraForm) Blur() tea.Cmd {
	f.insertMode = false
	f.focusedField().Blur()
	return nil
}

// Ensure JiraForm implements tea.Model.
var _ tea.Model = (*JiraForm)(nil)

// InInsertMode reports whether the form is in insert mode.
func (f *JiraForm) InInsertMode() bool { return f.insertMode }

// Ensure JiraForm implements tui.FocusReporter.
var _ tui.FocusReporter = (*JiraForm)(nil)

// Ensure JiraForm implements tui.InsertModeReporter.
var _ tui.InsertModeReporter = (*JiraForm)(nil)

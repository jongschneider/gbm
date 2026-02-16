package config

import (
	"gbm/pkg/tui"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// allFields returns all form fields in order for index-based access.
func (f *JiraForm) allFields() []tui.Field {
	return []tui.Field{
		f.enableField,                     // 0
		f.serverHostField,                 // 1
		f.serverUserField,                 // 2
		f.serverTokenField,                // 3
		f.filtersStatusField,              // 4
		f.filtersPriorityField,            // 5
		f.filtersTypeField,                // 6
		f.attachmentsEnabledField,         // 7
		f.attachmentsMaxSizeField,         // 8
		f.attachmentsDirField,             // 9
		f.markdownIncludeCommentsField,    // 10
		f.markdownIncludeAttachmentsField, // 11
		f.markdownUseRelativeLinksField,   // 12
		f.markdownFilenamePatternField,    // 13
	}
}

// focusedField returns the currently focused field.
func (f *JiraForm) focusedField() tui.Field {
	if !f.enabled && f.focusedFieldIdx != 0 {
		// If disabled, only enable field is available
		return f.enableField
	}

	allFields := f.allFields()
	if f.focusedFieldIdx >= 0 && f.focusedFieldIdx < len(allFields) {
		return allFields[f.focusedFieldIdx]
	}
	return f.enableField
}

// nextField moves focus to the next field.
func (f *JiraForm) nextField() (tea.Model, tea.Cmd) {
	f.focusedField().Blur()
	if f.enabled {
		maxFields := 14
		f.focusedFieldIdx = (f.focusedFieldIdx + 1) % maxFields
	} else {
		f.focusedFieldIdx = 0
	}
	return f, f.focusedField().Focus()
}

// prevField moves focus to the previous field.
func (f *JiraForm) prevField() (tea.Model, tea.Cmd) {
	f.focusedField().Blur()
	if f.enabled {
		maxFields := 14
		f.focusedFieldIdx = (f.focusedFieldIdx - 1 + maxFields) % maxFields
	} else {
		f.focusedFieldIdx = 0
	}
	return f, f.focusedField().Focus()
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
		// Type assertion failures return zero value which is acceptable
		hostVal, _ := f.serverHostField.GetValue().(string)
		userVal, _ := f.serverUserField.GetValue().(string)
		tokenVal, _ := f.serverTokenField.GetValue().(string)
		data["jira_host"] = hostVal
		data["jira_username"] = userVal
		data["jira_api_token"] = tokenVal

		// Type assertion failures return zero value which is acceptable
		statusVal, _ := f.filtersStatusField.GetValue().(string)
		priorityVal, _ := f.filtersPriorityField.GetValue().(string)
		typeVal, _ := f.filtersTypeField.GetValue().(string)
		data["jira_filters_status"] = statusVal
		data["jira_filters_priority"] = priorityVal
		data["jira_filters_type"] = typeVal

		// Type assertion failures return zero value which is acceptable
		attachEnabledVal, _ := f.attachmentsEnabledField.GetValue().(bool)
		attachMaxVal, _ := f.attachmentsMaxSizeField.GetValue().(string)
		attachDirVal, _ := f.attachmentsDirField.GetValue().(string)
		data["jira_attachments_enabled"] = attachEnabledVal
		data["jira_attachments_max_size"] = attachMaxVal
		data["jira_attachments_dir"] = attachDirVal

		// Type assertion failures return zero value which is acceptable
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

// FocusedYOffset returns the line number where the focused field starts.
// This implements the tui.FocusReporter interface for auto-scrolling support.
func (f *JiraForm) FocusedYOffset() int {
	lineCount := 0

	// Count lines helper
	countLines := func(s string) int {
		return strings.Count(s, "\n") + 1
	}

	// Title + empty line
	lineCount += 2 // "JIRA Configuration" + ""

	// Enable field (index 0)
	if f.focusedFieldIdx == 0 {
		return lineCount
	}
	lineCount += countLines(f.enableField.View()) + 1 // field + empty line

	// If disabled, no more fields to count
	if !f.enabled {
		return lineCount
	}

	// Define field groups with section headers
	type fieldGroup struct {
		header string
		fields []tui.Field
	}

	groups := []fieldGroup{
		{"Server Configuration", []tui.Field{f.serverHostField, f.serverUserField, f.serverTokenField}},
		{"Filters", []tui.Field{f.filtersStatusField, f.filtersPriorityField, f.filtersTypeField}},
		{"Attachments", []tui.Field{f.attachmentsEnabledField, f.attachmentsMaxSizeField, f.attachmentsDirField}},
		{"Markdown", []tui.Field{f.markdownIncludeCommentsField, f.markdownIncludeAttachmentsField, f.markdownUseRelativeLinksField, f.markdownFilenamePatternField}},
	}

	fieldIdx := 1 // Start after enable field
	for _, group := range groups {
		// Section header + empty line
		lineCount += 2

		for _, field := range group.fields {
			if f.focusedFieldIdx == fieldIdx {
				return lineCount
			}
			lineCount += countLines(field.View()) + 1 // field + empty line
			fieldIdx++
		}
	}

	return lineCount
}

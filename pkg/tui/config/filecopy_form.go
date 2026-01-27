// Package config provides TUI components for configuration management.
package config

import (
	"gbm/pkg/tui"
	"gbm/pkg/tui/fields"
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
)

// FileCopyRule represents a file copy rule with source worktree and files.
type FileCopyRule struct {
	SourceWorktree string
	Files          []string
}

// FileCopyFormConfig holds configuration for the FileCopy form.
type FileCopyFormConfig struct {
	OnSave func(rules []FileCopyRule) error
	Theme  *tui.Theme
	Rules  []FileCopyRule
}

// ModalState represents the current modal being displayed.
type ModalState int

const (
	ModalNone ModalState = iota
	ModalAdd
	ModalEdit
	ModalDelete
	ModalDiscard
	ModalFilePicker
	ModalHelp
)

// FileCopyForm renders a table of file copy rules with add/edit/delete modals.
type FileCopyForm struct {
	sourceField     tui.Field
	confirmField    tui.Field
	filesField      tui.Field
	onSave          func(rules []FileCopyRule) error
	table           *tui.Table
	theme           *tui.Theme
	filepickerField *fields.FilePicker
	helpOverlay     *tui.HelpOverlay
	rules           []FileCopyRule
	modalState      ModalState
	height          int
	width           int
	editingIdx      int
	prevModalState  ModalState
	modalFocusIdx   int
	submitted       bool
	cancelled       bool
}

// NewFileCopyForm creates a new FileCopy configuration form.
func NewFileCopyForm(config FileCopyFormConfig) *FileCopyForm {
	if config.Theme == nil {
		config.Theme = tui.DefaultTheme()
	}

	ctx := &tui.Context{Theme: config.Theme}
	t := tui.NewTable(ctx).
		WithColumns([]tui.Column{
			{Title: "Source Worktree", Width: 20},
			{Title: "Files", Width: 50},
		}).
		WithHeight(8).
		WithFocused(true).
		WithCycling(true)

	rules := make([]FileCopyRule, len(config.Rules))
	copy(rules, config.Rules)

	rows := buildTableRows(rules)
	t = t.WithRows(rows).Build()

	return &FileCopyForm{
		theme:      config.Theme,
		onSave:     config.OnSave,
		table:      t,
		rules:      rules,
		modalState: ModalNone,
		editingIdx: -1,
	}
}

// buildTableRows converts rules to table rows.
func buildTableRows(rules []FileCopyRule) []table.Row {
	rows := make([]table.Row, len(rules))
	for i, rule := range rules {
		filesPreview := formatFilesPreview(rule.Files)
		rows[i] = table.Row{rule.SourceWorktree, filesPreview}
	}
	return rows
}

// formatFilesPreview creates a short preview of the files list.
func formatFilesPreview(files []string) string {
	if len(files) == 0 {
		return "(no files)"
	}
	if len(files) == 1 {
		return files[0]
	}
	if len(files) == 2 {
		return files[0] + ", " + files[1]
	}
	return files[0] + ", " + files[1] + ", ..."
}

// Init implements tea.Model.
func (f *FileCopyForm) Init() tea.Cmd {
	return f.table.Init()
}

// Update implements tea.Model.
func (f *FileCopyForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch f.modalState {
	case ModalHelp:
		return f.handleHelpModal(msg)
	case ModalAdd, ModalEdit:
		return f.handleEditModal(msg)
	case ModalDelete:
		return f.handleDeleteModal(msg)
	case ModalDiscard:
		return f.handleDiscardModal(msg)
	case ModalFilePicker:
		return f.handleFilePickerModal(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return f.handleKeyMsg(msg)
	case tea.WindowSizeMsg:
		return f.handleWindowSize(msg)
	}

	newTable, cmd := f.table.Update(msg)
	if t, ok := newTable.(*tui.Table); ok {
		f.table = t
	}
	return f, cmd
}

// handleKeyMsg processes keyboard input when no modal is open.
func (f *FileCopyForm) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "?":
		return f.openHelpModal()
	case "a":
		return f.openAddModal()
	case "e":
		return f.openEditModal()
	case "d":
		return f.openDeleteModal()
	case "s":
		return f.save()
	case "q":
		return f.openDiscardModal()
	case "esc":
		f.cancelled = true
		return f, func() tea.Msg {
			return tui.BackBoundaryMsg{}
		}
	}

	newTable, cmd := f.table.Update(msg)
	if t, ok := newTable.(*tui.Table); ok {
		f.table = t
	}
	return f, cmd
}

// openAddModal opens the add rule modal.
func (f *FileCopyForm) openAddModal() (tea.Model, tea.Cmd) {
	f.modalState = ModalAdd
	f.editingIdx = -1
	f.initModalFields("", "")
	return f, f.sourceField.Focus()
}

// openEditModal opens the edit rule modal for the selected rule.
func (f *FileCopyForm) openEditModal() (tea.Model, tea.Cmd) {
	cursor := f.table.Cursor()
	if cursor < 0 || cursor >= len(f.rules) {
		return f, nil
	}

	f.modalState = ModalEdit
	f.editingIdx = cursor
	rule := f.rules[cursor]
	f.initModalFields(rule.SourceWorktree, strings.Join(rule.Files, ", "))
	return f, f.sourceField.Focus()
}

// openDeleteModal opens the delete confirmation modal.
func (f *FileCopyForm) openDeleteModal() (tea.Model, tea.Cmd) {
	cursor := f.table.Cursor()
	if cursor < 0 || cursor >= len(f.rules) {
		return f, nil
	}

	f.modalState = ModalDelete
	f.editingIdx = cursor
	rule := f.rules[cursor]
	confirm := fields.NewConfirm("delete_confirm", "Delete rule for '"+rule.SourceWorktree+"'?")
	f.confirmField = confirm.WithTheme(f.theme)
	return f, f.confirmField.Focus()
}

// openDiscardModal opens the discard confirmation modal.
func (f *FileCopyForm) openDiscardModal() (tea.Model, tea.Cmd) {
	f.modalState = ModalDiscard
	confirm := fields.NewConfirm("discard_confirm", "Discard unsaved changes?")
	f.confirmField = confirm.WithTheme(f.theme)
	return f, f.confirmField.Focus()
}

// openHelpModal opens the help overlay.
func (f *FileCopyForm) openHelpModal() (tea.Model, tea.Cmd) {
	f.modalState = ModalHelp
	f.helpOverlay = tui.NewHelpOverlay().
		WithTheme(f.theme).
		WithWidth(f.width).
		WithHeight(f.height)
	return f, nil
}

// handleHelpModal processes input while showing the help overlay.
func (f *FileCopyForm) handleHelpModal(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return f, nil
	}

	switch keyMsg.String() {
	case "esc", "?", "enter":
		f.modalState = ModalNone
		f.helpOverlay = nil
		return f, nil
	}
	return f, nil
}

// initModalFields initializes the source and files fields for add/edit modals.
func (f *FileCopyForm) initModalFields(source, filesStr string) {
	sourceFieldPtr := fields.NewTextInput(
		"source_worktree",
		"Source Worktree",
		"Name of worktree to copy files from (e.g., main)",
	)
	f.sourceField = sourceFieldPtr.
		WithDefault(source).
		WithTheme(f.theme)

	filesFieldPtr := fields.NewTextInput(
		"files",
		"Files",
		"Files to copy (comma-separated, or press 'b' to browse)",
	)
	f.filesField = filesFieldPtr.
		WithDefault(filesStr).
		WithTheme(f.theme)

	f.modalFocusIdx = 0
}

// handleEditModal processes input in the add/edit modal.
func (f *FileCopyForm) handleEditModal(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		newField, cmd := f.focusedModalField().Update(msg)
		f.updateModalField(newField)
		return f, cmd
	}

	switch keyMsg.Type {
	case tea.KeyTab:
		f.focusedModalField().Blur()
		f.modalFocusIdx = (f.modalFocusIdx + 1) % 2
		return f, f.focusedModalField().Focus()

	case tea.KeyShiftTab:
		f.focusedModalField().Blur()
		f.modalFocusIdx = (f.modalFocusIdx - 1 + 2) % 2
		return f, f.focusedModalField().Focus()

	case tea.KeyEnter:
		return f.confirmEditModal()

	case tea.KeyEsc:
		f.modalState = ModalNone
		f.editingIdx = -1
		return f, nil
	}

	if keyMsg.String() == "b" && f.modalFocusIdx == 1 {
		return f.openFilePicker()
	}

	newField, cmd := f.focusedModalField().Update(keyMsg)
	f.updateModalField(newField)
	return f, cmd
}

// focusedModalField returns the currently focused field in the modal.
func (f *FileCopyForm) focusedModalField() tui.Field {
	if f.modalFocusIdx == 0 {
		return f.sourceField
	}
	return f.filesField
}

// updateModalField updates the focused modal field.
func (f *FileCopyForm) updateModalField(field tui.Field) {
	if f.modalFocusIdx == 0 {
		f.sourceField = field
	} else {
		f.filesField = field
	}
}

// confirmEditModal saves the add/edit modal data.
func (f *FileCopyForm) confirmEditModal() (tea.Model, tea.Cmd) {
	sourceVal, _ := f.sourceField.GetValue().(string)
	filesVal, _ := f.filesField.GetValue().(string)

	if sourceVal == "" {
		return f, nil
	}

	files := parseFilesList(filesVal)

	rule := FileCopyRule{
		SourceWorktree: sourceVal,
		Files:          files,
	}

	if f.modalState == ModalAdd {
		f.rules = append(f.rules, rule)
	} else if f.modalState == ModalEdit && f.editingIdx >= 0 && f.editingIdx < len(f.rules) {
		f.rules[f.editingIdx] = rule
	}

	f.refreshTable()
	f.modalState = ModalNone
	f.editingIdx = -1
	return f, nil
}

// parseFilesList splits comma-separated files into a slice.
func parseFilesList(filesStr string) []string {
	if filesStr == "" {
		return nil
	}

	parts := strings.Split(filesStr, ",")
	files := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			files = append(files, trimmed)
		}
	}
	return files
}

// handleDeleteModal processes input in the delete modal.
func (f *FileCopyForm) handleDeleteModal(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return f, nil
	}

	switch keyMsg.String() {
	case "y", "Y":
		return f.confirmDelete()
	case "n", "N", "esc":
		f.modalState = ModalNone
		f.editingIdx = -1
		return f, nil
	}

	newField, cmd := f.confirmField.Update(keyMsg)
	f.confirmField = newField
	return f, cmd
}

// confirmDelete removes the selected rule.
func (f *FileCopyForm) confirmDelete() (tea.Model, tea.Cmd) {
	if f.editingIdx >= 0 && f.editingIdx < len(f.rules) {
		f.rules = append(f.rules[:f.editingIdx], f.rules[f.editingIdx+1:]...)
		f.refreshTable()
	}
	f.modalState = ModalNone
	f.editingIdx = -1
	return f, nil
}

// handleDiscardModal processes input in the discard modal.
func (f *FileCopyForm) handleDiscardModal(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return f, nil
	}

	switch keyMsg.String() {
	case "y", "Y":
		f.cancelled = true
		f.modalState = ModalNone
		return f, func() tea.Msg {
			return tui.BackBoundaryMsg{}
		}
	case "n", "N", "esc":
		f.modalState = ModalNone
		return f, nil
	}

	newField, cmd := f.confirmField.Update(keyMsg)
	f.confirmField = newField
	return f, cmd
}

// openFilePicker opens the file picker modal.
func (f *FileCopyForm) openFilePicker() (tea.Model, tea.Cmd) {
	f.prevModalState = f.modalState
	f.modalState = ModalFilePicker

	f.filepickerField = fields.NewFilePicker(
		"files",
		"Select Files",
		"Navigate and select files to copy",
	)
	f.filepickerField = f.filepickerField.
		WithDirAllowed(true).
		WithMultiSelect(true).
		WithTheme(f.theme).(*fields.FilePicker)

	if f.height > 0 {
		f.filepickerField = f.filepickerField.WithHeight(f.height - 4).(*fields.FilePicker)
	}

	return f, f.filepickerField.Focus()
}

// handleFilePickerModal processes input in the file picker modal.
func (f *FileCopyForm) handleFilePickerModal(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.Type {
		case tea.KeyEsc:
			f.modalState = f.prevModalState
			return f, f.filesField.Focus()

		case tea.KeyEnter:
			selectedFiles := f.filepickerField.GetSelectedFiles()
			if len(selectedFiles) > 0 {
				currentFilesVal, _ := f.filesField.GetValue().(string)
				var existingFiles []string
				if currentFilesVal != "" {
					existingFiles = parseFilesList(currentFilesVal)
				}

				for _, file := range selectedFiles {
					if !containsPath(existingFiles, file) {
						existingFiles = append(existingFiles, file)
					}
				}

				newFilesStr := strings.Join(existingFiles, ", ")

				filesFieldPtr := fields.NewTextInput(
					"files",
					"Files",
					"Files to copy (comma-separated, or press 'b' to browse)",
				)
				f.filesField = filesFieldPtr.
					WithDefault(newFilesStr).
					WithTheme(f.theme)
			}

			f.modalState = f.prevModalState
			return f, f.filesField.Focus()
		}
	}

	newField, cmd := f.filepickerField.Update(msg)
	if fp, ok := newField.(*fields.FilePicker); ok {
		f.filepickerField = fp
	}

	if f.filepickerField.IsCancelled() {
		f.modalState = f.prevModalState
		return f, f.filesField.Focus()
	}

	return f, cmd
}

// containsPath checks if a path exists in a slice.
func containsPath(paths []string, path string) bool {
	return slices.Contains(paths, path)
}

// save persists the rules via the OnSave callback.
func (f *FileCopyForm) save() (tea.Model, tea.Cmd) {
	if f.onSave != nil {
		err := f.onSave(f.rules)
		if err != nil {
			return f, nil
		}
	}
	f.submitted = true
	return f, func() tea.Msg {
		return tui.BackBoundaryMsg{}
	}
}

// refreshTable updates the table with current rules.
func (f *FileCopyForm) refreshTable() {
	rows := buildTableRows(f.rules)
	f.table.SetRows(rows)
}

// handleWindowSize updates dimensions on terminal resize.
func (f *FileCopyForm) handleWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	f.width = msg.Width
	f.height = msg.Height
	f.table.SetHeight(min(8, msg.Height-10))
	return f, nil
}

// View implements tea.Model.
func (f *FileCopyForm) View() string {
	switch f.modalState {
	case ModalHelp:
		return f.helpOverlay.View()
	case ModalAdd:
		return f.renderEditModal("Add File Copy Rule")
	case ModalEdit:
		return f.renderEditModal("Edit File Copy Rule")
	case ModalDelete:
		return f.confirmField.View()
	case ModalDiscard:
		return f.confirmField.View()
	case ModalFilePicker:
		return f.filepickerField.View()
	}

	lines := []string{
		f.theme.Focused.Title.Render("File Copy Rules"),
		"",
		f.table.View(),
		"",
	}

	if len(f.rules) == 0 {
		lines = append(lines, f.theme.Blurred.Description.Render("No rules configured. Press 'a' to add a rule."))
		lines = append(lines, "")
	}

	lines = append(lines, f.theme.Blurred.Description.Render("a=add  e=edit  d=delete  s=save  ?=help  q=quit"))

	return strings.Join(lines, "\n")
}

// renderEditModal renders the add/edit modal.
func (f *FileCopyForm) renderEditModal(title string) string {
	lines := []string{
		f.theme.Focused.Title.Render(title),
		"",
		f.sourceField.View(),
		"",
		f.filesField.View(),
		"",
		f.theme.Blurred.Description.Render("Tab=next  b=browse files  Enter=confirm  Esc=cancel"),
	}
	return strings.Join(lines, "\n")
}

// GetRules returns the current rules.
func (f *FileCopyForm) GetRules() []FileCopyRule {
	return f.rules
}

// IsComplete returns whether the form has been submitted.
func (f *FileCopyForm) IsComplete() bool {
	return f.submitted
}

// IsCancelled returns whether the form was cancelled.
func (f *FileCopyForm) IsCancelled() bool {
	return f.cancelled
}

// ModalState returns the current modal state.
func (f *FileCopyForm) GetModalState() ModalState {
	return f.modalState
}

// Ensure FileCopyForm implements tea.Model.
var _ tea.Model = (*FileCopyForm)(nil)

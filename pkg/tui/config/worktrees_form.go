package config

import (
	"gbm/pkg/tui"
	"gbm/pkg/tui/fields"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
)

type WorktreeEntry struct {
	Name        string
	Branch      string
	MergeInto   string
	Description string
}

type WorktreesFormConfig struct {
	OnSave    func(worktrees []WorktreeEntry) error
	Theme     *tui.Theme
	Worktrees []WorktreeEntry
}

type WorktreeModalState int

const (
	WorktreeModalNone WorktreeModalState = iota
	WorktreeModalAdd
	WorktreeModalEdit
	WorktreeModalDelete
	WorktreeModalDiscard
	WorktreeModalHelp
)

type WorktreesForm struct {
	nameField        tui.Field
	confirmField     tui.Field
	descriptionField tui.Field
	mergeIntoField   tui.Field
	branchField      tui.Field
	onSave           func(worktrees []WorktreeEntry) error
	table            *tui.Table
	theme            *tui.Theme
	helpOverlay      *tui.HelpOverlay
	validationError  string
	worktrees        []WorktreeEntry
	modalState       WorktreeModalState
	height           int
	width            int
	modalFocusIdx    int
	editingIdx       int
	cancelled        bool
	submitted        bool
	insertMode       bool // vim-style insert mode for text inputs
}

var validWorktreeNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

func NewWorktreesForm(config WorktreesFormConfig) *WorktreesForm {
	if config.Theme == nil {
		config.Theme = tui.DefaultTheme()
	}

	ctx := &tui.Context{Theme: config.Theme}
	t := tui.NewTable(ctx).
		WithColumns([]tui.Column{
			{Title: "Name", Width: 20},
			{Title: "Branch", Width: 25},
			{Title: "Merge Into", Width: 20},
		}).
		WithHeight(8).
		WithFocused(true).
		WithCycling(true)

	worktrees := make([]WorktreeEntry, len(config.Worktrees))
	copy(worktrees, config.Worktrees)

	rows := buildWorktreeRows(worktrees)
	t = t.WithRows(rows).Build()

	return &WorktreesForm{
		theme:      config.Theme,
		onSave:     config.OnSave,
		table:      t,
		worktrees:  worktrees,
		modalState: WorktreeModalNone,
		editingIdx: -1,
	}
}

func buildWorktreeRows(worktrees []WorktreeEntry) []table.Row {
	rows := make([]table.Row, len(worktrees))
	for i, wt := range worktrees {
		mergeInto := wt.MergeInto
		if mergeInto == "" {
			mergeInto = "-"
		}
		rows[i] = table.Row{wt.Name, wt.Branch, mergeInto}
	}
	return rows
}

func (f *WorktreesForm) Init() tea.Cmd {
	return f.table.Init()
}

func (f *WorktreesForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch f.modalState {
	case WorktreeModalNone:
		// Fall through to normal handling below
	case WorktreeModalHelp:
		return f.handleHelpModal(msg)
	case WorktreeModalAdd, WorktreeModalEdit:
		return f.handleEditModal(msg)
	case WorktreeModalDelete:
		return f.handleDeleteModal(msg)
	case WorktreeModalDiscard:
		return f.handleDiscardModal(msg)
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

func (f *WorktreesForm) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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

func (f *WorktreesForm) openAddModal() (tea.Model, tea.Cmd) {
	f.modalState = WorktreeModalAdd
	f.editingIdx = -1
	f.validationError = ""
	f.initModalFields("", "", "", "")
	return f, f.nameField.Focus()
}

func (f *WorktreesForm) openEditModal() (tea.Model, tea.Cmd) {
	cursor := f.table.Cursor()
	if cursor < 0 || cursor >= len(f.worktrees) {
		return f, nil
	}

	f.modalState = WorktreeModalEdit
	f.editingIdx = cursor
	f.validationError = ""
	wt := f.worktrees[cursor]
	f.initModalFields(wt.Name, wt.Branch, wt.MergeInto, wt.Description)
	return f, f.nameField.Focus()
}

func (f *WorktreesForm) openDeleteModal() (tea.Model, tea.Cmd) {
	cursor := f.table.Cursor()
	if cursor < 0 || cursor >= len(f.worktrees) {
		return f, nil
	}

	f.modalState = WorktreeModalDelete
	f.editingIdx = cursor
	wt := f.worktrees[cursor]
	confirm := fields.NewConfirm("delete_confirm", "Delete worktree '"+wt.Name+"'?")
	f.confirmField = confirm.WithTheme(f.theme)
	return f, f.confirmField.Focus()
}

func (f *WorktreesForm) openDiscardModal() (tea.Model, tea.Cmd) {
	f.modalState = WorktreeModalDiscard
	confirm := fields.NewConfirm("discard_confirm", "Discard unsaved changes?")
	f.confirmField = confirm.WithTheme(f.theme)
	return f, f.confirmField.Focus()
}

func (f *WorktreesForm) openHelpModal() (tea.Model, tea.Cmd) {
	f.modalState = WorktreeModalHelp
	f.helpOverlay = tui.NewHelpOverlay().
		WithTheme(f.theme).
		WithWidth(f.width).
		WithHeight(f.height)
	return f, nil
}

func (f *WorktreesForm) handleHelpModal(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return f, nil
	}

	switch keyMsg.String() {
	case "esc", "?", "enter":
		f.modalState = WorktreeModalNone
		f.helpOverlay = nil
		return f, nil
	}
	return f, nil
}

func (f *WorktreesForm) initModalFields(name, branch, mergeInto, description string) {
	nameFieldPtr := fields.NewTextInput("name", "Name", "Worktree name (alphanumeric, -, _)")
	f.nameField = nameFieldPtr.WithDefault(name).WithTheme(f.theme)

	branchFieldPtr := fields.NewTextInput("branch", "Branch", "Git branch name")
	f.branchField = branchFieldPtr.WithDefault(branch).WithTheme(f.theme)

	mergeFieldPtr := fields.NewTextInput("merge_into", "Merge Into", "Branch to merge into (optional)")
	f.mergeIntoField = mergeFieldPtr.WithDefault(mergeInto).WithTheme(f.theme)

	descFieldPtr := fields.NewTextInput("description", "Description", "Brief description (optional)")
	f.descriptionField = descFieldPtr.WithDefault(description).WithTheme(f.theme)

	f.modalFocusIdx = 0
}

func (f *WorktreesForm) handleEditModal(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		newField, cmd := f.focusedModalField().Update(msg)
		f.updateModalField(newField)
		return f, cmd
	}

	// Insert mode: pass all keys to the field except Esc
	if f.insertMode {
		if keyMsg.Type == tea.KeyEsc {
			f.insertMode = false
			return f, nil
		}
		newField, cmd := f.focusedModalField().Update(keyMsg)
		f.updateModalField(newField)
		return f, cmd
	}

	// Normal mode (vim-style navigation)

	// Handle vim-style keys in normal mode
	if keyMsg.Type == tea.KeyRunes && len(keyMsg.Runes) == 1 {
		switch keyMsg.Runes[0] {
		case 'j':
			return f.cycleModalFocus(false)
		case 'k':
			return f.cycleModalFocus(true)
		case 'i':
			f.insertMode = true
			return f, nil
		}
	}

	switch keyMsg.Type { //nolint:exhaustive // Only handling relevant keys
	case tea.KeyTab:
		return f.cycleModalFocus(false)
	case tea.KeyShiftTab:
		return f.cycleModalFocus(true)
	case tea.KeyEnter:
		return f.confirmEdit()
	case tea.KeyEsc:
		f.modalState = WorktreeModalNone
		f.editingIdx = -1
		f.validationError = ""
		f.insertMode = false
		return f, nil
	}

	return f, nil
}

func (f *WorktreesForm) focusedModalField() tui.Field {
	switch f.modalFocusIdx {
	case 0:
		return f.nameField
	case 1:
		return f.branchField
	case 2:
		return f.mergeIntoField
	case 3:
		return f.descriptionField
	default:
		return f.nameField
	}
}

func (f *WorktreesForm) updateModalField(field tui.Field) {
	switch f.modalFocusIdx {
	case 0:
		f.nameField = field
	case 1:
		f.branchField = field
	case 2:
		f.mergeIntoField = field
	case 3:
		f.descriptionField = field
	}
}

func (f *WorktreesForm) cycleModalFocus(backward bool) (tea.Model, tea.Cmd) {
	f.focusedModalField().Blur()

	if backward {
		f.modalFocusIdx--
		if f.modalFocusIdx < 0 {
			f.modalFocusIdx = 3
		}
	} else {
		f.modalFocusIdx++
		if f.modalFocusIdx > 3 {
			f.modalFocusIdx = 0
		}
	}

	return f, f.focusedModalField().Focus()
}

func (f *WorktreesForm) confirmEdit() (tea.Model, tea.Cmd) {
	// Type assertion failures return zero value which is acceptable
	name := strings.TrimSpace(f.nameField.GetValue().(string))
	branch := strings.TrimSpace(f.branchField.GetValue().(string))
	mergeInto := strings.TrimSpace(f.mergeIntoField.GetValue().(string))
	description := strings.TrimSpace(f.descriptionField.GetValue().(string))

	if name == "" {
		f.validationError = "Name is required"
		return f, nil
	}
	if !validWorktreeNameRegex.MatchString(name) {
		f.validationError = "Invalid name: use only alphanumeric, -, _"
		return f, nil
	}
	if branch == "" {
		f.validationError = "Branch is required"
		return f, nil
	}

	if f.modalState == WorktreeModalAdd {
		for _, wt := range f.worktrees {
			if wt.Name == name {
				f.validationError = "Worktree '" + name + "' already exists"
				return f, nil
			}
		}
	} else if f.editingIdx >= 0 {
		for i, wt := range f.worktrees {
			if i != f.editingIdx && wt.Name == name {
				f.validationError = "Worktree '" + name + "' already exists"
				return f, nil
			}
		}
	}

	entry := WorktreeEntry{
		Name:        name,
		Branch:      branch,
		MergeInto:   mergeInto,
		Description: description,
	}

	if f.modalState == WorktreeModalAdd {
		f.worktrees = append(f.worktrees, entry)
	} else if f.editingIdx >= 0 && f.editingIdx < len(f.worktrees) {
		f.worktrees[f.editingIdx] = entry
	}

	f.refreshTable()
	f.modalState = WorktreeModalNone
	f.editingIdx = -1
	f.validationError = ""
	return f, nil
}

func (f *WorktreesForm) handleDeleteModal(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return f, nil
	}

	switch keyMsg.String() {
	case "y", "Y":
		return f.confirmDelete()
	case "n", "N", "esc":
		f.modalState = WorktreeModalNone
		f.editingIdx = -1
		return f, nil
	}

	newField, cmd := f.confirmField.Update(keyMsg)
	f.confirmField = newField
	return f, cmd
}

func (f *WorktreesForm) confirmDelete() (tea.Model, tea.Cmd) {
	if f.editingIdx >= 0 && f.editingIdx < len(f.worktrees) {
		f.worktrees = append(f.worktrees[:f.editingIdx], f.worktrees[f.editingIdx+1:]...)
		f.refreshTable()
	}
	f.modalState = WorktreeModalNone
	f.editingIdx = -1
	return f, nil
}

func (f *WorktreesForm) handleDiscardModal(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return f, nil
	}

	switch keyMsg.String() {
	case "y", "Y":
		f.cancelled = true
		f.modalState = WorktreeModalNone
		return f, func() tea.Msg {
			return tui.BackBoundaryMsg{}
		}
	case "n", "N", "esc":
		f.modalState = WorktreeModalNone
		return f, nil
	}

	newField, cmd := f.confirmField.Update(keyMsg)
	f.confirmField = newField
	return f, cmd
}

func (f *WorktreesForm) save() (tea.Model, tea.Cmd) {
	if f.onSave != nil {
		err := f.onSave(f.worktrees)
		if err != nil {
			return f, nil
		}
	}
	f.submitted = true
	return f, func() tea.Msg {
		return tui.BackBoundaryMsg{}
	}
}

func (f *WorktreesForm) refreshTable() {
	rows := buildWorktreeRows(f.worktrees)
	f.table.SetRows(rows)
}

func (f *WorktreesForm) handleWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	f.width = msg.Width
	f.height = msg.Height
	f.table.SetHeight(min(8, msg.Height-10))
	return f, nil
}

func (f *WorktreesForm) View() string {
	switch f.modalState {
	case WorktreeModalNone:
		// Fall through to normal view below
	case WorktreeModalHelp:
		return f.helpOverlay.View()
	case WorktreeModalAdd:
		return f.renderEditModal("Add Worktree")
	case WorktreeModalEdit:
		return f.renderEditModal("Edit Worktree")
	case WorktreeModalDelete:
		return f.confirmField.View()
	case WorktreeModalDiscard:
		return f.confirmField.View()
	}

	lines := []string{
		f.theme.Focused.Title.Render("Worktrees"),
		"",
		f.table.View(),
		"",
	}

	if len(f.worktrees) == 0 {
		lines = append(lines, f.theme.Blurred.Description.Render("No worktrees configured. Press 'a' to add."))
		lines = append(lines, "")
	}

	lines = append(lines, f.theme.Blurred.Description.Render("a=add  e=edit  d=delete  s=save  ?=help  q=quit"))

	return strings.Join(lines, "\n")
}

func (f *WorktreesForm) renderEditModal(title string) string {
	lines := []string{
		f.theme.Focused.Title.Render(title),
		"",
		f.nameField.View(),
		"",
		f.branchField.View(),
		"",
		f.mergeIntoField.View(),
		"",
		f.descriptionField.View(),
		"",
	}

	if f.validationError != "" {
		lines = append(lines, f.theme.Focused.Error.Render("Error: "+f.validationError))
		lines = append(lines, "")
	}

	lines = append(lines, f.theme.Blurred.Description.Render("Tab=next  Enter=confirm  Esc=cancel"))
	return strings.Join(lines, "\n")
}

func (f *WorktreesForm) GetWorktrees() []WorktreeEntry {
	return f.worktrees
}

func (f *WorktreesForm) IsComplete() bool {
	return f.submitted
}

func (f *WorktreesForm) IsCancelled() bool {
	return f.cancelled
}

func (f *WorktreesForm) GetModalState() WorktreeModalState {
	return f.modalState
}

// Focus gives the form keyboard focus.
func (f *WorktreesForm) Focus() tea.Cmd {
	return f.table.Init()
}

// Blur removes keyboard focus from the form.
func (f *WorktreesForm) Blur() tea.Cmd {
	f.insertMode = false
	return nil
}

// FocusedYOffset returns the line number where the focused element starts.
// This implements the tui.FocusReporter interface for auto-scrolling support.
func (f *WorktreesForm) FocusedYOffset() int {
	// Count lines helper
	countLines := func(s string) int {
		return strings.Count(s, "\n") + 1
	}

	// In modal mode, focus is on form fields
	if f.modalState == WorktreeModalAdd || f.modalState == WorktreeModalEdit {
		lineCount := 2 // Title + empty line

		formFields := []tui.Field{f.nameField, f.branchField, f.mergeIntoField, f.descriptionField}
		for i, field := range formFields {
			if f.modalFocusIdx == i {
				return lineCount
			}
			lineCount += countLines(field.View()) + 1 // field + empty line
		}
		return lineCount
	}

	// In normal view, the table handles its own scrolling
	// Return position after title
	return 2
}

var _ tea.Model = (*WorktreesForm)(nil)

// InInsertMode reports whether the form is in insert mode.
func (f *WorktreesForm) InInsertMode() bool { return f.insertMode }

// Ensure WorktreesForm implements tui.FocusReporter.
var _ tui.FocusReporter = (*WorktreesForm)(nil)

// Ensure WorktreesForm implements tui.InsertModeReporter.
var _ tui.InsertModeReporter = (*WorktreesForm)(nil)

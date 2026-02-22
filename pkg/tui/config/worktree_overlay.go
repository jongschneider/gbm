package config

import (
	"errors"
	"fmt"
	"gbm/pkg/tui"
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// WorktreeOverlayState represents the interaction state of the worktree editor overlay.
type WorktreeOverlayState int

const (
	// WOBrowsing is the default state: navigating between the three fields.
	WOBrowsing WorktreeOverlayState = iota
	// WOEditing is active when an inline text input is open for a field.
	WOEditing
	// WORenaming is active when the worktree name (map key) is being changed.
	WORenaming
	// WOPromptingName is active when a new worktree is being created and the
	// user is prompted to enter the name before the editor opens.
	WOPromptingName
)

// WorktreeOverlay is an editor overlay for a single worktree entry.
// It provides three navigable fields (Branch, Merge Into, Description),
// inline editing via `e`, renaming via `r`, and validation for the
// worktree name (map key).
//
// The overlay follows the same visual pattern as other overlays in the
// Config TUI: a bordered box with a title, field rows, and a hint line.
type WorktreeOverlay struct {
	theme         *tui.Theme
	origValues    [3]string
	name          string
	origName      string
	nameErr       string
	keys          EditorOverlayKeyMap
	existingNames []string
	worktreeNames []string
	branchNames   []string
	fields        []*FieldRow
	nameInput     textinput.Model
	renameInput   textinput.Model
	focusIndex    int
	width         int
	state         WorktreeOverlayState
	isNew         bool
	confirmed     bool
}

// WorktreeOverlayOption configures a WorktreeOverlay during construction.
type WorktreeOverlayOption func(*WorktreeOverlay)

// WithWorktreeTheme sets the theme for the overlay.
func WithWorktreeTheme(theme *tui.Theme) WorktreeOverlayOption {
	return func(o *WorktreeOverlay) {
		if theme != nil {
			o.theme = theme
		}
	}
}

// WithWorktreeWidth sets the rendering width.
func WithWorktreeWidth(w int) WorktreeOverlayOption {
	return func(o *WorktreeOverlay) {
		if w > 0 {
			o.width = w
		}
	}
}

// WithWorktreeNames sets the list of worktree names for autocomplete suggestions
// on the merge_into field.
func WithWorktreeNames(names []string) WorktreeOverlayOption {
	return func(o *WorktreeOverlay) {
		o.worktreeNames = names
	}
}

// WithBranchNames sets the list of branch names for autocomplete suggestions
// on the branch field.
func WithBranchNames(names []string) WorktreeOverlayOption {
	return func(o *WorktreeOverlay) {
		o.branchNames = names
	}
}

// WithExistingNames sets the list of other worktree names for duplicate detection.
func WithExistingNames(names []string) WorktreeOverlayOption {
	return func(o *WorktreeOverlay) {
		o.existingNames = names
	}
}

// NewWorktreeOverlay creates a worktree editor overlay for an existing entry.
// name is the map key, values are the current field values in order:
// [branch, merge_into, description].
func NewWorktreeOverlay(name string, values [3]string, opts ...WorktreeOverlayOption) *WorktreeOverlay {
	o := &WorktreeOverlay{
		theme: tui.DefaultTheme(),
		keys:  NewEditorOverlayKeys(),
		name:  name,
		width: 50,
	}
	for _, opt := range opts {
		opt(o)
	}

	o.origName = name
	o.origValues = values
	o.initFields()
	o.loadValues(values)

	return o
}

// NewWorktreeOverlayForNew creates a worktree editor overlay for a new entry.
// It starts in the WOPromptingName state, prompting the user to enter a name
// before the editor fields are shown.
func NewWorktreeOverlayForNew(opts ...WorktreeOverlayOption) *WorktreeOverlay {
	o := &WorktreeOverlay{
		theme: tui.DefaultTheme(),
		keys:  NewEditorOverlayKeys(),
		width: 50,
		isNew: true,
		state: WOPromptingName,
	}
	for _, opt := range opts {
		opt(o)
	}

	o.initFields()
	o.initNameInput()

	return o
}

// initFields creates the three FieldRow instances from worktreeEntryFields.
func (o *WorktreeOverlay) initFields() {
	lw := 0
	for _, fm := range worktreeEntryFields {
		if len(fm.Label) > lw {
			lw = len(fm.Label)
		}
	}
	if lw < 10 {
		lw = 10
	}

	o.fields = make([]*FieldRow, len(worktreeEntryFields))
	for i, fm := range worktreeEntryFields {
		fr := NewFieldRow(fm, o.theme)
		fr.SetLabelWidth(lw)
		fr.SetWidth(o.innerWidth())
		o.fields[i] = fr
	}
	// Set dynamic suggestions on the branch field (index 0).
	if len(o.fields) > 0 && len(o.branchNames) > 0 {
		names := o.branchNames // capture for closure
		o.fields[0].SetSuggestions(func() []string { return names })
	}
	// Set dynamic suggestions on the merge_into field (index 1).
	if len(o.fields) > 1 && len(o.worktreeNames) > 0 {
		names := o.worktreeNames // capture for closure
		o.fields[1].SetSuggestions(func() []string { return names })
	}
	if len(o.fields) > 0 {
		o.fields[0].SetFocused(true)
	}
}

// initNameInput sets up the text input used for new worktree name prompting.
func (o *WorktreeOverlay) initNameInput() {
	ti := textinput.New()
	ti.CharLimit = 256
	ti.Prompt = ""
	ti.Placeholder = "worktree name"
	ti.Focus()
	ti.Width = max(o.innerWidth()-20, 10)
	o.nameInput = ti
}

// initRenameInput sets up the text input used for renaming.
func (o *WorktreeOverlay) initRenameInput() {
	ti := textinput.New()
	ti.CharLimit = 256
	ti.Prompt = ""
	ti.SetValue(o.name)
	ti.SetCursor(len(o.name))
	ti.Focus()
	ti.Width = max(o.innerWidth()-20, 10)
	o.renameInput = ti
}

// loadValues populates the field rows with the given values.
func (o *WorktreeOverlay) loadValues(values [3]string) {
	for i, v := range values {
		if i < len(o.fields) {
			o.fields[i].SetValue(v)
		}
	}
}

// --- State accessors ---.

// State returns the current overlay state.
func (o *WorktreeOverlay) State() WorktreeOverlayState {
	return o.state
}

// Name returns the current worktree name (map key).
func (o *WorktreeOverlay) Name() string {
	return o.name
}

// IsNew reports whether this overlay is creating a new worktree entry.
func (o *WorktreeOverlay) IsNew() bool {
	return o.isNew
}

// IsConfirmed reports whether the overlay was closed via enter (confirm).
func (o *WorktreeOverlay) IsConfirmed() bool {
	return o.confirmed
}

// FocusIndex returns the currently focused field index.
func (o *WorktreeOverlay) FocusIndex() int {
	return o.focusIndex
}

// Values returns the current field values as [branch, merge_into, description].
func (o *WorktreeOverlay) Values() [3]string {
	var result [3]string
	for i := range 3 {
		if i < len(o.fields) {
			if s, ok := o.fields[i].Value().(string); ok {
				result[i] = s
			}
		}
	}
	return result
}

// Fields returns the FieldRow slice for testing/inspection.
func (o *WorktreeOverlay) Fields() []*FieldRow {
	return o.fields
}

// NameError returns the current name validation error, if any.
func (o *WorktreeOverlay) NameError() string {
	return o.nameErr
}

// --- Key handling ---.

// HandleKey processes a tea.KeyMsg and returns a tea.Cmd if any.
// Returns true if the overlay should be closed (confirmed or cancelled).
func (o *WorktreeOverlay) HandleKey(msg tea.KeyMsg) (tea.Cmd, bool) {
	switch o.state {
	case WOPromptingName:
		return o.handlePromptingNameKey(msg)
	case WOBrowsing:
		return o.handleBrowsingKey(msg)
	case WOEditing:
		return o.handleEditingKey(msg)
	case WORenaming:
		return o.handleRenamingKey(msg)
	}
	return nil, false
}

// handlePromptingNameKey processes keys during the new-worktree name prompt.
func (o *WorktreeOverlay) handlePromptingNameKey(msg tea.KeyMsg) (tea.Cmd, bool) {
	switch msg.String() {
	case "enter":
		name := strings.TrimSpace(o.nameInput.Value())
		if err := o.validateName(name); err != nil {
			o.nameErr = err.Error()
			return nil, false
		}
		o.name = name
		o.origName = name
		o.nameErr = ""
		o.state = WOBrowsing
		return nil, false

	case "esc":
		return nil, true
	}

	// Clear error on new input.
	if msg.Type == tea.KeyRunes || msg.Type == tea.KeyBackspace {
		o.nameErr = ""
	}

	var cmd tea.Cmd
	o.nameInput, cmd = o.nameInput.Update(msg)
	return cmd, false
}

// handleBrowsingKey processes keys during field navigation.
func (o *WorktreeOverlay) handleBrowsingKey(msg tea.KeyMsg) (tea.Cmd, bool) {
	switch msg.String() {
	case "up", "k":
		o.moveFocusUp()
		return nil, false

	case "down", "j":
		o.moveFocusDown()
		return nil, false

	case "e":
		cmd := o.enterEditing()
		return cmd, false

	case "r":
		o.enterRenaming()
		return textinput.Blink, false

	case "enter":
		o.confirmed = true
		return nil, true

	case "esc":
		o.discard()
		return nil, true
	}

	return nil, false
}

// handleEditingKey processes keys during inline field editing.
func (o *WorktreeOverlay) handleEditingKey(msg tea.KeyMsg) (tea.Cmd, bool) {
	switch msg.String() {
	case "enter":
		_, err := o.fields[o.focusIndex].CommitEditing()
		if err != nil {
			// Stay in editing state; error is displayed by FieldRow.
			return nil, false
		}
		o.state = WOBrowsing
		return nil, false

	case "esc":
		o.fields[o.focusIndex].CancelEditing()
		o.state = WOBrowsing
		return nil, false
	}

	// Forward to the active FieldRow's text input.
	cmd := o.fields[o.focusIndex].UpdateInput(msg)
	return cmd, false
}

// handleRenamingKey processes keys during name renaming.
func (o *WorktreeOverlay) handleRenamingKey(msg tea.KeyMsg) (tea.Cmd, bool) {
	switch msg.String() {
	case "enter":
		newName := strings.TrimSpace(o.renameInput.Value())
		if err := o.validateName(newName); err != nil {
			o.nameErr = err.Error()
			return nil, false
		}
		o.name = newName
		o.nameErr = ""
		o.state = WOBrowsing
		return nil, false

	case "esc":
		o.nameErr = ""
		o.state = WOBrowsing
		return nil, false
	}

	// Clear error on new input.
	if msg.Type == tea.KeyRunes || msg.Type == tea.KeyBackspace {
		o.nameErr = ""
	}

	var cmd tea.Cmd
	o.renameInput, cmd = o.renameInput.Update(msg)
	return cmd, false
}

// --- Navigation ---.

// moveFocusDown moves focus to the next field, wrapping around.
func (o *WorktreeOverlay) moveFocusDown() {
	if len(o.fields) == 0 {
		return
	}
	o.fields[o.focusIndex].SetFocused(false)
	o.focusIndex = (o.focusIndex + 1) % len(o.fields)
	o.fields[o.focusIndex].SetFocused(true)
}

// moveFocusUp moves focus to the previous field, wrapping around.
func (o *WorktreeOverlay) moveFocusUp() {
	if len(o.fields) == 0 {
		return
	}
	o.fields[o.focusIndex].SetFocused(false)
	o.focusIndex = (o.focusIndex + len(o.fields) - 1) % len(o.fields)
	o.fields[o.focusIndex].SetFocused(true)
}

// enterEditing transitions the focused field into inline editing mode.
func (o *WorktreeOverlay) enterEditing() tea.Cmd {
	if len(o.fields) == 0 {
		return nil
	}
	cmd := o.fields[o.focusIndex].EnterEditing()
	if cmd != nil {
		o.state = WOEditing
	}
	return cmd
}

// enterRenaming transitions the overlay into rename mode.
func (o *WorktreeOverlay) enterRenaming() {
	o.initRenameInput()
	o.nameErr = ""
	o.state = WORenaming
}

// discard restores original values and name before closing.
func (o *WorktreeOverlay) discard() {
	o.name = o.origName
	o.loadValues(o.origValues)
	o.confirmed = false
}

// --- Name validation ---.

// validateName checks that the given name is valid for use as a worktree map
// key. It rejects empty names, duplicates (against existingNames), and names
// containing characters that git disallows in branch names.
func (o *WorktreeOverlay) validateName(name string) error {
	if name == "" {
		return errors.New("name is required")
	}

	if err := validateGitBranchChars(name); err != nil {
		return err
	}

	if slices.Contains(o.existingNames, name) {
		return fmt.Errorf("worktree %q already exists", name)
	}

	return nil
}

// validateGitBranchChars rejects characters that git disallows in ref names.
// Based on git-check-ref-format rules: no space, ~, ^, :, ?, *, [, \, null,
// DEL, or control characters; no ".." sequence; no leading/trailing ".";
// no trailing ".lock"; no leading "-".
func validateGitBranchChars(name string) error {
	if strings.HasPrefix(name, "-") {
		return errors.New("name cannot start with '-'")
	}
	if strings.HasPrefix(name, ".") {
		return errors.New("name cannot start with '.'")
	}
	if strings.HasSuffix(name, ".") {
		return errors.New("name cannot end with '.'")
	}
	if strings.HasSuffix(name, ".lock") {
		return errors.New("name cannot end with '.lock'")
	}
	if strings.Contains(name, "..") {
		return errors.New("name cannot contain '..'")
	}

	for _, r := range name {
		if r <= 0x1f || r == 0x7f {
			return errors.New("name cannot contain control characters")
		}
		switch r {
		case ' ', '~', '^', ':', '?', '*', '[', '\\':
			return fmt.Errorf("name cannot contain %q", string(r))
		}
	}

	return nil
}

// --- Rendering ---.

// innerWidth returns the usable width inside the overlay border.
func (o *WorktreeOverlay) innerWidth() int {
	// Reserve 4 for border + padding.
	return max(o.width-4, 20)
}

// View renders the full overlay as a bordered box.
func (o *WorktreeOverlay) View() string {
	iw := o.innerWidth()

	var content string
	switch o.state {
	case WOPromptingName:
		content = o.viewNamePrompt(iw)
	default:
		content = o.viewEditor(iw)
	}

	// Build title.
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(o.theme.Accent)

	title := o.buildTitle(titleStyle)

	// Hints line.
	hints := o.buildHints(iw)

	// Assemble: title + blank + content + blank + hints.
	body := title + "\n\n" + content + "\n\n" + hints

	// Box style.
	boxStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(o.theme.Border).
		Padding(0, 1).
		Width(iw + 2) // +2 for padding

	return boxStyle.Render(body)
}

// buildTitle returns the styled overlay title line.
func (o *WorktreeOverlay) buildTitle(titleStyle lipgloss.Style) string {
	if o.state == WOPromptingName {
		return titleStyle.Render("Worktrees > New")
	}
	if o.state == WORenaming {
		return titleStyle.Render(fmt.Sprintf("Worktrees > Rename %q", o.name))
	}
	return titleStyle.Render(fmt.Sprintf("Worktrees > Edit %q", o.name))
}

// viewNamePrompt renders the new-worktree name input.
func (o *WorktreeOverlay) viewNamePrompt(width int) string {
	promptStyle := lipgloss.NewStyle().
		Foreground(o.theme.Accent).
		Bold(true)

	var lines []string
	lines = append(lines, promptStyle.Render("Enter worktree name:"))
	lines = append(lines, "  "+o.nameInput.View())

	if o.nameErr != "" {
		errStyle := lipgloss.NewStyle().
			Foreground(o.theme.ErrorAccent).Bold(true)
		lines = append(lines, "  "+errStyle.Render(o.nameErr))
	}

	_ = width
	return strings.Join(lines, "\n")
}

// viewEditor renders the three-field editor.
func (o *WorktreeOverlay) viewEditor(width int) string {
	var lines []string

	if o.state == WORenaming {
		lines = append(lines, o.viewRenameInput())
	}

	for _, fr := range o.fields {
		lines = append(lines, fr.View())
	}

	_ = width
	return strings.Join(lines, "\n")
}

// viewRenameInput renders the rename text input above the fields.
func (o *WorktreeOverlay) viewRenameInput() string {
	labelStyle := lipgloss.NewStyle().
		Foreground(o.theme.Accent).Bold(true)

	line := "  " + labelStyle.Render("Name") + "  " + o.renameInput.View()

	if o.nameErr != "" {
		errStyle := lipgloss.NewStyle().
			Foreground(o.theme.ErrorAccent).Bold(true)
		line += "\n  " + errStyle.Render(o.nameErr)
	}

	return line
}

// buildHints returns the context-sensitive keybinding hints.
func (o *WorktreeOverlay) buildHints(width int) string {
	hintStyle := lipgloss.NewStyle().Foreground(o.theme.Muted)
	keyStyle := lipgloss.NewStyle().Foreground(o.theme.Highlight)
	sep := hintStyle.Render(" . ")

	formatHint := func(k, desc string) string {
		return keyStyle.Render(k) + " " + hintStyle.Render(desc)
	}

	var hints string

	switch o.state {
	case WOPromptingName:
		hints = formatHint("enter", "confirm") + sep + formatHint("esc", "cancel")

	case WOEditing:
		hints = formatHint("enter", "confirm") + sep + formatHint("esc", "cancel")

	case WORenaming:
		hints = formatHint("enter", "confirm") + sep + formatHint("esc", "cancel")

	default: // WOBrowsing
		hints = strings.Join([]string{
			formatHint("up/dn", "navigate"),
			formatHint("e", "edit"),
			formatHint("r", "rename"),
			formatHint("enter", "confirm"),
			formatHint("esc", "cancel"),
		}, sep)
	}

	_ = width
	return hints
}

// SetWidth updates the rendering width and propagates to child field rows.
func (o *WorktreeOverlay) SetWidth(w int) {
	if w <= 0 {
		return
	}
	o.width = w
	iw := o.innerWidth()
	for _, fr := range o.fields {
		fr.SetWidth(iw)
	}
	o.nameInput.Width = max(iw-20, 10)
	o.renameInput.Width = max(iw-20, 10)
}

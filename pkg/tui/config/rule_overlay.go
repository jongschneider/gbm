package config

import (
	"fmt"
	"gbm/pkg/tui"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// RuleOverlayState represents the interaction state of the rule editor overlay.
type RuleOverlayState int

const (
	// ROBrowsing is the default state: navigating between the two fields.
	ROBrowsing RuleOverlayState = iota
	// ROEditingField is active when an inline text input is open for the
	// Source Worktree field.
	ROEditingField
	// ROListOverlay is active when the ListOverlay is open for the Files field.
	ROListOverlay
	// ROConfirmDiscard is active when a discard-changes confirmation is shown.
	ROConfirmDiscard
)

// RuleOverlayResultMsg is sent by the RuleOverlay when the user commits
// or discards changes. The parent model uses this to update the rule.
type RuleOverlayResultMsg struct {
	// SourceWorktree is the edited source worktree value.
	SourceWorktree string
	// Files is the edited list of file patterns.
	Files []string
	// Committed is true when the user pressed enter to confirm.
	Committed bool
}

// RuleOverlay is an editor overlay for a single file copy rule.
// It provides two navigable fields (Source Worktree, Files), inline
// editing for the string field via `e`, and a nested ListOverlay for
// the Files field via `e`.
//
// The overlay follows the same visual pattern as WorktreeOverlay:
// a bordered box with a title, field rows, and a hint line.
type RuleOverlay struct {
	theme       *tui.Theme
	keys        EditorOverlayKeyMap
	fields      []*FieldRow
	listOverlay *ListOverlay
	origSource  string
	origFiles   []string
	focusIndex  int
	width       int
	state       RuleOverlayState
	isNew       bool
}

// RuleOverlayOption configures a RuleOverlay during construction.
type RuleOverlayOption func(*RuleOverlay)

// WithRuleTheme sets the theme for the overlay.
func WithRuleTheme(theme *tui.Theme) RuleOverlayOption {
	return func(o *RuleOverlay) {
		if theme != nil {
			o.theme = theme
		}
	}
}

// WithRuleWidth sets the rendering width.
func WithRuleWidth(w int) RuleOverlayOption {
	return func(o *RuleOverlay) {
		if w > 0 {
			o.width = w
		}
	}
}

// NewRuleOverlay creates a rule editor overlay for an existing file copy rule.
// sourceWorktree is the current source worktree value, files is the current
// list of file patterns.
func NewRuleOverlay(sourceWorktree string, files []string, opts ...RuleOverlayOption) *RuleOverlay {
	o := &RuleOverlay{
		theme: tui.DefaultTheme(),
		keys:  NewEditorOverlayKeys(),
		width: 50,
	}
	for _, opt := range opts {
		opt(o)
	}

	o.origSource = sourceWorktree
	o.origFiles = copyStrings(files)
	o.initFields()
	o.loadValues(sourceWorktree, files)

	return o
}

// NewRuleOverlayForNew creates a rule editor overlay for a new file copy rule.
// It starts with empty fields.
func NewRuleOverlayForNew(opts ...RuleOverlayOption) *RuleOverlay {
	o := &RuleOverlay{
		theme: tui.DefaultTheme(),
		keys:  NewEditorOverlayKeys(),
		width: 50,
		isNew: true,
	}
	for _, opt := range opts {
		opt(o)
	}

	o.initFields()

	return o
}

// initFields creates the two FieldRow instances from fileCopyRuleFields.
func (o *RuleOverlay) initFields() {
	lw := 0
	for _, fm := range fileCopyRuleFields {
		if len(fm.Label) > lw {
			lw = len(fm.Label)
		}
	}
	if lw < 10 {
		lw = 10
	}

	o.fields = make([]*FieldRow, len(fileCopyRuleFields))
	for i, fm := range fileCopyRuleFields {
		fr := NewFieldRow(fm, o.theme)
		fr.SetLabelWidth(lw)
		fr.SetWidth(o.innerWidth())
		o.fields[i] = fr
	}
	if len(o.fields) > 0 {
		o.fields[0].SetFocused(true)
	}
}

// loadValues populates the field rows with the given values.
func (o *RuleOverlay) loadValues(sourceWorktree string, files []string) {
	if len(o.fields) > 0 {
		o.fields[0].SetValue(sourceWorktree)
	}
	if len(o.fields) > 1 {
		o.fields[1].SetValue(copyStrings(files))
	}
}

// --- State accessors ---.

// State returns the current overlay state.
func (o *RuleOverlay) State() RuleOverlayState {
	return o.state
}

// IsNew reports whether this overlay is creating a new rule entry.
func (o *RuleOverlay) IsNew() bool {
	return o.isNew
}

// FocusIndex returns the currently focused field index.
func (o *RuleOverlay) FocusIndex() int {
	return o.focusIndex
}

// SourceWorktree returns the current source worktree value.
func (o *RuleOverlay) SourceWorktree() string {
	if len(o.fields) > 0 {
		if s, ok := o.fields[0].Value().(string); ok {
			return s
		}
	}
	return ""
}

// Files returns the current files list.
func (o *RuleOverlay) Files() []string {
	if len(o.fields) > 1 {
		if sl, ok := o.fields[1].Value().([]string); ok {
			return copyStrings(sl)
		}
	}
	return nil
}

// Fields returns the FieldRow slice for testing/inspection.
func (o *RuleOverlay) Fields() []*FieldRow {
	return o.fields
}

// ListOverlay returns the nested list overlay, or nil if not active.
func (o *RuleOverlay) ListOverlay() *ListOverlay {
	return o.listOverlay
}

// HasChanges reports whether the current values differ from the originals.
func (o *RuleOverlay) HasChanges() bool {
	if o.SourceWorktree() != o.origSource {
		return true
	}
	files := o.Files()
	if len(files) != len(o.origFiles) {
		return true
	}
	for i := range files {
		if files[i] != o.origFiles[i] {
			return true
		}
	}
	return false
}

// --- Key handling ---.

// HandleKey processes a tea.KeyMsg and returns a result message if the overlay
// should close, along with any tea.Cmd to forward. Returns nil result while
// the overlay remains open.
func (o *RuleOverlay) HandleKey(msg tea.KeyMsg) (*RuleOverlayResultMsg, tea.Cmd) {
	switch o.state {
	case ROBrowsing:
		return o.handleBrowsingKey(msg)
	case ROEditingField:
		return o.handleEditingKey(msg)
	case ROListOverlay:
		return o.handleListOverlayKey(msg)
	case ROConfirmDiscard:
		return o.handleConfirmDiscardKey(msg)
	}
	return nil, nil
}

// handleBrowsingKey processes keys during field navigation.
func (o *RuleOverlay) handleBrowsingKey(msg tea.KeyMsg) (*RuleOverlayResultMsg, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		o.moveFocusUp()
		return nil, nil

	case "down", "j":
		o.moveFocusDown()
		return nil, nil

	case "e":
		return o.enterEdit()

	case "enter":
		return o.confirm(), nil

	case "esc":
		return o.tryDiscard()
	}

	return nil, nil
}

// handleEditingKey processes keys during inline field editing (Source Worktree).
func (o *RuleOverlay) handleEditingKey(msg tea.KeyMsg) (*RuleOverlayResultMsg, tea.Cmd) {
	switch msg.String() {
	case "enter":
		_, err := o.fields[o.focusIndex].CommitEditing()
		if err != nil {
			return nil, nil
		}
		o.state = ROBrowsing
		return nil, nil

	case "esc":
		o.fields[o.focusIndex].CancelEditing()
		o.state = ROBrowsing
		return nil, nil
	}

	cmd := o.fields[o.focusIndex].UpdateInput(msg)
	return nil, cmd
}

// handleListOverlayKey forwards keys to the nested ListOverlay.
func (o *RuleOverlay) handleListOverlayKey(msg tea.KeyMsg) (*RuleOverlayResultMsg, tea.Cmd) {
	if o.listOverlay == nil {
		o.state = ROBrowsing
		return nil, nil
	}

	result, cmd := o.listOverlay.Update(msg)
	if result == nil {
		return nil, cmd
	}

	// ListOverlay closed. Apply result if committed.
	if result.Committed && len(o.fields) > 1 {
		o.fields[1].SetValue(copyStrings(result.Items))
	}
	o.listOverlay = nil
	o.state = ROBrowsing
	return nil, cmd
}

// handleConfirmDiscardKey processes keys during the discard confirmation.
func (o *RuleOverlay) handleConfirmDiscardKey(msg tea.KeyMsg) (*RuleOverlayResultMsg, tea.Cmd) {
	switch msg.String() {
	case "y":
		return o.discard(), nil

	case "n", "esc":
		o.state = ROBrowsing
		return nil, nil
	}

	return nil, nil
}

// --- Navigation ---.

// moveFocusDown moves focus to the next field, wrapping around.
func (o *RuleOverlay) moveFocusDown() {
	if len(o.fields) == 0 {
		return
	}
	o.fields[o.focusIndex].SetFocused(false)
	o.focusIndex = (o.focusIndex + 1) % len(o.fields)
	o.fields[o.focusIndex].SetFocused(true)
}

// moveFocusUp moves focus to the previous field, wrapping around.
func (o *RuleOverlay) moveFocusUp() {
	if len(o.fields) == 0 {
		return
	}
	o.fields[o.focusIndex].SetFocused(false)
	o.focusIndex = (o.focusIndex + len(o.fields) - 1) % len(o.fields)
	o.fields[o.focusIndex].SetFocused(true)
}

// enterEdit transitions the focused field into the appropriate editing mode.
// Source Worktree (index 0): inline text input.
// Files (index 1): nested ListOverlay.
func (o *RuleOverlay) enterEdit() (*RuleOverlayResultMsg, tea.Cmd) {
	if len(o.fields) == 0 {
		return nil, nil
	}

	focused := o.fields[o.focusIndex]

	// Files field: open nested ListOverlay.
	if focused.Meta().Type == StringList {
		var items []string
		if sl, ok := focused.Value().([]string); ok {
			items = copyStrings(sl)
		}
		title := "File Copy > Files"
		o.listOverlay = NewListOverlay(title, items, o.theme)
		o.listOverlay.SetSize(o.width, 20)
		o.state = ROListOverlay
		return nil, nil
	}

	// String field: inline text input.
	cmd := focused.EnterEditing()
	if cmd != nil {
		o.state = ROEditingField
	}
	return nil, cmd
}

// confirm builds the result message with current values.
func (o *RuleOverlay) confirm() *RuleOverlayResultMsg {
	return &RuleOverlayResultMsg{
		SourceWorktree: o.SourceWorktree(),
		Files:          o.Files(),
		Committed:      true,
	}
}

// tryDiscard initiates discard: if changes were made, asks for confirmation;
// otherwise discards immediately.
func (o *RuleOverlay) tryDiscard() (*RuleOverlayResultMsg, tea.Cmd) {
	if !o.HasChanges() {
		return o.discard(), nil
	}
	o.state = ROConfirmDiscard
	return nil, nil
}

// discard returns a non-committed result.
func (o *RuleOverlay) discard() *RuleOverlayResultMsg {
	return &RuleOverlayResultMsg{
		Committed: false,
	}
}

// --- Rendering ---.

// innerWidth returns the usable width inside the overlay border.
func (o *RuleOverlay) innerWidth() int {
	return max(o.width-4, 20)
}

// View renders the full overlay as a bordered box.
func (o *RuleOverlay) View() string {
	iw := o.innerWidth()

	// Build title.
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(o.theme.Accent)

	var title string
	if o.isNew {
		title = titleStyle.Render("File Copy > New Rule")
	} else {
		title = titleStyle.Render("File Copy > Edit Rule")
	}

	// Render field rows.
	var fieldLines []string
	for _, fr := range o.fields {
		fieldLines = append(fieldLines, fr.View())
	}
	content := strings.Join(fieldLines, "\n")

	// Discard confirmation prompt.
	if o.state == ROConfirmDiscard {
		promptStyle := lipgloss.NewStyle().
			Foreground(o.theme.ErrorAccent).Bold(true)
		content += "\n\n" + promptStyle.Render("  Discard changes? y/n")
	}

	// Hints line.
	hints := o.buildHints()

	// Assemble: title + blank + content + blank + hints.
	body := title + "\n\n" + content + "\n\n" + hints

	// Box style.
	boxStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(o.theme.Border).
		Padding(0, 1).
		Width(iw + 2) // +2 for padding

	rendered := boxStyle.Render(body)

	// If the nested list overlay is active, render it on top.
	if o.state == ROListOverlay && o.listOverlay != nil {
		rendered += "\n" + o.listOverlay.View(o.width, 20)
	}

	return rendered
}

// buildHints returns the context-sensitive keybinding hints.
func (o *RuleOverlay) buildHints() string {
	hintStyle := lipgloss.NewStyle().Foreground(o.theme.Muted)
	keyStyle := lipgloss.NewStyle().Foreground(o.theme.Highlight)
	sep := hintStyle.Render(" . ")

	formatHint := func(k, desc string) string {
		return keyStyle.Render(k) + " " + hintStyle.Render(desc)
	}

	switch o.state {
	case ROEditingField:
		return formatHint("enter", "confirm") + sep + formatHint("esc", "cancel")

	case ROConfirmDiscard:
		return formatHint("y", "yes") + sep + formatHint("n", "no")

	default: // ROBrowsing (ROListOverlay uses its own hints)
		return strings.Join([]string{
			formatHint("up/dn", "navigate"),
			formatHint("e", "edit"),
			formatHint("enter", "confirm"),
			formatHint("esc", "cancel"),
		}, sep)
	}
}

// SetWidth updates the rendering width and propagates to child field rows.
func (o *RuleOverlay) SetWidth(w int) {
	if w <= 0 {
		return
	}
	o.width = w
	iw := o.innerWidth()
	for _, fr := range o.fields {
		fr.SetWidth(iw)
	}
}

// --- Helpers ---.

// copyStrings returns a deep copy of a string slice.
func copyStrings(src []string) []string {
	if src == nil {
		return nil
	}
	dst := make([]string, len(src))
	copy(dst, src)
	return dst
}

// formatFileSummary returns a short summary of a file list for display.
func formatFileSummary(files []string) string {
	if len(files) == 0 {
		return "(none)"
	}
	return fmt.Sprintf("%d file(s)", len(files))
}

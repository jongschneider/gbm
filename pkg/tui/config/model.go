package config

import (
	"fmt"
	"gbm/pkg/tui"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"gopkg.in/yaml.v3"
)

// Minimum terminal dimensions required by the Config TUI.
const (
	MinTermWidth  = 60
	MinTermHeight = 16
)

// ModelState represents the current interaction state of the ConfigModel.
type ModelState int

const (
	// StateBrowsing is the default state: navigating fields with arrow keys.
	StateBrowsing ModelState = iota
	// StateEditing is active when an inline text input is open.
	StateEditing
	// StateHelp is active when the help overlay is displayed.
	StateHelp
	// StateErrors is active when the validation errors overlay is displayed.
	StateErrors
	// StateSaving is active while a save operation is in progress.
	StateSaving
	// StateOverwriteConfirm is active when the overwrite confirmation overlay
	// is displayed (external change detected).
	StateOverwriteConfirm
	// StateWriteError is active when a write error overlay is displayed.
	StateWriteError
)

// SectionTab identifies which tab is active.
type SectionTab int

const (
	TabGeneral SectionTab = iota
	TabJira
	TabFileCopy
	TabWorktrees
)

// tabCount is the total number of tabs in the Config TUI.
const tabCount = 4

// tabLabels maps each tab to its display label.
var tabLabels = [tabCount]string{
	"General",
	"JIRA",
	"File Copy",
	"Worktrees",
}

// flashClearMsg is sent after a timer to clear the status bar flash message.
type flashClearMsg struct{}

// ConfigModel is the root Bubble Tea model for the Config TUI.
// It orchestrates tab navigation, state transitions, dirty tracking, and
// delegates rendering/updates to per-section models (future ticket).
type ConfigModel struct {
	modTime          time.Time
	accessor         ConfigAccessor
	dirty            *DirtyTracker
	helpOverlay      *HelpOverlay
	errorOverlay     *ErrorOverlay
	root             *yaml.Node
	theme            *tui.Theme
	filePath         string
	flashMessage     string
	writeErrorMsg    string
	browsingKeys     BrowsingKeyMap
	editingKeys      EditingKeyMap
	confirmKeys      ConfirmationKeyMap
	activeTab        SectionTab
	width            int
	height           int
	state            ModelState
	focusedFieldType FieldType
	tabBadges        [tabCount]bool
	isNewFile        bool
	quitAfterSave    bool
}

// NewConfigModel creates a new ConfigModel with the given options.
// The model starts in browsing state on the General tab.
func NewConfigModel(opts ...ConfigModelOption) *ConfigModel {
	defaultTheme := tui.DefaultTheme()
	m := &ConfigModel{
		theme:        defaultTheme,
		dirty:        NewDirtyTracker(nil),
		helpOverlay:  NewHelpOverlay(defaultTheme),
		errorOverlay: NewErrorOverlay(nil, defaultTheme),
		browsingKeys: NewBrowsingKeys(),
		editingKeys:  NewEditingKeys(),
		confirmKeys:  NewConfirmationKeys(),
		activeTab:    TabGeneral,
		state:        StateBrowsing,
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// ConfigModelOption configures a ConfigModel during construction.
type ConfigModelOption func(*ConfigModel)

// WithTheme sets the theme for the ConfigModel.
func WithTheme(theme *tui.Theme) ConfigModelOption {
	return func(m *ConfigModel) {
		if theme != nil {
			m.theme = theme
			m.helpOverlay = NewHelpOverlay(theme)
			m.errorOverlay = NewErrorOverlay(nil, theme)
		}
	}
}

// WithDirtyTracker sets the dirty tracker for the ConfigModel.
func WithDirtyTracker(dt *DirtyTracker) ConfigModelOption {
	return func(m *ConfigModel) {
		if dt != nil {
			m.dirty = dt
		}
	}
}

// WithFilePath sets the config file path.
func WithFilePath(path string) ConfigModelOption {
	return func(m *ConfigModel) {
		m.filePath = path
	}
}

// WithNewFile marks the model as editing a new (not-yet-saved) config file.
func WithNewFile(isNew bool) ConfigModelOption {
	return func(m *ConfigModel) {
		m.isNewFile = isNew
	}
}

// WithAccessor sets the config accessor for reading field values during save.
func WithAccessor(accessor ConfigAccessor) ConfigModelOption {
	return func(m *ConfigModel) {
		m.accessor = accessor
	}
}

// WithYAMLRoot sets the YAML node root for comment-preserving writes.
func WithYAMLRoot(root *yaml.Node) ConfigModelOption {
	return func(m *ConfigModel) {
		m.root = root
	}
}

// WithModTime sets the file modification time for external change detection.
func WithModTime(t time.Time) ConfigModelOption {
	return func(m *ConfigModel) {
		m.modTime = t
	}
}

// Init implements tea.Model. It returns nil (no initial command).
func (m *ConfigModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model. It routes messages by type and delegates
// key handling based on the current state.
func (m *ConfigModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case flashClearMsg:
		m.flashMessage = ""
		return m, nil

	case SaveResultMsg:
		return m.handleSaveResult(msg)

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	}

	return m, nil
}

// handleKeyMsg routes key presses based on the current state.
func (m *ConfigModel) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.state {
	case StateBrowsing:
		return m.handleBrowsingKey(msg)
	case StateEditing:
		return m.handleEditingKey(msg)
	case StateHelp:
		return m.handleHelpKey(msg)
	case StateErrors:
		return m.handleErrorsKey(msg)
	case StateOverwriteConfirm:
		return m.handleOverwriteConfirmKey(msg)
	case StateWriteError:
		return m.handleWriteErrorKey(msg)
	case StateSaving:
		// Ignore all keys while saving.
		return m, nil
	}
	return m, nil
}

// handleBrowsingKey processes key presses in browsing state.
func (m *ConfigModel) handleBrowsingKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.browsingKeys.NextTab):
		m.nextTab()
		return m, nil

	case key.Matches(msg, m.browsingKeys.PrevTab):
		m.prevTab()
		return m, nil

	case key.Matches(msg, m.browsingKeys.Help):
		m.helpOverlay.ResetScroll()
		m.state = StateHelp
		return m, nil

	case key.Matches(msg, m.browsingKeys.Save):
		return m.startSave(false)

	case key.Matches(msg, m.browsingKeys.SaveQuit):
		return m.startSave(true)

	case key.Matches(msg, m.browsingKeys.Quit):
		return m, tea.Quit

	case key.Matches(msg, m.browsingKeys.ForceQuit):
		return m, tea.Quit
	}

	return m, nil
}

// handleHelpKey processes key presses in help overlay state.
// ? or esc closes the overlay. up/down/j/k scroll the content.
func (m *ConfigModel) handleHelpKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	viewportHeight := max(m.height-6, 1) // account for overlay chrome
	shouldClose := m.helpOverlay.HandleKey(msg, viewportHeight)
	if shouldClose {
		m.state = StateBrowsing
	}
	return m, nil
}

// handleEditingKey processes key presses in editing state.
// Tab and shift-tab are explicitly ignored (disabled in EditingKeyMap).
func (m *ConfigModel) handleEditingKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.editingKeys.Cancel):
		m.state = StateBrowsing
		return m, nil

	case key.Matches(msg, m.editingKeys.Confirm):
		m.state = StateBrowsing
		return m, nil

	case key.Matches(msg, m.editingKeys.ForceQuit):
		// First ctrl-c during editing cancels the edit.
		m.state = StateBrowsing
		return m, nil
	}

	return m, nil
}

// handleErrorsKey processes key presses in the errors overlay state.
// up/down navigate the error list. enter jumps to the error's field.
// esc closes the overlay.
func (m *ConfigModel) handleErrorsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	action := m.errorOverlay.HandleKey(msg)
	switch action {
	case ErrorActionNone:
		// Navigation handled internally by the overlay.
	case ErrorActionClose:
		m.state = StateBrowsing
	case ErrorActionJump:
		selected := m.errorOverlay.SelectedError()
		m.activeTab = selected.Tab
		m.state = StateBrowsing
	}
	return m, nil
}

// startSave begins the save flow: validate, check external changes, then save.
// If quit is true, the TUI will quit after a successful save.
func (m *ConfigModel) startSave(quit bool) (tea.Model, tea.Cmd) {
	m.quitAfterSave = quit

	// Step 1: Validate.
	if m.accessor != nil {
		if errs := ValidateSave(m.accessor); len(errs) > 0 {
			m.ShowErrorOverlay(errs)
			return m, nil
		}
	}

	// Step 2: Check external changes.
	sf := NewSaveFlow(m.filePath, m.modTime, m.root, m.dirty, m.accessor, m.isNewFile)
	needsConfirm, err := sf.NeedsOverwriteConfirmation()
	if err != nil {
		m.writeErrorMsg = err.Error()
		m.state = StateWriteError
		return m, nil
	}
	if needsConfirm {
		m.state = StateOverwriteConfirm
		return m, nil
	}

	// Step 3: Execute save.
	m.state = StateSaving
	return m, executeSaveCmd(sf)
}

// handleSaveResult processes the result of a save operation.
func (m *ConfigModel) handleSaveResult(msg SaveResultMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		m.writeErrorMsg = msg.Err.Error()
		m.state = StateWriteError
		return m, nil
	}

	// Save succeeded.
	m.isNewFile = false

	// Update mod time from the written file.
	if info, err := os.Stat(m.filePath); err == nil {
		m.modTime = info.ModTime()
	}

	m.state = StateBrowsing

	flashMsg := "ok saved " + filepath.Base(m.filePath)
	cmd := m.SetFlash(flashMsg)

	if m.quitAfterSave {
		return m, tea.Batch(cmd, tea.Quit)
	}
	return m, cmd
}

// handleOverwriteConfirmKey processes key presses in the overwrite
// confirmation overlay. y/enter overwrites, n/esc cancels.
func (m *ConfigModel) handleOverwriteConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.confirmKeys.Confirm):
		// User confirmed overwrite -- execute save.
		sf := NewSaveFlow(m.filePath, m.modTime, m.root, m.dirty, m.accessor, m.isNewFile)
		m.state = StateSaving
		return m, executeSaveCmd(sf)

	case key.Matches(msg, m.confirmKeys.Deny), key.Matches(msg, m.confirmKeys.Cancel):
		m.quitAfterSave = false
		m.state = StateBrowsing
		return m, nil
	}

	return m, nil
}

// handleWriteErrorKey processes key presses in the write error overlay.
// esc closes the overlay and returns to browsing.
func (m *ConfigModel) handleWriteErrorKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "esc" {
		m.writeErrorMsg = ""
		m.state = StateBrowsing
	}
	return m, nil
}

// ShowErrorOverlay populates the error overlay with the given errors and
// switches to the errors state. If errors is empty, this is a no-op.
func (m *ConfigModel) ShowErrorOverlay(errs []ValidationError) {
	if len(errs) == 0 {
		return
	}
	m.errorOverlay.SetErrors(errs)
	m.state = StateErrors
	m.tabBadges = TabsWithErrors(errs)
}

// ClearValidationErrors clears all validation errors and badges.
func (m *ConfigModel) ClearValidationErrors() {
	m.errorOverlay.SetErrors(nil)
	m.tabBadges = [tabCount]bool{}
}

// UpdateTabBadges recalculates tab badges from the given errors.
func (m *ConfigModel) UpdateTabBadges(errs []ValidationError) {
	m.tabBadges = TabsWithErrors(errs)
}

// ErrorOverlay returns the error overlay for external inspection.
func (m *ConfigModel) ErrorOverlay() *ErrorOverlay {
	return m.errorOverlay
}

// nextTab advances to the next tab, wrapping around.
func (m *ConfigModel) nextTab() {
	m.activeTab = SectionTab((int(m.activeTab) + 1) % tabCount)
}

// prevTab moves to the previous tab, wrapping around.
func (m *ConfigModel) prevTab() {
	m.activeTab = SectionTab((int(m.activeTab) + tabCount - 1) % tabCount)
}

// View implements tea.Model. It renders the tab bar, section content area,
// and status bar. If the terminal is too small, it shows a size warning.
func (m *ConfigModel) View() string {
	if m.width < MinTermWidth || m.height < MinTermHeight {
		return m.viewTooSmall()
	}

	if m.state == StateHelp {
		return m.helpOverlay.View(m.width, m.height)
	}

	if m.state == StateErrors {
		return m.errorOverlay.View(m.width, m.height)
	}

	if m.state == StateOverwriteConfirm {
		return m.viewOverwriteConfirm()
	}

	if m.state == StateWriteError {
		return m.viewWriteError()
	}

	var b strings.Builder
	b.WriteString(m.viewTabBar())
	b.WriteString("\n")
	b.WriteString(m.viewContent())
	b.WriteString("\n")
	b.WriteString(m.viewStatusBar())
	return b.String()
}

// viewTooSmall renders the "terminal too small" message.
func (m *ConfigModel) viewTooSmall() string {
	msg := fmt.Sprintf(
		"Terminal too small (%dx%d). Minimum: %dx%d.",
		m.width, m.height, MinTermWidth, MinTermHeight,
	)
	style := lipgloss.NewStyle().
		Foreground(m.theme.ErrorAccent).
		Bold(true).
		Width(m.width).
		Align(lipgloss.Center)

	// Center vertically by padding with newlines.
	padding := ""
	if m.height > 1 {
		topPad := m.height / 2
		padding = strings.Repeat("\n", topPad)
	}
	return padding + style.Render(msg)
}

// viewTabBar renders the tab bar with active tab highlighting and error badges.
func (m *ConfigModel) viewTabBar() string {
	activeStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.theme.Accent).
		Padding(0, 1)

	inactiveStyle := lipgloss.NewStyle().
		Foreground(m.theme.Muted).
		Padding(0, 1)

	var tabs []string
	for i := range tabCount {
		label := tabLabels[i]

		// Add validation error badge.
		if m.tabBadges[i] {
			label += " (!)"
		}

		if SectionTab(i) == m.activeTab {
			label = "[" + label + "]"
			tabs = append(tabs, activeStyle.Render(label))
		} else {
			tabs = append(tabs, inactiveStyle.Render(label))
		}
	}

	bar := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)

	// Add a separator line under the tab bar.
	separator := lipgloss.NewStyle().
		Foreground(m.theme.Border).
		Width(m.width).
		Render(strings.Repeat("\u2500", m.width))

	return bar + "\n" + separator
}

// viewContent renders the main content area for the active tab.
// This is a placeholder that will be replaced by SectionModel rendering
// in a future ticket.
func (m *ConfigModel) viewContent() string {
	// Calculate available height: total height minus tab bar (2 lines) minus
	// status bar (2 lines: separator + content).
	contentHeight := max(m.height-4, 1)

	label := tabLabels[m.activeTab]
	placeholder := fmt.Sprintf("  %s section content", label)

	// Pad to fill the content area.
	lines := make([]string, contentHeight)
	lines[0] = placeholder
	for i := 1; i < contentHeight; i++ {
		lines[i] = ""
	}
	return strings.Join(lines, "\n")
}

// viewStatusBar renders the status bar with dirty count, file status,
// context-sensitive keybindings, and flash messages.
func (m *ConfigModel) viewStatusBar() string {
	separator := lipgloss.NewStyle().
		Foreground(m.theme.Border).
		Width(m.width).
		Render(strings.Repeat("\u2500", m.width))

	var parts []string

	// Dirty count or new file indicator.
	if m.isNewFile {
		newStyle := lipgloss.NewStyle().
			Foreground(m.theme.SuccessAccent).
			Bold(true)
		parts = append(parts, newStyle.Render("[new file]"))
	} else if count := m.dirty.DirtyCount(); count > 0 {
		dirtyStyle := lipgloss.NewStyle().
			Foreground(m.theme.Highlight).
			Bold(true)
		parts = append(parts, dirtyStyle.Render(fmt.Sprintf("[%d modified]", count)))
	}

	// Flash message takes priority over keybinding hints.
	if m.flashMessage != "" {
		flashStyle := lipgloss.NewStyle().
			Foreground(m.theme.SuccessAccent)
		parts = append(parts, flashStyle.Render(m.flashMessage))
	} else {
		// Context-sensitive keybinding hints.
		parts = append(parts, m.statusKeyHints())
	}

	content := lipgloss.NewStyle().
		Width(m.width).
		Render(strings.Join(parts, "  "))

	return separator + "\n" + content
}

// statusKeyHints returns the context-sensitive keybinding hints for the
// current state, showing 3-4 of the most relevant keys.
func (m *ConfigModel) statusKeyHints() string {
	hintStyle := lipgloss.NewStyle().Foreground(m.theme.Muted)
	keyStyle := lipgloss.NewStyle().Foreground(m.theme.Highlight)

	formatHint := func(b key.Binding) string {
		keys := b.Help().Key
		desc := b.Help().Desc
		return keyStyle.Render(keys) + " " + hintStyle.Render(desc)
	}

	sep := hintStyle.Render(" . ")

	switch m.state {
	case StateEditing:
		return formatHint(m.editingKeys.Confirm) + sep + formatHint(m.editingKeys.Cancel)

	default: // StateBrowsing
		// Build the edit verb based on focused field type.
		editVerb := EditHelpVerb(fieldTypeToString(m.focusedFieldType))
		editKey := keyStyle.Render("e") + " " + hintStyle.Render(editVerb)

		return strings.Join([]string{
			formatHint(m.browsingKeys.NextTab) +
				"/" + keyStyle.Render("S-tab") + " " + hintStyle.Render("section"),
			formatHint(m.browsingKeys.Up),
			editKey,
			formatHint(m.browsingKeys.Help),
			formatHint(m.browsingKeys.SaveQuit),
		}, sep)
	}
}

// viewOverwriteConfirm renders the overwrite confirmation overlay.
func (m *ConfigModel) viewOverwriteConfirm() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.theme.Highlight)

	bodyStyle := lipgloss.NewStyle().
		Foreground(m.theme.Muted)

	hintStyle := lipgloss.NewStyle().
		Foreground(m.theme.Accent)

	content := titleStyle.Render("File changed externally") + "\n\n" +
		bodyStyle.Render("The config file has been modified outside the editor.\nOverwrite with your changes?") + "\n\n" +
		hintStyle.Render("y") + bodyStyle.Render(" overwrite  ") +
		hintStyle.Render("n") + bodyStyle.Render("/") +
		hintStyle.Render("esc") + bodyStyle.Render(" cancel")

	innerWidth := max(m.width-4, 30)
	boxStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Highlight).
		Padding(1, 2).
		Width(innerWidth)

	box := boxStyle.Render(content)

	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(box)
}

// viewWriteError renders the write error overlay.
func (m *ConfigModel) viewWriteError() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.theme.ErrorAccent)

	bodyStyle := lipgloss.NewStyle().
		Foreground(m.theme.Muted)

	hintStyle := lipgloss.NewStyle().
		Foreground(m.theme.Accent)

	content := titleStyle.Render("Write Error") + "\n\n" +
		bodyStyle.Render(m.writeErrorMsg) + "\n\n" +
		hintStyle.Render("esc") + bodyStyle.Render(" close")

	innerWidth := max(m.width-4, 30)
	boxStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.ErrorAccent).
		Padding(1, 2).
		Width(innerWidth)

	box := boxStyle.Render(content)

	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(box)
}

// SetFlash sets a flash message on the status bar that auto-clears after 3s.
func (m *ConfigModel) SetFlash(msg string) tea.Cmd {
	m.flashMessage = msg
	return tea.Tick(3*time.Second, func(time.Time) tea.Msg {
		return flashClearMsg{}
	})
}

// SetTabBadge sets or clears the error badge on a tab.
func (m *ConfigModel) SetTabBadge(tab SectionTab, hasBadge bool) {
	if int(tab) >= 0 && int(tab) < tabCount {
		m.tabBadges[tab] = hasBadge
	}
}

// ActiveTab returns the currently active tab.
func (m *ConfigModel) ActiveTab() SectionTab {
	return m.activeTab
}

// State returns the current interaction state.
func (m *ConfigModel) State() ModelState {
	return m.state
}

// Width returns the current terminal width.
func (m *ConfigModel) Width() int {
	return m.width
}

// Height returns the current terminal height.
func (m *ConfigModel) Height() int {
	return m.height
}

// SetFocusedFieldType sets the field type of the currently focused field.
// This drives context-sensitive status bar rendering.
func (m *ConfigModel) SetFocusedFieldType(ft FieldType) {
	m.focusedFieldType = ft
}

// WriteErrorMsg returns the current write error message, if any.
func (m *ConfigModel) WriteErrorMsg() string {
	return m.writeErrorMsg
}

// IsNewFile reports whether the model is editing a new (not-yet-saved) file.
func (m *ConfigModel) IsNewFile() bool {
	return m.isNewFile
}

// ModTime returns the file modification time used for external change detection.
func (m *ConfigModel) ModTime() time.Time {
	return m.modTime
}

// SetModTime updates the file modification time.
func (m *ConfigModel) SetModTime(t time.Time) {
	m.modTime = t
}

// fieldTypeToString converts a FieldType to the string format expected by
// EditHelpVerb. This bridges the typed enum and the string-based verb lookup.
func fieldTypeToString(ft FieldType) string {
	switch ft {
	case String:
		return "string"
	case SensitiveString:
		return "sensitive_string"
	case Int:
		return "int"
	case Bool:
		return "bool"
	case StringList:
		return "string_list"
	case ObjectList:
		return "object_list"
	default:
		return "string"
	}
}

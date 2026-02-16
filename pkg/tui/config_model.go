package tui

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// PaneFocus represents which pane currently has keyboard focus.
type PaneFocus int

const (
	// SidebarFocused means the sidebar has keyboard focus.
	SidebarFocused PaneFocus = iota
	// ContentFocused means the content pane has keyboard focus.
	ContentFocused
)

// ConfigState holds all editable config values in memory.
// This state persists across navigation between sections.
type ConfigState struct {
	JiraFiltersType                string
	JiraMarkdownFilenamePattern    string
	JiraUsername                   string
	JiraAPIToken                   string
	JiraFiltersPriority            string
	WorktreesDir                   string
	DefaultBranch                  string
	JiraHost                       string
	JiraAttachmentsDir             string
	Worktrees                      []WorktreeEntryState
	FileCopyRules                  []FileCopyRuleState
	JiraFiltersStatus              []string
	JiraAttachmentsMaxSize         int
	JiraMarkdownIncludeAttachments bool
	JiraMarkdownUseRelativeLinks   bool
	JiraMarkdownIncludeComments    bool
	JiraAttachmentsEnabled         bool
	JiraEnabled                    bool
	dirty                          bool
}

// FileCopyRuleState represents a file copy rule in the config state.
type FileCopyRuleState struct {
	SourceWorktree string
	Files          []string
}

// WorktreeEntryState represents a worktree entry in the config state.
type WorktreeEntryState struct {
	Name        string
	Branch      string
	MergeInto   string
	Description string
}

// MarkDirty sets the dirty flag to true, indicating unsaved changes.
func (s *ConfigState) MarkDirty() {
	s.dirty = true
}

// ClearDirty sets the dirty flag to false, indicating no unsaved changes.
func (s *ConfigState) ClearDirty() {
	s.dirty = false
}

// IsDirty returns whether the state has unsaved changes.
func (s *ConfigState) IsDirty() bool {
	return s.dirty
}

// FormFactory creates forms for config sections.
// The factory receives the section name and returns a tea.Model for that section.
// It also receives a callback to update the shared state when the form saves.
type FormFactory func(section string, state *ConfigState, theme *Theme, onUpdate func()) tea.Model

// ConfigModelOption is a function that configures a ConfigModel.
type ConfigModelOption func(*ConfigModel)

// WithInitialState sets the initial config state.
func WithInitialState(state *ConfigState) ConfigModelOption {
	return func(m *ConfigModel) {
		if state != nil {
			m.state = state
		}
	}
}

// WithOnSave sets the callback for saving config.
func WithOnSave(fn func(*ConfigState) error) ConfigModelOption {
	return func(m *ConfigModel) {
		m.onSave = fn
	}
}

// WithOnReset sets the callback for reloading config from file.
func WithOnReset(fn func() (*ConfigState, error)) ConfigModelOption {
	return func(m *ConfigModel) {
		m.onReset = fn
	}
}

// WithFormFactory sets the factory for creating section forms.
func WithFormFactory(factory FormFactory) ConfigModelOption {
	return func(m *ConfigModel) {
		m.formFactory = factory
	}
}

// ConfigModel is the root model for the config TUI.
// It manages a two-pane layout with sidebar on the left and content form on the right.
// It holds the in-memory config state that sections can modify.
// Each pane uses a viewport for independent scrolling within terminal height.
type ConfigModel struct {
	sidebar            *Sidebar
	theme              *Theme
	state              *ConfigState
	onSave             func(*ConfigState) error
	onReset            func() (*ConfigState, error)
	formFactory        FormFactory
	helpOverlay        *HelpOverlay
	validationOverlay  *validationOverlay
	saveConfirmField   Field
	currentForm        tea.Model
	formCache          map[string]tea.Model
	saveError          string // error message from last failed save, cleared on next keypress
	sidebarViewport    viewport.Model
	contentViewport    viewport.Model
	saveConfirmContext SaveConfirmContext
	saveConfirmReturn  PaneFocus // pane to return to after save confirm dismissal
	paneFocus          PaneFocus
	width              int
	height             int
	ready              bool // true after first WindowSizeMsg
	showSaveConfirm    bool
}

// NewConfigModel creates a new ConfigModel with a Sidebar as the initial view.
func NewConfigModel(theme *Theme, opts ...ConfigModelOption) *ConfigModel {
	if theme == nil {
		theme = DefaultTheme()
	}

	sidebar := NewSidebar(theme)

	m := &ConfigModel{
		sidebar:   sidebar,
		theme:     theme,
		state:     &ConfigState{}, // Empty state by default
		formCache: make(map[string]tea.Model),
		paneFocus: SidebarFocused,
	}

	for _, opt := range opts {
		opt(m)
	}

	// Initialize the first form for the initial sidebar selection
	m.currentForm = m.getOrCreateForm(sidebar.FocusedSection())

	return m
}

// Init implements tea.Model.
func (m *ConfigModel) Init() tea.Cmd {
	cmds := []tea.Cmd{m.sidebar.Init()}
	if m.currentForm != nil {
		cmds = append(cmds, m.currentForm.Init())
	}
	// Focus the sidebar initially
	m.sidebar.Focus()
	return tea.Batch(cmds...)
}

// Update implements tea.Model.
func (m *ConfigModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.handleWindowSize(msg)

	case tea.KeyMsg:
		// Ctrl+C always quits, regardless of modal state or pane focus
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		// Handle validation overlay if showing (Esc/Enter only)
		if m.validationOverlay != nil {
			return m.handleValidationOverlay(msg)
		}
		// Handle save confirmation dialog if showing
		if m.showSaveConfirm {
			return m.handleSaveConfirmation(msg)
		}
		// Handle help overlay if showing
		if m.helpOverlay != nil {
			return m.handleHelpOverlay(msg)
		}
		return m.handleKeyMsg(msg)

	case SidebarSelectionChangedMsg:
		// Preview mode - update content without focus change
		m.currentForm = m.getOrCreateForm(msg.Section)
		return m, nil

	case SidebarSelectionMsg:
		// Enter pressed in sidebar - focus the content pane
		return m.focusContent()

	case BackBoundaryMsg:
		// Content requested to go back - focus the sidebar
		return m.focusSidebar()

	case ValidationOverlayDismissedMsg:
		// Validation overlay dismissed - return to invoking pane
		m.validationOverlay = nil
		return m, m.restoreFocusAfterOverlay()
	}

	// Delegate to the appropriate pane based on focus
	return m.delegateToFocusedPane(msg)
}

// handleWindowSize updates dimensions on terminal resize.
func (m *ConfigModel) handleWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height

	// Calculate pane widths and content height (reserve 1 line for footer)
	sidebarWidth, contentWidth := m.calculatePaneWidths()
	contentHeight := msg.Height - 1

	// Update sidebar dimensions
	m.sidebar.WithWidth(sidebarWidth).WithHeight(contentHeight)

	// Initialize or resize viewports
	if !m.ready {
		// First time - create viewports
		m.sidebarViewport = viewport.New(sidebarWidth, contentHeight)
		m.contentViewport = viewport.New(contentWidth, contentHeight)
		m.ready = true
	} else {
		// Resize existing viewports
		m.sidebarViewport.Width = sidebarWidth
		m.sidebarViewport.Height = contentHeight
		m.contentViewport.Width = contentWidth
		m.contentViewport.Height = contentHeight
	}

	// Propagate to current form if it exists
	var cmd tea.Cmd
	if m.currentForm != nil {
		contentSizeMsg := tea.WindowSizeMsg{Width: contentWidth, Height: contentHeight}
		m.currentForm, cmd = m.currentForm.Update(contentSizeMsg)
	}

	return m, cmd
}

// handleKeyMsg processes keyboard input.
func (m *ConfigModel) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Clear any transient error message on the next keypress
	m.saveError = ""

	// Ctrl+S triggers save from either pane
	if msg.String() == "ctrl+s" {
		return m.triggerSaveFlow(SaveContextSave)
	}

	if m.paneFocus == SidebarFocused {
		return m.handleSidebarKeys(msg)
	}

	return m.handleContentKeys(msg)
}

// handleSidebarKeys processes keyboard input when the sidebar pane has focus.
func (m *ConfigModel) handleSidebarKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		if m.IsDirty() {
			m.saveConfirmContext = SaveContextQuit
			m.saveConfirmReturn = SidebarFocused
			return m.showSaveConfirmDialog(SaveContextQuit)
		}
		return m, tea.Quit
	case "?":
		return m.showHelp()
	case "l", "right":
		return m.focusContent()
	case "r":
		return m.handleReset()
	}
	// Delegate to sidebar
	newSidebar, cmd := m.sidebar.Update(msg)
	if s, ok := newSidebar.(*Sidebar); ok {
		m.sidebar = s
	}
	return m, cmd
}

// handleContentKeys processes keyboard input when the content pane has focus.
func (m *ConfigModel) handleContentKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// All key events are delegated to the focused form.
	// Forms handle Esc (emit BackBoundaryMsg), navigation, and text input directly.
	switch msg.String() {
	case "pgup", "pgdown", "ctrl+u", "ctrl+d", "home", "end":
		// Scroll keys go to viewport
		var cmd tea.Cmd
		m.contentViewport, cmd = m.contentViewport.Update(msg)
		return m, cmd
	}

	return m.delegateToForm(msg)
}

// delegateToForm passes a key message to the current form.
func (m *ConfigModel) delegateToForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.currentForm == nil {
		return m, nil
	}

	newForm, cmd := m.currentForm.Update(msg)
	m.currentForm = newForm
	// Auto-scroll to keep focused field visible
	m.scrollToFocusedField()
	return m, cmd
}

// triggerSaveFlow flushes all forms, validates all sections, and either shows
// validation errors or the save confirmation dialog.
func (m *ConfigModel) triggerSaveFlow(ctx SaveConfirmContext) (tea.Model, tea.Cmd) {
	// Step 1: Flush all cached forms to ConfigState
	m.flushAllForms()

	// Step 2: Validate all sections
	errs := m.validateAllSections()
	if len(errs) > 0 {
		// Show validation error overlay (S5)
		return m.showValidationErrors(errs)
	}

	// Step 3: Show save confirmation with context
	m.saveConfirmContext = ctx
	m.saveConfirmReturn = m.paneFocus
	return m.showSaveConfirmDialog(ctx)
}

// flushAllForms copies current field values from all cached forms into ConfigState.
// Forms that have not been visited (not in cache) don't need flushing -- their
// data in ConfigState is already the initial values.
func (m *ConfigModel) flushAllForms() {
	if m.state == nil {
		return
	}
	for _, form := range m.formCache {
		if flusher, ok := form.(Flusher); ok {
			flusher.FlushToState(m.state)
		}
	}
}

// validateAllSections runs Validate() on all cached forms and aggregates errors.
// Returns a slice of human-readable error strings.
func (m *ConfigModel) validateAllSections() []string {
	var errs []string
	for _, form := range m.formCache {
		if validator, ok := form.(Validator); ok {
			errs = append(errs, validator.Validate()...)
		}
	}
	return errs
}

// showValidationErrors displays the validation error overlay.
func (m *ConfigModel) showValidationErrors(errs []string) (tea.Model, tea.Cmd) {
	m.validationOverlay = newValidationOverlay(errs, m.theme, m.width)
	return m, nil
}

// handleValidationOverlay processes input while the validation overlay is showing.
func (m *ConfigModel) handleValidationOverlay(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	cmd := m.validationOverlay.Update(msg)
	return m, cmd
}

// restoreFocusAfterOverlay returns a Cmd that restores focus to the appropriate
// pane after an overlay (validation errors, save confirmation) is dismissed.
func (m *ConfigModel) restoreFocusAfterOverlay() tea.Cmd {
	if m.paneFocus == ContentFocused {
		return m.refocusCurrentForm()
	}
	m.sidebar.Focus()
	return nil
}

// showSaveConfirmDialog displays the save confirmation dialog.
func (m *ConfigModel) showSaveConfirmDialog(ctx SaveConfirmContext) (tea.Model, tea.Cmd) {
	m.showSaveConfirm = true

	confirm := newSaveConfirm(m.theme)
	if ctx == SaveContextQuit {
		confirm.WithTitle("Save changes before quitting?")
	}
	m.saveConfirmField = confirm

	return m, m.saveConfirmField.Focus()
}

// handleSaveConfirmation processes input while the save confirmation dialog is showing.
func (m *ConfigModel) handleSaveConfirmation(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.showSaveConfirm = false
		m.saveConfirmField = nil
		return m, m.restoreFocusAfterOverlay()
	}

	newField, cmd := m.saveConfirmField.Update(msg)
	m.saveConfirmField = newField

	if !m.saveConfirmField.IsComplete() {
		return m, cmd
	}

	confirmed, _ := m.saveConfirmField.GetValue().(bool)
	m.showSaveConfirm = false
	m.saveConfirmField = nil

	if m.saveConfirmContext == SaveContextQuit {
		return m.handleSaveConfirmQuit(confirmed)
	}

	return m.handleSaveConfirmSave(confirmed)
}

// handleSaveConfirmSave handles the save confirmation result for SaveContextSave (Ctrl+S).
func (m *ConfigModel) handleSaveConfirmSave(confirmed bool) (tea.Model, tea.Cmd) {
	if confirmed {
		if err := m.handleSave(); err != nil {
			m.saveError = "Save failed: " + err.Error()
		}
	}
	return m, m.restoreFocusAfterOverlay()
}

// handleSaveConfirmQuit handles the save confirmation result for SaveContextQuit (q while dirty).
func (m *ConfigModel) handleSaveConfirmQuit(confirmed bool) (tea.Model, tea.Cmd) {
	if !confirmed {
		// "No" in quit context: discard changes and quit
		return m, tea.Quit
	}

	// "Yes" in quit context: save, then quit on success or stay with error on failure
	if err := m.handleSave(); err != nil {
		m.saveError = "Save failed: " + err.Error()
		return m, m.restoreFocusAfterOverlay()
	}
	return m, tea.Quit
}

// delegateToFocusedPane passes messages to the currently focused pane.
func (m *ConfigModel) delegateToFocusedPane(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle mouse events for viewport scrolling
	if _, ok := msg.(tea.MouseMsg); ok {
		if m.paneFocus == ContentFocused {
			var cmd tea.Cmd
			m.contentViewport, cmd = m.contentViewport.Update(msg)
			return m, cmd
		}
	}

	if m.paneFocus == SidebarFocused {
		newSidebar, cmd := m.sidebar.Update(msg)
		if s, ok := newSidebar.(*Sidebar); ok {
			m.sidebar = s
		}
		return m, cmd
	}

	if m.currentForm != nil {
		newForm, cmd := m.currentForm.Update(msg)
		m.currentForm = newForm
		return m, cmd
	}

	return m, nil
}

// focusContent moves focus from sidebar to content pane.
func (m *ConfigModel) focusContent() (tea.Model, tea.Cmd) {
	m.paneFocus = ContentFocused
	m.sidebar.Blur()

	var cmd tea.Cmd
	if m.currentForm != nil {
		if focusable, ok := m.currentForm.(interface{ Focus() tea.Cmd }); ok {
			cmd = focusable.Focus()
		}
	}
	return m, cmd
}

// refocusCurrentForm calls Focus() on the current form to restore cursor blink.
// Used after dismissing overlays (save confirmation, etc.) to ensure the cursor
// blink ticker is restarted.
func (m *ConfigModel) refocusCurrentForm() tea.Cmd {
	if m.currentForm != nil {
		if focusable, ok := m.currentForm.(interface{ Focus() tea.Cmd }); ok {
			return focusable.Focus()
		}
	}
	return nil
}

// focusSidebar moves focus from content pane to sidebar.
func (m *ConfigModel) focusSidebar() (tea.Model, tea.Cmd) {
	m.paneFocus = SidebarFocused

	if m.currentForm != nil {
		if blurrable, ok := m.currentForm.(interface{ Blur() tea.Cmd }); ok {
			blurrable.Blur()
		}
	}

	m.sidebar.Focus()
	return m, nil
}

// scrollToFocusedField adjusts the content viewport to keep the focused field visible.
// If the form implements FocusReporter, it uses the reported position.
func (m *ConfigModel) scrollToFocusedField() {
	if m.currentForm == nil || !m.ready {
		return
	}

	// Check if form implements FocusReporter interface
	reporter, ok := m.currentForm.(FocusReporter)
	if !ok {
		return
	}

	focusLine := reporter.FocusedYOffset()
	if focusLine < 0 {
		return
	}

	// Ensure the focused line is visible in the viewport
	// Add some padding (2 lines) so the focused field isn't at the very edge
	viewportHeight := m.contentViewport.Height
	currentOffset := m.contentViewport.YOffset
	padding := 2

	// If focused line is below the visible area, scroll down
	if focusLine >= currentOffset+viewportHeight-padding {
		m.contentViewport.SetYOffset(focusLine - viewportHeight + padding + 1)
	}

	// If focused line is above the visible area, scroll up
	if focusLine < currentOffset+padding {
		newOffset := max(focusLine-padding, 0)
		m.contentViewport.SetYOffset(newOffset)
	}
}

// getOrCreateForm returns the cached form for a section, or creates a new one.
func (m *ConfigModel) getOrCreateForm(section string) tea.Model {
	if m.formFactory == nil {
		return nil
	}

	// Check cache first
	if form, ok := m.formCache[section]; ok {
		return form
	}

	// Create new form
	onUpdate := func() {
		m.state.dirty = true
	}

	form := m.formFactory(section, m.state, m.theme, onUpdate)
	if form != nil {
		m.formCache[section] = form
	}

	return form
}

// calculatePaneWidths returns the widths for sidebar and content panes.
func (m *ConfigModel) calculatePaneWidths() (sidebarWidth, contentWidth int) {
	if m.width < 40 {
		// Very narrow terminal - give more to sidebar
		sidebarWidth = m.width / 2
	} else {
		// Normal terminal - ~25% for sidebar
		sidebarWidth = min(max(m.width/4, 20), 30)
	}
	contentWidth = m.width - sidebarWidth - 3 // Account for border
	return
}

// handleReset reloads config from file.
func (m *ConfigModel) handleReset() (tea.Model, tea.Cmd) {
	if m.onReset == nil {
		return m, nil
	}

	newState, err := m.onReset()
	if err != nil {
		// Could show error message, for now just ignore
		return m, nil
	}

	m.state = newState
	m.formCache = make(map[string]tea.Model)
	m.currentForm = m.getOrCreateForm(m.sidebar.FocusedSection())
	return m, nil
}

// handleSave saves the current state to file and clears the dirty flag.
// Returns the error from onSave so the caller can surface it to the user.
func (m *ConfigModel) handleSave() error {
	if m.onSave == nil || m.state == nil {
		return nil
	}

	err := m.onSave(m.state)
	if err != nil {
		return err
	}

	m.state.dirty = false
	return nil
}

// showHelp creates and shows the help overlay.
func (m *ConfigModel) showHelp() (tea.Model, tea.Cmd) {
	m.helpOverlay = NewHelpOverlay().
		WithTheme(m.theme).
		WithWidth(m.width).
		WithHeight(m.height)
	return m, nil
}

// handleHelpOverlay processes input while showing the help overlay.
func (m *ConfigModel) handleHelpOverlay(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "?", "enter":
		m.helpOverlay = nil
		return m, nil
	}
	return m, nil
}

// View implements tea.Model.
func (m *ConfigModel) View() string {
	// Show validation error overlay if active (full screen)
	if m.validationOverlay != nil {
		return m.validationOverlay.View()
	}

	// Show save confirmation dialog if active (full screen)
	if m.showSaveConfirm && m.saveConfirmField != nil {
		return m.saveConfirmField.View()
	}

	// Show help overlay if active (full screen)
	if m.helpOverlay != nil {
		return m.helpOverlay.View()
	}

	// Before first WindowSizeMsg, show loading state
	if !m.ready {
		return "Loading..."
	}

	// Calculate pane widths
	sidebarWidth, contentWidth := m.calculatePaneWidths()

	// Set sidebar content in viewport
	m.sidebarViewport.SetContent(m.sidebar.View())

	// Set content pane content in viewport
	contentView := ""
	if m.currentForm != nil {
		contentView = m.currentForm.View()
	} else {
		contentView = m.theme.Blurred.Description.Render("Select a section from the sidebar")
	}
	m.contentViewport.SetContent(contentView)

	// Create sidebar style with right border
	sidebarStyle := lipgloss.NewStyle().
		Width(sidebarWidth).
		BorderStyle(lipgloss.NormalBorder()).
		BorderRight(true).
		BorderForeground(m.theme.Border)

	// Create content style
	contentStyle := lipgloss.NewStyle().
		Width(contentWidth).
		PaddingLeft(1)

	// Join panes horizontally - viewports handle their own heights
	mainView := lipgloss.JoinHorizontal(lipgloss.Top,
		sidebarStyle.Render(m.sidebarViewport.View()),
		contentStyle.Render(m.contentViewport.View()))

	// Add footer
	footer := m.renderFooter()

	return mainView + "\n" + footer
}

// renderFooter renders the help footer with context-sensitive hints.
func (m *ConfigModel) renderFooter() string {
	// Show save error prominently if present
	if m.saveError != "" {
		return lipgloss.NewStyle().
			Foreground(m.theme.ErrorAccent).
			Bold(true).
			Render(m.saveError)
	}

	dirtyIndicator := ""
	if m.IsDirty() {
		dirtyIndicator = " [modified]"
	}

	var helpText string
	if m.paneFocus == SidebarFocused {
		helpText = "↑↓=navigate  l/→=enter  r=reset  Ctrl+S=save  ?=help  q=quit"
	} else {
		helpText = "Esc=back  Tab=next  PgUp/Dn=scroll  Ctrl+S=save"
	}

	return m.theme.Blurred.Description.Render(helpText + dirtyIndicator)
}

// GetSidebar returns the sidebar component.
func (m *ConfigModel) GetSidebar() *Sidebar {
	return m.sidebar
}

// GetPaneFocus returns which pane currently has focus.
func (m *ConfigModel) GetPaneFocus() PaneFocus {
	return m.paneFocus
}

// GetCurrentForm returns the current form being displayed.
func (m *ConfigModel) GetCurrentForm() tea.Model {
	return m.currentForm
}

// GetTheme returns the current theme.
func (m *ConfigModel) GetTheme() *Theme {
	return m.theme
}

// GetState returns the current config state.
func (m *ConfigModel) GetState() *ConfigState {
	return m.state
}

// SetState updates the config state.
func (m *ConfigModel) SetState(state *ConfigState) {
	m.state = state
}

// GetFormCache returns the form cache map.
func (m *ConfigModel) GetFormCache() map[string]tea.Model {
	return m.formCache
}

// IsDirty returns whether the config has unsaved changes.
func (m *ConfigModel) IsDirty() bool {
	return m.state != nil && m.state.dirty
}

// ShowSaveConfirm returns whether the save confirmation dialog is visible.
func (m *ConfigModel) ShowSaveConfirm() bool {
	return m.showSaveConfirm
}

// GetSaveError returns the current save error message, if any.
func (m *ConfigModel) GetSaveError() string {
	return m.saveError
}

// GetSaveConfirmContext returns the context that triggered the save confirmation.
func (m *ConfigModel) GetSaveConfirmContext() SaveConfirmContext {
	return m.saveConfirmContext
}

// GetValidationOverlay returns the current validation overlay, if any.
func (m *ConfigModel) GetValidationOverlay() *validationOverlay {
	return m.validationOverlay
}

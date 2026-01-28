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
	sidebar         *Sidebar
	theme           *Theme
	state           *ConfigState
	onSave          func(*ConfigState) error
	onReset         func() (*ConfigState, error)
	formFactory     FormFactory
	helpOverlay     *HelpOverlay
	currentForm     tea.Model
	formCache       map[string]tea.Model
	sidebarViewport viewport.Model
	contentViewport viewport.Model
	paneFocus       PaneFocus
	width           int
	height          int
	ready           bool // true after first WindowSizeMsg
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

	case configSavedMsg:
		// Form saved its data to state, focus sidebar
		return m.focusSidebar()
	}

	// Delegate to the appropriate pane based on focus
	return m.delegateToFocusedPane(msg)
}

// configSavedMsg signals that a form saved its data to state.
type configSavedMsg struct{}

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
	// Global keys work in both panes
	switch msg.String() {
	case "?":
		return m.showHelp()
	case "s":
		return m.handleSave()
	case "q", "ctrl+c":
		return m, tea.Quit
	}

	// Focus-specific navigation
	if m.paneFocus == SidebarFocused {
		switch msg.String() {
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

	// Content pane has focus
	switch msg.String() {
	case "h", "left":
		return m.focusSidebar()
	case "esc":
		return m.focusSidebar()
	case "pgup", "pgdown", "ctrl+u", "ctrl+d", "home", "end":
		// Scroll keys go to viewport
		var cmd tea.Cmd
		m.contentViewport, cmd = m.contentViewport.Update(msg)
		return m, cmd
	}

	// Delegate to current form
	if m.currentForm != nil {
		newForm, cmd := m.currentForm.Update(msg)
		m.currentForm = newForm
		// Auto-scroll to keep focused field visible
		m.scrollToFocusedField()
		return m, cmd
	}

	return m, nil
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
	return m, nil
}

// handleSave saves the current state to file.
func (m *ConfigModel) handleSave() (tea.Model, tea.Cmd) {
	if m.onSave == nil || m.state == nil {
		return m, nil
	}

	err := m.onSave(m.state)
	if err != nil {
		// Could show error message, for now just ignore
		return m, nil
	}

	m.state.dirty = false
	return m, nil
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
	dirtyIndicator := ""
	if m.IsDirty() {
		dirtyIndicator = " [modified]"
	}

	var helpText string
	if m.paneFocus == SidebarFocused {
		helpText = "↑↓=navigate  l/→=enter  s=save  r=reset  ?=help  q=quit"
	} else {
		helpText = "h/←/Esc=back  Tab=next  PgUp/Dn=scroll  s=save  ?=help  q=quit"
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

// IsDirty returns whether the config has unsaved changes.
func (m *ConfigModel) IsDirty() bool {
	return m.state != nil && m.state.dirty
}

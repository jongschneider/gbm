package tui

import (
	tea "github.com/charmbracelet/bubbletea"
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
// It manages navigation between Sidebar and section forms using Navigator.
// It holds the in-memory config state that sections can modify.
type ConfigModel struct {
	sidebar     *Sidebar
	nav         *Navigator
	theme       *Theme
	state       *ConfigState
	onSave      func(*ConfigState) error
	onReset     func() (*ConfigState, error)
	formFactory FormFactory
	helpOverlay *HelpOverlay
	width       int
	height      int
}

// NewConfigModel creates a new ConfigModel with a Sidebar as the initial view.
func NewConfigModel(theme *Theme, opts ...ConfigModelOption) *ConfigModel {
	if theme == nil {
		theme = DefaultTheme()
	}

	sidebar := NewSidebar(theme)
	navigator := NewNavigator(sidebar)

	m := &ConfigModel{
		sidebar: sidebar,
		nav:     navigator,
		theme:   theme,
		state:   &ConfigState{}, // Empty state by default
	}

	for _, opt := range opts {
		opt(m)
	}

	return m
}

// Init implements tea.Model.
func (m *ConfigModel) Init() tea.Cmd {
	return m.nav.Init()
}

// Update implements tea.Model.
func (m *ConfigModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.sidebar.WithWidth(msg.Width).WithHeight(msg.Height)
		// Propagate to navigator
		_, cmd := m.nav.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		// Handle help overlay if showing
		if m.helpOverlay != nil {
			return m.handleHelpOverlay(msg)
		}

		// Handle global keys when at sidebar level
		if m.nav.Depth() == 1 {
			switch msg.String() {
			case "?":
				return m.showHelp()
			case "r":
				return m.handleReset()
			case "s":
				return m.handleSave()
			case "q", "ctrl+c":
				return m, tea.Quit
			}
		}

	case SidebarSelectionMsg:
		return m.handleSidebarSelection(msg)

	case BackBoundaryMsg:
		return m.handleBackBoundary()

	case configSavedMsg:
		// Form saved its data to state, pop back to sidebar
		return m.handleBackBoundary()
	}

	// Delegate to navigator
	_, cmd := m.nav.Update(msg)
	return m, cmd
}

// configSavedMsg signals that a form saved its data to state.
type configSavedMsg struct{}

// handleSidebarSelection creates and pushes the appropriate form.
func (m *ConfigModel) handleSidebarSelection(msg SidebarSelectionMsg) (tea.Model, tea.Cmd) {
	if m.formFactory == nil {
		return m, nil
	}

	onUpdate := func() {
		m.state.dirty = true
	}

	form := m.formFactory(msg.Section, m.state, m.theme, onUpdate)
	if form == nil {
		return m, nil
	}

	m.nav.Push(form)
	return m, form.Init()
}

// handleBackBoundary pops the current form and returns to sidebar.
func (m *ConfigModel) handleBackBoundary() (tea.Model, tea.Cmd) {
	if m.nav.Depth() > 1 {
		m.nav.Pop()
		return m, m.nav.Current().Init()
	}
	return m, nil
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
	// Show help overlay if active
	if m.helpOverlay != nil {
		return m.helpOverlay.View()
	}

	view := m.nav.View()

	// Add help footer when at sidebar level
	if m.nav.Depth() == 1 {
		dirtyIndicator := ""
		if m.IsDirty() {
			dirtyIndicator = " [modified]"
		}
		help := m.theme.Blurred.Description.Render("↑↓=navigate  Enter=select  s=save  r=reset  ?=help  q=quit" + dirtyIndicator)
		view = view + "\n\n" + help
	}

	return view
}

// GetSidebar returns the sidebar component.
func (m *ConfigModel) GetSidebar() *Sidebar {
	return m.sidebar
}

// GetNavigator returns the navigator.
func (m *ConfigModel) GetNavigator() *Navigator {
	return m.nav
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

package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// ConfigModel is the root model for the config TUI.
// It manages navigation between Sidebar and section forms using Navigator.
// It holds the in-memory config state that sections can modify.
type ConfigModel struct {
	sidebar *Sidebar
	nav     *Navigator
	theme   *Theme
	width   int
	height  int
	// Config state will be added when persisting config loading is integrated
}

// NewConfigModel creates a new ConfigModel with a Sidebar as the initial view.
func NewConfigModel(theme *Theme) *ConfigModel {
	if theme == nil {
		theme = DefaultTheme()
	}

	sidebar := NewSidebar(theme)
	navigator := NewNavigator(sidebar)

	return &ConfigModel{
		sidebar: sidebar,
		nav:     navigator,
		theme:   theme,
	}
}

// Init implements tea.Model
func (m *ConfigModel) Init() tea.Cmd {
	return m.nav.Init()
}

// Update implements tea.Model
func (m *ConfigModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle window size
	if sizeMsg, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = sizeMsg.Width
		m.height = sizeMsg.Height
		m.sidebar.WithWidth(sizeMsg.Width).WithHeight(sizeMsg.Height)
	}

	// Delegate to navigator (it updates itself internally)
	_, cmd := m.nav.Update(msg)

	return m, cmd
}

// View implements tea.Model
func (m *ConfigModel) View() string {
	return m.nav.View()
}

// GetSidebar returns the sidebar component
func (m *ConfigModel) GetSidebar() *Sidebar {
	return m.sidebar
}

// GetNavigator returns the navigator
func (m *ConfigModel) GetNavigator() *Navigator {
	return m.nav
}

// GetTheme returns the current theme
func (m *ConfigModel) GetTheme() *Theme {
	return m.theme
}

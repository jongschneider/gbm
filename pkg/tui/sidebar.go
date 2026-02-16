// Package tui provides terminal user interface components.
package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// SidebarSection represents a top-level section in the config sidebar.
type SidebarSection struct {
	Name     string // "Basics", "JIRA", "FileCopy", "Worktrees"
	HasError bool   // Shows validation badge
}

// Sidebar manages navigation between config sections with visual indicators.
type Sidebar struct {
	theme      *Theme
	hasErrors  map[string]bool
	sections   []SidebarSection
	focusedIdx int
	width      int
	height     int
	focused    bool // Whether the sidebar has keyboard focus
}

// NewSidebar creates a new Sidebar with default sections.
func NewSidebar(theme *Theme) *Sidebar {
	if theme == nil {
		theme = DefaultTheme()
	}
	return &Sidebar{
		sections: []SidebarSection{
			{Name: "Basics"},
			{Name: "JIRA"},
			{Name: "FileCopy"},
			{Name: "Worktrees"},
		},
		focusedIdx: 0,
		theme:      theme,
		hasErrors:  make(map[string]bool),
		focused:    true, // Sidebar is focused by default
	}
}

// Init implements tea.Model.
func (s *Sidebar) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (s *Sidebar) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle j/k vim-style navigation
		if msg.Type == tea.KeyRunes && len(msg.Runes) == 1 {
			switch msg.Runes[0] {
			case 'k':
				return s.moveUp()
			case 'j':
				return s.moveDown()
			}
		}

		switch msg.Type { //nolint:exhaustive // Only handling relevant keys
		case tea.KeyUp:
			return s.moveUp()
		case tea.KeyDown:
			return s.moveDown()
		case tea.KeyEnter:
			// Emit message that this section was selected (for focus change)
			return s, func() tea.Msg {
				return SidebarSelectionMsg{Section: s.sections[s.focusedIdx].Name}
			}
		}
	case tea.WindowSizeMsg:
		s.width = msg.Width
		s.height = msg.Height
	case SetErrorMsg:
		// Update error state for a section
		s.hasErrors[msg.Section] = msg.HasError
		if sectionIdx := s.findSectionIndex(msg.Section); sectionIdx >= 0 {
			s.sections[sectionIdx].HasError = msg.HasError
		}
	}
	return s, nil
}

// View implements tea.Model.
func (s *Sidebar) View() string {
	var lines []string
	for i, section := range s.sections {
		line := s.renderSection(section, i == s.focusedIdx)
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

// renderSection renders a single section line.
func (s *Sidebar) renderSection(section SidebarSection, selected bool) string {
	label := "• " + section.Name

	// Add error badge if section has validation errors
	if section.HasError {
		label += " ⚠"
	}

	// Apply styling based on selection and focus state
	if selected {
		if s.focused {
			// Selected and sidebar has focus - use focused style
			return s.theme.Focused.Title.Render(label)
		}
		// Selected but sidebar doesn't have focus - use blurred selection style
		return s.theme.SidebarSelectedBlurred.Render(label)
	}
	return s.theme.Blurred.Title.Render(label)
}

// Focus gives the sidebar keyboard focus.
func (s *Sidebar) Focus() {
	s.focused = true
}

// Blur removes keyboard focus from the sidebar.
func (s *Sidebar) Blur() {
	s.focused = false
}

// IsFocused returns whether the sidebar has keyboard focus.
func (s *Sidebar) IsFocused() bool {
	return s.focused
}

// FocusedSection returns the currently focused section name.
func (s *Sidebar) FocusedSection() string {
	if s.focusedIdx < len(s.sections) {
		return s.sections[s.focusedIdx].Name
	}
	return ""
}

// SetError updates the error badge for a section.
func (s *Sidebar) SetError(sectionName string, hasError bool) {
	s.hasErrors[sectionName] = hasError
	if idx := s.findSectionIndex(sectionName); idx >= 0 {
		s.sections[idx].HasError = hasError
	}
}

// findSectionIndex returns the index of a section by name.
func (s *Sidebar) findSectionIndex(name string) int {
	for i, section := range s.sections {
		if section.Name == name {
			return i
		}
	}
	return -1
}

// moveUp moves focus to the previous section.
func (s *Sidebar) moveUp() (tea.Model, tea.Cmd) {
	if s.focusedIdx > 0 {
		s.focusedIdx--
		return s, func() tea.Msg {
			return SidebarSelectionChangedMsg{Section: s.sections[s.focusedIdx].Name}
		}
	}
	return s, nil
}

// moveDown moves focus to the next section.
func (s *Sidebar) moveDown() (tea.Model, tea.Cmd) {
	if s.focusedIdx < len(s.sections)-1 {
		s.focusedIdx++
		return s, func() tea.Msg {
			return SidebarSelectionChangedMsg{Section: s.sections[s.focusedIdx].Name}
		}
	}
	return s, nil
}

// WithWidth sets the sidebar width.
func (s *Sidebar) WithWidth(width int) *Sidebar {
	s.width = width
	return s
}

// WithHeight sets the sidebar height.
func (s *Sidebar) WithHeight(height int) *Sidebar {
	s.height = height
	return s
}

// SidebarSelectionMsg is sent when a section is selected (Enter key).
// This triggers focus change to the content pane.
type SidebarSelectionMsg struct {
	Section string
}

// SidebarSelectionChangedMsg is sent when the selection changes (up/down navigation).
// This is used for preview mode where content updates without focus change.
type SidebarSelectionChangedMsg struct {
	Section string
}

// SetErrorMsg updates error state for a section.
type SetErrorMsg struct {
	Section  string
	HasError bool
}

// NewSetErrorMsg creates a message to update error state.
func NewSetErrorMsg(section string, hasError bool) SetErrorMsg {
	return SetErrorMsg{Section: section, HasError: hasError}
}

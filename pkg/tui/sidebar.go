// Package tui provides terminal user interface components
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// SidebarSection represents a top-level section in the config sidebar
type SidebarSection struct {
	Name     string // "Basics", "JIRA", "FileCopy", "Worktrees"
	Expanded bool
	HasError bool // Shows validation badge
}

// Sidebar manages navigation between config sections with visual indicators
type Sidebar struct {
	sections   []SidebarSection
	focusedIdx int
	width      int
	height     int
	theme      *Theme
	hasErrors  map[string]bool // Track which sections have validation errors
}

// NewSidebar creates a new Sidebar with default sections
func NewSidebar(theme *Theme) *Sidebar {
	if theme == nil {
		theme = DefaultTheme()
	}
	return &Sidebar{
		sections: []SidebarSection{
			{Name: "Basics", Expanded: true, HasError: false},
			{Name: "JIRA", Expanded: false, HasError: false},
			{Name: "FileCopy", Expanded: false, HasError: false},
			{Name: "Worktrees", Expanded: false, HasError: false},
		},
		focusedIdx: 0,
		theme:      theme,
		hasErrors:  make(map[string]bool),
	}
}

// Init implements tea.Model
func (s *Sidebar) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (s *Sidebar) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyUp:
			if s.focusedIdx > 0 {
				s.focusedIdx--
			}
			return s, nil
		case tea.KeyDown:
			if s.focusedIdx < len(s.sections)-1 {
				s.focusedIdx++
			}
			return s, nil
		case tea.KeyLeft:
			// Collapse focused section
			s.sections[s.focusedIdx].Expanded = false
			return s, nil
		case tea.KeyRight:
			// Expand focused section
			s.sections[s.focusedIdx].Expanded = true
			return s, nil
		case tea.KeyEnter:
			// Emit message that this section was selected
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

// View implements tea.Model
func (s *Sidebar) View() string {
	var lines []string
	for i, section := range s.sections {
		line := s.renderSection(section, i == s.focusedIdx)
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

// renderSection renders a single section with expand/collapse indicator
func (s *Sidebar) renderSection(section SidebarSection, focused bool) string {
	indicator := "▸"
	if section.Expanded {
		indicator = "▾"
	}

	label := fmt.Sprintf("%s %s", indicator, section.Name)

	// Add error badge if section has validation errors
	if section.HasError {
		label = fmt.Sprintf("%s ⚠", label)
	}

	// Apply focus styling
	if focused {
		return s.theme.Focused.Title.Render(label)
	}
	return s.theme.Blurred.Title.Render(label)
}

// FocusedSection returns the currently focused section name
func (s *Sidebar) FocusedSection() string {
	if s.focusedIdx < len(s.sections) {
		return s.sections[s.focusedIdx].Name
	}
	return ""
}

// SetExpanded updates the expanded state of a section
func (s *Sidebar) SetExpanded(sectionName string, expanded bool) {
	if idx := s.findSectionIndex(sectionName); idx >= 0 {
		s.sections[idx].Expanded = expanded
	}
}

// IsExpanded returns whether a section is expanded
func (s *Sidebar) IsExpanded(sectionName string) bool {
	if idx := s.findSectionIndex(sectionName); idx >= 0 {
		return s.sections[idx].Expanded
	}
	return false
}

// SetError updates the error badge for a section
func (s *Sidebar) SetError(sectionName string, hasError bool) {
	s.hasErrors[sectionName] = hasError
	if idx := s.findSectionIndex(sectionName); idx >= 0 {
		s.sections[idx].HasError = hasError
	}
}

// findSectionIndex returns the index of a section by name
func (s *Sidebar) findSectionIndex(name string) int {
	for i, section := range s.sections {
		if section.Name == name {
			return i
		}
	}
	return -1
}

// WithWidth sets the sidebar width
func (s *Sidebar) WithWidth(width int) *Sidebar {
	s.width = width
	return s
}

// WithHeight sets the sidebar height
func (s *Sidebar) WithHeight(height int) *Sidebar {
	s.height = height
	return s
}

// SidebarSelectionMsg is sent when a section is selected
type SidebarSelectionMsg struct {
	Section string
}

// SetErrorMsg updates error state for a section
type SetErrorMsg struct {
	Section  string
	HasError bool
}

// NewSetErrorMsg creates a message to update error state
func NewSetErrorMsg(section string, hasError bool) SetErrorMsg {
	return SetErrorMsg{Section: section, HasError: hasError}
}

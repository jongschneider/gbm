package service

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// FilterableItem represents an item in the filterable list
type FilterableItem struct {
	Label string // Display text (e.g., "ABC-123: Fix the bug")
	Value string // Actual value (e.g., "ABC-123")
}

func (i FilterableItem) FilterValue() string { return i.Label }
func (i FilterableItem) Title() string       { return i.Label }
func (i FilterableItem) Description() string { return "" }

// FilterableSelectModel is a Bubble Tea model for a filterable select
type FilterableSelectModel struct {
	textInput    textinput.Model
	list         list.Model
	allItems     []FilterableItem
	filteredList []list.Item
	title        string
	description  string
	selected     string
	cancelled    bool
	width        int
	height       int
}

// NewFilterableSelect creates a new filterable select component
func NewFilterableSelect(title, description string, items []FilterableItem) FilterableSelectModel {
	ti := textinput.New()
	ti.Placeholder = "Type to filter or enter custom value..."
	ti.Focus()
	ti.CharLimit = 200
	ti.Width = 80

	// Convert FilterableItems to list.Items
	listItems := make([]list.Item, len(items))
	for i, item := range items {
		listItems[i] = item
	}

	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false
	delegate.SetHeight(1)

	l := list.New(listItems, delegate, 80, 10)
	l.Title = ""
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(false) // We handle filtering manually

	return FilterableSelectModel{
		textInput:    ti,
		list:         l,
		allItems:     items,
		filteredList: listItems,
		title:        title,
		description:  description,
		width:        80,
		height:       20,
	}
}

func (m FilterableSelectModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m FilterableSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			// Ctrl+C sets cancelled flag to signal "cancel entirely"
			m.cancelled = true
			return m, tea.Quit

		case "esc":
			// ESC quits without setting cancelled to signal "go back"
			return m, tea.Quit

		case "enter":
			// If list has items and one is selected, use that
			if len(m.filteredList) > 0 {
				selectedItem := m.list.SelectedItem()
				if selectedItem != nil {
					if item, ok := selectedItem.(FilterableItem); ok {
						m.selected = item.Value
						return m, tea.Quit
					}
				}
			}
			// Otherwise, use the text input value
			m.selected = strings.TrimSpace(m.textInput.Value())
			return m, tea.Quit

		case "down", "up":
			// If there are filtered items, navigate the list
			if len(m.filteredList) > 0 {
				m.list, cmd = m.list.Update(msg)
				cmds = append(cmds, cmd)
				return m, tea.Batch(cmds...)
			}

		default:
			// Update text input
			m.textInput, cmd = m.textInput.Update(msg)
			cmds = append(cmds, cmd)

			// Filter the list based on text input
			m.filterList()

			return m, tea.Batch(cmds...)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetWidth(msg.Width - 4)
		m.list.SetHeight(msg.Height - 10)
	}

	return m, tea.Batch(cmds...)
}

func (m *FilterableSelectModel) filterList() {
	query := strings.ToLower(strings.TrimSpace(m.textInput.Value()))

	if query == "" {
		// No filter, show all items
		m.filteredList = make([]list.Item, len(m.allItems))
		for i, item := range m.allItems {
			m.filteredList[i] = item
		}
	} else {
		// Filter items that match the query
		m.filteredList = []list.Item{}
		for _, item := range m.allItems {
			if strings.Contains(strings.ToLower(item.Label), query) ||
				strings.Contains(strings.ToLower(item.Value), query) {
				m.filteredList = append(m.filteredList, item)
			}
		}
	}

	// Update the list with filtered items
	m.list.SetItems(m.filteredList)
}

func (m FilterableSelectModel) View() string {
	var s strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	s.WriteString(titleStyle.Render(m.title))
	s.WriteString("\n")

	// Description
	if m.description != "" {
		descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
		s.WriteString(descStyle.Render(m.description))
		s.WriteString("\n")
	}

	s.WriteString("\n")

	// Text input
	s.WriteString(m.textInput.View())
	s.WriteString("\n\n")

	// List or "No matches" message
	if len(m.filteredList) == 0 {
		noMatchStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Italic(true)
		inputValue := strings.TrimSpace(m.textInput.Value())
		if inputValue != "" {
			s.WriteString(noMatchStyle.Render(fmt.Sprintf("No matches found. Press Enter to use: %q", inputValue)))
		} else {
			s.WriteString(noMatchStyle.Render("Start typing to filter tickets or enter a custom value"))
		}
	} else {
		s.WriteString(m.list.View())
	}

	s.WriteString("\n\n")

	// Help text
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	s.WriteString(helpStyle.Render("↑/↓: navigate • enter: select • esc: cancel"))

	return s.String()
}

// Run executes the filterable select and returns the selected value
func (m FilterableSelectModel) Run() (string, error) {
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return "", err
	}

	if finalModel, ok := finalModel.(FilterableSelectModel); ok {
		if finalModel.cancelled {
			return "", fmt.Errorf("cancelled")
		}
		return finalModel.selected, nil
	}

	return "", fmt.Errorf("unexpected model type")
}

// IsComplete implements StepModel interface
func (m FilterableSelectModel) IsComplete() bool {
	return m.selected != ""
}

// IsCancelled implements StepModel interface
func (m FilterableSelectModel) IsCancelled() bool {
	return m.cancelled
}

// GetSelected returns the selected value
func (m FilterableSelectModel) GetSelected() string {
	return m.selected
}

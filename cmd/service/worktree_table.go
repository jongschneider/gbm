package service

import (
	"fmt"
	"gbm/internal/git"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	baseStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240"))
)

type worktreeTableModel struct {
	table           table.Model
	worktrees       []git.Worktree
	trackedBranches map[string]bool
	svc             *Service
	confirmingDelete bool
	deleteTarget     string
	message          string
}

func newWorktreeTable(worktrees []git.Worktree, trackedBranches map[string]bool, svc *Service) worktreeTableModel {
	columns := []table.Column{
		{Title: "Name", Width: 30},
		{Title: "Branch", Width: 60},
		{Title: "Status", Width: 10},
	}

	rows := []table.Row{}
	for _, wt := range worktrees {
		status := "ad hoc"
		if trackedBranches[wt.Branch] {
			status = "tracked"
		}

		rows = append(rows, table.Row{
			wt.Name,
			wt.Branch,
			status,
		})
	}

	// Calculate appropriate height (show all rows, or 15 max)
	height := len(rows)
	if height > 15 {
		height = 15
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(height),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	return worktreeTableModel{
		table:           t,
		worktrees:       worktrees,
		trackedBranches: trackedBranches,
		svc:             svc,
	}
}

func (m worktreeTableModel) Init() tea.Cmd {
	return nil
}

func (m worktreeTableModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle confirmation state
		if m.confirmingDelete {
			switch msg.String() {
			case "y", "Y":
				// Delete the worktree
				_, err := m.svc.Git.RemoveWorktree(m.deleteTarget, false, false)
				if err != nil {
					m.message = fmt.Sprintf("Error: %v", err)
					m.confirmingDelete = false
					return m, nil
				}

				// Refresh worktrees list
				worktrees, err := m.svc.Git.ListWorktrees(false)
				if err != nil {
					m.message = fmt.Sprintf("Error refreshing: %v", err)
					m.confirmingDelete = false
					return m, nil
				}

				// Categorize worktrees: tracked, ad hoc (exclude bare)
				var trackedWorktrees []git.Worktree
				var adHocWorktrees []git.Worktree

				for _, wt := range worktrees {
					if wt.IsBare {
						// Skip bare repository
						continue
					} else if m.trackedBranches[wt.Branch] {
						trackedWorktrees = append(trackedWorktrees, wt)
					} else {
						adHocWorktrees = append(adHocWorktrees, wt)
					}
				}

				// Combine in priority order: tracked, ad hoc
				sortedWorktrees := make([]git.Worktree, 0, len(trackedWorktrees)+len(adHocWorktrees))
				sortedWorktrees = append(sortedWorktrees, trackedWorktrees...)
				sortedWorktrees = append(sortedWorktrees, adHocWorktrees...)

				// Save the deleted target before rebuilding
				deletedTarget := m.deleteTarget

				// Rebuild the table with updated worktrees
				m = newWorktreeTable(sortedWorktrees, m.trackedBranches, m.svc)
				m.message = fmt.Sprintf("Deleted worktree '%s'", deletedTarget)
				return m, nil

			case "n", "N", "esc":
				// Cancel deletion
				m.confirmingDelete = false
				m.message = ""
				return m, nil
			}
			return m, nil
		}

		// Normal mode key handling
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "d":
			// Get selected worktree
			cursor := m.table.Cursor()
			if cursor >= 0 && cursor < len(m.worktrees) {
				m.deleteTarget = m.worktrees[cursor].Name
				m.confirmingDelete = true
			}
			return m, nil
		}
	}

	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m worktreeTableModel) View() string {
	var output string

	// Show table
	output = baseStyle.Render(m.table.View())

	// Show confirmation prompt if in delete mode
	if m.confirmingDelete {
		confirmStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("208")).
			Bold(true)
		confirmMsg := fmt.Sprintf("\n\nDelete worktree '%s'? (y/n): ", m.deleteTarget)
		output += confirmStyle.Render(confirmMsg)
	} else {
		// Show help text
		helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
		help := "\n↑/↓: navigate • d: delete • q/esc: quit\n"

		// Show message if any
		if m.message != "" {
			messageStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("120"))
			output += messageStyle.Render("\n" + m.message)
		}

		output += helpStyle.Render(help)
	}

	return output
}

func runWorktreeTable(worktrees []git.Worktree, trackedBranches map[string]bool, svc *Service) error {
	m := newWorktreeTable(worktrees, trackedBranches, svc)
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running table: %w", err)
	}
	return nil
}

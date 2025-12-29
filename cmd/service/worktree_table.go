package service

import (
	"fmt"
	"gbm/internal/git"
	"os"
	"os/exec"
	"path/filepath"

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
	table            table.Model
	worktrees        []git.Worktree
	trackedBranches  map[string]bool
	gitStatuses      map[string]string // Maps worktree name to git status symbol
	svc              *Service
	currentWorktree  *git.Worktree // Track current worktree for state updates
	confirmingDelete bool
	deleteTarget     string
	message          string
	switchOutput     string // Output from switch command to print after exit
}

func newWorktreeTable(worktrees []git.Worktree, trackedBranches map[string]bool, currentWorktree *git.Worktree, svc *Service) worktreeTableModel {
	columns := []table.Column{
		{Title: "Name", Width: 30},
		{Title: "Branch", Width: 60},
		{Title: "Kind", Width: 10},
		{Title: "Git Status", Width: 12},
	}

	// Fetch git statuses for all worktrees (non-bare only)
	gitStatuses := make(map[string]string)
	for _, wt := range worktrees {
		if wt.IsBare {
			continue
		}

		status, err := svc.Git.GetBranchStatus(wt.Path)
		if err == nil {
			gitStatuses[wt.Name] = statusToSymbol(status)
		}
	}

	rows := []table.Row{}
	for _, wt := range worktrees {
		kind := "ad hoc"
		if trackedBranches[wt.Branch] {
			kind = "tracked"
		}

		// Add * indicator if this is the current worktree
		name := wt.Name
		if currentWorktree != nil && wt.Name == currentWorktree.Name {
			name = "* " + name
		}

		// Get git status for this worktree
		gitStatus := gitStatuses[wt.Name]
		if gitStatus == "" {
			gitStatus = "-"
		}

		rows = append(rows, table.Row{
			name,
			wt.Branch,
			kind,
			gitStatus,
		})
	}

	// Calculate appropriate height (show all rows, or 25 max)
	// Add 1 to account for header row
	height := len(rows) + 1
	if height > 26 {
		height = 26
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
		gitStatuses:     gitStatuses,
		currentWorktree: currentWorktree,
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

				// Save the deleted target before rebuilding
				deletedTarget := m.deleteTarget

				// Rebuild the table with updated worktrees
				m = *m.rebuildTable()
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
		case "l":
			// Pull selected worktree
			cursor := m.table.Cursor()
			if cursor >= 0 && cursor < len(m.worktrees) {
				targetWorktree := m.worktrees[cursor]
				m.message = fmt.Sprintf("Pulling worktree '%s'...", targetWorktree.Name)

				// Pull the worktree
				err := m.svc.Git.PullWorktree(targetWorktree.Path, false)
				if err != nil {
					m.message = fmt.Sprintf("Error pulling '%s': %v", targetWorktree.Name, err)
				} else {
					m.message = fmt.Sprintf("Successfully pulled '%s'", targetWorktree.Name)
					// Rebuild table to refresh git status
					m = *m.rebuildTable()
				}
			}
			return m, nil
		case "p":
			// Push selected worktree
			cursor := m.table.Cursor()
			if cursor >= 0 && cursor < len(m.worktrees) {
				targetWorktree := m.worktrees[cursor]
				m.message = fmt.Sprintf("Pushing worktree '%s'...", targetWorktree.Name)

				// Push the worktree
				err := m.svc.Git.PushWorktree(targetWorktree.Path, false)
				if err != nil {
					m.message = fmt.Sprintf("Error pushing '%s': %v", targetWorktree.Name, err)
				} else {
					m.message = fmt.Sprintf("Successfully pushed '%s'", targetWorktree.Name)
					// Rebuild table to refresh git status
					m = *m.rebuildTable()
				}
			}
			return m, nil
		case " ", "enter":
			// Switch to selected worktree by invoking gbm2 wt switch
			cursor := m.table.Cursor()
			if cursor >= 0 && cursor < len(m.worktrees) {
				targetWorktree := m.worktrees[cursor]

				// Execute gbm wt switch <name>
				cmd := exec.Command(os.Args[0], "wt", "switch", targetWorktree.Name)
				// Inherit environment variables (including GBM_SHELL_INTEGRATION)
				envVars := []string{"GBM_SHELL_INTEGRATION=1"}
				// Pass current worktree name via env var so subprocess knows where we're switching from
				if m.currentWorktree != nil {
					envVars = append(envVars, fmt.Sprintf("GBM_CURRENT_WORKTREE=%s", m.currentWorktree.Name))
				}
				cmd.Env = append(cmd.Environ(), envVars...)
				output, err := cmd.CombinedOutput()
				if err != nil {
					m.message = fmt.Sprintf("Error switching: %v", err)
					return m, nil
				}

				// Store the output to print after the program exits
				m.switchOutput = string(output)
				return m, tea.Quit
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
		help := "\n↑/↓: navigate • space/enter: switch • l: pull • p: push • d: delete • q/esc: quit\n"

		// Show message if any
		if m.message != "" {
			messageStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("120"))
			output += messageStyle.Render("\n" + m.message)
		}

		output += helpStyle.Render(help)
	}

	return output
}

// rebuildTable refreshes the worktree list and rebuilds the entire table UI
func (m *worktreeTableModel) rebuildTable() *worktreeTableModel {
	// Refresh worktrees list
	worktrees, err := m.svc.Git.ListWorktrees(false)
	if err != nil {
		// If refresh fails, keep existing worktrees but show error
		m.message = fmt.Sprintf("Error refreshing worktrees: %v", err)
		return m
	}

	// Categorize worktrees: tracked, ad hoc (exclude bare)
	var trackedWorktrees []git.Worktree
	var adHocWorktrees []git.Worktree

	for _, wt := range worktrees {
		if wt.IsBare {
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

	// Rebuild table with fresh data
	newModel := newWorktreeTable(sortedWorktrees, m.trackedBranches, m.currentWorktree, m.svc)
	// Preserve the message from the caller
	newModel.message = m.message
	return &newModel
}

// statusToSymbol converts a BranchTrackingStatus to a display symbol
func statusToSymbol(status *git.BranchTrackingStatus) string {
	if status == nil || !status.Tracked {
		return "-" // No upstream
	}

	if status.AheadCount > 0 && status.BehindCount > 0 {
		return "⇄" // Diverged
	} else if status.AheadCount > 0 {
		return "⇢" // Ahead
	} else if status.BehindCount > 0 {
		return "⇠" // Behind
	}

	return "=" // Up-to-date
}

func runWorktreeTable(worktrees []git.Worktree, trackedBranches map[string]bool, currentWorktree *git.Worktree, svc *Service) error {
	m := newWorktreeTable(worktrees, trackedBranches, currentWorktree, svc)
	p := tea.NewProgram(m, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("error running table: %w", err)
	}

	// Write switch output to a temp file for shell integration to read
	if model, ok := finalModel.(worktreeTableModel); ok {
		if model.switchOutput != "" {
			// Write cd command to a temp file that shell integration can read
			// Use PPID (parent process ID, i.e., the shell's PID) for the filename
			ppid := os.Getppid()
			tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf(".gbm-switch-%d", ppid))

			// Clean up any stale temp file from previous run
			_ = os.Remove(tmpFile)

			// Write the switch output
			if err := os.WriteFile(tmpFile, []byte(model.switchOutput), 0600); err != nil {
				return fmt.Errorf("failed to write switch file: %w", err)
			}
			// Note: The shell integration is responsible for cleaning up this file after reading

			// Also print it for non-shell-integration users
			fmt.Print(model.switchOutput)
		}
	}

	return nil
}

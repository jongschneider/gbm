package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"gbm/internal/git"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

type worktreeTableModel struct {
	table            table.Model
	worktrees        []git.Worktree
	trackedBranches  map[string]bool
	svc              *Service
	currentWorktree  *git.Worktree // Track current worktree for state updates
	confirmingDelete bool
	deleteTarget     string
	message          string
	switchOutput     string // Output from switch command to print after exit
	branchStatuses   map[string]*git.BranchStatus
}

func newWorktreeTable(worktrees []git.Worktree, trackedBranches map[string]bool, currentWorktree *git.Worktree, svc *Service) worktreeTableModel {
	columns := []table.Column{
		{Title: "Name", Width: 30},
		{Title: "Branch", Width: 50},
		{Title: "Kind", Width: 10},
		{Title: "Git Status", Width: 15},
	}

	// Fetch all worktrees once at repo level for efficiency
	repoRoot, _ := svc.Git.FindGitRoot(".")
	if repoRoot != "" {
		cmd := exec.Command("git", "-C", repoRoot, "fetch", "--all", "--quiet")
		_ = cmd.Run() // Ignore errors, continue with stale info if fetch fails
	}

	// Fetch branch statuses for all worktrees concurrently
	branchStatuses := make(map[string]*git.BranchStatus)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, wt := range worktrees {
		if wt.IsBare {
			continue
		}

		wg.Add(1)
		go func(worktree git.Worktree) {
			defer wg.Done()
			status, err := svc.Git.GetBranchStatus(worktree.Path)
			if err == nil && status != nil {
				mu.Lock()
				branchStatuses[worktree.Name] = status
				mu.Unlock()
			}
		}(wt)

	}
	wg.Wait()

	rows := []table.Row{}
	for _, wt := range worktrees {
		kind := "ad hoc"
		if trackedBranches[wt.Branch] {
			kind = "tracked"
		}

		// Get git status symbol
		gitStatus := "—"
		if status, ok := branchStatuses[wt.Name]; ok && status != nil {
			if status.NoRemote {
				gitStatus = "?"
			} else if status.UpToDate {
				gitStatus = "✓"
			} else if status.Ahead > 0 && status.Behind > 0 {
				gitStatus = fmt.Sprintf("↕ %d↑%d↓", status.Ahead, status.Behind)
			} else if status.Ahead > 0 {
				gitStatus = fmt.Sprintf("↑ %d", status.Ahead)
			} else if status.Behind > 0 {
				gitStatus = fmt.Sprintf("↓ %d", status.Behind)
			}
		}

		// Add * indicator if this is the current worktree
		name := wt.Name
		if currentWorktree != nil && wt.Name == currentWorktree.Name {
			name = "* " + name
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
	height := min(len(rows)+1, 26)

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
		currentWorktree: currentWorktree,
		svc:             svc,
		branchStatuses:  branchStatuses,
	}
}

func (m worktreeTableModel) Init() tea.Cmd {
	return nil
}

// refreshTable re-fetches worktrees and rebuilds the table with fresh data
func (m worktreeTableModel) refreshTable() worktreeTableModel {
	// Refresh worktrees list
	worktrees, err := m.svc.Git.ListWorktrees(false)
	if err != nil {
		m.message = fmt.Sprintf("Error refreshing: %v", err)
		return m
	}

	// Categorize worktrees: current first (if found), then tracked, then ad hoc (exclude bare)
	var sortedWorktrees []git.Worktree
	var trackedWorktrees []git.Worktree
	var adHocWorktrees []git.Worktree

	for _, wt := range worktrees {
		if wt.IsBare {
			// Skip bare repository
			continue
		}
		if m.currentWorktree != nil && wt.Name == m.currentWorktree.Name {
			// Add current worktree first and skip categorization
			sortedWorktrees = append(sortedWorktrees, wt)
			continue
		}

		if m.trackedBranches[wt.Branch] {
			trackedWorktrees = append(trackedWorktrees, wt)
			continue
		}
		adHocWorktrees = append(adHocWorktrees, wt)
	}

	// Append categorized worktrees in priority order: tracked, ad hoc
	sortedWorktrees = append(sortedWorktrees, trackedWorktrees...)
	sortedWorktrees = append(sortedWorktrees, adHocWorktrees...)

	// Rebuild the table with updated worktrees (will fetch branch statuses again)
	return newWorktreeTable(sortedWorktrees, m.trackedBranches, m.currentWorktree, m.svc)
}

// refreshWorktreeStatus updates the git status for a specific worktree (by index)
// This is more performant than refreshTable() when only one worktree changed
func (m worktreeTableModel) refreshWorktreeStatus(index int) worktreeTableModel {
	if index < 0 || index >= len(m.worktrees) {
		return m
	}

	wt := m.worktrees[index]

	// Fetch only this worktree's remote tracking branch
	cmd := exec.Command("git", "-C", wt.Path, "fetch", "origin", wt.Branch, "--quiet")
	_ = cmd.Run() // Ignore errors, continue with stale info if fetch fails

	// Get updated branch status for just this worktree
	status, err := m.svc.Git.GetBranchStatus(wt.Path)
	if err == nil && status != nil {
		m.branchStatuses[wt.Name] = status
	}

	// Regenerate the git status string
	gitStatus := "—"
	if status != nil {
		if status.NoRemote {
			gitStatus = "?"
		} else if status.UpToDate {
			gitStatus = "✓"
		} else if status.Ahead > 0 && status.Behind > 0 {
			gitStatus = fmt.Sprintf("↕ %d↑%d↓", status.Ahead, status.Behind)
		} else if status.Ahead > 0 {
			gitStatus = fmt.Sprintf("↑ %d", status.Ahead)
		} else if status.Behind > 0 {
			gitStatus = fmt.Sprintf("↓ %d", status.Behind)
		}
	}

	// Update just this row in the table
	rows := m.table.Rows()
	if index < len(rows) {
		// Keep name, branch, kind columns the same, update only git status
		rows[index][3] = gitStatus
		m.table.SetRows(rows)
	}

	return m
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

				// Refresh the table with updated data
				m = m.refreshTable()
				m.message = fmt.Sprintf("Deleted worktree '%s'", deletedTarget)
				m.confirmingDelete = false
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
				targetName := targetWorktree.Name
				m.message = fmt.Sprintf("Pulling worktree '%s'...", targetName)

				// Pull the worktree
				err := m.svc.Git.PullWorktree(targetWorktree.Path, false)
				if err != nil {
					m.message = fmt.Sprintf("Error pulling '%s': %v", targetName, err)
				} else {
					// Refresh only this worktree's status for performance
					m = m.refreshWorktreeStatus(cursor)
					m.message = fmt.Sprintf("Successfully pulled '%s'", targetName)
				}
			}
			return m, nil
		case "p":
			// Push selected worktree
			cursor := m.table.Cursor()
			if cursor >= 0 && cursor < len(m.worktrees) {
				targetWorktree := m.worktrees[cursor]
				targetName := targetWorktree.Name
				m.message = fmt.Sprintf("Pushing worktree '%s'...", targetName)

				// Push the worktree
				err := m.svc.Git.PushWorktree(targetWorktree.Path, false)
				if err != nil {
					m.message = fmt.Sprintf("Error pushing '%s': %v", targetName, err)
				} else {
					// Refresh only this worktree's status for performance
					m = m.refreshWorktreeStatus(cursor)
					m.message = fmt.Sprintf("Successfully pushed '%s'", targetName)
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
			if err := os.WriteFile(tmpFile, []byte(model.switchOutput), 0o600); err != nil {
				return fmt.Errorf("failed to write switch file: %w", err)
			}
			// Note: The shell integration is responsible for cleaning up this file after reading

			// Also print it for non-shell-integration users
			fmt.Print(model.switchOutput)
		}
	}

	return nil
}

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
	"github.com/muesli/termenv"
)

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

// FormatGitStatus formats a BranchStatus into a display string for the table.
// Returns: ✓ (up to date), ↑ N (ahead), ↓ N (behind), ↕ N↑M↓ (diverged), ? (no remote), — (unknown).
func FormatGitStatus(status *git.BranchStatus) string {
	if status == nil {
		return "—"
	}
	if status.NoRemote {
		return "?"
	}
	if status.UpToDate {
		return "✓"
	}
	if status.Ahead > 0 && status.Behind > 0 {
		return fmt.Sprintf("↕ %d↑%d↓", status.Ahead, status.Behind)
	}
	if status.Ahead > 0 {
		return fmt.Sprintf("↑ %d", status.Ahead)
	}
	if status.Behind > 0 {
		return fmt.Sprintf("↓ %d", status.Behind)
	}
	return "—"
}

// FormatWorktreeName formats a worktree name with a * prefix if it's the current worktree.
func FormatWorktreeName(wt git.Worktree, currentWorktree *git.Worktree) string {
	if currentWorktree != nil && wt.Name == currentWorktree.Name {
		return "* " + wt.Name
	}
	return wt.Name
}

// FormatWorktreeKind returns "tracked" or "ad hoc" based on whether the branch is tracked.
func FormatWorktreeKind(wt git.Worktree, trackedBranches map[string]bool) string {
	if trackedBranches[wt.Branch] {
		return "tracked"
	}
	return "ad hoc"
}

// BuildWorktreeRow creates a table row for a worktree using the shared formatting helpers.
func BuildWorktreeRow(wt git.Worktree, currentWorktree *git.Worktree, trackedBranches map[string]bool, status *git.BranchStatus) table.Row {
	return table.Row{
		FormatWorktreeName(wt, currentWorktree),
		wt.Branch,
		FormatWorktreeKind(wt, trackedBranches),
		FormatGitStatus(status),
	}
}

// CalculateTableColumns returns responsive column widths based on terminal width.
// Column ratios: Name 25% (min 15), Branch 45% (min 20), Kind 10% (min 8), Status 20% (min 10).
func CalculateTableColumns(terminalWidth int) []table.Column {
	// Account for table borders (~4 chars), minimum 60 chars
	availableWidth := max(terminalWidth-4, 60)

	nameWidth := max(availableWidth*25/100, 15)
	branchWidth := max(availableWidth*45/100, 20)
	kindWidth := max(availableWidth*10/100, 8)
	statusWidth := max(availableWidth*20/100, 10)

	return []table.Column{
		{Title: "Name", Width: nameWidth},
		{Title: "Branch", Width: branchWidth},
		{Title: "Kind", Width: kindWidth},
		{Title: "Git Status", Width: statusWidth},
	}
}

// CalculateTableHeight returns the table height based on terminal height and row count.
// Leaves room for help text and messages (4 lines).
func CalculateTableHeight(terminalHeight, rowCount int) int {
	// Reserve space for help/status (4 lines) plus header (1 line), minimum 5
	maxHeight := max(terminalHeight-5, 5)
	// Show all rows up to max, plus 1 for header
	return min(rowCount+1, maxHeight)
}

// SortWorktrees sorts worktrees by priority: current first, then tracked, then ad hoc.
// Bare worktrees are excluded from the result.
func SortWorktrees(worktrees []git.Worktree, currentWorktree *git.Worktree, trackedBranches map[string]bool) []git.Worktree {
	var sorted []git.Worktree
	var tracked []git.Worktree
	var adHoc []git.Worktree

	for _, wt := range worktrees {
		if wt.IsBare {
			continue
		}
		if currentWorktree != nil && wt.Name == currentWorktree.Name {
			sorted = append(sorted, wt)
			continue
		}
		if trackedBranches[wt.Branch] {
			tracked = append(tracked, wt)
			continue
		}
		adHoc = append(adHoc, wt)
	}

	sorted = append(sorted, tracked...)
	sorted = append(sorted, adHoc...)
	return sorted
}

// DefaultTableStyles returns the standard table styling.
func DefaultTableStyles() table.Styles {
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
	return s
}

type worktreeTableModel struct {
	table            table.Model
	worktrees        []git.Worktree
	trackedBranches  map[string]bool
	svc              *Service
	currentWorktree  *git.Worktree // Track current worktree for state updates
	confirmingDelete bool
	deleteTarget     string
	message          string
	switchOutput     string // Worktree path to output after TUI exit
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

	// Build rows using shared helpers
	rows := make([]table.Row, 0, len(worktrees))
	for _, wt := range worktrees {
		status := branchStatuses[wt.Name]
		rows = append(rows, BuildWorktreeRow(wt, currentWorktree, trackedBranches, status))
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
	t.SetStyles(DefaultTableStyles())

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

	// Sort worktrees: current first, then tracked, then ad hoc (excludes bare)
	sortedWorktrees := SortWorktrees(worktrees, m.currentWorktree, m.trackedBranches)

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

	// Regenerate the git status string using shared helper
	gitStatus := FormatGitStatus(status)

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

				// Check if this is a tracked worktree
				if m.trackedBranches[targetWorktree.Branch] {
					m.message = fmt.Sprintf("Cannot push tracked worktree '%s' - tracked worktrees should not be pushed", targetWorktree.Name)
					return m, nil
				}

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
			// Store selected worktree path for output after TUI exits
			cursor := m.table.Cursor()
			if cursor >= 0 && cursor < len(m.worktrees) {
				targetWorktree := m.worktrees[cursor]
				// Store just the path (not "cd <path>")
				m.switchOutput = targetWorktree.Path
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
		// Show help text (conditionally show push option)
		helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

		// Check if currently selected worktree is tracked
		cursor := m.table.Cursor()
		showPush := true
		if cursor >= 0 && cursor < len(m.worktrees) {
			selectedWorktree := m.worktrees[cursor]
			if m.trackedBranches[selectedWorktree.Branch] {
				showPush = false
			}
		}

		help := "\n↑/↓: navigate • space/enter: switch • l: pull"
		if showPush {
			help += " • p: push"
		}
		help += " • d: delete • q/esc: quit\n"

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
	// Open /dev/tty for TUI rendering
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("failed to open /dev/tty: %w (TUI requires an interactive terminal)", err)
	}
	defer func() {
		_ = tty.Close()
	}()

	// Explicitly detect and set color profile for the tty output
	// This ensures lipgloss renders with colors even though we're using a custom file handle
	// IMPORTANT: Must be set BEFORE creating the table model so styles are applied correctly
	renderer := lipgloss.NewRenderer(tty,
		termenv.WithColorCache(true),
		termenv.WithTTY(true),
		termenv.WithProfile(termenv.TrueColor),
	)
	lipgloss.SetDefaultRenderer(renderer)

	// Create table model AFTER setting the renderer
	m := newWorktreeTable(worktrees, trackedBranches, currentWorktree, svc)

	// TUI renders to /dev/tty, leaving stdout clean
	// Use WithInput/WithOutput to explicitly direct I/O to /dev/tty
	p := tea.NewProgram(m,
		tea.WithInput(tty),
		tea.WithOutput(tty),
		tea.WithAltScreen(),
	)
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("error running table: %w", err)
	}

	// Output path to stdout if user selected a worktree (universal pattern)
	if model, ok := finalModel.(worktreeTableModel); ok {
		if model.switchOutput != "" {
			// Always output path to stdout (machine-readable)
			fmt.Println(model.switchOutput)

			// Always output message to stderr (human-readable)
			fmt.Fprintf(os.Stderr, "✓ Selected worktree: %s\n", filepath.Base(model.switchOutput))
		}
	}

	return nil
}

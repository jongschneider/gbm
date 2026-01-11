package service

import (
	"fmt"
	"io"
	"os"
	"time"

	"gbm/pkg/tui/async"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

// testlsModel manages the async table display for testing the Table component.
type testlsModel struct {
	table           table.Model
	worktrees       []mockWorktree
	trackedBranches map[string]bool
	delay           time.Duration
	// Async cell tracking
	asyncStatuses   map[int]*async.Cell[string] // Row index -> async cell for git status
	asyncOperations map[int]*async.Cell[string] // Row index -> async cell for pull/push/delete operations
	operationStates map[int]operationState      // Row index -> current operation state
	messageDump     io.Writer                   // Debug: dump all messages
}

type operationState struct {
	operation string    //nolint:unused // used in future steps - "pull", "push", "delete", ""
	result    string    //nolint:unused // used in future steps - Result message; empty = not started/cleared
	clearAt   time.Time //nolint:unused // used in future steps - When to clear result (after 2 seconds)
}

type mockWorktree struct {
	Name   string
	Branch string
	Path   string
}

// Init initializes the table model and starts async loads.
func (m *testlsModel) Init() tea.Cmd {
	cmds := []tea.Cmd{m.tickCmd()}

	// Set initial column widths
	m.updateColumns(m.table.Width())

	// Set initial rows
	m.updateTableRows()

	// Start async git status loads for each worktree
	for i := range m.worktrees {
		rowIdx := i
		// Create async cell for git status
		eval := async.New(func() (string, error) {
			mockGitService := &MockTableGitService{delay: m.delay}
			status, err := mockGitService.GetBranchStatus(m.worktrees[rowIdx].Path)
			if err != nil {
				return "error", err
			}
			return status, nil
		})

		cell := async.NewCell(eval)
		m.asyncStatuses[rowIdx] = cell

		// Create placeholder async cell for operations (not started yet)
		opEval := async.New(func() (string, error) {
			return "", nil
		})
		opCell := async.NewCell(opEval)
		m.asyncOperations[rowIdx] = opCell

		// Start loading
		cmd := cell.StartLoading()
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return tea.Batch(cmds...)
}

// tickCmd returns a command that sends tick messages for animation
func (m *testlsModel) tickCmd() tea.Cmd {
	return tea.Every(time.Millisecond*100, func(t time.Time) tea.Msg {
		return tickMsg{}
	})
}

// clearOperationCmd returns a command that clears an operation after the specified delay
func clearOperationCmd(rowIdx int, delay time.Duration) tea.Cmd {
	return tea.Tick(delay, func(t time.Time) tea.Msg {
		return clearOperationMsg{rowIdx: rowIdx}
	})
}

type tickMsg struct{}

type operationTriggeredMsg struct { //nolint:unused // used in future steps
	rowIdx    int
	operation string
}

type clearOperationMsg struct {
	rowIdx int
}

// Update handles input and state changes.
func (m *testlsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Dump message for debugging if messageDump is set
	if m.messageDump != nil {
		_, _ = fmt.Fprintf(m.messageDump, "[%s] %T\n", time.Now().Format("15:04:05.000"), msg)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.table.SetWidth(msg.Width)
		m.table.SetHeight(msg.Height - 4)
		m.updateColumns(msg.Width)

	case async.CellLoadedMsg:
		// A cell finished loading - update operation state with result if it's an operation cell
		// Check all operation cells to find which one loaded
		for rowIdx, cell := range m.asyncOperations {
			if cell.IsLoaded() {
				if opState, ok := m.operationStates[rowIdx]; ok && opState.operation != "" && opState.result == "" {
					// This is an operation cell that just finished - update with result
					opState.result = cell.View()
					opState.clearAt = time.Now().Add(2 * time.Second)
					m.operationStates[rowIdx] = opState
					
					// Handle delete: remove worktree from shared state
					if opState.operation == "delete" && rowIdx < len(m.worktrees) {
						m.worktrees = append(m.worktrees[:rowIdx], m.worktrees[rowIdx+1:]...)
						
						// Clear operation state after delete since row is gone
						delete(m.operationStates, rowIdx)
						delete(m.asyncOperations, rowIdx)
						
						// Reindex all maps for rows > rowIdx
						for i := rowIdx + 1; i < len(m.worktrees)+1; i++ {
							if state, ok := m.operationStates[i]; ok {
								m.operationStates[i-1] = state
								delete(m.operationStates, i)
							}
							if cell, ok := m.asyncOperations[i]; ok {
								m.asyncOperations[i-1] = cell
								delete(m.asyncOperations, i)
							}
							if status, ok := m.asyncStatuses[i]; ok {
								m.asyncStatuses[i-1] = status
								delete(m.asyncStatuses, i)
							}
						}
						
						// Adjust cursor: move up by 1 if we deleted the last row, stay otherwise
						newCursor := rowIdx
						if rowIdx >= len(m.worktrees) && len(m.worktrees) > 0 {
							newCursor = len(m.worktrees) - 1
						}
						m.table.SetCursor(newCursor)
						
						m.updateTableRows()
						return m, tea.Batch(cmds...)
					}
					
					// Schedule the clear operation
					cmds = append(cmds, clearOperationCmd(rowIdx, 2*time.Second))
				}
			}
		}
		m.updateTableRows()

	case clearOperationMsg:
		// Clear the operation state for this row
		if opState, ok := m.operationStates[msg.rowIdx]; ok {
			opState.operation = ""
			opState.result = ""
			opState.clearAt = time.Time{}
			m.operationStates[msg.rowIdx] = opState
			m.updateTableRows()
		}

	case tickMsg:
		// Tick all async cells to advance spinner animation
		for _, asyncCell := range m.asyncStatuses {
			asyncCell.Tick()
		}
		// Tick operation cells too
		for _, asyncCell := range m.asyncOperations {
			asyncCell.Tick()
		}
		// Refresh display to show new spinner frame
		m.updateTableRows()
		// Schedule the next tick
		cmds = append(cmds, m.tickCmd())

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			return m, tea.Quit
		case "enter", " ":
			// Output selected worktree path and quit
			fmt.Printf("%s\n", m.worktrees[m.table.Cursor()].Path)
			return m, tea.Quit
		case "l": // Pull
			rowIdx := m.table.Cursor()
			if rowIdx < len(m.worktrees) {
				cmd := m.triggerOperation(rowIdx, "pull")
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			}
			// Delegate to table for navigation
			tableModel, tableCmd := m.table.Update(msg)
			m.table = tableModel
			if tableCmd != nil {
				cmds = append(cmds, tableCmd)
			}
		case "p": // Push
			rowIdx := m.table.Cursor()
			if rowIdx < len(m.worktrees) {
				kind := "ad hoc"
				if m.trackedBranches[m.worktrees[rowIdx].Branch] {
					kind = "tracked"
				}
				if kind != "tracked" {
					cmd := m.triggerOperation(rowIdx, "push")
					if cmd != nil {
						cmds = append(cmds, cmd)
					}
				}
			}
			// Delegate to table for navigation
			tableModel, tableCmd := m.table.Update(msg)
			m.table = tableModel
			if tableCmd != nil {
				cmds = append(cmds, tableCmd)
			}
		case "d": // Delete - don't pass to table, maintain cursor position
			rowIdx := m.table.Cursor()
			if rowIdx < len(m.worktrees) {
				cmd := m.triggerOperation(rowIdx, "delete")
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			}
			// Skip table.Update for delete - we handle cursor ourselves
		default:
			// For all other keys, delegate to table for navigation
			tableModel, tableCmd := m.table.Update(msg)
			m.table = tableModel
			if tableCmd != nil {
				cmds = append(cmds, tableCmd)
			}
		}
	default:
		// For non-KeyMsg, always delegate to table
		tableModel, tableCmd := m.table.Update(msg)
		m.table = tableModel

		if tableCmd != nil {
			cmds = append(cmds, tableCmd)
		}
	}

	if len(cmds) == 0 {
		return m, nil
	}
	if len(cmds) == 1 {
		return m, cmds[0]
	}
	return m, tea.Batch(cmds...)
}

// updateColumns sets column widths to span the available width
func (m *testlsModel) updateColumns(width int) {
	// Reserve space for borders and padding (~4 chars)
	availableWidth := width - 4
	if availableWidth < 60 {
		availableWidth = 60 // Minimum acceptable width
	}

	// Distribute width: Name (30%), Branch (45%), Kind (10%), GitStatus (15%)
	nameWidth := max(15, availableWidth*30/100)
	branchWidth := max(25, availableWidth*45/100)
	kindWidth := max(8, availableWidth*10/100)
	gitStatusWidth := max(12, availableWidth*15/100)

	columns := []table.Column{
		{Title: "Name", Width: nameWidth},
		{Title: "Branch", Width: branchWidth},
		{Title: "Kind", Width: kindWidth},
		{Title: "Git Status", Width: gitStatusWidth},
	}

	m.table.SetColumns(columns)
}

// updateTableRows refreshes table rows with current async cell values
func (m *testlsModel) updateTableRows() {
	rows := []table.Row{}
	for i, wt := range m.worktrees {
		kind := "ad hoc"
		if m.trackedBranches[wt.Branch] {
			kind = "tracked"
		}

		// Get git status: spinner or loaded value
		gitStatus := "—"
		if asyncCell, ok := m.asyncStatuses[i]; ok {
			gitStatus = asyncCell.View()
		}

		// Append operation result if present
		if opState, ok := m.operationStates[i]; ok && opState.result != "" {
			gitStatus = gitStatus + " [" + opState.result + "]"
		}

		rows = append(rows, table.Row{
			wt.Name,
			wt.Branch,
			kind,
			gitStatus,
		})
	}
	m.table.SetRows(rows)
}

// triggerOperation starts an async operation for the given row.
func (m *testlsModel) triggerOperation(rowIdx int, op string) tea.Cmd {
	var opFunc func(path string) (string, error)
	mockGitService := &MockTableGitService{delay: m.delay}

	switch op {
	case "pull":
		opFunc = mockGitService.Pull
	case "push":
		opFunc = mockGitService.Push
	case "delete":
		opFunc = mockGitService.Delete
	default:
		return nil
	}

	// Create async cell for operation
	eval := async.New(func() (string, error) {
		if rowIdx >= len(m.worktrees) {
			return "", fmt.Errorf("invalid row index")
		}
		return opFunc(m.worktrees[rowIdx].Path)
	})

	cell := async.NewCell(eval)
	m.asyncOperations[rowIdx] = cell

	// Update operation state
	m.operationStates[rowIdx] = operationState{
		operation: op,
		result:    "",
		clearAt:   time.Time{},
	}

	// Start loading and return cmd
	return cell.StartLoading()
}

// View renders the table with footer help text.
func (m *testlsModel) View() string {
	output := m.table.View()

	// Show help text (conditionally show push option for ad-hoc worktrees)
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	// Determine if selected worktree is tracked
	isTracked := false
	cursor := m.table.Cursor()
	if cursor < len(m.worktrees) {
		isTracked = m.trackedBranches[m.worktrees[cursor].Branch]
	}

	help := "\n↑/↓: navigate • space/enter: select • l: pull"
	if !isTracked {
		help += " • p: push"
	}
	help += " • d: delete • q/esc: quit"

	// Show operation info if one is in progress
	if cursor < len(m.worktrees) {
		if opState, ok := m.operationStates[cursor]; ok && opState.operation != "" {
			help += "\n[" + opState.operation + " in progress]"
		}
	}

	help += "\n"
	output += helpStyle.Render(help)
	return output
}

// MockTableGitService provides mock git operations for testing.
type MockTableGitService struct {
	delay time.Duration
}

// GetBranchStatus returns a mock git status with delay.
func (m *MockTableGitService) GetBranchStatus(path string) (string, error) {
	time.Sleep(m.delay)
	statuses := []string{"✓", "↑ 3", "↓ 2", "↕ 1↑2", "?"}
	hash := 0
	for _, c := range path {
		hash = (hash*31 + int(c)) % len(statuses)
	}
	return statuses[hash], nil
}

// Pull simulates a git pull operation.
func (m *MockTableGitService) Pull(path string) (string, error) {
	time.Sleep(m.delay)
	return "✓ pulled", nil
}

// Push simulates a git push operation.
func (m *MockTableGitService) Push(path string) (string, error) {
	time.Sleep(m.delay)
	return "✓ pushed", nil
}

// Delete simulates a worktree delete operation.
func (m *MockTableGitService) Delete(path string) (string, error) {
	time.Sleep(m.delay)
	return "✓ deleted", nil
}

// Mock data for testing
var (
	mockWorktrees = []mockWorktree{
		{"main", "main", "/tmp/git-repo"},
		{"feature/auth", "feature/auth", "/tmp/feature-auth"},
		{"bugfix/login", "bugfix/login", "/tmp/bugfix-login"},
		{"wip/dashboard", "wip/dashboard", "/tmp/wip-dashboard"},
		{"hotfix/crash", "hotfix/crash", "/tmp/hotfix-crash"},
		{"release/v1.0", "release/v1.0", "/tmp/release-v1.0"},
		{"experiment/ml", "experiment/ml", "/tmp/experiment-ml"},
		{"archived/old", "archived/old", "/tmp/archived-old"},
	}

	// Tracked branches (cannot be pushed)
	trackedBranches = map[string]bool{
		"main":         true,
		"develop":      true,
		"release/v1.0": true,
	}
)

// newWorktreeTestlsCommand creates the testls command for testing the Table component.
func newWorktreeTestlsCommand(svc *Service) *cobra.Command {
	var delay int

	cmd := &cobra.Command{
		Use:   "testls",
		Short: "Test the async table component with mock data",
		Long:  "Displays a table of mock worktrees with async git status loading",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTestLS(time.Duration(delay) * time.Millisecond)
		},
	}

	cmd.Flags().IntVar(&delay, "delay", 1000, "Simulated network delay in milliseconds (0-5000)")

	return cmd
}

// runTestLS executes the testls command.
func runTestLS(delay time.Duration) error {
	// Build table with mock data - matching original worktree_table.go styling
	columns := []table.Column{
		{Title: "Name", Width: 30},
		{Title: "Branch", Width: 50},
		{Title: "Kind", Width: 10},
		{Title: "Git Status", Width: 15},
	}

	// Create table with same styling as original worktree table
	// Rows will be set by Init() via updateTableRows()
	height := min(len(mockWorktrees)+1, 26)
	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(height),
	)

	// Apply original worktree table styles
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

	// Open debug log file if DEBUG=1
	var messageDump io.Writer
	if os.Getenv("DEBUG") == "1" {
		debugFile, err := os.Create("messages.log")
		if err != nil {
			return fmt.Errorf("failed to create debug log: %w", err)
		}
		defer func() {
			_ = debugFile.Close()
		}()
		messageDump = debugFile
	}

	// Create model
	model := &testlsModel{
		table:           t,
		worktrees:       mockWorktrees,
		trackedBranches: trackedBranches,
		delay:           delay,
		asyncStatuses:   make(map[int]*async.Cell[string]),
		asyncOperations: make(map[int]*async.Cell[string]),
		operationStates: make(map[int]operationState),
		messageDump:     messageDump,
	}

	// Open input for TUI
	input, err := os.Open("/dev/tty")
	if err != nil {
		return fmt.Errorf("failed to open /dev/tty: %w", err)
	}
	defer func() {
		_ = input.Close()
	}()

	// Run TUI program
	p := tea.NewProgram(model, tea.WithInput(input), tea.WithAltScreen())
	_, err = p.Run()
	if err != nil {
		return fmt.Errorf("testls error: %w", err)
	}

	return nil
}

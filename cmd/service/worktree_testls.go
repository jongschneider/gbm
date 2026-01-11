package service

import (
	"fmt"
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
	selectedRow     int
	// Async cell tracking
	asyncStatuses map[int]*async.Cell[string] // Row index -> async cell for git status
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

type tickMsg struct{}

// Update handles input and state changes.
func (m *testlsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	consumeKey := false

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.table.SetWidth(msg.Width)
		m.table.SetHeight(msg.Height - 4)
		m.updateColumns(msg.Width)
		
	case tickMsg:
		// Tick all async cells to advance spinner animation
		for _, asyncCell := range m.asyncStatuses {
			asyncCell.Tick()
		}
		// Schedule the next tick
		cmds = append(cmds, m.tickCmd())
		
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			return m, tea.Quit
		case "up":
			if m.selectedRow > 0 {
				m.selectedRow--
				m.table.SetCursor(m.selectedRow)
			}
			consumeKey = true
		case "down":
			if m.selectedRow < len(m.worktrees)-1 {
				m.selectedRow++
				m.table.SetCursor(m.selectedRow)
			}
			consumeKey = true
		case "enter", " ":
			// Output selected worktree path and quit
			fmt.Printf("%s\n", m.worktrees[m.selectedRow].Path)
			return m, tea.Quit
		case "l": // Pull
			selectedName := m.worktrees[m.selectedRow].Name
			fmt.Fprintf(os.Stderr, "Would pull: %s\n", selectedName)
			consumeKey = true
		case "p": // Push
			kind := "ad hoc"
			if m.trackedBranches[m.worktrees[m.selectedRow].Branch] {
				kind = "tracked"
			}
			if kind == "tracked" {
				fmt.Fprintf(os.Stderr, "Cannot push tracked worktree\n")
			} else {
				fmt.Fprintf(os.Stderr, "Would push: %s\n", m.worktrees[m.selectedRow].Name)
			}
			consumeKey = true
		case "d": // Delete
			fmt.Fprintf(os.Stderr, "Would delete: %s\n", m.worktrees[m.selectedRow].Name)
			consumeKey = true
		}
	}

	// Update table based on current async cell states
	m.updateTableRows()

	// Delegate to table (unless we consumed the key)
	if !consumeKey {
		tableModel, tableCmd := m.table.Update(msg)
		m.table = tableModel
		m.selectedRow = m.table.Cursor()

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

		rows = append(rows, table.Row{
			wt.Name,
			wt.Branch,
			kind,
			gitStatus,
		})
	}
	m.table.SetRows(rows)
}

// View renders the table with footer help text.
func (m *testlsModel) View() string {
	output := m.table.View()

	// Show help text (conditionally show push option for ad-hoc worktrees)
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	// Determine if selected worktree is tracked
	isTracked := false
	if m.selectedRow < len(m.worktrees) {
		isTracked = m.trackedBranches[m.worktrees[m.selectedRow].Branch]
	}

	help := "\n↑/↓: navigate • space/enter: select • l: pull"
	if !isTracked {
		help += " • p: push"
	}
	help += " • d: delete • q/esc: quit\n"

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
		hash = (hash * 31 + int(c)) % len(statuses)
	}
	return statuses[hash], nil
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
		"main":        true,
		"develop":     true,
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

	// Create initial rows with placeholder for git status
	rows := []table.Row{}
	for _, wt := range mockWorktrees {
		kind := "ad hoc"
		if trackedBranches[wt.Branch] {
			kind = "tracked"
		}
		rows = append(rows, table.Row{wt.Name, wt.Branch, kind, "—"})
	}

	// Create table with same styling as original worktree table
	height := min(len(rows)+1, 26)
	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
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

	// Create model
	model := &testlsModel{
		table:           t,
		worktrees:       mockWorktrees,
		trackedBranches: trackedBranches,
		delay:           delay,
		selectedRow:     0,
		asyncStatuses:   make(map[int]*async.Cell[string]),
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

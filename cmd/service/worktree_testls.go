package service

import (
	"fmt"
	"os"
	"time"

	"gbm/pkg/tui"
	"gbm/pkg/tui/async"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

// testlsModel manages the async table display for testing the Table component.
type testlsModel struct {
	table       *tui.Table
	ctx         *tui.Context
	delay       time.Duration
	confirmed   bool
	selectedRow int
}

// Init initializes the table model.
func (m *testlsModel) Init() tea.Cmd {
	// Start async loads for git status cells
	cmds := []tea.Cmd{}
	for i := 0; i < len(mockWorktrees); i++ {
		// Column 3 is git status - async load it
		rowIdx := i
		colIdx := 3

		// Create an async cell that fetches git status with delay
		eval := async.New(func() (string, error) {
			mockGitService := &MockTableGitService{delay: m.delay}
			status, err := mockGitService.GetBranchStatus(mockWorktrees[rowIdx].Path)
			if err != nil {
				return "error", err
			}
			return status, nil
		})

		cell := async.NewCell(eval)
		m.table.SetAsyncCell(rowIdx, colIdx, cell)

		// StartLoading() returns a Cmd that fetches the value
		cmd := cell.StartLoading()
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	if len(cmds) > 0 {
		return tea.Batch(cmds...)
	}
	return nil
}

// Update handles input and state changes.
func (m *testlsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			return m, tea.Quit
		case "up":
			if m.selectedRow > 0 {
				m.selectedRow--
				m.table.SetCursor(m.selectedRow)
			}
		case "down":
			if m.selectedRow < len(mockWorktrees)-1 {
				m.selectedRow++
				m.table.SetCursor(m.selectedRow)
			}
		case "enter", " ":
			// Output selected worktree path and quit
			fmt.Printf("%s\n", mockWorktrees[m.selectedRow].Path)
			return m, tea.Quit
		case "l": // Pull
			// Simulate pull operation
			selectedName := mockWorktrees[m.selectedRow].Name
			// For now, just show it was pulled (mock doesn't actually do anything)
			fmt.Fprintf(os.Stderr, "Would pull: %s\n", selectedName)
		case "p": // Push
			// Only allowed for ad-hoc worktrees
			kind := "ad hoc"
			if trackedBranches[mockWorktrees[m.selectedRow].Branch] {
				kind = "tracked"
			}
			if kind == "tracked" {
				fmt.Fprintf(os.Stderr, "Cannot push tracked worktree\n")
			} else {
				fmt.Fprintf(os.Stderr, "Would push: %s\n", mockWorktrees[m.selectedRow].Name)
			}
		case "d": // Delete
			fmt.Fprintf(os.Stderr, "Would delete: %s\n", mockWorktrees[m.selectedRow].Name)
		}
	}

	// Delegate to table
	_, cmd := m.table.Update(msg)
	m.selectedRow = m.table.Cursor()
	return m, cmd
}

// View renders the table with footer help text.
func (m *testlsModel) View() string {
	output := m.table.View()

	// Show help text (conditionally show push option for ad-hoc worktrees)
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	// Determine if selected worktree is tracked
	istracked := false
	if m.selectedRow < len(mockWorktrees) {
		istracked = trackedBranches[mockWorktrees[m.selectedRow].Branch]
	}

	help := "\n↑/↓: navigate • space/enter: select • l: pull"
	if !istracked {
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
	mockWorktrees = []struct {
		Name   string
		Branch string
		Path   string
	}{
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
	// Create TUI context
	ctx := tui.NewContext()

	// Build table with mock data
	columns := []tui.Column{
		{Title: "Name", Width: 20},
		{Title: "Branch", Width: 25},
		{Title: "Kind", Width: 10},
		{Title: "Git Status", Width: 15},
	}

	tableRows := make([]table.Row, len(mockWorktrees))
	for i, wt := range mockWorktrees {
		kind := "ad hoc"
		if trackedBranches[wt.Branch] {
			kind = "tracked"
		}
		tableRows[i] = table.Row{wt.Name, wt.Branch, kind, ""}
	}

	table := tui.NewTable(ctx).
		WithColumns(columns).
		WithRows(tableRows).
		WithHeight(min(len(tableRows)+1, 26)).
		Build()

	// Create model
	model := &testlsModel{
		table:   table,
		ctx:     ctx,
		delay:   delay,
		selectedRow: 0,
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

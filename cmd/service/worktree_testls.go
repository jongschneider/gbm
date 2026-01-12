package service

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gbm/internal/git"
	"gbm/pkg/tui"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/spf13/cobra"
)

// testlsGitOps defines the git operations needed by the testls TUI.
// This interface enables dependency injection and mock testing.
type testlsGitOps interface {
	ListWorktrees(dryRun bool) ([]git.Worktree, error)
	GetBranchStatus(worktreePath string) (*git.BranchStatus, error)
	RemoveWorktree(worktreeName string, force, dryRun bool) (*git.Worktree, error)
	PullWorktree(worktreePath string, dryRun bool) error
	PushWorktree(worktreePath string, dryRun bool) error
}

// operationState represents the FSM states for the testls TUI.
type operationState int

const (
	stateIdle operationState = iota
	stateConfirming
	stateOperating
)

// operationResultMsg is sent when an async git operation completes.
type operationResultMsg struct {
	opType     string // "pull", "push", "delete"
	targetName string
	newStatus  *git.BranchStatus // updated status after operation
	err        error
}

// clearMessageMsg is sent after a delay to clear the result message.
type clearMessageMsg struct{}

// testlsModel is the Bubble Tea model for the testls TUI.
type testlsModel struct {
	// Display components
	ctx   *tui.Context
	table *tui.Table

	// Data
	worktrees       []git.Worktree
	trackedBranches map[string]bool
	branchStatuses  map[string]*git.BranchStatus

	// Dependencies
	gitOps          testlsGitOps
	currentWorktree *git.Worktree

	// State machine
	state           operationState
	currentOp       string        // "pull", "push", "delete"
	operationTarget string        // worktree name being operated on
	operationIndex  int           // row index being operated on
	spinner         spinner.Model // spinner for animation

	// Output
	message      string
	switchOutput string // worktree path to output on exit
}

// newTestlsModel creates a new testlsModel with pre-fetched data.
func newTestlsModel(
	worktrees []git.Worktree,
	trackedBranches map[string]bool,
	branchStatuses map[string]*git.BranchStatus,
	currentWorktree *git.Worktree,
	gitOps testlsGitOps,
) *testlsModel {
	ctx := tui.NewContext()

	// Build rows using shared helpers
	rows := make([]table.Row, 0, len(worktrees))
	for _, wt := range worktrees {
		status := branchStatuses[wt.Name]
		rows = append(rows, BuildWorktreeRow(wt, currentWorktree, trackedBranches, status))
	}

	// Default columns (will be resized on WindowSizeMsg)
	columns := []tui.Column{
		{Title: "Name", Width: 30},
		{Title: "Branch", Width: 50},
		{Title: "Kind", Width: 10},
		{Title: "Status", Width: 15},
	}

	// Create table
	tbl := tui.NewTable(ctx).
		WithColumns(columns).
		WithRows(rows).
		WithHeight(min(len(rows)+1, 26)).
		WithFocused(true).
		Build()

	// Create spinner
	sp := spinner.New()
	sp.Spinner = spinner.Line
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return &testlsModel{
		ctx:             ctx,
		table:           tbl,
		worktrees:       worktrees,
		trackedBranches: trackedBranches,
		branchStatuses:  branchStatuses,
		gitOps:          gitOps,
		currentWorktree: currentWorktree,
		state:           stateIdle,
		spinner:         sp,
	}
}

// Init implements tea.Model.
func (m *testlsModel) Init() tea.Cmd {
	return m.spinner.Tick
}

// Update implements tea.Model with the state machine.
func (m *testlsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.ctx = m.ctx.WithDimensions(msg.Width, msg.Height)
		columns := CalculateTableColumns(msg.Width)
		m.table.SetColumns(convertToTuiColumns(columns))
		m.table.SetHeight(CalculateTableHeight(msg.Height, len(m.worktrees)))
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case spinner.TickMsg:
		if m.state == stateOperating {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			// Update the row to show spinner
			m.updateOperatingRow()
			return m, cmd
		}
		return m, nil

	case operationResultMsg:
		return m.handleOperationResult(msg)

	case clearMessageMsg:
		m.message = ""
		return m, nil
	}

	// Forward other messages to table
	_, cmd := m.table.Update(msg)
	return m, cmd
}

// handleKeyMsg processes key presses based on current state.
func (m *testlsModel) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.state {
	case stateIdle:
		return m.handleIdleKeyMsg(msg)
	case stateConfirming:
		return m.handleConfirmingKeyMsg(msg)
	case stateOperating:
		// Only allow quit during operation
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		}
		return m, nil
	}
	return m, nil
}

// handleIdleKeyMsg handles keys in idle state.
func (m *testlsModel) handleIdleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c", "esc":
		return m, tea.Quit

	case "up", "k":
		m.table.SetCursor(max(0, m.table.Cursor()-1))
		return m, nil

	case "down", "j":
		m.table.SetCursor(min(len(m.worktrees)-1, m.table.Cursor()+1))
		return m, nil

	case " ", "enter":
		cursor := m.table.Cursor()
		if cursor >= 0 && cursor < len(m.worktrees) {
			m.switchOutput = m.worktrees[cursor].Path
			return m, tea.Quit
		}
		return m, nil

	case "l":
		return m.startOperation("pull")

	case "p":
		cursor := m.table.Cursor()
		if cursor >= 0 && cursor < len(m.worktrees) {
			wt := m.worktrees[cursor]
			if m.trackedBranches[wt.Branch] {
				m.message = fmt.Sprintf("Cannot push tracked branch '%s'", wt.Branch)
				return m, m.scheduleClearMessage()
			}
		}
		return m.startOperation("push")

	case "d":
		cursor := m.table.Cursor()
		if cursor >= 0 && cursor < len(m.worktrees) {
			m.operationTarget = m.worktrees[cursor].Name
			m.operationIndex = cursor
			m.state = stateConfirming
		}
		return m, nil
	}

	// Forward navigation to table
	_, cmd := m.table.Update(msg)
	// Guard against cursor going out of bounds (bubbles/table handles additional keys)
	m.clampCursor()
	return m, cmd
}

// handleConfirmingKeyMsg handles keys in confirmation state.
func (m *testlsModel) handleConfirmingKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		return m.startOperation("delete")
	case "n", "N", "esc":
		m.state = stateIdle
		m.operationTarget = ""
		m.operationIndex = 0
		return m, nil
	}
	return m, nil
}

// startOperation initiates an async git operation.
func (m *testlsModel) startOperation(opType string) (tea.Model, tea.Cmd) {
	cursor := m.table.Cursor()
	if cursor < 0 || cursor >= len(m.worktrees) {
		return m, nil
	}

	wt := m.worktrees[cursor]
	m.state = stateOperating
	m.currentOp = opType
	m.operationTarget = wt.Name
	m.operationIndex = cursor
	m.message = ""

	// Update row to show spinner immediately
	m.updateOperatingRow()

	// Create async command for the git operation
	cmd := m.createOperationCmd(opType, wt)

	return m, tea.Batch(m.spinner.Tick, cmd)
}

// createOperationCmd creates a tea.Cmd for the git operation.
func (m *testlsModel) createOperationCmd(opType string, wt git.Worktree) tea.Cmd {
	return func() tea.Msg {
		var err error
		var newStatus *git.BranchStatus

		switch opType {
		case "pull":
			err = m.gitOps.PullWorktree(wt.Path, false)
			if err == nil {
				newStatus, _ = m.gitOps.GetBranchStatus(wt.Path)
			}
		case "push":
			err = m.gitOps.PushWorktree(wt.Path, false)
			if err == nil {
				newStatus, _ = m.gitOps.GetBranchStatus(wt.Path)
			}
		case "delete":
			_, err = m.gitOps.RemoveWorktree(wt.Name, false, false)
		}

		return operationResultMsg{
			opType:     opType,
			targetName: wt.Name,
			newStatus:  newStatus,
			err:        err,
		}
	}
}

// handleOperationResult processes the result of an async operation.
func (m *testlsModel) handleOperationResult(msg operationResultMsg) (tea.Model, tea.Cmd) {
	m.state = stateIdle
	m.currentOp = ""

	if msg.err != nil {
		m.message = fmt.Sprintf("Error %s '%s': %v", msg.opType, msg.targetName, msg.err)
		// Restore the row to previous state
		if m.operationIndex < len(m.worktrees) {
			wt := m.worktrees[m.operationIndex]
			status := m.branchStatuses[wt.Name]
			m.updateRow(m.operationIndex, BuildWorktreeRow(wt, m.currentWorktree, m.trackedBranches, status))
		}
	} else {
		switch msg.opType {
		case "delete":
			m.message = fmt.Sprintf("Deleted worktree '%s'", msg.targetName)
			// Refresh the worktree list
			return m.refreshAfterDelete()
		default:
			m.message = fmt.Sprintf("Successfully %sed '%s'", msg.opType, msg.targetName)
			// Update the status
			if msg.newStatus != nil && m.operationIndex < len(m.worktrees) {
				wt := m.worktrees[m.operationIndex]
				m.branchStatuses[wt.Name] = msg.newStatus
				m.updateRow(m.operationIndex, BuildWorktreeRow(wt, m.currentWorktree, m.trackedBranches, msg.newStatus))
			}
		}
	}

	m.operationTarget = ""
	m.operationIndex = 0

	return m, m.scheduleClearMessage()
}

// refreshAfterDelete reloads the worktree list after a delete.
func (m *testlsModel) refreshAfterDelete() (tea.Model, tea.Cmd) {
	worktrees, err := m.gitOps.ListWorktrees(false)
	if err != nil {
		m.message = fmt.Sprintf("Error refreshing: %v", err)
		return m, m.scheduleClearMessage()
	}

	// Filter out bare worktrees and sort
	var filtered []git.Worktree
	for _, wt := range worktrees {
		if !wt.IsBare {
			filtered = append(filtered, wt)
		}
	}
	m.worktrees = filtered

	// Rebuild rows
	rows := make([]table.Row, 0, len(m.worktrees))
	for _, wt := range m.worktrees {
		status := m.branchStatuses[wt.Name]
		rows = append(rows, BuildWorktreeRow(wt, m.currentWorktree, m.trackedBranches, status))
	}
	m.table.SetRows(rows)

	// Adjust cursor if needed
	if m.table.Cursor() >= len(m.worktrees) {
		m.table.SetCursor(max(0, len(m.worktrees)-1))
	}

	return m, m.scheduleClearMessage()
}

// updateOperatingRow updates the current row to show spinner.
func (m *testlsModel) updateOperatingRow() {
	if m.operationIndex >= len(m.worktrees) {
		return
	}

	wt := m.worktrees[m.operationIndex]
	opLabel := m.currentOp
	if opLabel == "delete" {
		opLabel = "del"
	}

	// Create row with spinner in status column
	row := table.Row{
		FormatWorktreeName(wt, m.currentWorktree),
		wt.Branch,
		FormatWorktreeKind(wt, m.trackedBranches),
		m.spinner.View() + " " + opLabel,
	}
	m.updateRow(m.operationIndex, row)
}

// updateRow updates a specific row in the table.
func (m *testlsModel) updateRow(index int, row table.Row) {
	rows := m.table.Rows()
	if index < len(rows) {
		rows[index] = row
		m.table.SetRows(rows)
	}
}

// clampCursor ensures the table cursor stays within valid bounds.
func (m *testlsModel) clampCursor() {
	if len(m.worktrees) == 0 {
		return
	}
	cursor := m.table.Cursor()
	if cursor < 0 {
		m.table.SetCursor(0)
	} else if cursor >= len(m.worktrees) {
		m.table.SetCursor(len(m.worktrees) - 1)
	}
}

// scheduleClearMessage returns a command to clear the message after 2 seconds.
func (m *testlsModel) scheduleClearMessage() tea.Cmd {
	return tea.Tick(2*time.Second, func(time.Time) tea.Msg {
		return clearMessageMsg{}
	})
}

// View implements tea.Model.
func (m *testlsModel) View() string {
	var output string

	// Render table
	output = m.table.View()

	// Show confirmation prompt if in confirming state
	if m.state == stateConfirming {
		confirmStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("208")).
			Bold(true)
		confirmMsg := fmt.Sprintf("\n\nDelete worktree '%s'? (y/n): ", m.operationTarget)
		output += confirmStyle.Render(confirmMsg)
		return output
	}

	// Show message if any
	if m.message != "" {
		messageStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("120"))
		output += messageStyle.Render("\n" + m.message)
	}

	// Show help text
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	cursor := m.table.Cursor()
	showPush := true
	if cursor >= 0 && cursor < len(m.worktrees) {
		if m.trackedBranches[m.worktrees[cursor].Branch] {
			showPush = false
		}
	}

	help := "\n↑/↓: navigate • space/enter: switch • l: pull"
	if showPush {
		help += " • p: push"
	}
	help += " • d: delete • q/esc: quit\n"

	if m.state == stateOperating {
		help = "\n" + m.spinner.View() + " Operation in progress... (q to quit)\n"
	}

	output += helpStyle.Render(help)

	return output
}

// convertToTuiColumns converts table.Column to tui.Column.
func convertToTuiColumns(cols []table.Column) []tui.Column {
	result := make([]tui.Column, len(cols))
	for i, c := range cols {
		result[i] = tui.Column{Title: c.Title, Width: c.Width}
	}
	return result
}

// runTestlsTable runs the testls TUI.
func runTestlsTable(worktrees []git.Worktree, trackedBranches map[string]bool, currentWorktree *git.Worktree, svc *Service) error {
	// Open /dev/tty for TUI rendering
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("failed to open /dev/tty: %w (TUI requires an interactive terminal)", err)
	}
	defer func() {
		_ = tty.Close()
	}()

	// Set up color renderer
	renderer := lipgloss.NewRenderer(tty,
		termenv.WithColorCache(true),
		termenv.WithTTY(true),
		termenv.WithProfile(termenv.TrueColor),
	)
	lipgloss.SetDefaultRenderer(renderer)

	// Fetch branch statuses concurrently
	branchStatuses := fetchBranchStatuses(worktrees, svc.Git)

	// Create model
	m := newTestlsModel(worktrees, trackedBranches, branchStatuses, currentWorktree, svc.Git)

	// Run TUI
	p := tea.NewProgram(m,
		tea.WithInput(tty),
		tea.WithOutput(tty),
		tea.WithAltScreen(),
	)

	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("error running TUI: %w", err)
	}

	// Output path to stdout if user selected a worktree
	if model, ok := finalModel.(*testlsModel); ok {
		if model.switchOutput != "" {
			fmt.Println(model.switchOutput)
			fmt.Fprintf(os.Stderr, "✓ Selected worktree: %s\n", filepath.Base(model.switchOutput))
		}
	}

	return nil
}

// fetchBranchStatuses fetches branch statuses for all worktrees concurrently.
func fetchBranchStatuses(worktrees []git.Worktree, gitSvc testlsGitOps) map[string]*git.BranchStatus {
	statuses := make(map[string]*git.BranchStatus)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, wt := range worktrees {
		if wt.IsBare {
			continue
		}

		wg.Add(1)
		go func(worktree git.Worktree) {
			defer wg.Done()
			status, err := gitSvc.GetBranchStatus(worktree.Path)
			if err == nil && status != nil {
				mu.Lock()
				statuses[worktree.Name] = status
				mu.Unlock()
			}
		}(wt)
	}
	wg.Wait()

	return statuses
}

// newWorktreeTestlsCommand creates the testls subcommand.
func newWorktreeTestlsCommand(svc *Service) *cobra.Command {
	return &cobra.Command{
		Use:   "testls",
		Short: "List worktrees with async operations (prototype)",
		Long:  `Interactive TUI to list and manage worktrees with non-blocking async git operations.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get worktrees
			worktrees, err := svc.Git.ListWorktrees(false)
			if err != nil {
				return fmt.Errorf("failed to list worktrees: %w", err)
			}

			// Filter out bare worktrees and sort by priority
			var filtered []git.Worktree
			for _, wt := range worktrees {
				if !wt.IsBare {
					filtered = append(filtered, wt)
				}
			}

			if len(filtered) == 0 {
				fmt.Fprintln(os.Stderr, "No worktrees found")
				return nil
			}

			// Get tracked branches from config
			config := svc.GetConfig()
			trackedBranches := make(map[string]bool)
			for _, wtConfig := range config.Worktrees {
				trackedBranches[wtConfig.Branch] = true
			}

			// Get current worktree
			currentWorktree, _ := svc.Git.GetCurrentWorktree()

			return runTestlsTable(filtered, trackedBranches, currentWorktree, svc)
		},
	}
}

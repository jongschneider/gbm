package service

import (
	"fmt"
	"gbm/internal/git"
	"gbm/pkg/tui"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

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
// The table height is set to show all available worktrees (rowCount).
// The table library handles its own viewport rendering with header + data rows.
func CalculateTableHeight(terminalHeight, rowCount int) int {
	// Reserve space for header (1), message (1), help text (1), and padding (1)
	reservedLines := 4
	availableHeight := terminalHeight - reservedLines

	// Use all available space or fit all rows, whichever is smaller
	if rowCount+2 <= availableHeight {
		return rowCount + 2 // 1 for header, 1 for border
	}
	return availableHeight
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

// WorktreeConfigService defines the configuration service interface needed by the TUI.
type WorktreeConfigService interface {
	GetConfig() *Config
}

// WorktreeTableGitOps defines the git operations needed by the worktree table TUI.
// This interface enables dependency injection and mock testing.
type WorktreeTableGitOps interface {
	ListWorktrees(dryRun bool) ([]git.Worktree, error)
	GetCurrentWorktree() (*git.Worktree, error)
	GetBranchStatus(worktreePath string) (*git.BranchStatus, error)
	RemoveWorktree(worktreeName string, force, dryRun bool) (*git.Worktree, error)
	PullWorktree(worktreePath string, dryRun bool) error
	PushWorktree(worktreePath string, dryRun bool) error
	ListBranches(dryRun bool) ([]string, error)
	BranchExists(name string) (bool, error)
	DeleteBranch(branchName string, force, dryRun bool) error
}

// operationState represents the FSM states for the testls TUI.
type operationState int

const (
	stateIdle operationState = iota
	stateConfirming
	stateOperating
	stateConfirmingBranchDelete
)

// operationResultMsg is sent when an async git operation completes.
type operationResultMsg struct {
	err        error
	newStatus  *git.BranchStatus
	opType     string
	targetName string
	branchName string
}

// clearMessageMsg is sent after a delay to clear the result message.
type clearMessageMsg struct{}

// worktreeListModel is the Bubble Tea model for the worktree table TUI.
type worktreeListModel struct {
	gitOps            WorktreeTableGitOps
	trackedBranches   map[string]bool
	branchStatuses    map[string]*git.BranchStatus
	loadingStatuses   map[string]bool
	table             *tui.Table
	currentWorktree   *git.Worktree
	ctx               *tui.Context
	deletedBranchName string
	switchOutput      string
	currentOp         string
	operationTarget   string
	message           string
	worktrees         []git.Worktree
	spinner           spinner.Model
	operationIndex    int
	state             operationState
}

// newWorktreeListModel creates a new testlsModel with pre-fetched data.
func newWorktreeListModel(
	worktrees []git.Worktree,
	trackedBranches map[string]bool,
	branchStatuses map[string]*git.BranchStatus,
	currentWorktree *git.Worktree,
	gitOps WorktreeTableGitOps,
) *worktreeListModel {
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

	// Create table with height for all rows (will be recalculated on WindowSizeMsg)
	// Start with a reasonable default that fits all rows plus header
	tableHeight := len(rows) + 1

	// Build initial rows with loading spinners and initialize loading statuses
	initialRows := make([]table.Row, 0, len(worktrees))
	loadingStatuses := make(map[string]bool)
	for _, wt := range worktrees {
		if wt.IsBare {
			continue
		}
		row := table.Row{
			FormatWorktreeName(wt, currentWorktree),
			wt.Branch,
			FormatWorktreeKind(wt, trackedBranches),
			"", // Loading spinner placeholder
		}
		initialRows = append(initialRows, row)
		loadingStatuses[wt.Name] = true
	}

	tbl := tui.NewTable(ctx).
		WithColumns(columns).
		WithRows(initialRows).
		WithHeight(tableHeight).
		WithFocused(true).
		WithFilterable(true).
		WithCycling(true).
		Build()

	// Create spinner with dots style
	sp := spinner.New(spinner.WithSpinner(spinner.Dot))
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return &worktreeListModel{
		ctx:             ctx,
		table:           tbl,
		worktrees:       worktrees,
		trackedBranches: trackedBranches,
		branchStatuses:  branchStatuses,
		loadingStatuses: loadingStatuses,
		gitOps:          gitOps,
		currentWorktree: currentWorktree,
		state:           stateIdle,
		spinner:         sp,
	}
}

// Init implements tea.Model.
func (m *worktreeListModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.loadBranchStatusesAsync())
}

// statusFetchMsg wraps a FetchMsg with the worktree name.
type statusFetchMsg struct {
	err          error
	status       *git.BranchStatus
	worktreeName string
}

// loadBranchStatusesAsync returns a command that loads all branch statuses asynchronously.
func (m *worktreeListModel) loadBranchStatusesAsync() tea.Cmd {
	return func() tea.Msg {
		var cmds []tea.Cmd
		for _, wt := range m.worktrees {
			if wt.IsBare {
				continue
			}
			// Capture wt in closure
			worktree := wt
			cmd := func() tea.Msg {
				status, err := m.gitOps.GetBranchStatus(worktree.Path)
				return statusFetchMsg{
					worktreeName: worktree.Name,
					status:       status,
					err:          err,
				}
			}
			cmds = append(cmds, cmd)
		}
		return tea.Batch(cmds...)()
	}
}

// Update implements tea.Model with the state machine.
func (m *worktreeListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)

		if m.state == stateOperating {
			// Update the row to show spinner
			m.updateOperatingRow()
		}

		// Update loading status rows with spinner
		if len(m.loadingStatuses) > 0 {
			m.updateLoadingRows()
		}

		return m, cmd

	case operationResultMsg:
		return m.handleOperationResult(msg)

	case clearMessageMsg:
		m.message = ""
		return m, nil

	case statusFetchMsg:
		return m.handleBranchStatusLoaded(msg)
	}

	// Forward other messages to table
	_, cmd := m.table.Update(msg)
	return m, cmd
}

// handleBranchStatusLoaded updates the model when a branch status is loaded.
func (m *worktreeListModel) handleBranchStatusLoaded(msg statusFetchMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		// Silently ignore errors loading individual statuses, but clear loading state
		delete(m.loadingStatuses, msg.worktreeName)
		return m, nil
	}

	// Store the status and update the corresponding row
	m.branchStatuses[msg.worktreeName] = msg.status
	delete(m.loadingStatuses, msg.worktreeName)

	// Find the worktree index and update its row
	for i, wt := range m.worktrees {
		if wt.Name == msg.worktreeName {
			newRow := BuildWorktreeRow(wt, m.currentWorktree, m.trackedBranches, msg.status)
			m.updateRow(i, newRow)
			break
		}
	}

	return m, nil
}

// handleKeyMsg processes key presses based on current state.
func (m *worktreeListModel) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.state {
	case stateIdle:
		return m.handleIdleKeyMsg(msg)
	case stateConfirming:
		return m.handleConfirmingKeyMsg(msg)
	case stateConfirmingBranchDelete:
		return m.handleConfirmingBranchDeleteKeyMsg(msg)
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
func (m *worktreeListModel) handleIdleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.table.IsFilterActive() {
		_, cmd := m.table.Update(msg)
		return m, cmd
	}
	switch msg.String() {
	case "q", "ctrl+c", "esc":
		if msg.String() == "esc" && m.table.FilterQuery() != "" {
			m.table.ClearFilter()
			return m, nil
		}
		return m, tea.Quit
	case " ", "enter":
		return m.handleSelectWorktree()
	case "l":
		return m.startOperation("pull")
	case "p":
		return m.handlePush()
	case "d":
		return m.handleDeleteInit()
	}
	_, cmd := m.table.Update(msg)
	return m, cmd
}

func (m *worktreeListModel) handleSelectWorktree() (tea.Model, tea.Cmd) {
	if idx := m.table.OriginalIndex(); idx >= 0 && idx < len(m.worktrees) {
		m.switchOutput = m.worktrees[idx].Path
		return m, tea.Quit
	}
	return m, nil
}

func (m *worktreeListModel) handlePush() (tea.Model, tea.Cmd) {
	if idx := m.table.OriginalIndex(); idx >= 0 && idx < len(m.worktrees) && m.trackedBranches[m.worktrees[idx].Branch] {
		m.message = fmt.Sprintf("Cannot push tracked branch '%s'", m.worktrees[idx].Branch)
		return m, m.scheduleClearMessage()
	}
	return m.startOperation("push")
}

func (m *worktreeListModel) handleDeleteInit() (tea.Model, tea.Cmd) {
	if idx := m.table.OriginalIndex(); idx >= 0 && idx < len(m.worktrees) {
		m.operationTarget = m.worktrees[idx].Name
		m.operationIndex = idx
		m.state = stateConfirming
	}
	return m, nil
}

// handleConfirmingKeyMsg handles keys in confirmation state.
func (m *worktreeListModel) handleConfirmingKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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

// handleConfirmingBranchDeleteKeyMsg handles keys in branch delete confirmation state.
func (m *worktreeListModel) handleConfirmingBranchDeleteKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		// Delete the branch
		branchName := m.deletedBranchName
		err := m.gitOps.DeleteBranch(branchName, true, false)
		if err != nil {
			m.message = fmt.Sprintf("Error deleting branch '%s': %v", branchName, err)
		} else {
			m.message = fmt.Sprintf("Deleted branch '%s'", branchName)
		}
		m.deletedBranchName = ""
		m.state = stateIdle
		m.operationTarget = ""
		m.operationIndex = 0
		return m, m.scheduleClearMessage()
	case "n", "N", "esc":
		m.deletedBranchName = ""
		m.state = stateIdle
		m.operationTarget = ""
		m.operationIndex = 0
		return m, m.scheduleClearMessage()
	}
	return m, nil
}

// startOperation initiates an async git operation.
func (m *worktreeListModel) startOperation(opType string) (tea.Model, tea.Cmd) {
	idx := m.table.OriginalIndex()
	if idx < 0 || idx >= len(m.worktrees) {
		return m, nil
	}

	wt := m.worktrees[idx]
	m.state = stateOperating
	m.currentOp = opType
	m.operationTarget = wt.Name
	m.operationIndex = idx
	m.message = ""

	// Update row to show spinner immediately
	m.updateOperatingRow()

	// Create async command for the git operation
	cmd := m.createOperationCmd(opType, wt)

	return m, tea.Batch(m.spinner.Tick, cmd)
}

// createOperationCmd creates a tea.Cmd for the git operation.
func (m *worktreeListModel) createOperationCmd(opType string, wt git.Worktree) tea.Cmd {
	return func() tea.Msg {
		var err error
		var newStatus *git.BranchStatus

		switch opType {
		case "pull":
			err = m.gitOps.PullWorktree(wt.Path, false)
			if err == nil {
				newStatus, _ = m.gitOps.GetBranchStatus(wt.Path) //nolint:errcheck // Status refresh is best-effort
			}
		case "push":
			err = m.gitOps.PushWorktree(wt.Path, false)
			if err == nil {
				newStatus, _ = m.gitOps.GetBranchStatus(wt.Path) //nolint:errcheck // Status refresh is best-effort
			}
		case "delete":
			_, err = m.gitOps.RemoveWorktree(wt.Name, false, false)
		}

		return operationResultMsg{
			opType:     opType,
			targetName: wt.Name,
			branchName: wt.Branch,
			newStatus:  newStatus,
			err:        err,
		}
	}
}

// handleOperationResult processes the result of an async operation.
func (m *worktreeListModel) handleOperationResult(msg operationResultMsg) (tea.Model, tea.Cmd) {
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
			// Check if branch still exists and prompt to delete it
			if msg.branchName != "" {
				exists, err := m.gitOps.BranchExists(msg.branchName)
				if err == nil && exists {
					m.deletedBranchName = msg.branchName
					m.state = stateConfirmingBranchDelete
					// Refresh the worktree list first
					m.refreshWorktreeList()
					return m, nil
				}
			}
			// Branch doesn't exist or error checking, just refresh
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
func (m *worktreeListModel) refreshAfterDelete() (tea.Model, tea.Cmd) {
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
	m.worktrees = SortWorktrees(filtered, m.currentWorktree, m.trackedBranches)

	// Rebuild rows
	rows := make([]table.Row, 0, len(m.worktrees))
	for _, wt := range m.worktrees {
		status := m.branchStatuses[wt.Name]
		rows = append(rows, BuildWorktreeRow(wt, m.currentWorktree, m.trackedBranches, status))
	}
	m.table.SetRows(rows)

	// Update table height to match new row count
	m.table.SetHeight(CalculateTableHeight(m.ctx.Height, len(m.worktrees)))

	// Adjust cursor if needed
	if m.table.Cursor() >= len(m.worktrees) {
		m.table.SetCursor(max(0, len(m.worktrees)-1))
	}

	return m, m.scheduleClearMessage()
}

// refreshWorktreeList reloads the worktree list without scheduling a message clear.
// Used when transitioning to another state that will handle its own message.
func (m *worktreeListModel) refreshWorktreeList() {
	worktrees, err := m.gitOps.ListWorktrees(false)
	if err != nil {
		return
	}

	// Filter out bare worktrees and sort
	var filtered []git.Worktree
	for _, wt := range worktrees {
		if !wt.IsBare {
			filtered = append(filtered, wt)
		}
	}
	m.worktrees = SortWorktrees(filtered, m.currentWorktree, m.trackedBranches)

	// Rebuild rows
	rows := make([]table.Row, 0, len(m.worktrees))
	for _, wt := range m.worktrees {
		status := m.branchStatuses[wt.Name]
		rows = append(rows, BuildWorktreeRow(wt, m.currentWorktree, m.trackedBranches, status))
	}
	m.table.SetRows(rows)

	// Update table height to match new row count
	m.table.SetHeight(CalculateTableHeight(m.ctx.Height, len(m.worktrees)))

	// Adjust cursor if needed
	if m.table.Cursor() >= len(m.worktrees) {
		m.table.SetCursor(max(0, len(m.worktrees)-1))
	}
}

// updateOperatingRow updates the current row to show spinner.
func (m *worktreeListModel) updateOperatingRow() {
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

// updateLoadingRows updates rows that are still loading their status.
func (m *worktreeListModel) updateLoadingRows() {
	for i, wt := range m.worktrees {
		if m.loadingStatuses[wt.Name] {
			row := table.Row{
				FormatWorktreeName(wt, m.currentWorktree),
				wt.Branch,
				FormatWorktreeKind(wt, m.trackedBranches),
				m.spinner.View(),
			}
			m.updateRow(i, row)
		}
	}
}

// updateRow updates a specific row in the table.
func (m *worktreeListModel) updateRow(index int, row table.Row) {
	rows := m.table.Rows()
	if index < len(rows) {
		rows[index] = row
		m.table.SetRows(rows)
	}
}

// scheduleClearMessage returns a command to clear the message after 2 seconds.
func (m *worktreeListModel) scheduleClearMessage() tea.Cmd {
	return tea.Tick(2*time.Second, func(time.Time) tea.Msg {
		return clearMessageMsg{}
	})
}

// View implements tea.Model.
func (m *worktreeListModel) View() string {
	var output string

	// Render table
	output = m.table.View()

	// Show confirmation prompt if in confirming state
	if m.state == stateConfirming {
		// Note: Styles created here (not at package level) because the Lipgloss
		// renderer is configured at TUI startup time, after package initialization.
		confirmStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("208")).
			Bold(true)
		confirmMsg := fmt.Sprintf("\n\nDelete worktree '%s'? (y/n): ", m.operationTarget)
		output += confirmStyle.Render(confirmMsg)
		return output
	}

	// Show branch delete confirmation prompt
	if m.state == stateConfirmingBranchDelete {
		confirmStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("208")).
			Bold(true)
		messageStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("120"))
		output += messageStyle.Render("\n" + m.message)
		confirmMsg := fmt.Sprintf("\n\nAlso delete local branch '%s'? (y/n): ", m.deletedBranchName)
		output += confirmStyle.Render(confirmMsg)
		return output
	}

	// Show message if any
	if m.message != "" {
		messageStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("120"))
		output += messageStyle.Render("\n" + m.message)
	}

	// Show help text (skip if filter is active - table shows its own UI)
	if !m.table.IsFilterActive() {
		helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
		idx := m.table.OriginalIndex()
		showPush := true
		if idx >= 0 && idx < len(m.worktrees) {
			if m.trackedBranches[m.worktrees[idx].Branch] {
				showPush = false
			}
		}

		help := "\n↑/↓: navigate • /: filter • space/enter: switch • l: pull"
		if showPush {
			help += " • p: push"
		}
		help += " • d: delete • q/esc: quit\n"

		if m.state == stateOperating {
			help = "\n" + m.spinner.View() + " Operation in progress... (q to quit)\n"
		}

		output += helpStyle.Render(help)
	}

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

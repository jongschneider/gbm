package service

import (
	"fmt"
	"gbm/internal/git"

	"github.com/charmbracelet/bubbles/table"
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

// FormatWorktreeKind returns "tracked" or "ad hoc" based on whether the
// worktree's name is a configured slot. Matches sync's identity model:
// a worktree is gbm-managed when its directory name is a key in
// config.Worktrees, regardless of which branch it currently has checked
// out (the branch can drift between syncs).
func FormatWorktreeKind(wt git.Worktree, trackedNames map[string]bool) string {
	if trackedNames[wt.Name] {
		return "tracked"
	}
	return "ad hoc"
}

// BuildWorktreeRow creates a table row for a worktree using the shared formatting helpers.
func BuildWorktreeRow(wt git.Worktree, currentWorktree *git.Worktree, trackedNames map[string]bool, status *git.BranchStatus) table.Row {
	return table.Row{
		FormatWorktreeName(wt, currentWorktree),
		wt.Branch,
		FormatWorktreeKind(wt, trackedNames),
		FormatGitStatus(status),
	}
}

// CalculateTableColumns returns responsive column widths based on terminal width.
// Column ratios: Name 25% (min 15), Branch 45% (min 20), Kind 10% (min 8), Status 20% (min 10).
func CalculateTableColumns(terminalWidth int) []table.Column {
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
func CalculateTableHeight(terminalHeight, rowCount int) int {
	reservedLines := 4
	availableHeight := terminalHeight - reservedLines

	if rowCount+2 <= availableHeight {
		return rowCount + 2
	}
	return availableHeight
}

// SortWorktrees sorts worktrees by priority: current first, then tracked, then ad hoc.
// Bare worktrees are excluded from the result.
func SortWorktrees(worktrees []git.Worktree, currentWorktree *git.Worktree, trackedNames map[string]bool) []git.Worktree {
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
		if trackedNames[wt.Name] {
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

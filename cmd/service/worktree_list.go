package service

import (
	"fmt"

	"gbm/internal/git"

	"github.com/spf13/cobra"
)

func newWorktreeListCommand(svc *Service) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls", "l"},
		Short:   "List all worktrees",
		Long: `List all worktrees in the repository.

Examples:
  # List all worktrees
  gbm worktree list

  # List all worktrees in JSON format
  gbm --json worktree list`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			worktrees, err := svc.Git.ListWorktrees(ShouldUseDryRun())
			if err != nil {
				if ShouldUseJSON() {
					return HandleError(err.Error())
				}
				return err
			}

			if len(worktrees) == 0 {
				if ShouldUseJSON() {
					return OutputJSONArray([]map[string]interface{}{})
				}
				fmt.Println("No worktrees found.")
				return nil
			}

			// Get current worktree first
			currentWorktree, _ := svc.Git.GetCurrentWorktree()

			// Get config to identify tracked worktrees
			config := svc.GetConfig()

			// Create a map of tracked branches for quick lookup
			trackedBranches := make(map[string]bool)
			for _, wtConfig := range config.Worktrees {
				trackedBranches[wtConfig.Branch] = true
			}

			// Initialize sorted list and categorization lists
			sortedWorktrees := make([]git.Worktree, 0, len(worktrees))
			var trackedWorktrees []git.Worktree
			var adHocWorktrees []git.Worktree

			// Categorize worktrees: current first (if found), then tracked, then ad hoc (exclude bare)
			for _, wt := range worktrees {
				if wt.IsBare {
					// Skip bare repository
					continue
				}
				if currentWorktree != nil && wt.Name == currentWorktree.Name {
					// Add current worktree first and skip categorization
					sortedWorktrees = append(sortedWorktrees, wt)
					continue
				}

				if trackedBranches[wt.Branch] {
					trackedWorktrees = append(trackedWorktrees, wt)
					continue
				}
				adHocWorktrees = append(adHocWorktrees, wt)
			}

			// Append categorized worktrees in priority order: tracked, ad hoc
			sortedWorktrees = append(sortedWorktrees, trackedWorktrees...)
			sortedWorktrees = append(sortedWorktrees, adHocWorktrees...)

			// Handle JSON output
			if ShouldUseJSON() {
				// Convert worktrees to structured response
				wtList := make([]WorktreeListItemResponse, len(sortedWorktrees))
				for i, wt := range sortedWorktrees {
					isCurrent := currentWorktree != nil && wt.Name == currentWorktree.Name
					isTracked := trackedBranches[wt.Branch]
					wtList[i] = WorktreeListItemResponse{
						Name:    wt.Name,
						Path:    wt.Path,
						Branch:  wt.Branch,
						Current: isCurrent,
						Tracked: isTracked,
					}
				}
				response := WorktreeListResponse{
					Count:     len(wtList),
					Worktrees: wtList,
				}
				return OutputJSONArray(response)
			}

			// TUI table requires interactive input
			if !ShouldAllowInput() {
				return fmt.Errorf("TUI requires interactive input. Use 'gbm --json worktree list' for non-interactive output, or 'gbm worktree switch <name>' to switch directly")
			}

			// Display using bubbletea table
			return runWorktreeTable(sortedWorktrees, trackedBranches, currentWorktree, svc)
		},
	}

	return cmd
}

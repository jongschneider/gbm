package service

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gbm/internal/git"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
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
			// return runWorktreeTable(sortedWorktrees, trackedBranches, currentWorktree, svc)

			// Run TUI and handle output
			return handleWorktreeTableTUI(svc, svc.Git)
		},
	}

	return cmd
}

// newWorktreeTableTUI creates a worktree table TUI model with injected dependencies.
func newWorktreeTableTUI(
	cfgSvc WorktreeConfigService,
	gitOps WorktreeTableGitOps,
) (*worktreeListModel, error) {
	// Get worktrees
	worktrees, err := gitOps.ListWorktrees(false)
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	// Get tracked branches from config
	config := cfgSvc.GetConfig()
	trackedBranches := make(map[string]bool)
	for _, wtConfig := range config.Worktrees {
		trackedBranches[wtConfig.Branch] = true
	}

	// Get current worktree
	currentWorktree, _ := gitOps.GetCurrentWorktree()

	// Sort worktrees: current first, then tracked, then ad hoc (excludes bare)
	sorted := SortWorktrees(worktrees, currentWorktree, trackedBranches)

	// Fetch branch statuses concurrently
	branchStatuses := fetchBranchStatuses(sorted, gitOps)

	// Create model
	return newWorktreeListModel(sorted, trackedBranches, branchStatuses, currentWorktree, gitOps), nil
}

// handleWorktreeTableTUI runs the TUI and handles the final output.
func handleWorktreeTableTUI(svc WorktreeConfigService, gitOps WorktreeTableGitOps) error {
	// Open /dev/tty for TUI rendering FIRST, before creating any models/styles
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("failed to open /dev/tty: %w (TUI requires an interactive terminal)", err)
	}
	defer func() {
		_ = tty.Close()
	}()

	// Set up color renderer BEFORE creating the model, so styles are created with the correct renderer
	renderer := lipgloss.NewRenderer(tty,
		termenv.WithColorCache(true),
		termenv.WithTTY(true),
		termenv.WithProfile(termenv.TrueColor),
	)
	lipgloss.SetDefaultRenderer(renderer)

	// Create model with mocked dependencies
	m, err := newWorktreeTableTUI(svc, gitOps)
	if err != nil {
		return err
	}

	if len(m.worktrees) == 0 {
		fmt.Fprintln(os.Stderr, "No worktrees found")
		return nil
	}

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
	if model, ok := finalModel.(*worktreeListModel); ok {
		if model.switchOutput != "" {
			fmt.Println(model.switchOutput)
			fmt.Fprintf(os.Stderr, "✓ Selected worktree: %s\n", filepath.Base(model.switchOutput))
		}
	}

	return nil
}

// fetchBranchStatuses fetches branch statuses for all worktrees concurrently.
func fetchBranchStatuses(worktrees []git.Worktree, gitSvc WorktreeTableGitOps) map[string]*git.BranchStatus {
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

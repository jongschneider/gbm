package service

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"gbm/internal/git"
	"gbm/internal/jira"
	"gbm/internal/utils"

	"github.com/spf13/cobra"
)

func newWorktreeCommand(svc *Service) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "worktree",
		Aliases: []string{"wt"},
		Short:   "Manage git worktrees",
		Long:    `Create, list, and manage git worktrees.`,
	}

	cmd.AddCommand(newWorktreeAddCommand(svc))
	cmd.AddCommand(newWorktreeListCommand(svc))
	cmd.AddCommand(newWorktreeRemoveCommand(svc))
	cmd.AddCommand(newWorktreeSwitchCommand(svc))
	cmd.AddCommand(newWorktreePushCommand(svc))
	cmd.AddCommand(newWorktreePullCommand(svc))
	cmd.AddCommand(newWorktreeTestaddCommand(svc))
	cmd.AddCommand(newWorktreeTestlsCommand(svc))

	return cmd
}

func newWorktreeAddCommand(svc *Service) *cobra.Command {
	var (
		createBranch bool
		baseBranch   string
		visualizeFSM bool
		fsmGraphType string
	)

	cmd := &cobra.Command{
		Use:     "add [name] [branch]",
		Aliases: []string{"a"},
		Short:   "Add a new worktree in the worktrees directory",
		Long: `Add a new worktree in the worktrees directory for the given branch.
The worktree will be created at <repo-root>/worktrees/<name>.

When called with no arguments, launches an interactive TUI workflow.
When called with arguments, uses the traditional CLI mode.

Examples:
  # Interactive TUI mode
  gbm worktree add

  # CLI mode: Create worktree for existing branch
  gbm worktree add feature-x feature-x

  # CLI mode: Create worktree with new branch from current HEAD
  gbm worktree add feature-y feature-y -b

  # CLI mode: Create worktree with new branch from specific base
  gbm worktree add feature-z feature-z -b --base main`,
		Args: func(cmd *cobra.Command, args []string) error {
			// Accept 0 args (TUI mode) or 2 args (CLI mode)
			if len(args) == 0 || len(args) == 2 {
				return nil
			}
			return fmt.Errorf("accepts 0 or 2 arg(s), received %d", len(args))
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// If no args provided, launch TUI mode
			if len(args) == 0 {
				// TUI mode requires interactive input
				if !ShouldAllowInput() {
					if ShouldUseJSON() {
						return HandleError("TUI mode requires interactive input. Use 'gbm worktree add <name> <branch>' for non-interactive mode")
					}
					return fmt.Errorf("TUI mode requires interactive input. Use 'gbm worktree add <name> <branch>' for non-interactive mode")
				}
				return runWorktreeAddTUI(cmd, svc, visualizeFSM, fsmGraphType)
			}

			// CLI mode (existing behavior)
			worktreeName := args[0]
			branchName := args[1]

			// Get worktrees directory from service (reads from config)
			worktreesDir, err := svc.GetWorktreesPath()
			if err != nil {
				if ShouldUseJSON() {
					return HandleError(fmt.Sprintf("failed to get worktrees directory: %v", err))
				}
				return fmt.Errorf("failed to get worktrees directory: %w", err)
			}

			// Create worktrees directory if it doesn't exist
			if err := utils.MkdirAll(worktreesDir, ShouldUseDryRun()); err != nil {
				if ShouldUseJSON() {
					return HandleError(err.Error())
				}
				return err
			}

			// Flag override pattern: explicit flag > config > default
			baseBranch = utils.GetStringFlagOrConfig(cmd, "base", svc.GetConfig().DefaultBranch)
			if baseBranch == "" {
				baseBranch = "master" // Ultimate fallback
			}

			// Try to add the worktree
			wt, err := svc.Git.AddWorktree(worktreesDir, worktreeName, branchName, createBranch, baseBranch, ShouldUseDryRun())
			if err == nil {
				if !ShouldUseDryRun() {
					if ShouldUseJSON() {
						response := WorktreeAddResponse{
							Worktree: WorktreeResponse{
								Name:   wt.Name,
								Path:   wt.Path,
								Branch: wt.Branch,
							},
							Created: true,
						}
						return OutputJSONWithMessage(response, fmt.Sprintf("Created worktree '%s' for branch '%s'", wt.Name, wt.Branch))
					}

					// Text output: path to stdout, message to stderr
					fmt.Println(wt.Path)
					PrintSuccess(fmt.Sprintf("Created worktree '%s' for branch '%s'", wt.Name, wt.Branch))
				}
				return nil
			}

			// If it's not a "branch doesn't exist" error, or user already specified -b, return the error
			errMsg := err.Error()
			isBranchNotExist := strings.Contains(errMsg, "does not exist") || strings.Contains(errMsg, "invalid reference")
			if !isBranchNotExist || createBranch {
				if ShouldUseJSON() {
					return HandleError(errMsg)
				}
				return err
			}

			// Handle no-input mode
			if !ShouldAllowInput() {
				if ShouldUseJSON() {
					return HandleError(fmt.Sprintf("Branch '%s' does not exist and --no-input mode prevents prompting", branchName))
				}
				return fmt.Errorf("branch '%s' does not exist. Use -b to create it", branchName)
			}

			// Prompt user if they want to create a new branch
			fmt.Printf("Branch '%s' does not exist. Create it as a new branch? (y/N): ", branchName)

			reader := bufio.NewReader(os.Stdin)
			response, err := reader.ReadString('\n')
			if err != nil {
				if ShouldUseJSON() {
					return HandleError(fmt.Sprintf("failed to read user input: %v", err))
				}
				return fmt.Errorf("failed to read user input: %w", err)
			}

			response = strings.TrimSpace(strings.ToLower(response))
			if response != "y" && response != "yes" {
				return ErrBranchCreationCancelled
			}

			// Retry with createBranch = true
			wt, err = svc.Git.AddWorktree(worktreesDir, worktreeName, branchName, true, baseBranch, ShouldUseDryRun())
			if err != nil {
				if ShouldUseJSON() {
					return HandleError(err.Error())
				}
				return err
			}
			if !ShouldUseDryRun() {
				if ShouldUseJSON() {
					response := WorktreeAddResponse{
						Worktree: WorktreeResponse{
							Name:   wt.Name,
							Path:   wt.Path,
							Branch: wt.Branch,
						},
						Created: true,
					}
					return OutputJSONWithMessage(response, fmt.Sprintf("Created worktree '%s' for branch '%s'", wt.Name, wt.Branch))
				}

				// Text output: path to stdout, message to stderr
				fmt.Println(wt.Path)
				PrintSuccess(fmt.Sprintf("Created worktree '%s' for branch '%s'", wt.Name, wt.Branch))
			}
			return nil
		},
		PostRunE: func(cmd *cobra.Command, args []string) error {
			// Skip PostRunE in TUI mode (no args) - TUI already handles file copying
			if len(args) == 0 {
				return nil
			}
			if ShouldUseDryRun() {
				return nil
			}
			worktreeName := args[0]
			// Copy files from source worktrees based on config rules
			if err := svc.CopyFilesToWorktree(worktreeName); err != nil {
				PrintWarning(fmt.Sprintf("failed to copy files to worktree: %v", err))
			}
			// Create JIRA markdown if applicable
			if err := svc.CreateJiraMarkdownFile(worktreeName); err != nil {
				PrintWarning(fmt.Sprintf("failed to create JIRA markdown: %v", err))
			}
			return nil
		},
	}

	cmd.Flags().BoolVarP(&createBranch, "create-branch", "b", false, "Create a new branch for the worktree")
	cmd.Flags().StringVar(&baseBranch, "base", "", "Base branch to create new branch from (defaults to config.default_branch)")
	cmd.Flags().BoolVar(&visualizeFSM, "visualize-fsm", false, "Print FSM diagram before running TUI (TUI mode only)")
	cmd.Flags().StringVar(&fsmGraphType, "fsm-graph-type", "statediagram", "FSM graph type: 'statediagram' or 'flowchart' (default: statediagram)")

	// Add JIRA key completions for the first positional argument
	cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			// Complete JIRA keys with summaries for context
			filters := svc.GetJiraFilters()
			jiraIssues, err := svc.Jira.GetJiraIssues(filters, false)
			if err != nil {
				// If JIRA CLI is not available, fall back to no completions
				return nil, cobra.ShellCompDirectiveNoFileComp
			}

			var completions []string
			for _, issue := range jiraIssues {
				// Format: "KEY\tSummary" - clean completion of just the key with summary context
				completion := fmt.Sprintf("%s\t%s", issue.Key, issue.Summary)
				completions = append(completions, completion)
			}
			return completions, cobra.ShellCompDirectiveNoFileComp
		} else if len(args) == 1 {
			// Complete branch name based on the JIRA key
			worktreeName := args[0]
			if jira.IsJiraKey(worktreeName) {
				branchName, err := svc.Jira.GenerateBranchFromJira(worktreeName, false)
				if err != nil {
					// Fallback to default branch name generation
					branchName = fmt.Sprintf("%s%s", FeatureBranchPrefix, strings.ToLower(worktreeName))
				}
				return []string{branchName}, cobra.ShellCompDirectiveNoFileComp
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return cmd
}

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

func newWorktreeRemoveCommand(svc *Service) *cobra.Command {
	var (
		force bool
	)

	cmd := &cobra.Command{
		Use:     "remove <name>",
		Aliases: []string{"rm", "r"},
		Short:   "Remove a worktree from the worktrees directory",
		Long: `Remove a worktree from the worktrees directory.

The worktree directory is moved to Trash/Recycle Bin (with timestamp)
before removal, providing a safety mechanism to recover files if needed.

Use "." to remove the current worktree.

Examples:
  # Remove a worktree by name
  gbm worktree remove feature-x

  # Remove the current worktree
  gbm worktree remove .

  # Force remove a worktree (even if it has uncommitted changes)
  gbm worktree remove feature-x --force`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			worktreeName := args[0]

			// If "." is provided, get the current worktree name
			if worktreeName == "." {
				currentWorktree, err := svc.Git.GetCurrentWorktree()
				if err != nil {
					return fmt.Errorf("failed to get current worktree: %w", err)
				}
				worktreeName = currentWorktree.Name
			}

			// Check if we're trying to remove the current worktree
			// If so, we need to change directory first to avoid issues with subsequent git commands
			currentWorktree, err := svc.Git.GetCurrentWorktree()
			isCurrentWorktree := (err == nil && currentWorktree.Name == worktreeName)

			if isCurrentWorktree && !ShouldUseDryRun() {
				// Get current working directory
				cwd, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("failed to get current directory: %w", err)
				}

				// Get the repo root to switch to
				repoRoot, err := svc.Git.FindGitRoot(cwd)
				if err != nil {
					return fmt.Errorf("failed to find repository root: %w", err)
				}

				// Change to the repo root before removing the current worktree
				if err := os.Chdir(repoRoot); err != nil {
					return fmt.Errorf("failed to change directory to repo root: %w", err)
				}
				PrintInfo("Switching to repository root before removing current worktree")
			}

			// Remove the worktree (this validates it exists and returns its info)
			removedWorktree, err := svc.Git.RemoveWorktree(worktreeName, force, ShouldUseDryRun())
			if err != nil {
				return err
			}

			// If we're in dry-run mode or there's no branch, return early
			if ShouldUseDryRun() || removedWorktree.Branch == "" {
				return nil
			}

			branchName := removedWorktree.Branch

			// Handle no-input mode: use default (don't delete branch)
			if !ShouldAllowInput() {
				PrintMessage("Branch '%s' was not deleted (--no-input mode uses default: keep branch).\n", branchName)
				return nil
			}

			// Prompt to delete the branch
			fmt.Printf("Delete branch '%s' associated with this worktree? (y/N): ", branchName)

			reader := bufio.NewReader(os.Stdin)
			response, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read user input: %w", err)
			}

			response = strings.TrimSpace(strings.ToLower(response))
			if response != "y" && response != "yes" {
				PrintMessage("Branch '%s' was not deleted.\n", branchName)
				return nil
			}

			// Delete the branch
			if err := svc.Git.DeleteBranch(branchName, force, ShouldUseDryRun()); err != nil {
				return fmt.Errorf("failed to delete branch: %w", err)
			}
			PrintSuccess(fmt.Sprintf("Branch '%s' deleted successfully", branchName))

			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force removal even if worktree has uncommitted changes")

	// Add shell completions for worktree names
	cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			// Start with "." as a special option for current worktree
			completions := []string{"."}

			// Add all worktrees
			worktreeCompletions, err := generateWorktreeCompletions(svc)
			if err != nil {
				return completions, cobra.ShellCompDirectiveNoFileComp
			}

			completions = append(completions, worktreeCompletions...)
			return completions, cobra.ShellCompDirectiveNoFileComp
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return cmd
}

func newWorktreePushCommand(svc *Service) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "push [worktree-name]",
		Short: "Push worktree changes to remote",
		Long: `Push changes from a worktree to the remote repository.

Usage:
  gbm wt push                    # Push current worktree (if in a worktree)
  gbm wt push <worktree-name>    # Push specific worktree

The command will automatically set upstream (-u) if not already set.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return handlePushCurrent(svc)
			}

			// Handle "." as current worktree
			if args[0] == "." {
				return handlePushCurrent(svc)
			}

			return handlePushNamed(svc, args[0])
		},
	}

	// Add completion for worktree names
	cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		completions, err := generateWorktreeCompletions(svc)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		return completions, cobra.ShellCompDirectiveNoFileComp
	}

	return cmd
}

func newWorktreePullCommand(svc *Service) *cobra.Command {
	var pullAll bool

	cmd := &cobra.Command{
		Use:   "pull [worktree-name]",
		Short: "Pull worktree changes from remote",
		Long: `Pull changes from the remote repository to a worktree.

Usage:
  gbm wt pull                    # Pull current worktree (if in a worktree)
  gbm wt pull <worktree-name>    # Pull specific worktree
  gbm wt pull --all              # Pull all worktrees`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if pullAll {
				return handlePullAll(svc)
			}

			if len(args) == 0 {
				return handlePullCurrent(svc)
			}

			// Handle "." as current worktree
			if args[0] == "." {
				return handlePullCurrent(svc)
			}

			return handlePullNamed(svc, args[0])
		},
	}

	cmd.Flags().BoolVar(&pullAll, "all", false, "Pull all worktrees")

	// Add completion for worktree names
	cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		completions, err := generateWorktreeCompletions(svc)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		return completions, cobra.ShellCompDirectiveNoFileComp
	}

	return cmd
}

func newWorktreeSwitchCommand(svc *Service) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "switch <name>",
		Aliases: []string{"sw", "s"},
		Short:   "Switch to a worktree directory",
		Long: `Switch to a worktree directory.

This command outputs the path to the worktree. To actually change your shell's directory,
you need to use shell integration:

  # Setup (add to ~/.zshrc or ~/.bashrc)
  eval "$(gbm shell-integration)"

  # Then you can use
  gbm worktree switch feature-x

Use "-" to switch to the previous worktree (like "cd -"):
  gbm worktree switch -

Examples:
  # With shell integration enabled, switch to a worktree
  gbm worktree switch feature-x

  # Switch to the previous worktree
  gbm worktree switch -

  # Capture path without message
  path=$(gbm worktree switch feature-x 2>/dev/null)`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			worktreeName := args[0]

			// Get current worktree to track for history
			var currentWorktree *git.Worktree

			// First check if parent process passed the current worktree via env var
			// This is used when switching from the TUI table
			if envCurrentWt := os.Getenv("GBM_CURRENT_WORKTREE"); envCurrentWt != "" {
				// Look up the worktree by name to get full info
				wts, _ := svc.Git.ListWorktrees(false)
				for i, wt := range wts {
					if wt.Name == envCurrentWt {
						currentWorktree = &wts[i]
						break
					}
				}
			} else {
				// Otherwise, detect from working directory
				currentWorktree, _ = svc.Git.GetCurrentWorktree() // Ignore error if not in a worktree
			}

			// Handle "-" to switch to previous worktree
			if worktreeName == "-" {
				state := svc.GetState()
				if state.PreviousWorktree == "" {
					if ShouldUseJSON() {
						return HandleError(ErrNoPreviousWorktree.Error())
					}
					return ErrNoPreviousWorktree
				}
				worktreeName = state.PreviousWorktree
				PrintMessage("Switching to previous worktree: %s\n", worktreeName)
			}

			// List all worktrees to find the target
			worktrees, err := svc.Git.ListWorktrees(false)
			if err != nil {
				if ShouldUseJSON() {
					return HandleError(fmt.Sprintf("failed to list worktrees: %v", err))
				}
				return fmt.Errorf("failed to list worktrees: %w", err)
			}

			// Find the worktree by name
			var targetWorktree *git.Worktree
			for i, wt := range worktrees {
				if wt.Name == worktreeName {
					targetWorktree = &worktrees[i]
					break
				}
			}

			if targetWorktree == nil {
				if ShouldUseJSON() {
					return HandleError(fmt.Sprintf("worktree '%s' not found", worktreeName))
				}
				return fmt.Errorf("worktree '%s' not found", worktreeName)
			}

			// Update state to track worktree history
			state := svc.GetState()
			if currentWorktree != nil && currentWorktree.Name != targetWorktree.Name {
				// Save current worktree as previous before switching
				state.PreviousWorktree = currentWorktree.Name
			}
			state.CurrentWorktree = targetWorktree.Name
			_ = svc.SaveState() // Ignore errors - state tracking is optional

			// Handle output based on format
			if ShouldUseJSON() {
				response := WorktreeSwitchResponse{
					Worktree: WorktreeResponse{
						Name:   targetWorktree.Name,
						Path:   targetWorktree.Path,
						Branch: targetWorktree.Branch,
					},
					Previous: state.PreviousWorktree,
				}
				return OutputJSONWithMessage(response, fmt.Sprintf("Switched to worktree '%s'", worktreeName))
			}

			// Universal pattern: Always output path to stdout, messages to stderr
			// This enables shell integration without environment variables
			// Users can suppress the message with: gbm wt switch foo 2>/dev/null

			// Always output path to stdout (machine-readable)
			fmt.Println(targetWorktree.Path)

			// Always output message to stderr (human-readable)
			PrintSuccess(fmt.Sprintf("Switched to worktree '%s'", worktreeName))
			return nil
		},
	}

	// Add shell completions for worktree names
	cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			var completions []string

			// Add "-" option if there's a previous worktree
			state := svc.GetState()
			if state.PreviousWorktree != "" {
				completions = append(completions, fmt.Sprintf("-\t%s (previous)", state.PreviousWorktree))
			}

			// Add all worktrees
			worktreeCompletions, err := generateWorktreeCompletions(svc)
			if err != nil {
				return completions, cobra.ShellCompDirectiveNoFileComp
			}

			completions = append(completions, worktreeCompletions...)
			return completions, cobra.ShellCompDirectiveNoFileComp
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return cmd
}

// Handler functions for push and pull commands

func handlePushCurrent(svc *Service) error {
	// Get current working directory
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Check if we're in a worktree
	inWorktree, worktreeName, err := svc.Git.IsInWorktree(wd)
	if err != nil {
		return fmt.Errorf("failed to check if in worktree: %w", err)
	}

	if !inWorktree {
		return fmt.Errorf("not currently in a worktree. Use 'gbm wt push <worktree-name>' to push a specific worktree")
	}

	PrintInfo(fmt.Sprintf("Pushing current worktree '%s'", worktreeName))

	// Get the worktree path
	worktrees, err := svc.Git.ListWorktrees(false)
	if err != nil {
		return fmt.Errorf("failed to list worktrees: %w", err)
	}

	for _, wt := range worktrees {
		if wt.Name == worktreeName {
			return svc.Git.PushWorktree(wt.Path, false)
		}
	}

	return fmt.Errorf("worktree '%s' not found", worktreeName)
}

func handlePushNamed(svc *Service, worktreeName string) error {
	// List all worktrees to find the target
	worktrees, err := svc.Git.ListWorktrees(false)
	if err != nil {
		return fmt.Errorf("failed to list worktrees: %w", err)
	}

	// Find the worktree by name
	for _, wt := range worktrees {
		if wt.Name == worktreeName {
			if wt.IsBare {
				return fmt.Errorf("cannot push bare repository")
			}

			PrintInfo(fmt.Sprintf("Pushing worktree '%s'", worktreeName))
			return svc.Git.PushWorktree(wt.Path, false)
		}
	}

	return fmt.Errorf("worktree '%s' does not exist", worktreeName)
}

func handlePullAll(svc *Service) error {
	PrintInfo("Pulling all worktrees")

	worktrees, err := svc.Git.ListWorktrees(false)
	if err != nil {
		return fmt.Errorf("failed to get worktrees: %w", err)
	}

	hasErrors := false
	for _, wt := range worktrees {
		// Skip bare repository
		if wt.IsBare {
			continue
		}

		PrintMessage("Pulling worktree '%s'...\n", wt.Name)
		if err := svc.Git.PullWorktree(wt.Path, false); err != nil {
			PrintError("failed to pull worktree '%s': %v\n", wt.Name, err)
			hasErrors = true
			continue
		}
		PrintSuccess(fmt.Sprintf("Successfully pulled worktree '%s'", wt.Name))
	}

	if hasErrors {
		return fmt.Errorf("some worktrees failed to pull")
	}

	return nil
}

func handlePullCurrent(svc *Service) error {
	// Get current working directory
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Check if we're in a worktree
	inWorktree, worktreeName, err := svc.Git.IsInWorktree(wd)
	if err != nil {
		return fmt.Errorf("failed to check if in worktree: %w", err)
	}

	if !inWorktree {
		return fmt.Errorf("not currently in a worktree. Use 'gbm wt pull <worktree-name>' to pull a specific worktree")
	}

	PrintMessage("Pulling current worktree '%s'...\n", worktreeName)

	// Get the worktree path
	worktrees, err := svc.Git.ListWorktrees(false)
	if err != nil {
		return fmt.Errorf("failed to list worktrees: %w", err)
	}

	for _, wt := range worktrees {
		if wt.Name == worktreeName {
			return svc.Git.PullWorktree(wt.Path, false)
		}
	}

	return fmt.Errorf("worktree '%s' not found", worktreeName)
}

func handlePullNamed(svc *Service, worktreeName string) error {
	// List all worktrees to find the target
	worktrees, err := svc.Git.ListWorktrees(false)
	if err != nil {
		return fmt.Errorf("failed to list worktrees: %w", err)
	}

	// Find the worktree by name
	for _, wt := range worktrees {
		if wt.Name == worktreeName {
			if wt.IsBare {
				return fmt.Errorf("cannot pull bare repository")
			}

			PrintMessage("Pulling worktree '%s'...\n", worktreeName)
			return svc.Git.PullWorktree(wt.Path, false)
		}
	}

	return fmt.Errorf("worktree '%s' does not exist", worktreeName)
}

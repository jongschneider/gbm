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

	return cmd
}

func newWorktreeAddCommand(svc *Service) *cobra.Command {
	var (
		createBranch bool
		baseBranch   string
		dryRun       bool
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
				return runWorktreeAddTUI(cmd, svc, visualizeFSM, fsmGraphType)
			}

			// CLI mode (existing behavior)
			worktreeName := args[0]
			branchName := args[1]

			// Get worktrees directory from service (reads from config)
			worktreesDir, err := svc.GetWorktreesPath()
			if err != nil {
				return fmt.Errorf("failed to get worktrees directory: %w", err)
			}

			// Create worktrees directory if it doesn't exist
			if err := utils.MkdirAll(worktreesDir, dryRun); err != nil {
				return err
			}

			// Try to add the worktree
			wt, err := svc.Git.AddWorktree(worktreesDir, worktreeName, branchName, createBranch, baseBranch, dryRun)
			if err == nil {
				if !dryRun {
					fmt.Printf("Created worktree '%s' at %s for branch '%s'\n", wt.Name, wt.Path, wt.Branch)
				}
				return nil
			}

			// If it's not a "branch doesn't exist" error, or user already specified -b, return the error
			errMsg := err.Error()
			isBranchNotExist := strings.Contains(errMsg, "does not exist") || strings.Contains(errMsg, "invalid reference")
			if !isBranchNotExist || createBranch {
				return err
			}

			// Prompt user if they want to create a new branch
			fmt.Printf("Branch '%s' does not exist. Create it as a new branch? (y/N): ", branchName)

			reader := bufio.NewReader(os.Stdin)
			response, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read user input: %w", err)
			}

			response = strings.TrimSpace(strings.ToLower(response))
			if response != "y" && response != "yes" {
				return ErrBranchCreationCancelled
			}

			// Retry with createBranch = true
			wt, err = svc.Git.AddWorktree(worktreesDir, worktreeName, branchName, true, baseBranch, dryRun)
			if err != nil {
				return err
			}
			if !dryRun {
				fmt.Printf("Created worktree '%s' at %s for branch '%s'\n", wt.Name, wt.Path, wt.Branch)
			}
			return nil
		},
		PostRunE: func(cmd *cobra.Command, args []string) error {
			// Skip PostRunE in TUI mode (no args) - TUI already handles file copying
			if len(args) == 0 {
				return nil
			}
			if dryRun {
				return nil
			}
			worktreeName := args[0]
			// Copy files from source worktrees based on config rules
			if err := svc.CopyFilesToWorktree(worktreeName); err != nil {
				fmt.Printf("Warning: failed to copy files to worktree: %v\n", err)
			}
			return nil
		},
	}

	cmd.Flags().BoolVarP(&createBranch, "create-branch", "b", false, "Create a new branch for the worktree")
	cmd.Flags().StringVar(&baseBranch, "base", "", "Base branch to create new branch from (defaults to 'main')")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print commands without executing them")
	cmd.Flags().BoolVar(&visualizeFSM, "visualize-fsm", false, "Print FSM diagram before running TUI (TUI mode only)")
	cmd.Flags().StringVar(&fsmGraphType, "fsm-graph-type", "statediagram", "FSM graph type: 'statediagram' or 'flowchart' (default: statediagram)")

	// Add JIRA key completions for the first positional argument
	cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			// Complete JIRA keys with summaries for context
			jiraIssues, err := svc.Jira.GetJiraIssues(false)
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
	var dryRun bool

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls", "l"},
		Short:   "List all worktrees",
		Long: `List all worktrees in the repository.

Examples:
  # List all worktrees
  gbm worktree list`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			worktrees, err := svc.Git.ListWorktrees(dryRun)
			if err != nil {
				return err
			}

			if len(worktrees) == 0 {
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

			// Display using bubbletea table
			return runWorktreeTable(sortedWorktrees, trackedBranches, currentWorktree, svc)
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print commands without executing them")

	return cmd
}

func newWorktreeRemoveCommand(svc *Service) *cobra.Command {
	var (
		force  bool
		dryRun bool
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

			if isCurrentWorktree && !dryRun {
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
				fmt.Printf("Switching to repository root before removing current worktree...\n")
			}

			// Remove the worktree (this validates it exists and returns its info)
			removedWorktree, err := svc.Git.RemoveWorktree(worktreeName, force, dryRun)
			if err != nil {
				return err
			}

			// If we're in dry-run mode or there's no branch, return early
			if dryRun || removedWorktree.Branch == "" {
				return nil
			}

			branchName := removedWorktree.Branch

			// Prompt to delete the branch
			fmt.Printf("Delete branch '%s' associated with this worktree? (y/N): ", branchName)

			reader := bufio.NewReader(os.Stdin)
			response, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read user input: %w", err)
			}

			response = strings.TrimSpace(strings.ToLower(response))
			if response != "y" && response != "yes" {
				fmt.Printf("Branch '%s' was not deleted.\n", branchName)
				return nil
			}

			// Delete the branch
			if err := svc.Git.DeleteBranch(branchName, force, dryRun); err != nil {
				return fmt.Errorf("failed to delete branch: %w", err)
			}
			fmt.Printf("Branch '%s' deleted successfully.\n", branchName)

			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force removal even if worktree has uncommitted changes")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print commands without executing them")

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
	var printPath bool

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
  # Print the path to a worktree
  gbm worktree switch --print-path feature-x

  # With shell integration enabled, switch to a worktree
  gbm worktree switch feature-x

  # Switch to the previous worktree
  gbm worktree switch -`,
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
					return ErrNoPreviousWorktree
				}
				worktreeName = state.PreviousWorktree
				fmt.Printf("Switching to previous worktree: %s\n", worktreeName)
			}

			// List all worktrees to find the target
			worktrees, err := svc.Git.ListWorktrees(false)
			if err != nil {
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
				return fmt.Errorf("worktree '%s' not found", worktreeName)
			}

			// If --print-path is set, just print the path
			if printPath {
				fmt.Print(targetWorktree.Path)
				return nil
			}

			// Update state to track worktree history
			state := svc.GetState()
			if currentWorktree != nil && currentWorktree.Name != targetWorktree.Name {
				// Save current worktree as previous before switching
				state.PreviousWorktree = currentWorktree.Name
			}
			state.CurrentWorktree = targetWorktree.Name
			_ = svc.SaveState() // Ignore errors - state tracking is optional

			// Check if shell integration is enabled
			if os.Getenv("GBM_SHELL_INTEGRATION") != "" {
				// Output cd command for shell wrapper to execute
				fmt.Printf("cd %s\n", targetWorktree.Path)
				return nil
			}

			// No shell integration, output instructions
			fmt.Printf("To switch to worktree '%s':\n", worktreeName)
			fmt.Printf("  cd %s\n\n", targetWorktree.Path)
			fmt.Printf("To enable automatic directory switching, set up shell integration:\n")
			fmt.Printf("  eval \"$(gbm shell-integration)\"\n")
			return nil
		},
	}

	cmd.Flags().BoolVar(&printPath, "print-path", false, "Print only the path to the worktree")

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

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

	return cmd
}

func newWorktreeAddCommand(svc *Service) *cobra.Command {
	var (
		createBranch bool
		baseBranch   string
		dryRun       bool
	)

	cmd := &cobra.Command{
		Use:     "add <name> <branch>",
		Aliases: []string{"a"},
		Short:   "Add a new worktree in the worktrees directory",
		Long: `Add a new worktree in the worktrees directory for the given branch.
The worktree will be created at <repo-root>/worktrees/<name>.

Examples:
  # Create worktree for existing branch
  gbm worktree add feature-x feature-x

  # Create worktree with new branch from current HEAD
  gbm worktree add feature-y feature-y -b

  # Create worktree with new branch from specific base
  gbm worktree add feature-z feature-z -b --base main`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
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
				return fmt.Errorf("branch creation cancelled")
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
	}

	cmd.Flags().BoolVarP(&createBranch, "create-branch", "b", false, "Create a new branch for the worktree")
	cmd.Flags().StringVar(&baseBranch, "base", "", "Base branch to create new branch from (defaults to 'main')")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print commands without executing them")

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
					branchName = fmt.Sprintf("feature/%s", strings.ToLower(worktreeName))
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

			// Format and print worktrees
			for _, wt := range worktrees {
				status := ""
				if wt.IsBare {
					status = "(bare)"
				} else if wt.Branch != "" {
					status = fmt.Sprintf("[%s]", wt.Branch)
				}
				fmt.Printf("%s  %s %s\n", wt.Path, wt.Commit, status)
			}
			return nil
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
			// List all worktrees
			worktrees, err := svc.Git.ListWorktrees(false)
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}

			// Start with "." as a special option for current worktree
			completions := []string{"."}

			// Add all non-bare worktrees
			for _, wt := range worktrees {
				// Exclude the bare repo
				if wt.IsBare {
					continue
				}
				// Add completion with branch context
				completion := wt.Name
				if wt.Branch != "" {
					completion = fmt.Sprintf("%s\t%s", wt.Name, wt.Branch)
				}
				completions = append(completions, completion)
			}

			return completions, cobra.ShellCompDirectiveNoFileComp
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return cmd
}

func newWorktreeSwitchCommand(svc *Service) *cobra.Command {
	var (
		printPath bool
	)

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

Examples:
  # Print the path to a worktree
  gbm worktree switch --print-path feature-x

  # With shell integration enabled, switch to a worktree
  gbm worktree switch feature-x`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			worktreeName := args[0]

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
			// List all worktrees
			worktrees, err := svc.Git.ListWorktrees(false)
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}

			var completions []string
			// Add all non-bare worktrees
			for _, wt := range worktrees {
				// Exclude the bare repo
				if wt.IsBare {
					continue
				}
				// Add completion with branch context
				completion := wt.Name
				if wt.Branch != "" {
					completion = fmt.Sprintf("%s\t%s", wt.Name, wt.Branch)
				}
				completions = append(completions, completion)
			}

			return completions, cobra.ShellCompDirectiveNoFileComp
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return cmd
}

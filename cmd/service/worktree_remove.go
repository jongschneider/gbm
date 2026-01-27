package service

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func newWorktreeRemoveCommand(svc *Service) *cobra.Command {
	var force bool

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
			return runWorktreeRemove(svc, args[0], force)
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

// runWorktreeRemove handles the worktree removal logic.
func runWorktreeRemove(svc *Service, worktreeName string, force bool) error {
	// Resolve "." to current worktree
	if worktreeName == "." {
		currentWorktree, err := svc.Git.GetCurrentWorktree()
		if err != nil {
			return fmt.Errorf("failed to get current worktree: %w", err)
		}
		worktreeName = currentWorktree.Name
	}

	// Handle current worktree removal by changing directory first
	if err := prepareCurrentWorktreeRemoval(svc, worktreeName); err != nil {
		return err
	}

	// Remove the worktree
	removedWorktree, err := svc.Git.RemoveWorktree(worktreeName, force, ShouldUseDryRun())
	if err != nil {
		return err
	}

	if ShouldUseDryRun() || removedWorktree.Branch == "" {
		return nil
	}

	// Prompt to delete associated branch
	return promptDeleteBranch(svc, removedWorktree.Branch, force)
}

// prepareCurrentWorktreeRemoval changes to repo root if removing current worktree.
func prepareCurrentWorktreeRemoval(svc *Service, worktreeName string) error {
	currentWorktree, err := svc.Git.GetCurrentWorktree()
	if err != nil || currentWorktree.Name != worktreeName || ShouldUseDryRun() {
		return nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	repoRoot, err := svc.Git.FindGitRoot(cwd)
	if err != nil {
		return fmt.Errorf("failed to find repository root: %w", err)
	}

	if err := os.Chdir(repoRoot); err != nil {
		return fmt.Errorf("failed to change directory to repo root: %w", err)
	}
	PrintInfo("Switching to repository root before removing current worktree")
	return nil
}

// promptDeleteBranch prompts user to delete the branch associated with the removed worktree.
func promptDeleteBranch(svc *Service, branchName string, force bool) error {
	if !ShouldAllowInput() {
		PrintMessage("Branch '%s' was not deleted (--no-input mode uses default: keep branch).\n", branchName)
		return nil
	}

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

	if err := svc.Git.DeleteBranch(branchName, force, ShouldUseDryRun()); err != nil {
		return fmt.Errorf("failed to delete branch: %w", err)
	}
	PrintSuccess(fmt.Sprintf("Branch '%s' deleted successfully", branchName))
	return nil
}

package service

import (
	"fmt"
	"os"

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
			if dryRun {
				fmt.Printf("[DRY RUN] mkdir -p %s\n", worktreesDir)
			} else {
				if err := os.MkdirAll(worktreesDir, 0755); err != nil {
					return fmt.Errorf("failed to create worktrees directory: %w", err)
				}
			}

			return svc.Git.AddWorktree(worktreesDir, worktreeName, branchName, createBranch, baseBranch, dryRun)
		},
	}

	cmd.Flags().BoolVarP(&createBranch, "create-branch", "b", false, "Create a new branch for the worktree")
	cmd.Flags().StringVar(&baseBranch, "base", "", "Base branch to create new branch from (defaults to 'main')")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print commands without executing them")

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
			output, err := svc.Git.ListWorktrees(dryRun)
			if err != nil {
				return err
			}

			fmt.Print(output)
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
				worktreeName = currentWorktree
			}

			// Get worktrees directory from service (reads from config)
			worktreesDir, err := svc.GetWorktreesPath()
			if err != nil {
				return fmt.Errorf("failed to get worktrees directory: %w", err)
			}

			return svc.Git.RemoveWorktree(worktreesDir, worktreeName, force, dryRun)
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force removal even if worktree has uncommitted changes")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print commands without executing them")

	return cmd
}

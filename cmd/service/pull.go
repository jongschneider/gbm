package service

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newPullCommand(svc *Service) *cobra.Command {
	var pullAll bool

	cmd := &cobra.Command{
		Use:   "pull [worktree-name]",
		Short: "Pull worktree changes from remote",
		Long: `Pull changes from the remote repository to a worktree.

Usage:
  gbm pull                    # Pull current worktree (if in a worktree)
  gbm pull <worktree-name>    # Pull specific worktree
  gbm pull --all              # Pull all worktrees`,
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

func handlePullAll(svc *Service) error {
	fmt.Println("Pulling all worktrees...")

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

		fmt.Printf("Pulling worktree '%s'...\n", wt.Name)
		if err := svc.Git.PullWorktree(wt.Path, false); err != nil {
			fmt.Printf("Failed to pull worktree '%s': %v\n", wt.Name, err)
			hasErrors = true
			continue
		}
		fmt.Printf("Successfully pulled worktree '%s'\n", wt.Name)
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
		return fmt.Errorf("not currently in a worktree. Use 'gbm pull <worktree-name>' to pull a specific worktree")
	}

	fmt.Printf("Pulling current worktree '%s'...\n", worktreeName)

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

			fmt.Printf("Pulling worktree '%s'...\n", worktreeName)
			return svc.Git.PullWorktree(wt.Path, false)
		}
	}

	return fmt.Errorf("worktree '%s' does not exist", worktreeName)
}

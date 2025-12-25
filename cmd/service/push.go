package service

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newPushCommand(svc *Service) *cobra.Command {
	var pushAll bool

	cmd := &cobra.Command{
		Use:   "push [worktree-name]",
		Short: "Push worktree changes to remote",
		Long: `Push changes from a worktree to the remote repository.

Usage:
  gbm push                    # Push current worktree (if in a worktree)
  gbm push <worktree-name>    # Push specific worktree
  gbm push --all              # Push all worktrees

The command will automatically set upstream (-u) if not already set.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if pushAll {
				return handlePushAll(svc)
			}

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

	cmd.Flags().BoolVar(&pushAll, "all", false, "Push all worktrees")

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

func handlePushAll(svc *Service) error {
	fmt.Println("Pushing all worktrees...")

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

		fmt.Printf("Pushing worktree '%s'...\n", wt.Name)
		if err := svc.Git.PushWorktree(wt.Path, false); err != nil {
			fmt.Printf("Failed to push worktree '%s': %v\n", wt.Name, err)
			hasErrors = true
			continue
		}
		fmt.Printf("Successfully pushed worktree '%s'\n", wt.Name)
	}

	if hasErrors {
		return fmt.Errorf("some worktrees failed to push")
	}

	return nil
}

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
		return fmt.Errorf("not currently in a worktree. Use 'gbm push <worktree-name>' to push a specific worktree")
	}

	fmt.Printf("Pushing current worktree '%s'...\n", worktreeName)

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

			fmt.Printf("Pushing worktree '%s'...\n", worktreeName)
			return svc.Git.PushWorktree(wt.Path, false)
		}
	}

	return fmt.Errorf("worktree '%s' does not exist", worktreeName)
}

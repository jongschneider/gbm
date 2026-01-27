package service

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

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
		return errors.New("not currently in a worktree. Use 'gbm wt push <worktree-name>' to push a specific worktree")
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
				return errors.New("cannot push bare repository")
			}

			PrintInfo(fmt.Sprintf("Pushing worktree '%s'", worktreeName))
			return svc.Git.PushWorktree(wt.Path, false)
		}
	}

	return fmt.Errorf("worktree '%s' does not exist", worktreeName)
}

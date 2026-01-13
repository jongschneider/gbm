package service

import (
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

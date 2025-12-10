package service

import (
	"gbm/internal/git"

	"github.com/spf13/cobra"
)

func newInitCommand(gitSvc *git.Service) *cobra.Command {
	var (
		name              string
		defaultBranchName string
		dryRun            bool
	)

	cmd := &cobra.Command{
		Use:   "init [name]",
		Short: "Initialize a new git repository with worktree structure",
		Long: `Initialize a new git repository with worktree structure:
- <name>/.git (bare repository)
- <name>/worktrees/<defaultBranch>/ (main worktree)
- <name>/.gbm/config.yaml (configuration file)

If name is not provided, initializes in the current directory.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				name = args[0]
			}

			return gitSvc.Init(name, defaultBranchName, dryRun)
		},
	}

	cmd.Flags().StringVarP(&defaultBranchName, "branch", "b", "main", "Default branch name")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print commands without executing them")

	return cmd
}

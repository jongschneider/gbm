package service

import (
	"github.com/spf13/cobra"
)

func newCloneCommand(svc *Service) *cobra.Command {
	var (
		name   string
		dryRun bool
	)

	cmd := &cobra.Command{
		Use:   "clone <repo-url> [name]",
		Short: "Clone a remote git repository with worktree structure",
		Long: `Clone a remote git repository with worktree structure:
- <name>/.git (bare repository)
- <name>/worktrees/<defaultBranch>/ (main worktree)
- <name>/.gbm/config.yaml (configuration file)

If name is not provided, extracts the repository name from the URL.`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			repoURL := args[0]
			if len(args) > 1 {
				name = args[1]
			}

			return svc.Git.Clone(repoURL, name, dryRun)
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print commands without executing them")

	return cmd
}

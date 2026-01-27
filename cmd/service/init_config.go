package service

import (
	"fmt"
	"gbm/internal/git"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func newInitConfigCommand() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "init-config",
		Short: "Generate example configuration file",
		Long: `Generate an example .gbm/config.yaml file with comments explaining all options.

The generated config includes examples for:
  • Git remotes
  • JIRA integration
  • File copying rules
  • Path templates

By default, fails if config already exists. Use --force to overwrite.`,
		Example: `  # Generate example config
  gbm init-config

  # Overwrite existing config
  gbm init-config --force`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Find git root
			gitSvc := git.NewService()
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current directory: %w", err)
			}

			gitRoot, err := gitSvc.FindGitRoot(cwd)
			if err != nil {
				return fmt.Errorf("not in a git repository: %w", err)
			}

			configPath := filepath.Join(gitRoot, ".gbm", "config.yaml")

			// Check if exists (unless --force)
			if !force {
				if _, err := os.Stat(configPath); err == nil {
					return fmt.Errorf("config already exists at %s\nUse --force to overwrite", configPath)
				}
			} else {
				// With --force, remove existing config if it exists
				_ = os.Remove(configPath)
			}

			// Generate config
			if err := GenerateExampleConfig(configPath); err != nil {
				return err
			}

			// Success message
			defaultBranch := getDefaultBranch()
			fmt.Fprintf(os.Stderr, "✓ Created example config at %s\n\n", configPath)
			fmt.Fprintf(os.Stderr, "Configured with:\n")
			fmt.Fprintf(os.Stderr, "  • Default branch: %s (detected from git config)\n\n", defaultBranch)
			fmt.Fprintf(os.Stderr, "Edit the file to configure:\n")
			fmt.Fprintf(os.Stderr, "  • Git remotes\n")
			fmt.Fprintf(os.Stderr, "  • JIRA integration (optional)\n")
			fmt.Fprintf(os.Stderr, "  • File copying rules (optional)\n\n")
			fmt.Fprintf(os.Stderr, "Next steps:\n")
			fmt.Fprintf(os.Stderr, "  1. Edit %s\n", configPath)
			fmt.Fprintf(os.Stderr, "  2. Run: gbm init (if creating new repo)\n")

			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing config")

	return cmd
}

package service

import (
	"os"

	"github.com/spf13/cobra"
)

var logFile *os.File

func newRootCommand() *cobra.Command {
	// Create flags struct to bind command-line flags
	var flags CLIFlags

	rootCmd := &cobra.Command{
		Use:   "gbm2",
		Short: "Git Branch Manager - Manage Git worktrees based on .gbm/config.yaml",
		Long: `Git Branch Manager (gbm2) is a CLI tool that manages Git repository branches
and worktrees based on configuration defined in .gbm/config.yaml.

The tool synchronizes local worktrees with branch definitions and provides
notifications when configurations drift out of sync.`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Make flags available to all subcommands
			SetGlobalFlags(&flags)
		},
	}

	// Register persistent flags (available to all subcommands)
	rootCmd.PersistentFlags().BoolVarP(&flags.JSON, "json", "j", false, "Output in JSON format")
	rootCmd.PersistentFlags().BoolVar(&flags.NoColor, "no-color", false, "Disable colored output (also respects NO_COLOR environment variable)")
	rootCmd.PersistentFlags().BoolVarP(&flags.Quiet, "quiet", "q", false, "Suppress non-essential messages (errors still shown)")
	rootCmd.PersistentFlags().BoolVar(&flags.NoInput, "no-input", false, "Disable interactive prompts (uses defaults)")
	rootCmd.PersistentFlags().BoolVar(&flags.DryRun, "dry-run", false, "Preview operations without executing them")
	rootCmd.PersistentFlags().BoolVarP(&flags.Verbose, "verbose", "v", false, "Enable verbose output")

	// Create service with git and config
	svc := NewService()

	// Add all subcommands
	rootCmd.AddCommand(newInitCommand(svc))
	rootCmd.AddCommand(newInitConfigCommand())
	rootCmd.AddCommand(newCloneCommand(svc))
	rootCmd.AddCommand(newWorktreeCommand(svc))
	rootCmd.AddCommand(newSyncCommand(svc))
	rootCmd.AddCommand(newConfigCommand(svc))
	rootCmd.AddCommand(newShellIntegrationCommand())

	// Add completion command for shell completions
	rootCmd.CompletionOptions.HiddenDefaultCmd = false

	return rootCmd
}

// Execute runs the root command.
func Execute() error {
	return newRootCommand().Execute()
}

// CloseLogFile closes the log file if it was opened.
func CloseLogFile() {
	if logFile != nil {
		//nolint:errcheck // Best-effort cleanup during shutdown
		logFile.Close()
	}
}

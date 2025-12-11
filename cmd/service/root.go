package service

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	logFile *os.File
)

func newRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "gbm",
		Short: "Git Branch Manager - Manage Git worktrees based on .gbm/config.yaml",
		Long: `Git Branch Manager (gbm) is a CLI tool that manages Git repository branches
and worktrees based on configuration defined in .gbm/config.yaml.

The tool synchronizes local worktrees with branch definitions and provides
notifications when configurations drift out of sync.`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Initialize logging or other common setup here if needed
		},
	}

	// Create service with git and config
	svc := NewService()

	// Add all subcommands
	rootCmd.AddCommand(newInitCommand(svc))
	rootCmd.AddCommand(newCloneCommand(svc))
	rootCmd.AddCommand(newWorktreeCommand(svc))

	return rootCmd
}

// Execute runs the root command
func Execute() error {
	return newRootCommand().Execute()
}

// CloseLogFile closes the log file if it was opened
func CloseLogFile() {
	if logFile != nil {
		_ = logFile.Close()
	}
}

// PrintError prints an error message to stderr
func PrintError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}
}

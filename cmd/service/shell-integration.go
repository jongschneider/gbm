package service

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newShellIntegrationCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "shell-integration",
		Short: "Generate shell integration for directory changing",
		Long: `Generate shell integration code that allows 'gbm worktree switch' to change your shell's directory.

This outputs shell code that should be evaluated in your shell startup file.

Setup:
  # For Zsh (add to ~/.zshrc)
  eval "$(gbm shell-integration)"

  # For Bash (add to ~/.bashrc)
  eval "$(gbm shell-integration)"

After setup, 'gbm worktree switch <name>' will automatically cd to the worktree directory.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Output the shell wrapper function
			fmt.Print(shellIntegrationScript)
			return nil
		},
	}

	return cmd
}

const shellIntegrationScript = `# gbm shell integration

gbm2() {
    # All worktree commands that output a path to stdout
    # Handles: switch, sw, s, add, a, list, ls, l (and their worktree/wt forms)
    if [[ ("$1" = "worktree" || "$1" = "wt") && \
          ("$2" = "switch" || "$2" = "sw" || "$2" = "s" || \
           "$2" = "add" || "$2" = "a" || \
           "$2" = "list" || "$2" = "ls" || "$2" = "l") ]]; then

        # Capture stdout (path) while letting stderr through for messages
        local result
        result=$(command gbm2 "$@" 2>/dev/stderr)
        local exit_code=$?

        # If successful and result is a directory, cd to it
        if [ $exit_code -eq 0 ] && [ -n "$result" ] && [ -d "$result" ]; then
            cd "$result"
        fi

        return $exit_code

    # All other commands - pass through unchanged
    else
        command gbm2 "$@"
    fi
}
`

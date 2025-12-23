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
export GBM_SHELL_INTEGRATION=1

gbm2() {
    if [ "$1" = "worktree" ] && [ "$2" = "switch" ] && [ $# -gt 2 ]; then
        local cmd_output=$(command gbm2 "$@" 2>/dev/null)
        if [ $? -eq 0 ]; then
            # Extract the cd command from the output (might have other text before it)
            local cd_cmd=$(echo "$cmd_output" | grep '^cd ')
            if [ -n "$cd_cmd" ]; then
                eval "$cd_cmd"  # Execute cd in current shell
            else
                echo "$cmd_output"
            fi
        else
            command gbm2 "$@"
        fi
    elif [ "$1" = "wt" ] && [ "$2" = "switch" ] && [ $# -gt 2 ]; then
        # Support 'wt' alias as well
        local cmd_output=$(command gbm2 "$@" 2>/dev/null)
        if [ $? -eq 0 ]; then
            # Extract the cd command from the output (might have other text before it)
            local cd_cmd=$(echo "$cmd_output" | grep '^cd ')
            if [ -n "$cd_cmd" ]; then
                eval "$cd_cmd"  # Execute cd in current shell
            else
                echo "$cmd_output"
            fi
        else
            command gbm2 "$@"
        fi
    elif [ "$1" = "worktree" ] && [ "$2" = "list" ]; then
        # Handle 'worktree list' interactive switching
        command gbm2 "$@"
        local exit_code=$?
        # Check for a temp file containing cd command (written by the Go program)
        local switch_file="$TMPDIR/.gbm-switch-$$"
        if [ $exit_code -eq 0 ] && [ -f "$switch_file" ]; then
            local cd_cmd=$(cat "$switch_file" 2>/dev/null | grep '^cd ')
            if [ -n "$cd_cmd" ]; then
                eval "$cd_cmd"
            fi
            rm -f "$switch_file"
        fi
    elif [ "$1" = "wt" ] && ([ "$2" = "list" ] || [ "$2" = "ls" ] || [ "$2" = "l" ]); then
        # Support 'wt list/ls/l' aliases
        command gbm2 "$@"
        local exit_code=$?
        # Check for a temp file containing cd command (written by the Go program)
        local switch_file="$TMPDIR/.gbm-switch-$$"
        if [ $exit_code -eq 0 ] && [ -f "$switch_file" ]; then
            local cd_cmd=$(cat "$switch_file" 2>/dev/null | grep '^cd ')
            if [ -n "$cd_cmd" ]; then
                eval "$cd_cmd"
            fi
            rm -f "$switch_file"
        fi
    else
        command gbm2 "$@"
    fi
}
`

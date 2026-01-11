package service

import (
	"fmt"
	"os"

	"gbm/internal/git"

	"github.com/spf13/cobra"
)

func newWorktreeSwitchCommand(svc *Service) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "switch <name>",
		Aliases: []string{"sw", "s"},
		Short:   "Switch to a worktree directory",
		Long: `Switch to a worktree directory.

This command outputs the path to the worktree. To actually change your shell's directory,
you need to use shell integration:

  # Setup (add to ~/.zshrc or ~/.bashrc)
  eval "$(gbm shell-integration)"

  # Then you can use
  gbm worktree switch feature-x

Use "-" to switch to the previous worktree (like "cd -"):
  gbm worktree switch -

Examples:
  # With shell integration enabled, switch to a worktree
  gbm worktree switch feature-x

  # Switch to the previous worktree
  gbm worktree switch -

  # Capture path without message
  path=$(gbm worktree switch feature-x 2>/dev/null)`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			worktreeName := args[0]

			// Get current worktree to track for history
			var currentWorktree *git.Worktree

			// First check if parent process passed the current worktree via env var
			// This is used when switching from the TUI table
			if envCurrentWt := os.Getenv("GBM_CURRENT_WORKTREE"); envCurrentWt != "" {
				// Look up the worktree by name to get full info
				wts, _ := svc.Git.ListWorktrees(false)
				for i, wt := range wts {
					if wt.Name == envCurrentWt {
						currentWorktree = &wts[i]
						break
					}
				}
			} else {
				// Otherwise, detect from working directory
				currentWorktree, _ = svc.Git.GetCurrentWorktree() // Ignore error if not in a worktree
			}

			// Handle "-" to switch to previous worktree
			if worktreeName == "-" {
				state := svc.GetState()
				if state.PreviousWorktree == "" {
					if ShouldUseJSON() {
						return HandleError(ErrNoPreviousWorktree.Error())
					}
					return ErrNoPreviousWorktree
				}
				worktreeName = state.PreviousWorktree
				PrintMessage("Switching to previous worktree: %s\n", worktreeName)
			}

			// List all worktrees to find the target
			worktrees, err := svc.Git.ListWorktrees(false)
			if err != nil {
				if ShouldUseJSON() {
					return HandleError(fmt.Sprintf("failed to list worktrees: %v", err))
				}
				return fmt.Errorf("failed to list worktrees: %w", err)
			}

			// Find the worktree by name
			var targetWorktree *git.Worktree
			for i, wt := range worktrees {
				if wt.Name == worktreeName {
					targetWorktree = &worktrees[i]
					break
				}
			}

			if targetWorktree == nil {
				if ShouldUseJSON() {
					return HandleError(fmt.Sprintf("worktree '%s' not found", worktreeName))
				}
				return fmt.Errorf("worktree '%s' not found", worktreeName)
			}

			// Update state to track worktree history
			state := svc.GetState()
			if currentWorktree != nil && currentWorktree.Name != targetWorktree.Name {
				// Save current worktree as previous before switching
				state.PreviousWorktree = currentWorktree.Name
			}
			state.CurrentWorktree = targetWorktree.Name
			_ = svc.SaveState() // Ignore errors - state tracking is optional

			// Handle output based on format
			if ShouldUseJSON() {
				response := WorktreeSwitchResponse{
					Worktree: WorktreeResponse{
						Name:   targetWorktree.Name,
						Path:   targetWorktree.Path,
						Branch: targetWorktree.Branch,
					},
					Previous: state.PreviousWorktree,
				}
				return OutputJSONWithMessage(response, fmt.Sprintf("Switched to worktree '%s'", worktreeName))
			}

			// Universal pattern: Always output path to stdout, messages to stderr
			// This enables shell integration without environment variables
			// Users can suppress the message with: gbm wt switch foo 2>/dev/null

			// Always output path to stdout (machine-readable)
			fmt.Println(targetWorktree.Path)

			// Always output message to stderr (human-readable)
			PrintSuccess(fmt.Sprintf("Switched to worktree '%s'", worktreeName))
			return nil
		},
	}

	// Add shell completions for worktree names
	cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			var completions []string

			// Add "-" option if there's a previous worktree
			state := svc.GetState()
			if state.PreviousWorktree != "" {
				completions = append(completions, fmt.Sprintf("-\t%s (previous)", state.PreviousWorktree))
			}

			// Add all worktrees
			worktreeCompletions, err := generateWorktreeCompletions(svc)
			if err != nil {
				return completions, cobra.ShellCompDirectiveNoFileComp
			}

			completions = append(completions, worktreeCompletions...)
			return completions, cobra.ShellCompDirectiveNoFileComp
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return cmd
}

package service

import (
	"fmt"
	"gbm/internal/git"
	"os"

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
			return runWorktreeSwitch(svc, args[0])
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

// runWorktreeSwitch handles the worktree switch logic.
func runWorktreeSwitch(svc *Service, worktreeName string) error {
	currentWorktree := detectCurrentWorktree(svc)

	// Handle "-" to switch to previous worktree
	var err error
	worktreeName, err = resolvePreviousWorktree(svc, worktreeName)
	if err != nil {
		return err
	}

	// Find the target worktree
	targetWorktree, err := findWorktreeByName(svc, worktreeName)
	if err != nil {
		return err
	}

	// Update state and output result
	updateWorktreeState(svc, currentWorktree, targetWorktree)
	return outputSwitchResult(targetWorktree, svc.GetState().PreviousWorktree)
}

// detectCurrentWorktree gets the current worktree from env var or working directory.
func detectCurrentWorktree(svc *Service) *git.Worktree {
	if envCurrentWt := os.Getenv("GBM_CURRENT_WORKTREE"); envCurrentWt != "" {
		wts, _ := svc.Git.ListWorktrees(false)
		for i, wt := range wts {
			if wt.Name == envCurrentWt {
				return &wts[i]
			}
		}
	}
	current, _ := svc.Git.GetCurrentWorktree()
	return current
}

// resolvePreviousWorktree resolves "-" to the previous worktree name.
func resolvePreviousWorktree(svc *Service, name string) (string, error) {
	if name != "-" {
		return name, nil
	}

	state := svc.GetState()
	if state.PreviousWorktree == "" {
		if ShouldUseJSON() {
			return "", HandleError(ErrNoPreviousWorktree.Error())
		}
		return "", ErrNoPreviousWorktree
	}
	PrintMessage("Switching to previous worktree: %s\n", state.PreviousWorktree)
	return state.PreviousWorktree, nil
}

// findWorktreeByName finds a worktree by name from the list.
func findWorktreeByName(svc *Service, name string) (*git.Worktree, error) {
	worktrees, err := svc.Git.ListWorktrees(false)
	if err != nil {
		if ShouldUseJSON() {
			return nil, HandleError(fmt.Sprintf("failed to list worktrees: %v", err))
		}
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	for i, wt := range worktrees {
		if wt.Name == name {
			return &worktrees[i], nil
		}
	}

	if ShouldUseJSON() {
		return nil, HandleError(fmt.Sprintf("worktree '%s' not found", name))
	}
	return nil, fmt.Errorf("worktree '%s' not found", name)
}

// updateWorktreeState updates the state with current and previous worktree info.
func updateWorktreeState(svc *Service, current, target *git.Worktree) {
	state := svc.GetState()
	if current != nil && current.Name != target.Name {
		state.PreviousWorktree = current.Name
	}
	state.CurrentWorktree = target.Name
	_ = svc.SaveState()
}

// outputSwitchResult handles the output after switching worktrees.
func outputSwitchResult(target *git.Worktree, previous string) error {
	if ShouldUseJSON() {
		response := WorktreeSwitchResponse{
			Worktree: WorktreeResponse{
				Name:   target.Name,
				Path:   target.Path,
				Branch: target.Branch,
			},
			Previous: previous,
		}
		return OutputJSONWithMessage(response, fmt.Sprintf("Switched to worktree '%s'", target.Name))
	}

	fmt.Println(target.Path)
	PrintSuccess(fmt.Sprintf("Switched to worktree '%s'", target.Name))
	return nil
}

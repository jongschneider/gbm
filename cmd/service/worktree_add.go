package service

import (
	"bufio"
	"errors"
	"fmt"
	"gbm/internal/git"
	"gbm/internal/jira"
	"gbm/internal/utils"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// runTUIMode launches the interactive TUI workflow.
func runTUIMode(svc *Service) error {
	if !ShouldAllowInput() {
		if ShouldUseJSON() {
			return HandleError("TUI mode requires interactive input. Use 'gbm worktree add <name> <branch>' for non-interactive mode")
		}
		return errors.New("TUI mode requires interactive input. Use 'gbm worktree add <name> <branch>' for non-interactive mode")
	}
	return runWorktreeAddWizardTUI(svc)
}

// runCLIMode runs the command-line worktree add workflow.
func runCLIMode(svc *Service, worktreeName, branchName string, createBranch bool, baseBranch string) error {
	worktreesDir, err := svc.GetWorktreesPath()
	if err != nil {
		return handleAddError(fmt.Sprintf("failed to get worktrees directory: %v", err), err)
	}

	if err := utils.MkdirAll(worktreesDir, ShouldUseDryRun()); err != nil {
		return handleAddError(err.Error(), err)
	}

	wt, err := svc.Git.AddWorktree(worktreesDir, worktreeName, branchName, createBranch, baseBranch, ShouldUseDryRun())
	if err == nil {
		return outputWorktreeCreated(wt)
	}

	// Check if branch doesn't exist and we can prompt to create it
	if shouldPromptForBranchCreation(err, createBranch) {
		return promptAndCreateBranch(svc, worktreesDir, worktreeName, branchName, baseBranch)
	}

	return handleAddError(err.Error(), err)
}

// shouldPromptForBranchCreation checks if we should offer to create a missing branch.
func shouldPromptForBranchCreation(err error, alreadyCreating bool) bool {
	if alreadyCreating {
		return false
	}
	errMsg := err.Error()
	return strings.Contains(errMsg, "does not exist") || strings.Contains(errMsg, "invalid reference")
}

// promptAndCreateBranch prompts the user to create a missing branch and retries.
func promptAndCreateBranch(svc *Service, worktreesDir, worktreeName, branchName, baseBranch string) error {
	if !ShouldAllowInput() {
		if ShouldUseJSON() {
			return HandleError(fmt.Sprintf("Branch '%s' does not exist and --no-input mode prevents prompting", branchName))
		}
		return fmt.Errorf("branch '%s' does not exist. Use -b to create it", branchName)
	}

	fmt.Fprintf(os.Stderr, "Branch '%s' does not exist. Create it as a new branch? (y/N): ", branchName)
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return handleAddError(fmt.Sprintf("failed to read user input: %v", err), err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	if response != "y" && response != "yes" {
		return ErrBranchCreationCancelled
	}

	wt, err := svc.Git.AddWorktree(worktreesDir, worktreeName, branchName, true, baseBranch, ShouldUseDryRun())
	if err != nil {
		return handleAddError(err.Error(), err)
	}
	return outputWorktreeCreated(wt)
}

// outputWorktreeCreated handles the output after successfully creating a worktree.
func outputWorktreeCreated(wt *git.Worktree) error {
	if ShouldUseDryRun() {
		return nil
	}

	if ShouldUseJSON() {
		response := WorktreeAddResponse{
			Worktree: WorktreeResponse{
				Name:   wt.Name,
				Path:   wt.Path,
				Branch: wt.Branch,
			},
			Created: true,
		}
		return OutputJSONWithMessage(response, fmt.Sprintf("Created worktree '%s' for branch '%s'", wt.Name, wt.Branch))
	}

	fmt.Println(wt.Path)
	PrintSuccess(fmt.Sprintf("Created worktree '%s' for branch '%s'", wt.Name, wt.Branch))
	return nil
}

// handleAddError returns an error, formatting for JSON output if needed.
func handleAddError(msg string, err error) error {
	if ShouldUseJSON() {
		return HandleError(msg)
	}
	if err != nil {
		return err
	}
	return errors.New(msg)
}

func newWorktreeAddCommand(svc *Service) *cobra.Command {
	var (
		createBranch bool
		baseBranch   string
	)

	cmd := &cobra.Command{
		Use:     "add [name] [branch]",
		Aliases: []string{"a"},
		Short:   "Add a new worktree in the worktrees directory",
		Long: `Add a new worktree in the worktrees directory for the given branch.
The worktree will be created at <repo-root>/worktrees/<name>.

When called with no arguments, launches an interactive TUI workflow.
When called with arguments, uses the traditional CLI mode.

Examples:
  # Interactive TUI mode
  gbm worktree add

  # CLI mode: Create worktree for existing branch
  gbm worktree add feature-x feature-x

  # CLI mode: Create worktree with new branch from current HEAD
  gbm worktree add feature-y feature-y -b

  # CLI mode: Create worktree with new branch from specific base
  gbm worktree add feature-z feature-z -b --base main`,
		Args: func(cmd *cobra.Command, args []string) error {
			// Accept 0 args (TUI mode) or 2 args (CLI mode)
			if len(args) == 0 || len(args) == 2 {
				return nil
			}
			return fmt.Errorf("accepts 0 or 2 arg(s), received %d", len(args))
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// If no args provided, launch TUI mode
			if len(args) == 0 {
				return runTUIMode(svc)
			}

			// CLI mode
			baseBranch = utils.GetStringFlagOrConfig(cmd, "base", svc.GetConfig().DefaultBranch)
			if baseBranch == "" {
				baseBranch = "master"
			}
			return runCLIMode(svc, args[0], args[1], createBranch, baseBranch)
		},
		PostRunE: func(cmd *cobra.Command, args []string) error {
			// Skip PostRunE in TUI mode (no args) - TUI already handles file copying
			if len(args) == 0 {
				return nil
			}
			if ShouldUseDryRun() {
				return nil
			}
			worktreeName := args[0]
			// Copy files from source worktrees based on config rules
			err := svc.CopyFilesToWorktree(worktreeName)
			if err != nil {
				PrintWarning(fmt.Sprintf("failed to copy files to worktree: %v", err))
			}
			// Create JIRA markdown if applicable
			err = svc.CreateJiraMarkdownFile(worktreeName)
			if err != nil {
				PrintWarning(fmt.Sprintf("failed to create JIRA markdown: %v", err))
			}
			return nil
		},
	}

	cmd.Flags().BoolVarP(&createBranch, "create-branch", "b", false, "Create a new branch for the worktree")
	cmd.Flags().StringVar(&baseBranch, "base", "", "Base branch to create new branch from (defaults to config.default_branch)")

	// Add JIRA key completions for the first positional argument
	cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			// Complete JIRA keys with summaries for context
			filters := svc.GetJiraFilters()
			jiraIssues, err := svc.Jira.GetJiraIssues(filters, false)
			if err != nil {
				// If JIRA CLI is not available, fall back to no completions
				return nil, cobra.ShellCompDirectiveNoFileComp
			}

			var completions []string
			for _, issue := range jiraIssues {
				// Format: "KEY\tSummary" - clean completion of just the key with summary context
				completion := fmt.Sprintf("%s\t%s", issue.Key, issue.Summary)
				completions = append(completions, completion)
			}
			return completions, cobra.ShellCompDirectiveNoFileComp
		} else if len(args) == 1 {
			// Complete branch name based on the JIRA key
			worktreeName := args[0]
			if jira.IsJiraKey(worktreeName) {
				branchName, err := svc.Jira.GenerateBranchFromJira(worktreeName, false)
				if err != nil {
					// Fallback to default branch name generation
					branchName = fmt.Sprintf("%s%s", FeatureBranchPrefix, strings.ToLower(worktreeName))
				}
				return []string{branchName}, cobra.ShellCompDirectiveNoFileComp
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return cmd
}

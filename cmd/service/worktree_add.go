package service

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"gbm/internal/jira"
	"gbm/internal/utils"

	"github.com/spf13/cobra"
)

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
				// TUI mode requires interactive input
				if !ShouldAllowInput() {
					if ShouldUseJSON() {
						return HandleError("TUI mode requires interactive input. Use 'gbm worktree add <name> <branch>' for non-interactive mode")
					}
					return fmt.Errorf("TUI mode requires interactive input. Use 'gbm worktree add <name> <branch>' for non-interactive mode")
				}
				return runWorktreeAddWizardTUI(svc)
			}

			// CLI mode (existing behavior)
			worktreeName := args[0]
			branchName := args[1]

			// Get worktrees directory from service (reads from config)
			worktreesDir, err := svc.GetWorktreesPath()
			if err != nil {
				if ShouldUseJSON() {
					return HandleError(fmt.Sprintf("failed to get worktrees directory: %v", err))
				}
				return fmt.Errorf("failed to get worktrees directory: %w", err)
			}

			// Create worktrees directory if it doesn't exist
			if err := utils.MkdirAll(worktreesDir, ShouldUseDryRun()); err != nil {
				if ShouldUseJSON() {
					return HandleError(err.Error())
				}
				return err
			}

			// Flag override pattern: explicit flag > config > default
			baseBranch = utils.GetStringFlagOrConfig(cmd, "base", svc.GetConfig().DefaultBranch)
			if baseBranch == "" {
				baseBranch = "master" // Ultimate fallback
			}

			// Try to add the worktree
			wt, err := svc.Git.AddWorktree(worktreesDir, worktreeName, branchName, createBranch, baseBranch, ShouldUseDryRun())
			if err == nil {
				if !ShouldUseDryRun() {
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

					// Text output: path to stdout, message to stderr
					fmt.Println(wt.Path)
					PrintSuccess(fmt.Sprintf("Created worktree '%s' for branch '%s'", wt.Name, wt.Branch))
				}
				return nil
			}

			// If it's not a "branch doesn't exist" error, or user already specified -b, return the error
			errMsg := err.Error()
			isBranchNotExist := strings.Contains(errMsg, "does not exist") || strings.Contains(errMsg, "invalid reference")
			if !isBranchNotExist || createBranch {
				if ShouldUseJSON() {
					return HandleError(errMsg)
				}
				return err
			}

			// Handle no-input mode
			if !ShouldAllowInput() {
				if ShouldUseJSON() {
					return HandleError(fmt.Sprintf("Branch '%s' does not exist and --no-input mode prevents prompting", branchName))
				}
				return fmt.Errorf("branch '%s' does not exist. Use -b to create it", branchName)
			}

			// Prompt user if they want to create a new branch
			fmt.Fprintf(os.Stderr, "Branch '%s' does not exist. Create it as a new branch? (y/N): ", branchName)

			reader := bufio.NewReader(os.Stdin)
			response, err := reader.ReadString('\n')
			if err != nil {
				if ShouldUseJSON() {
					return HandleError(fmt.Sprintf("failed to read user input: %v", err))
				}
				return fmt.Errorf("failed to read user input: %w", err)
			}

			response = strings.TrimSpace(strings.ToLower(response))
			if response != "y" && response != "yes" {
				return ErrBranchCreationCancelled
			}

			// Retry with createBranch = true
			wt, err = svc.Git.AddWorktree(worktreesDir, worktreeName, branchName, true, baseBranch, ShouldUseDryRun())
			if err != nil {
				if ShouldUseJSON() {
					return HandleError(err.Error())
				}
				return err
			}
			if !ShouldUseDryRun() {
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

				// Text output: path to stdout, message to stderr
				fmt.Println(wt.Path)
				PrintSuccess(fmt.Sprintf("Created worktree '%s' for branch '%s'", wt.Name, wt.Branch))
			}
			return nil
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
			if err := svc.CopyFilesToWorktree(worktreeName); err != nil {
				PrintWarning(fmt.Sprintf("failed to copy files to worktree: %v", err))
			}
			// Create JIRA markdown if applicable
			if err := svc.CreateJiraMarkdownFile(worktreeName); err != nil {
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

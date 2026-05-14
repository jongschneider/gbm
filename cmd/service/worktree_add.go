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

// ExistingPolicy controls behavior when a worktree with the target name already exists.
type ExistingPolicy int

const (
	ErrorIfExists ExistingPolicy = iota
	PromptToReplace
)

// MissingBranchPolicy controls behavior when the target branch does not exist.
type MissingBranchPolicy int

const (
	ErrorIfBranchMissing MissingBranchPolicy = iota
	PromptToCreateBranch
)

// AddOptions controls the worktree-add flow.
type AddOptions struct {
	Name            string
	Branch          string
	BaseBranch      string
	CreateBranch    bool
	DryRun          bool
	OnExisting      ExistingPolicy
	OnMissingBranch MissingBranchPolicy
}

// addWorktree is the single entry point both CLI and TUI use to create a worktree.
// It applies the existing/missing-branch policies, creates the worktree, emits
// output (respecting --json/--dry-run globals), and runs post-create hooks
// (file copy + JIRA markdown).
//
// Returns (nil, nil) when the user cancels at an interactive prompt.
func addWorktree(svc *Service, opts AddOptions) (*git.Worktree, error) {
	worktreesDir, err := svc.GetWorktreesPath()
	if err != nil {
		return nil, handleAddError(fmt.Sprintf("failed to get worktrees directory: %v", err), err)
	}

	if err := handleExistingWorktreeForPolicy(svc, opts.Name, opts.OnExisting); err != nil {
		if errors.Is(err, ErrUserCancelled) {
			return nil, nil
		}
		return nil, handleAddError(err.Error(), err)
	}

	if err := utils.MkdirAll(worktreesDir, opts.DryRun); err != nil {
		return nil, handleAddError(err.Error(), err)
	}

	wt, err := svc.Git.AddWorktree(worktreesDir, opts.Name, opts.Branch, opts.CreateBranch, opts.BaseBranch, opts.DryRun)
	if err != nil && shouldPromptForBranchCreation(err, opts.CreateBranch) && opts.OnMissingBranch == PromptToCreateBranch {
		wt, err = promptAndCreateBranch(svc, worktreesDir, opts.Name, opts.Branch, opts.BaseBranch, opts.DryRun)
		if errors.Is(err, ErrBranchCreationCancelled) {
			return nil, err
		}
	}
	if err != nil {
		return nil, handleAddError(err.Error(), err)
	}

	if err := outputWorktreeCreated(wt); err != nil {
		return nil, err
	}

	if opts.DryRun {
		return wt, nil
	}

	if err := svc.CopyFilesToWorktree(opts.Name); err != nil {
		PrintWarning(fmt.Sprintf("failed to copy files to worktree: %v", err))
	}
	if err := svc.CreateJiraMarkdownFile(opts.Name); err != nil {
		PrintWarning(fmt.Sprintf("failed to create JIRA markdown: %v", err))
	}

	return wt, nil
}

// handleExistingWorktreeForPolicy checks whether a worktree named worktreeName
// already exists and applies the configured policy. Returns ErrUserCancelled if
// the user declines an interactive replace prompt.
func handleExistingWorktreeForPolicy(svc *Service, worktreeName string, policy ExistingPolicy) error {
	existing, err := svc.Git.ListWorktrees(false)
	if err != nil {
		return fmt.Errorf("failed to list worktrees: %w", err)
	}

	var exists bool
	for _, wt := range existing {
		if wt.Name == worktreeName {
			exists = true
			break
		}
	}
	if !exists {
		return nil
	}

	if policy == ErrorIfExists || !ShouldAllowInput() {
		return fmt.Errorf("worktree '%s' already exists", worktreeName)
	}

	fmt.Fprintf(os.Stderr, "Worktree '%s' already exists. Replace it? (y/N): ", worktreeName)
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read user input: %w", err)
	}
	response = strings.TrimSpace(strings.ToLower(response))
	if response != "y" && response != "yes" {
		fmt.Fprintln(os.Stderr, "Cancelled")
		return ErrUserCancelled
	}

	if _, err := svc.Git.RemoveWorktree(worktreeName, false, false); err != nil {
		return fmt.Errorf("failed to remove existing worktree: %w", err)
	}
	PrintSuccess(fmt.Sprintf("Removed existing worktree '%s'", worktreeName))
	return nil
}

// shouldPromptForBranchCreation checks if we should offer to create a missing branch.
func shouldPromptForBranchCreation(err error, alreadyCreating bool) bool {
	if alreadyCreating {
		return false
	}
	errMsg := err.Error()
	return strings.Contains(errMsg, "does not exist") || strings.Contains(errMsg, "invalid reference")
}

// promptAndCreateBranch prompts the user to create a missing branch and retries
// the worktree creation with createBranch=true. Returns ErrBranchCreationCancelled
// if the user declines.
func promptAndCreateBranch(svc *Service, worktreesDir, worktreeName, branchName, baseBranch string, dryRun bool) (*git.Worktree, error) {
	if !ShouldAllowInput() {
		if ShouldUseJSON() {
			return nil, HandleError(fmt.Sprintf("Branch '%s' does not exist and --no-input mode prevents prompting", branchName))
		}
		return nil, fmt.Errorf("branch '%s' does not exist. Use -b to create it", branchName)
	}

	fmt.Fprintf(os.Stderr, "Branch '%s' does not exist. Create it as a new branch? (y/N): ", branchName)
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read user input: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	if response != "y" && response != "yes" {
		return nil, ErrBranchCreationCancelled
	}

	return svc.Git.AddWorktree(worktreesDir, worktreeName, branchName, true, baseBranch, dryRun)
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
			if len(args) == 0 {
				return runTUIMode(svc)
			}

			baseBranch = utils.GetStringFlagOrConfig(cmd, "base", svc.GetConfig().DefaultBranch)
			if baseBranch == "" {
				baseBranch = "master"
			}

			onMissingBranch := ErrorIfBranchMissing
			if !createBranch {
				onMissingBranch = PromptToCreateBranch
			}

			_, err := addWorktree(svc, AddOptions{
				Name:            args[0],
				Branch:          args[1],
				BaseBranch:      baseBranch,
				CreateBranch:    createBranch,
				DryRun:          ShouldUseDryRun(),
				OnExisting:      ErrorIfExists,
				OnMissingBranch: onMissingBranch,
			})
			return err
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

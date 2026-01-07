package workflows

import (
	"fmt"

	"gbm/pkg/tui"
	"gbm/pkg/tui/fields"
)

// getFeatureSteps returns the steps for a feature workflow.
func getFeatureSteps(ctx *tui.Context) []tui.Step {
	return []tui.Step{
		// Step 1: JIRA issue selection or custom worktree name
		{
			Name: "worktree_name",
			Field: fields.NewFilterable(
				"worktree_name",
				"Select JIRA Issue or Enter Worktree Name",
				"Search JIRA issues or enter a custom name",
				[]fields.Option{},
			).WithOptionsFunc(func() ([]fields.Option, error) {
				if ctx.JiraService == nil {
					return []fields.Option{}, nil
				}
				issues, err := ctx.JiraService.FetchIssues()
				if err != nil {
					return nil, err
				}

				options := make([]fields.Option, len(issues))
				for i, issue := range issues {
					options[i] = fields.Option{
						Label: fmt.Sprintf("%s - %s", issue.Key, issue.Summary),
						Value: issue.Key,
					}
				}
				return options, nil
			}),
		},

		// Step 2: Branch name (with auto-generated default from JIRA issue)
		{
			Name: "branch_name",
			Field: fields.NewTextInput("branch_name", "Enter Branch Name", "Name for the new branch").
				WithPlaceholder("feature/KEY-description").
				WithValidator(validateBranchName),
		},

		// Step 3: Base branch selection (skipped if branch name already exists)
		{
			Name: "base_branch",
			Field: fields.NewFilterable(
				"base_branch",
				"Select Base Branch",
				"Choose the branch to base this feature on",
				[]fields.Option{},
			).WithOptionsFunc(func() ([]fields.Option, error) {
				if ctx.GitService == nil {
					return []fields.Option{}, nil
				}
				branches, err := ctx.GitService.ListBranches()
				if err != nil {
					return nil, err
				}
				options := make([]fields.Option, len(branches))
				for i, branch := range branches {
					options[i] = fields.Option{
						Label: branch,
						Value: branch,
					}
				}
				return options, nil
			}),

			// Skip this step if the branch name already exists
			Skip: func(state *tui.WorkflowState) bool {
				if ctx.GitService == nil || state.BranchName == "" {
					return false
				}
				exists, err := ctx.GitService.BranchExists(state.BranchName)
				return err == nil && exists
			},
		},

		// Step 4: Confirmation
		{
			Name:  "confirm",
			Field: fields.NewConfirm("confirm", "Create Feature Branch?"),
		},
	}
}

// getBugSteps returns the steps for a bug workflow.
// Identical to feature workflow steps (prefixing is handled in ProcessBugWorkflow).
func getBugSteps(ctx *tui.Context) []tui.Step {
	return []tui.Step{
		// Step 1: JIRA issue selection or custom worktree name
		{
			Name: "worktree_name",
			Field: fields.NewFilterable(
				"worktree_name",
				"Select JIRA Issue or Enter Worktree Name",
				"Search JIRA issues or enter a custom name",
				[]fields.Option{},
			).WithOptionsFunc(func() ([]fields.Option, error) {
				if ctx.JiraService == nil {
					return []fields.Option{}, nil
				}
				issues, err := ctx.JiraService.FetchIssues()
				if err != nil {
					return nil, err
				}

				options := make([]fields.Option, len(issues))
				for i, issue := range issues {
					options[i] = fields.Option{
						Label: fmt.Sprintf("%s - %s", issue.Key, issue.Summary),
						Value: issue.Key,
					}
				}
				return options, nil
			}),
		},

		// Step 2: Branch name (with auto-generated default from JIRA issue)
		{
			Name: "branch_name",
			Field: fields.NewTextInput("branch_name", "Enter Branch Name", "Name for the bug fix branch").
				WithPlaceholder("bug/KEY-description").
				WithValidator(validateBranchName),
		},

		// Step 3: Base branch selection (skipped if branch name already exists)
		{
			Name: "base_branch",
			Field: fields.NewFilterable(
				"base_branch",
				"Select Base Branch",
				"Choose the branch to base this bug fix on",
				[]fields.Option{},
			).WithOptionsFunc(func() ([]fields.Option, error) {
				if ctx.GitService == nil {
					return []fields.Option{}, nil
				}
				branches, err := ctx.GitService.ListBranches()
				if err != nil {
					return nil, err
				}
				options := make([]fields.Option, len(branches))
				for i, branch := range branches {
					options[i] = fields.Option{
						Label: branch,
						Value: branch,
					}
				}
				return options, nil
			}),

			// Skip this step if the branch name already exists
			Skip: func(state *tui.WorkflowState) bool {
				if ctx.GitService == nil || state.BranchName == "" {
					return false
				}
				exists, err := ctx.GitService.BranchExists(state.BranchName)
				return err == nil && exists
			},
		},

		// Step 4: Confirmation
		{
			Name:  "confirm",
			Field: fields.NewConfirm("confirm", "Create Bug Fix Branch?"),
		},
	}
}

// getHotfixSteps returns the steps for a hotfix workflow.
func getHotfixSteps(ctx *tui.Context) []tui.Step {
	return []tui.Step{
		// Step 1: JIRA issue selection or custom worktree name
		{
			Name: "worktree_name",
			Field: fields.NewFilterable(
				"worktree_name",
				"Select JIRA Issue or Enter Worktree Name",
				"Search JIRA issues or enter a custom name",
				[]fields.Option{},
			).WithOptionsFunc(func() ([]fields.Option, error) {
				if ctx.JiraService == nil {
					return []fields.Option{}, nil
				}
				issues, err := ctx.JiraService.FetchIssues()
				if err != nil {
					return nil, err
				}

				options := make([]fields.Option, len(issues))
				for i, issue := range issues {
					options[i] = fields.Option{
						Label: fmt.Sprintf("%s - %s", issue.Key, issue.Summary),
						Value: issue.Key,
					}
				}
				return options, nil
			}),
		},

		// Step 2: Base branch selection (mandatory - NOT skipped)
		{
			Name: "base_branch",
			Field: fields.NewFilterable(
				"base_branch",
				"Select Base Branch",
				"Choose the production or release branch to base this hotfix on",
				[]fields.Option{},
			).WithOptionsFunc(func() ([]fields.Option, error) {
				if ctx.GitService == nil {
					return []fields.Option{}, nil
				}
				branches, err := ctx.GitService.ListBranches()
				if err != nil {
					return nil, err
				}
				options := make([]fields.Option, len(branches))
				for i, branch := range branches {
					options[i] = fields.Option{
						Label: branch,
						Value: branch,
					}
				}
				return options, nil
			}),
		},

		// Step 3: Branch name (with auto-generated default from JIRA issue)
		{
			Name: "branch_name",
			Field: fields.NewTextInput("branch_name", "Enter Branch Name", "Name for the hotfix branch").
				WithPlaceholder("hotfix/KEY-description").
				WithValidator(validateBranchName),
		},

		// Step 4: Confirmation
		{
			Name:  "confirm",
			Field: fields.NewConfirm("confirm", "Create Hotfix Branch?"),
		},
	}
}

// getMergeSteps returns the steps for a merge workflow.
func getMergeSteps(ctx *tui.Context) []tui.Step {
	return []tui.Step{
		// Step 1: Source branch selection
		{
			Name: "source_branch",
			Field: fields.NewFilterable(
				"source_branch",
				"Select Source Branch",
				"Choose the branch to merge FROM",
				[]fields.Option{},
			).WithOptionsFunc(func() ([]fields.Option, error) {
				if ctx.GitService == nil {
					return []fields.Option{}, nil
				}
				branches, err := ctx.GitService.ListBranches()
				if err != nil {
					return nil, err
				}
				options := make([]fields.Option, len(branches))
				for i, branch := range branches {
					options[i] = fields.Option{
						Label: branch,
						Value: branch,
					}
				}
				return options, nil
			}),
		},

		// Step 2: Target branch selection
		{
			Name: "target_branch",
			Field: fields.NewFilterable(
				"target_branch",
				"Select Target Branch",
				"Choose the branch to merge INTO",
				[]fields.Option{},
			).WithOptionsFunc(func() ([]fields.Option, error) {
				if ctx.GitService == nil {
					return []fields.Option{}, nil
				}
				branches, err := ctx.GitService.ListBranches()
				if err != nil {
					return nil, err
				}
				options := make([]fields.Option, len(branches))
				for i, branch := range branches {
					options[i] = fields.Option{
						Label: branch,
						Value: branch,
					}
				}
				return options, nil
			}),
		},

		// Step 3: Confirmation (worktree and branch names auto-generated)
		{
			Name:  "confirm",
			Field: fields.NewConfirm("confirm", "Create Merge?"),
		},
	}
}

// GetWorkflowSteps returns the appropriate workflow steps for the given workflow type.
// Supported types: "feature", "bug", "hotfix", "merge"
// Returns an error for unknown workflow types.
func GetWorkflowSteps(workflowType string, ctx *tui.Context) ([]tui.Step, error) {
	switch workflowType {
	case "feature":
		return getFeatureSteps(ctx), nil
	case "bug":
		return getBugSteps(ctx), nil
	case "hotfix":
		return getHotfixSteps(ctx), nil
	case "merge":
		return getMergeSteps(ctx), nil
	default:
		return nil, fmt.Errorf("unknown workflow type: %s", workflowType)
	}
}

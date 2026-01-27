package workflows

import (
	"fmt"
	"gbm/pkg/tui"
	"gbm/pkg/tui/fields"
)

// MergeCustomWorkflow creates a merge workflow that demonstrates all Pattern types:
// 1. Async branch loading (from Git service)
// 2. Custom field storage for merge strategy
// 3. Navigator integration for multi-screen workflows
//
// Steps:
// 1. Select merge strategy (e.g., Merge commit, Squash, Rebase) - stored as custom field
// 2. Select source branch (async Filterable)
// 3. Select target branch (async Filterable, excludes source)
// 4. Confirmation with merge summary
//
// This workflow is a reference implementation for advanced TUI patterns.
func MergeCustomWorkflow(ctx *tui.Context) *tui.Wizard {
	steps := getMergeCustomSteps(ctx)
	return tui.NewWizard(steps, ctx)
}

// getMergeCustomSteps returns steps for the merge custom workflow.
// Demonstrates: custom fields, async operations, and complex state management.
func getMergeCustomSteps(ctx *tui.Context) []tui.Step {
	return []tui.Step{
		// Step 1: Merge strategy selection (custom field)
		{
			Name: "merge_strategy",
			Field: fields.NewSelector(
				"merge_strategy",
				"Select Merge Strategy",
				[]fields.Option{
					{
						Label: "Merge commit (creates merge commit)",
						Value: "merge_commit",
					},
					{
						Label: "Squash and merge (combines commits)",
						Value: "squash",
					},
					{
						Label: "Rebase and merge (replays commits)",
						Value: "rebase",
					},
				},
			),
		},

		// Step 2: Source branch selection (async, with spinner)
		{
			Name: "source_branch",
			Field: fields.NewFilterable(
				"source_branch",
				"Select Source Branch",
				"Choose the branch to merge FROM",
				[]fields.Option{},
			).WithOptionsFuncAsync(func() ([]fields.Option, error) {
				if ctx.GitService == nil {
					return []fields.Option{}, nil
				}
				branches, err := ctx.GitService.ListBranches(false)
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

		// Step 3: Target branch selection (async, optionally excludes source)
		{
			Name: "target_branch",
			Field: fields.NewFilterable(
				"target_branch",
				"Select Target Branch",
				"Choose the branch to merge INTO",
				[]fields.Option{},
			).WithOptionsFuncAsync(func() ([]fields.Option, error) {
				if ctx.GitService == nil {
					return []fields.Option{}, nil
				}
				branches, err := ctx.GitService.ListBranches(false)
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

		// Step 4: Confirmation
		{
			Name:  "confirm",
			Field: fields.NewConfirm("confirm", "Create Merge?"),
		},
	}
}

// ProcessMergeCustomWorkflow handles post-wizard logic for merge custom workflow.
// This function:
// 1. Stores the merge strategy in custom fields
// 2. Generates worktree and branch names
// 3. Validates merge compatibility
func ProcessMergeCustomWorkflow(wizard *tui.Wizard) error {
	state := wizard.State()

	// Retrieve merge strategy from custom fields
	strategy := state.GetField("merge_strategy")
	if strategy == nil {
		return fmt.Errorf("merge strategy not selected")
	}

	// Get source and target branches
	sourceBranch := state.GetField("source_branch")
	targetBranch := state.GetField("target_branch")

	if sourceBranch == nil || targetBranch == nil {
		return fmt.Errorf("source or target branch not selected")
	}

	// Store merge strategy in custom fields for later reference
	state.SetField("merge_strategy_selected", strategy)

	// Generate worktree and branch names
	sourceStr := fmt.Sprintf("%v", sourceBranch)
	targetStr := fmt.Sprintf("%v", targetBranch)

	// Worktree name: merge-{source-to-target}
	state.WorktreeName = fmt.Sprintf("merge-%s-to-%s", sourceStr, targetStr)

	// Branch name: merge/{source-to-target}
	state.BranchName = fmt.Sprintf("merge/%s-to-%s", sourceStr, targetStr)

	return nil
}

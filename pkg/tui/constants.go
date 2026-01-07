package tui

// Workflow type constants
const (
	WorkflowTypeFeature = "feature"
	WorkflowTypeBug     = "bug"
	WorkflowTypeHotfix  = "hotfix"
	WorkflowTypeMerge   = "merge"
)

// Field key constants
const (
	FieldKeyWorkflowType = "workflow_type"
	FieldKeyWorktreeName = "worktree_name"
	FieldKeyBranchName   = "branch_name"
	FieldKeyBaseBranch   = "base_branch"
	FieldKeyConfirm      = "confirm"
)

// Branch prefix constants
const (
	BranchPrefixFeature = "feature/"
	BranchPrefixBug     = "bug/"
	BranchPrefixHotfix  = "hotfix/"
	BranchPrefixMerge   = "merge/"
)

package service

// FSM States - Entry Point
const (
	StateSelectType = "select_type" // Initial state: select workflow type
)

// FSM States - Terminal
const (
	StateSuccess   = "success"   // Workflow completed successfully
	StateCancelled = "cancelled" // User pressed Ctrl+C or declined
	StateError     = "error"     // Error occurred during execution
)

// FSM States - Feature/Bug Workflow
const (
	StateFeatureWorktreeName  = "feature_worktree_name"
	StateFeatureBranchName    = "feature_branch_name"
	StateFeatureCheckBranch   = "feature_check_branch"
	StateFeatureBaseBranch    = "feature_base_branch"
	StateFeatureConfirmCreate = "feature_confirm_create"
	StateFeatureExecuteCreate = "feature_execute_create"
)

// FSM States - Hotfix Workflow
const (
	StateHotfixWorktreeName  = "hotfix_worktree_name"
	StateHotfixBaseBranch    = "hotfix_base_branch"
	StateHotfixBranchName    = "hotfix_branch_name"
	StateHotfixExecuteCreate = "hotfix_execute_create"
)

// FSM States - Mergeback Workflow
const (
	StateMergebackSourceBranch  = "mergeback_source_branch"
	StateMergebackTargetBranch  = "mergeback_target_branch"
	StateMergebackWorktreeName  = "mergeback_worktree_name"
	StateMergebackBranchName    = "mergeback_branch_name"
	StateMergebackConfirmMerge  = "mergeback_confirm_merge"
	StateMergebackExecuteCreate = "mergeback_execute_create"
	StateMergebackExecuteMerge  = "mergeback_execute_merge"
)

// FSM Events - Navigation
const (
	EventComplete   = "complete"    // User completed current step
	EventGoBack     = "go_back"     // User pressed ESC
	EventCancel     = "cancel"      // User pressed Ctrl+C
	EventConfirmYes = "confirm_yes" // User confirmed action
	EventConfirmNo  = "confirm_no"  // User declined action
	EventError      = "error"       // Error occurred
)

// FSM Events - Workflow Selection (from StateSelectType)
const (
	EventSelectFeature   = "select_feature"   // User chose feature workflow
	EventSelectBug       = "select_bug"       // User chose bug workflow
	EventSelectHotfix    = "select_hotfix"    // User chose hotfix workflow
	EventSelectMergeback = "select_mergeback" // User chose mergeback workflow
)

// FSM Events - Conditional
const (
	EventBranchExists  = "branch_exists"  // Branch check succeeded
	EventBranchMissing = "branch_missing" // Branch doesn't exist
)

// FSM Events - Loop
const (
	EventRetry = "retry" // Return to type selection after workflow completion
)

// Worktree Naming Constants
const (
	HotfixPrefix   = "HOTFIX_" // Prefix for hotfix worktree names
	MergebackPrefix = "MERGE_"  // Prefix for mergeback worktree names
)

// Branch Naming Constants
const (
	FeatureBranchPrefix   = "feature/" // Prefix for feature branch names
	BugBranchPrefix       = "bug/"     // Prefix for bug fix branch names
	HotfixBranchPrefix    = "hotfix/"  // Prefix for hotfix branch names
	MergebackBranchPrefix = "merge/"   // Prefix for mergeback branch names
)

// Workflow Type Constants
const (
	WorkflowTypeFeature   = "feature"   // Feature workflow type
	WorkflowTypeBug       = "bug"       // Bug fix workflow type
	WorkflowTypeHotfix    = "hotfix"    // Hotfix workflow type
	WorkflowTypeMergeback = "mergeback" // Mergeback workflow type
)

// Stage Constants - Testadd Wrapper Model
const (
	StageTypeSelection = "type_selection" // User is selecting workflow type
	StageWorkflow      = "workflow"       // User is in workflow-specific steps
)

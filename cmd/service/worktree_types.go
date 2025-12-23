package service

// WorkflowState represents common state across workflows
type WorkflowState struct {
	Service      *Service
	WorktreeName string
	BranchName   string
	BaseBranch   string
	DryRun       bool
}

// Workflow interface for all workflows
type Workflow interface {
	Run() error
}

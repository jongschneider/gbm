package tui

// GitService defines the interface for git operations needed by the TUI.
type GitService interface {
	BranchExists(branch string) (bool, error)
	ListBranches() ([]string, error)
}

// JiraService defines the interface for JIRA operations needed by the TUI.
type JiraService interface {
	FetchIssues() ([]JiraIssue, error)
}

// WorktreeConfig defines a worktree configuration with optional merge target.
type WorktreeConfig interface {
	GetBranch() string
	GetMergeInto() string
}

// RepoConfig defines the repository configuration needed by the TUI.
type RepoConfig interface {
	GetWorktrees() map[string]WorktreeConfig
}

// JiraIssue represents a JIRA issue for display in the TUI.
type JiraIssue struct {
	Key     string
	Summary string
}

// WorkflowState holds data collected across wizard steps.
// Standard fields (WorkflowType, WorktreeName, etc) are stored directly.
// Custom fields added dynamically are stored in CustomFields map.
type WorkflowState struct {
	WorkflowType string
	WorktreeName string
	BranchName   string
	BaseBranch   string
	JiraIssue    *JiraIssue
	CustomFields map[string]interface{}
}

// SetField stores a custom field in the workflow state.
// If the field name corresponds to a standard field (e.g., "workflow_type"),
// it updates the standard field instead of CustomFields.
func (ws *WorkflowState) SetField(key string, value interface{}) {
	if ws.CustomFields == nil {
		ws.CustomFields = make(map[string]interface{})
	}
	ws.CustomFields[key] = value
}

// GetField retrieves a custom field from the workflow state.
// Returns the value if found, nil otherwise.
// The caller is responsible for type assertion.
func (ws *WorkflowState) GetField(key string) interface{} {
	if ws.CustomFields == nil {
		return nil
	}
	return ws.CustomFields[key]
}

// Context provides shared state accessible to all TUI components.
type Context struct {
	Width       int
	Height      int
	Theme       *Theme
	State       *WorkflowState
	GitService  GitService
	JiraService JiraService
	Config      RepoConfig
}

// NewContext creates a new Context with default values.
func NewContext() *Context {
	return &Context{
		Theme: DefaultTheme(),
		State: &WorkflowState{},
	}
}

// WithDimensions sets the terminal dimensions and returns the Context.
func (c *Context) WithDimensions(width, height int) *Context {
	c.Width = width
	c.Height = height
	return c
}

// WithTheme sets the theme and returns the Context.
func (c *Context) WithTheme(theme *Theme) *Context {
	c.Theme = theme
	return c
}

// WithGitService sets the git service and returns the Context.
func (c *Context) WithGitService(svc GitService) *Context {
	c.GitService = svc
	return c
}

// WithJiraService sets the JIRA service and returns the Context.
func (c *Context) WithJiraService(svc JiraService) *Context {
	c.JiraService = svc
	return c
}

// WithConfig sets the repository configuration and returns the Context.
func (c *Context) WithConfig(cfg RepoConfig) *Context {
	c.Config = cfg
	return c
}

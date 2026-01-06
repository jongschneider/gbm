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

// JiraIssue represents a JIRA issue for display in the TUI.
type JiraIssue struct {
	Key     string
	Summary string
}

// WorkflowState holds data collected across wizard steps.
type WorkflowState struct {
	WorkflowType string
	WorktreeName string
	BranchName   string
	BaseBranch   string
	JiraIssue    *JiraIssue
}

// Context provides shared state accessible to all TUI components.
type Context struct {
	Width       int
	Height      int
	Theme       *Theme
	State       *WorkflowState
	GitService  GitService
	JiraService JiraService
	jiraIssues  []JiraIssue // Cached JIRA issues to avoid redundant fetches
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

// GetCachedJiraIssues returns cached JIRA issues, fetching them if needed.
// This avoids redundant fetches during wizard transitions.
func (c *Context) GetCachedJiraIssues() ([]JiraIssue, error) {
	// Return cached issues if available
	if len(c.jiraIssues) > 0 {
		return c.jiraIssues, nil
	}

	// Fetch from service if not cached
	if c.JiraService == nil {
		return nil, nil
	}

	issues, err := c.JiraService.FetchIssues()
	if err != nil {
		return nil, err
	}

	// Cache the results
	c.jiraIssues = issues
	return issues, nil
}

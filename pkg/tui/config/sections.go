package config

// Section field slices. Each slice declares the fields for a tab or overlay
// in the Config TUI. Keys use dot-path notation matching the YAML tags in
// cmd/service/service.go and internal/jira/types.go.

// generalFields defines fields for the General tab (top-level config).
// Maps to Config.DefaultBranch and Config.WorktreesDir.
var generalFields = []FieldMeta{
	{
		Key: "default_branch", Label: "Default Branch", Type: String,
		Description: "Base branch for new worktrees",
		Validate:    ValidateRequired,
		Suggestions: static("main", "master", "develop", "development"),
	},
	{
		Key: "worktrees_dir", Label: "Worktrees Directory", Type: String,
		Description: "Supports templates: {gitroot}, {branch}, {issue}",
		Validate:    ValidateRequired,
		Suggestions: static("worktrees", "../{gitroot}-wt", "~/dev/{gitroot}/wt"),
	},
}

// jiraFields defines fields for the JIRA tab, spanning four visual groups.
// Maps to JiraConfig, JiraFilters, MarkdownConfig, and AttachmentConfig.
//
// Groups: Connection (2), Filters (10), Markdown (6), Attachments (6) = 24 total.
var jiraFields = []FieldMeta{
	// -- Connection (2) --
	{
		Key: "jira.host", Label: "Host", Type: String, Group: "Connection",
		Description: "JIRA instance URL (e.g. https://jira.company.com)",
	},
	{
		Key: "jira.me", Label: "Username", Type: String, Group: "Connection",
		Description: "Your JIRA username for issue queries",
	},

	// -- Filters (10) --
	{
		Key: "jira.filters.priority", Label: "Priority", Type: String, Group: "Filters",
		Description: "Filter issues by priority level",
		Suggestions: static("Highest", "High", "Medium", "Low", "Lowest"),
	},
	{
		Key: "jira.filters.type", Label: "Type", Type: String, Group: "Filters",
		Description: "Filter issues by issue type",
		Suggestions: static("Bug", "Story", "Task", "Epic", "Sub-task"),
	},
	{
		Key: "jira.filters.component", Label: "Component", Type: String, Group: "Filters",
		Description: "Filter issues by component name",
	},
	{
		Key: "jira.filters.reporter", Label: "Reporter", Type: String, Group: "Filters",
		Description: "Filter issues by reporter username",
	},
	{
		Key: "jira.filters.assignee", Label: "Assignee", Type: String, Group: "Filters",
		Description: "Filter issues by assignee username",
	},
	{
		Key: "jira.filters.order_by", Label: "Order By", Type: String, Group: "Filters",
		Description: "Sort field for issue results",
		Suggestions: static("created", "updated", "priority", "status", "key"),
	},
	{
		Key: "jira.filters.status", Label: "Status", Type: StringList, Group: "Filters",
		Description: "Filter issues by status values",
	},
	{
		Key: "jira.filters.labels", Label: "Labels", Type: StringList, Group: "Filters",
		Description: "Filter issues by label names",
	},
	{
		Key: "jira.filters.custom_args", Label: "Custom Args", Type: StringList, Group: "Filters",
		Description: "Extra CLI args passed to the JIRA query",
	},
	{
		Key: "jira.filters.reverse", Label: "Reverse", Type: Bool, Group: "Filters",
		Description: "Reverse the sort order of results",
	},

	// -- Markdown (6) --
	{
		Key: "jira.markdown.filename_pattern", Label: "Filename Pattern", Type: String, Group: "Markdown",
		Description: "Output filename pattern for exported markdown",
		Suggestions: static("{key}.md", "issues/{key}.md"),
	},
	{
		Key: "jira.markdown.max_depth", Label: "Max Depth", Type: Int, Group: "Markdown",
		Description: "Max heading depth for markdown export (0 = unlimited)",
		Validate:    ValidateNonNegativeInt,
	},
	{
		Key: "jira.markdown.include_comments", Label: "Include Comments", Type: Bool, Group: "Markdown",
		Description: "Include issue comments in markdown export",
	},
	{
		Key: "jira.markdown.include_attachments", Label: "Include Attach.", Type: Bool, Group: "Markdown",
		Description: "Include attachment links in markdown export",
	},
	{
		Key: "jira.markdown.use_relative_links", Label: "Relative Links", Type: Bool, Group: "Markdown",
		Description: "Use relative paths instead of absolute URLs",
	},
	{
		Key: "jira.markdown.include_linked_issues", Label: "Linked Issues", Type: Bool, Group: "Markdown",
		Description: "Include linked issue references in export",
	},

	// -- Attachments (6) --
	{
		Key: "jira.attachments.enabled", Label: "Enabled", Type: Bool, Group: "Attachments",
		Description: "Enable automatic attachment downloading",
	},
	{
		Key: "jira.attachments.max_size_mb", Label: "Max Size (MB)", Type: Int, Group: "Attachments",
		Description: "Skip attachments larger than this size",
		Validate:    ValidateNonNegativeInt,
	},
	{
		Key: "jira.attachments.directory", Label: "Directory", Type: String, Group: "Attachments",
		Description: "Local directory for downloaded attachments",
	},
	{
		Key: "jira.attachments.download_timeout_seconds", Label: "Timeout (sec)", Type: Int, Group: "Attachments",
		Description: "HTTP timeout per attachment download",
		Validate:    ValidateNonNegativeInt,
	},
	{
		Key: "jira.attachments.retry_attempts", Label: "Retry Attempts", Type: Int, Group: "Attachments",
		Description: "Number of retries on download failure",
		Validate:    ValidateNonNegativeInt,
	},
	{
		Key: "jira.attachments.retry_backoff_ms", Label: "Retry Backoff", Type: Int, Group: "Attachments",
		Description: "Milliseconds between retry attempts",
		Validate:    ValidateNonNegativeInt,
	},
}

// fileCopyAutoFields defines fields for the File Copy tab's auto-copy section.
// Maps to AutoFileCopyConfig.
var fileCopyAutoFields = []FieldMeta{
	{
		Key: "file_copy.auto.enabled", Label: "Enabled", Type: Bool, Group: "Auto Copy",
		Description: "Enable automatic file copying on worktree creation",
	},
	{
		Key: "file_copy.auto.source_worktree", Label: "Source Worktree", Type: String, Group: "Auto Copy",
		Description: "Worktree to copy files from; supports template: {default}",
		Suggestions: static("{default}"),
	},
	{
		Key: "file_copy.auto.copy_ignored", Label: "Copy Ignored", Type: Bool, Group: "Auto Copy",
		Description: "Include git-ignored files when copying",
	},
	{
		Key: "file_copy.auto.copy_untracked", Label: "Copy Untracked", Type: Bool, Group: "Auto Copy",
		Description: "Include untracked files when copying",
	},
	{
		Key: "file_copy.auto.exclude", Label: "Exclude", Type: StringList, Group: "Auto Copy",
		Description: "Glob patterns to exclude from auto-copy",
	},
}

// fileCopyRuleFields defines fields for the file-copy rule editor overlay.
// Keys are relative to a single FileCopyRule entry.
var fileCopyRuleFields = []FieldMeta{
	{Key: "source_worktree", Label: "Source Worktree", Type: String},
	{Key: "files", Label: "Files", Type: StringList},
}

// worktreeEntryFields defines fields for the worktree editor overlay.
// Keys are relative to a single WorktreeConfig entry.
var worktreeEntryFields = []FieldMeta{
	{Key: "branch", Label: "Branch", Type: String},
	{Key: "merge_into", Label: "Merge Into", Type: String},
	{Key: "description", Label: "Description", Type: String},
}

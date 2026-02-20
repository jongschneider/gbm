package service

import (
	"fmt"
	"strings"

	tuiconfig "gbm/pkg/tui/config"
)

// ConfigAdapter wraps a *Config and implements tuiconfig.ConfigAccessor,
// providing get/set access to config values by dot-path key. All key
// mappings use an explicit switch (not reflection) to stay type-safe.
type ConfigAdapter struct {
	cfg *Config
}

// NewConfigAdapter creates a ConfigAccessor backed by the given Config.
func NewConfigAdapter(cfg *Config) *ConfigAdapter {
	return &ConfigAdapter{cfg: cfg}
}

// GetValue returns the current value for a dot-path key.
// Returns nil for unknown keys.
func (a *ConfigAdapter) GetValue(key string) any {
	section, _, _ := strings.Cut(key, ".")
	switch section {
	case "default_branch":
		return a.cfg.DefaultBranch
	case "worktrees_dir":
		return a.cfg.WorktreesDir
	case "jira":
		return a.getJiraValue(key)
	case "worktrees":
		return a.cfg.Worktrees
	case "file_copy":
		return a.getFileCopyValue(key)
	default:
		return nil
	}
}

func (a *ConfigAdapter) getJiraValue(key string) any {
	// Extract the sub-section: "jira.filters.priority" -> "filters".
	_, rest, _ := strings.Cut(key, ".")
	sub, _, _ := strings.Cut(rest, ".")
	switch sub {
	case "host":
		return a.cfg.Jira.Host
	case "me":
		return a.cfg.Jira.Me
	case "filters":
		return a.getJiraFiltersValue(key)
	case "markdown":
		return a.getJiraMarkdownValue(key)
	case "attachments":
		return a.getJiraAttachmentsValue(key)
	default:
		return nil
	}
}

func (a *ConfigAdapter) getJiraFiltersValue(key string) any {
	switch key {
	case "jira.filters.priority":
		return a.cfg.Jira.Filters.Priority
	case "jira.filters.type":
		return a.cfg.Jira.Filters.Type
	case "jira.filters.component":
		return a.cfg.Jira.Filters.Component
	case "jira.filters.reporter":
		return a.cfg.Jira.Filters.Reporter
	case "jira.filters.assignee":
		return a.cfg.Jira.Filters.Assignee
	case "jira.filters.order_by":
		return a.cfg.Jira.Filters.OrderBy
	case "jira.filters.status":
		return a.cfg.Jira.Filters.Status
	case "jira.filters.labels":
		return a.cfg.Jira.Filters.Labels
	case "jira.filters.custom_args":
		return a.cfg.Jira.Filters.CustomArgs
	case "jira.filters.reverse":
		return a.cfg.Jira.Filters.Reverse
	default:
		return nil
	}
}

func (a *ConfigAdapter) getJiraMarkdownValue(key string) any {
	switch key {
	case "jira.markdown.filename_pattern":
		return a.cfg.Jira.Markdown.FilenamePattern
	case "jira.markdown.max_depth":
		return a.cfg.Jira.Markdown.MaxDepth
	case "jira.markdown.include_comments":
		return a.cfg.Jira.Markdown.IncludeComments
	case "jira.markdown.include_attachments":
		return a.cfg.Jira.Markdown.IncludeAttachments
	case "jira.markdown.use_relative_links":
		return a.cfg.Jira.Markdown.UseRelativeLinks
	case "jira.markdown.include_linked_issues":
		return a.cfg.Jira.Markdown.IncludeLinkedIssues
	default:
		return nil
	}
}

func (a *ConfigAdapter) getJiraAttachmentsValue(key string) any {
	switch key {
	case "jira.attachments.enabled":
		return a.cfg.Jira.Attachments.Enabled
	case "jira.attachments.max_size_mb":
		return a.cfg.Jira.Attachments.MaxSizeMB
	case "jira.attachments.directory":
		return a.cfg.Jira.Attachments.Directory
	case "jira.attachments.download_timeout_seconds":
		return a.cfg.Jira.Attachments.DownloadTimeoutSec
	case "jira.attachments.retry_attempts":
		return a.cfg.Jira.Attachments.RetryAttempts
	case "jira.attachments.retry_backoff_ms":
		return a.cfg.Jira.Attachments.RetryBackoffMs
	default:
		return nil
	}
}

func (a *ConfigAdapter) getFileCopyValue(key string) any {
	switch key {
	case "file_copy.rules":
		return a.cfg.FileCopy.Rules
	case "file_copy.auto.enabled":
		return a.cfg.FileCopy.Auto.Enabled
	case "file_copy.auto.source_worktree":
		return a.cfg.FileCopy.Auto.SourceWorktree
	case "file_copy.auto.copy_ignored":
		return a.cfg.FileCopy.Auto.CopyIgnored
	case "file_copy.auto.copy_untracked":
		return a.cfg.FileCopy.Auto.CopyUntracked
	case "file_copy.auto.exclude":
		return a.cfg.FileCopy.Auto.Exclude
	default:
		return nil
	}
}

// SetValue updates the config value for a dot-path key.
// Returns tuiconfig.ErrUnknownKey for unrecognized keys, or a type error
// if the value cannot be assigned to the target field.
func (a *ConfigAdapter) SetValue(key string, value any) error {
	section, _, _ := strings.Cut(key, ".")
	switch section {
	case "default_branch":
		return setString(&a.cfg.DefaultBranch, value)
	case "worktrees_dir":
		return setString(&a.cfg.WorktreesDir, value)
	case "worktrees":
		return setWorktreeMap(&a.cfg.Worktrees, value)
	case "jira":
		return a.setJiraValue(key, value)
	case "file_copy":
		return a.setFileCopyValue(key, value)
	default:
		return fmt.Errorf("%w: %s", tuiconfig.ErrUnknownKey, key)
	}
}

func (a *ConfigAdapter) setJiraValue(key string, value any) error {
	_, rest, _ := strings.Cut(key, ".")
	sub, _, _ := strings.Cut(rest, ".")
	switch sub {
	case "host":
		return setString(&a.cfg.Jira.Host, value)
	case "me":
		return setString(&a.cfg.Jira.Me, value)
	case "filters":
		return a.setJiraFiltersValue(key, value)
	case "markdown":
		return a.setJiraMarkdownValue(key, value)
	case "attachments":
		return a.setJiraAttachmentsValue(key, value)
	default:
		return fmt.Errorf("%w: %s", tuiconfig.ErrUnknownKey, key)
	}
}

func (a *ConfigAdapter) setJiraFiltersValue(key string, value any) error {
	switch key {
	case "jira.filters.priority":
		return setString(&a.cfg.Jira.Filters.Priority, value)
	case "jira.filters.type":
		return setString(&a.cfg.Jira.Filters.Type, value)
	case "jira.filters.component":
		return setString(&a.cfg.Jira.Filters.Component, value)
	case "jira.filters.reporter":
		return setString(&a.cfg.Jira.Filters.Reporter, value)
	case "jira.filters.assignee":
		return setString(&a.cfg.Jira.Filters.Assignee, value)
	case "jira.filters.order_by":
		return setString(&a.cfg.Jira.Filters.OrderBy, value)
	case "jira.filters.status":
		return setStringSlice(&a.cfg.Jira.Filters.Status, value)
	case "jira.filters.labels":
		return setStringSlice(&a.cfg.Jira.Filters.Labels, value)
	case "jira.filters.custom_args":
		return setStringSlice(&a.cfg.Jira.Filters.CustomArgs, value)
	case "jira.filters.reverse":
		return setBool(&a.cfg.Jira.Filters.Reverse, value)
	default:
		return fmt.Errorf("%w: %s", tuiconfig.ErrUnknownKey, key)
	}
}

func (a *ConfigAdapter) setJiraMarkdownValue(key string, value any) error {
	switch key {
	case "jira.markdown.filename_pattern":
		return setString(&a.cfg.Jira.Markdown.FilenamePattern, value)
	case "jira.markdown.max_depth":
		return setInt(&a.cfg.Jira.Markdown.MaxDepth, value)
	case "jira.markdown.include_comments":
		return setBool(&a.cfg.Jira.Markdown.IncludeComments, value)
	case "jira.markdown.include_attachments":
		return setBool(&a.cfg.Jira.Markdown.IncludeAttachments, value)
	case "jira.markdown.use_relative_links":
		return setBool(&a.cfg.Jira.Markdown.UseRelativeLinks, value)
	case "jira.markdown.include_linked_issues":
		return setBool(&a.cfg.Jira.Markdown.IncludeLinkedIssues, value)
	default:
		return fmt.Errorf("%w: %s", tuiconfig.ErrUnknownKey, key)
	}
}

func (a *ConfigAdapter) setJiraAttachmentsValue(key string, value any) error {
	switch key {
	case "jira.attachments.enabled":
		return setBool(&a.cfg.Jira.Attachments.Enabled, value)
	case "jira.attachments.max_size_mb":
		return setInt64(&a.cfg.Jira.Attachments.MaxSizeMB, value)
	case "jira.attachments.directory":
		return setString(&a.cfg.Jira.Attachments.Directory, value)
	case "jira.attachments.download_timeout_seconds":
		return setInt(&a.cfg.Jira.Attachments.DownloadTimeoutSec, value)
	case "jira.attachments.retry_attempts":
		return setInt(&a.cfg.Jira.Attachments.RetryAttempts, value)
	case "jira.attachments.retry_backoff_ms":
		return setInt(&a.cfg.Jira.Attachments.RetryBackoffMs, value)
	default:
		return fmt.Errorf("%w: %s", tuiconfig.ErrUnknownKey, key)
	}
}

func (a *ConfigAdapter) setFileCopyValue(key string, value any) error {
	switch key {
	case "file_copy.rules":
		return setFileCopyRules(&a.cfg.FileCopy.Rules, value)
	case "file_copy.auto.enabled":
		return setBool(&a.cfg.FileCopy.Auto.Enabled, value)
	case "file_copy.auto.source_worktree":
		return setString(&a.cfg.FileCopy.Auto.SourceWorktree, value)
	case "file_copy.auto.copy_ignored":
		return setBool(&a.cfg.FileCopy.Auto.CopyIgnored, value)
	case "file_copy.auto.copy_untracked":
		return setBool(&a.cfg.FileCopy.Auto.CopyUntracked, value)
	case "file_copy.auto.exclude":
		return setStringSlice(&a.cfg.FileCopy.Auto.Exclude, value)
	default:
		return fmt.Errorf("%w: %s", tuiconfig.ErrUnknownKey, key)
	}
}

// --- typed setter helpers ---.

func setString(dst *string, value any) error {
	v, ok := value.(string)
	if !ok {
		return fmt.Errorf("expected string, got %T", value)
	}
	*dst = v
	return nil
}

func setInt(dst *int, value any) error {
	switch v := value.(type) {
	case int:
		*dst = v
	case int64:
		*dst = int(v)
	default:
		return fmt.Errorf("expected int, got %T", value)
	}
	return nil
}

func setInt64(dst *int64, value any) error {
	switch v := value.(type) {
	case int:
		*dst = int64(v)
	case int64:
		*dst = v
	default:
		return fmt.Errorf("expected int, got %T", value)
	}
	return nil
}

func setBool(dst *bool, value any) error {
	v, ok := value.(bool)
	if !ok {
		return fmt.Errorf("expected bool, got %T", value)
	}
	*dst = v
	return nil
}

func setStringSlice(dst *[]string, value any) error {
	v, ok := value.([]string)
	if !ok {
		return fmt.Errorf("expected []string, got %T", value)
	}
	*dst = v
	return nil
}

func setFileCopyRules(dst *[]FileCopyRule, value any) error {
	v, ok := value.([]FileCopyRule)
	if !ok {
		return fmt.Errorf("expected []FileCopyRule, got %T", value)
	}
	*dst = v
	return nil
}

//nolint:gocritic // ptrToRefParam: pointer needed to replace the entire map in the struct field.
func setWorktreeMap(dst *map[string]WorktreeConfig, value any) error {
	v, ok := value.(map[string]WorktreeConfig)
	if !ok {
		return fmt.Errorf("expected map[string]WorktreeConfig, got %T", value)
	}
	*dst = v
	return nil
}

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeneralFields(t *testing.T) {
	require.Len(t, generalFields, 2, "generalFields should have 2 entries")

	t.Run("default_branch", func(t *testing.T) {
		f := generalFields[0]
		assert.Equal(t, "default_branch", f.Key)
		assert.Equal(t, "Default Branch", f.Label)
		assert.Equal(t, String, f.Type)
		assert.Empty(t, f.Group)
		assert.NotNil(t, f.Validate, "default_branch should have a validator")
	})

	t.Run("worktrees_dir", func(t *testing.T) {
		f := generalFields[1]
		assert.Equal(t, "worktrees_dir", f.Key)
		assert.Equal(t, "Worktrees Directory", f.Label)
		assert.Equal(t, String, f.Type)
		assert.Empty(t, f.Group)
		assert.NotEmpty(t, f.Description)
		assert.NotNil(t, f.Validate, "worktrees_dir should have a validator")
	})
}

func TestJiraFields(t *testing.T) {
	require.Len(t, jiraFields, 24, "jiraFields should have 24 entries")

	// Count fields per group.
	groups := map[string]int{}
	for _, f := range jiraFields {
		groups[f.Group]++
	}
	assert.Equal(t, 2, groups["Connection"])
	assert.Equal(t, 10, groups["Filters"])
	assert.Equal(t, 6, groups["Markdown"])
	assert.Equal(t, 6, groups["Attachments"])

	// Verify all keys start with "jira.".
	for _, f := range jiraFields {
		assert.Regexp(t, `^jira\.`, f.Key, "JIRA field key %q should start with jira.", f.Key)
	}
}

func TestJiraFields_keys_match_yaml_tags(t *testing.T) {
	// Expected keys derived from YAML tags in service.go and jira/types.go.
	expected := []string{
		// Connection
		"jira.host",
		"jira.me",
		// Filters
		"jira.filters.priority",
		"jira.filters.type",
		"jira.filters.component",
		"jira.filters.reporter",
		"jira.filters.assignee",
		"jira.filters.order_by",
		"jira.filters.status",
		"jira.filters.labels",
		"jira.filters.custom_args",
		"jira.filters.reverse",
		// Markdown
		"jira.markdown.filename_pattern",
		"jira.markdown.max_depth",
		"jira.markdown.include_comments",
		"jira.markdown.include_attachments",
		"jira.markdown.use_relative_links",
		"jira.markdown.include_linked_issues",
		// Attachments
		"jira.attachments.enabled",
		"jira.attachments.max_size_mb",
		"jira.attachments.directory",
		"jira.attachments.download_timeout_seconds",
		"jira.attachments.retry_attempts",
		"jira.attachments.retry_backoff_ms",
	}

	require.Len(t, jiraFields, len(expected))
	for i, f := range jiraFields {
		assert.Equal(t, expected[i], f.Key, "jiraFields[%d] key mismatch", i)
	}
}

func TestJiraFields_types(t *testing.T) {
	typeByKey := map[string]FieldType{
		"jira.host":                                 String,
		"jira.me":                                   String,
		"jira.filters.priority":                     String,
		"jira.filters.type":                         String,
		"jira.filters.component":                    String,
		"jira.filters.reporter":                     String,
		"jira.filters.assignee":                     String,
		"jira.filters.order_by":                     String,
		"jira.filters.status":                       StringList,
		"jira.filters.labels":                       StringList,
		"jira.filters.custom_args":                  StringList,
		"jira.filters.reverse":                      Bool,
		"jira.markdown.filename_pattern":            String,
		"jira.markdown.max_depth":                   Int,
		"jira.markdown.include_comments":            Bool,
		"jira.markdown.include_attachments":         Bool,
		"jira.markdown.use_relative_links":          Bool,
		"jira.markdown.include_linked_issues":       Bool,
		"jira.attachments.enabled":                  Bool,
		"jira.attachments.max_size_mb":              Int,
		"jira.attachments.directory":                String,
		"jira.attachments.download_timeout_seconds": Int,
		"jira.attachments.retry_attempts":           Int,
		"jira.attachments.retry_backoff_ms":         Int,
	}
	for _, f := range jiraFields {
		expected, ok := typeByKey[f.Key]
		require.True(t, ok, "unexpected key %q", f.Key)
		assert.Equal(t, expected, f.Type, "type mismatch for %q", f.Key)
	}
}

func TestJiraFields_int_fields_have_validators(t *testing.T) {
	for _, f := range jiraFields {
		if f.Type == Int {
			assert.NotNil(t, f.Validate, "Int field %q should have a validator", f.Key)
		}
	}
}

func TestFileCopyAutoFields(t *testing.T) {
	require.Len(t, fileCopyAutoFields, 5, "fileCopyAutoFields should have 5 entries")

	expected := []struct {
		key   string
		label string
		ft    FieldType
	}{
		{"file_copy.auto.enabled", "Enabled", Bool},
		{"file_copy.auto.source_worktree", "Source Worktree", String},
		{"file_copy.auto.copy_ignored", "Copy Ignored", Bool},
		{"file_copy.auto.copy_untracked", "Copy Untracked", Bool},
		{"file_copy.auto.exclude", "Exclude", StringList},
	}
	for i, exp := range expected {
		f := fileCopyAutoFields[i]
		assert.Equal(t, exp.key, f.Key, "fileCopyAutoFields[%d] key", i)
		assert.Equal(t, exp.label, f.Label, "fileCopyAutoFields[%d] label", i)
		assert.Equal(t, exp.ft, f.Type, "fileCopyAutoFields[%d] type", i)
		assert.Equal(t, "Auto Copy", f.Group, "fileCopyAutoFields[%d] group", i)
	}
}

func TestFileCopyRuleFields(t *testing.T) {
	require.Len(t, fileCopyRuleFields, 2, "fileCopyRuleFields should have 2 entries")

	assert.Equal(t, "source_worktree", fileCopyRuleFields[0].Key)
	assert.Equal(t, String, fileCopyRuleFields[0].Type)

	assert.Equal(t, "files", fileCopyRuleFields[1].Key)
	assert.Equal(t, StringList, fileCopyRuleFields[1].Type)
}

func TestWorktreeEntryFields(t *testing.T) {
	require.Len(t, worktreeEntryFields, 3, "worktreeEntryFields should have 3 entries")

	expected := []struct {
		key   string
		label string
	}{
		{"branch", "Branch"},
		{"merge_into", "Merge Into"},
		{"description", "Description"},
	}
	for i, exp := range expected {
		f := worktreeEntryFields[i]
		assert.Equal(t, exp.key, f.Key, "worktreeEntryFields[%d] key", i)
		assert.Equal(t, exp.label, f.Label, "worktreeEntryFields[%d] label", i)
		assert.Equal(t, String, f.Type, "worktreeEntryFields[%d] type", i)
	}
}

func TestAllFieldsHaveLabels(t *testing.T) {
	allFields := [][]FieldMeta{
		generalFields,
		jiraFields,
		fileCopyAutoFields,
		fileCopyRuleFields,
		worktreeEntryFields,
	}
	for _, fields := range allFields {
		for _, f := range fields {
			assert.NotEmpty(t, f.Label, "field %q should have a label", f.Key)
		}
	}
}

func TestAllFieldsHaveKeys(t *testing.T) {
	allFields := [][]FieldMeta{
		generalFields,
		jiraFields,
		fileCopyAutoFields,
		fileCopyRuleFields,
		worktreeEntryFields,
	}
	for _, fields := range allFields {
		for _, f := range fields {
			assert.NotEmpty(t, f.Key, "field with label %q should have a key", f.Label)
		}
	}
}

func TestNoDuplicateKeys(t *testing.T) {
	// Check uniqueness within each section. Overlay sections (fileCopyRuleFields,
	// worktreeEntryFields) use short keys that may overlap with each other, so we
	// check each section independently.
	sections := map[string][]FieldMeta{
		"generalFields":       generalFields,
		"jiraFields":          jiraFields,
		"fileCopyAutoFields":  fileCopyAutoFields,
		"fileCopyRuleFields":  fileCopyRuleFields,
		"worktreeEntryFields": worktreeEntryFields,
	}
	for name, fields := range sections {
		seen := map[string]bool{}
		for _, f := range fields {
			assert.False(t, seen[f.Key], "duplicate key %q in %s", f.Key, name)
			seen[f.Key] = true
		}
	}
}

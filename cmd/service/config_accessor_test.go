package service

import (
	"gbm/internal/jira"
	"os"
	"testing"

	tuiconfig "gbm/pkg/tui/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestConfig returns a Config populated with distinct non-zero values
// so every GetValue assertion can verify the right field is accessed.
func newTestConfig() *Config {
	return &Config{
		DefaultBranch: "develop",
		WorktreesDir:  "wt",
		Worktrees: map[string]WorktreeConfig{
			"main":    {Branch: "main", Description: "primary worktree"},
			"feature": {Branch: "feature/x", MergeInto: "main", Description: "feature work"},
		},
		Jira: JiraConfig{
			Host: "https://jira.example.com",
			Me:   "alice",
			Filters: jira.JiraFilters{
				Priority:   "High",
				Type:       "Bug",
				Component:  "backend",
				Reporter:   "bob",
				Assignee:   "alice",
				OrderBy:    "created",
				Status:     []string{"Open", "In Progress"},
				Labels:     []string{"urgent"},
				CustomArgs: []string{"--foo"},
				Reverse:    true,
			},
			Markdown: MarkdownConfig{
				FilenamePattern:     "{key}.md",
				MaxDepth:            3,
				IncludeComments:     true,
				IncludeAttachments:  true,
				UseRelativeLinks:    true,
				IncludeLinkedIssues: false,
			},
			Attachments: AttachmentConfig{
				Enabled:            true,
				MaxSizeMB:          50,
				Directory:          ".attachments",
				DownloadTimeoutSec: 30,
				RetryAttempts:      3,
				RetryBackoffMs:     500,
			},
		},
		FileCopy: FileCopyConfig{
			Rules: []FileCopyRule{
				{SourceWorktree: "main", Files: []string{".env", ".env.local"}},
				{SourceWorktree: "develop", Files: []string{"config.json"}},
			},
			Auto: AutoFileCopyConfig{
				Enabled:        true,
				SourceWorktree: "{default}",
				CopyIgnored:    true,
				CopyUntracked:  false,
				Exclude:        []string{"*.log", "node_modules/"},
			},
		},
	}
}

func TestConfigAdapter_implements_ConfigAccessor(t *testing.T) {
	var _ tuiconfig.ConfigAccessor = NewConfigAdapter(&Config{})
}

func TestConfigAdapter_GetValue(t *testing.T) {
	cfg := newTestConfig()
	a := NewConfigAdapter(cfg)

	tests := []struct {
		assert func(t *testing.T, got any)
		key    string
	}{
		// General
		{key: "default_branch", assert: func(t *testing.T, got any) {
			t.Helper()
			assert.Equal(t, "develop", got)
		}},
		{key: "worktrees_dir", assert: func(t *testing.T, got any) {
			t.Helper()
			assert.Equal(t, "wt", got)
		}},

		// JIRA Connection
		{key: "jira.host", assert: func(t *testing.T, got any) {
			t.Helper()
			assert.Equal(t, "https://jira.example.com", got)
		}},
		{key: "jira.me", assert: func(t *testing.T, got any) {
			t.Helper()
			assert.Equal(t, "alice", got)
		}},

		// JIRA Filters
		{key: "jira.filters.priority", assert: func(t *testing.T, got any) {
			t.Helper()
			assert.Equal(t, "High", got)
		}},
		{key: "jira.filters.type", assert: func(t *testing.T, got any) {
			t.Helper()
			assert.Equal(t, "Bug", got)
		}},
		{key: "jira.filters.component", assert: func(t *testing.T, got any) {
			t.Helper()
			assert.Equal(t, "backend", got)
		}},
		{key: "jira.filters.reporter", assert: func(t *testing.T, got any) {
			t.Helper()
			assert.Equal(t, "bob", got)
		}},
		{key: "jira.filters.assignee", assert: func(t *testing.T, got any) {
			t.Helper()
			assert.Equal(t, "alice", got)
		}},
		{key: "jira.filters.order_by", assert: func(t *testing.T, got any) {
			t.Helper()
			assert.Equal(t, "created", got)
		}},
		{key: "jira.filters.status", assert: func(t *testing.T, got any) {
			t.Helper()
			assert.Equal(t, []string{"Open", "In Progress"}, got)
		}},
		{key: "jira.filters.labels", assert: func(t *testing.T, got any) {
			t.Helper()
			assert.Equal(t, []string{"urgent"}, got)
		}},
		{key: "jira.filters.custom_args", assert: func(t *testing.T, got any) {
			t.Helper()
			assert.Equal(t, []string{"--foo"}, got)
		}},
		{key: "jira.filters.reverse", assert: func(t *testing.T, got any) {
			t.Helper()
			assert.Equal(t, true, got)
		}},

		// JIRA Markdown
		{key: "jira.markdown.filename_pattern", assert: func(t *testing.T, got any) {
			t.Helper()
			assert.Equal(t, "{key}.md", got)
		}},
		{key: "jira.markdown.max_depth", assert: func(t *testing.T, got any) {
			t.Helper()
			assert.Equal(t, 3, got)
		}},
		{key: "jira.markdown.include_comments", assert: func(t *testing.T, got any) {
			t.Helper()
			assert.Equal(t, true, got)
		}},
		{key: "jira.markdown.include_attachments", assert: func(t *testing.T, got any) {
			t.Helper()
			assert.Equal(t, true, got)
		}},
		{key: "jira.markdown.use_relative_links", assert: func(t *testing.T, got any) {
			t.Helper()
			assert.Equal(t, true, got)
		}},
		{key: "jira.markdown.include_linked_issues", assert: func(t *testing.T, got any) {
			t.Helper()
			assert.Equal(t, false, got)
		}},

		// JIRA Attachments
		{key: "jira.attachments.enabled", assert: func(t *testing.T, got any) {
			t.Helper()
			assert.Equal(t, true, got)
		}},
		{key: "jira.attachments.max_size_mb", assert: func(t *testing.T, got any) {
			t.Helper()
			assert.Equal(t, int64(50), got)
		}},
		{key: "jira.attachments.directory", assert: func(t *testing.T, got any) {
			t.Helper()
			assert.Equal(t, ".attachments", got)
		}},
		{key: "jira.attachments.download_timeout_seconds", assert: func(t *testing.T, got any) {
			t.Helper()
			assert.Equal(t, 30, got)
		}},
		{key: "jira.attachments.retry_attempts", assert: func(t *testing.T, got any) {
			t.Helper()
			assert.Equal(t, 3, got)
		}},
		{key: "jira.attachments.retry_backoff_ms", assert: func(t *testing.T, got any) {
			t.Helper()
			assert.Equal(t, 500, got)
		}},

		// File Copy Auto
		{key: "file_copy.auto.enabled", assert: func(t *testing.T, got any) {
			t.Helper()
			assert.Equal(t, true, got)
		}},
		{key: "file_copy.auto.source_worktree", assert: func(t *testing.T, got any) {
			t.Helper()
			assert.Equal(t, "{default}", got)
		}},
		{key: "file_copy.auto.copy_ignored", assert: func(t *testing.T, got any) {
			t.Helper()
			assert.Equal(t, true, got)
		}},
		{key: "file_copy.auto.copy_untracked", assert: func(t *testing.T, got any) {
			t.Helper()
			assert.Equal(t, false, got)
		}},
		{key: "file_copy.auto.exclude", assert: func(t *testing.T, got any) {
			t.Helper()
			assert.Equal(t, []string{"*.log", "node_modules/"}, got)
		}},

		// File Copy Rules
		{key: "file_copy.rules", assert: func(t *testing.T, got any) {
			t.Helper()
			rules, ok := got.([]FileCopyRule)
			require.True(t, ok, "expected []FileCopyRule, got %T", got)
			assert.Len(t, rules, 2)
			assert.Equal(t, "main", rules[0].SourceWorktree)
			assert.Equal(t, []string{".env", ".env.local"}, rules[0].Files)
		}},

		// Worktrees
		{key: "worktrees", assert: func(t *testing.T, got any) {
			t.Helper()
			wt, ok := got.(map[string]WorktreeConfig)
			require.True(t, ok, "expected map[string]WorktreeConfig, got %T", got)
			assert.Len(t, wt, 2)
			assert.Equal(t, "main", wt["main"].Branch)
			assert.Equal(t, "feature/x", wt["feature"].Branch)
		}},

		// Unknown key
		{key: "nonexistent.key", assert: func(t *testing.T, got any) {
			t.Helper()
			assert.Nil(t, got)
		}},
	}
	for _, tc := range tests {
		t.Run(tc.key, func(t *testing.T) {
			got := a.GetValue(tc.key)
			tc.assert(t, got)
		})
	}
}

func TestConfigAdapter_SetValue(t *testing.T) {
	tests := []struct {
		value       any
		assert      func(t *testing.T, cfg *Config)
		assertError func(t *testing.T, err error)
		key         string
	}{
		// General
		{
			key: "default_branch", value: "main",
			assert: func(t *testing.T, cfg *Config) {
				t.Helper()
				assert.Equal(t, "main", cfg.DefaultBranch)
			},
			assertError: func(t *testing.T, err error) { t.Helper(); assert.NoError(t, err) },
		},
		{
			key: "worktrees_dir", value: "trees",
			assert: func(t *testing.T, cfg *Config) {
				t.Helper()
				assert.Equal(t, "trees", cfg.WorktreesDir)
			},
			assertError: func(t *testing.T, err error) { t.Helper(); assert.NoError(t, err) },
		},

		// JIRA Connection
		{
			key: "jira.host", value: "https://new.jira.com",
			assert: func(t *testing.T, cfg *Config) {
				t.Helper()
				assert.Equal(t, "https://new.jira.com", cfg.Jira.Host)
			},
			assertError: func(t *testing.T, err error) { t.Helper(); assert.NoError(t, err) },
		},
		{
			key: "jira.me", value: "bob",
			assert: func(t *testing.T, cfg *Config) {
				t.Helper()
				assert.Equal(t, "bob", cfg.Jira.Me)
			},
			assertError: func(t *testing.T, err error) { t.Helper(); assert.NoError(t, err) },
		},

		// JIRA Filters
		{
			key: "jira.filters.priority", value: "Low",
			assert: func(t *testing.T, cfg *Config) {
				t.Helper()
				assert.Equal(t, "Low", cfg.Jira.Filters.Priority)
			},
			assertError: func(t *testing.T, err error) { t.Helper(); assert.NoError(t, err) },
		},
		{
			key: "jira.filters.type", value: "Story",
			assert: func(t *testing.T, cfg *Config) {
				t.Helper()
				assert.Equal(t, "Story", cfg.Jira.Filters.Type)
			},
			assertError: func(t *testing.T, err error) { t.Helper(); assert.NoError(t, err) },
		},
		{
			key: "jira.filters.component", value: "frontend",
			assert: func(t *testing.T, cfg *Config) {
				t.Helper()
				assert.Equal(t, "frontend", cfg.Jira.Filters.Component)
			},
			assertError: func(t *testing.T, err error) { t.Helper(); assert.NoError(t, err) },
		},
		{
			key: "jira.filters.reporter", value: "charlie",
			assert: func(t *testing.T, cfg *Config) {
				t.Helper()
				assert.Equal(t, "charlie", cfg.Jira.Filters.Reporter)
			},
			assertError: func(t *testing.T, err error) { t.Helper(); assert.NoError(t, err) },
		},
		{
			key: "jira.filters.assignee", value: "dave",
			assert: func(t *testing.T, cfg *Config) {
				t.Helper()
				assert.Equal(t, "dave", cfg.Jira.Filters.Assignee)
			},
			assertError: func(t *testing.T, err error) { t.Helper(); assert.NoError(t, err) },
		},
		{
			key: "jira.filters.order_by", value: "updated",
			assert: func(t *testing.T, cfg *Config) {
				t.Helper()
				assert.Equal(t, "updated", cfg.Jira.Filters.OrderBy)
			},
			assertError: func(t *testing.T, err error) { t.Helper(); assert.NoError(t, err) },
		},
		{
			key: "jira.filters.status", value: []string{"Done"},
			assert: func(t *testing.T, cfg *Config) {
				t.Helper()
				assert.Equal(t, []string{"Done"}, cfg.Jira.Filters.Status)
			},
			assertError: func(t *testing.T, err error) { t.Helper(); assert.NoError(t, err) },
		},
		{
			key: "jira.filters.labels", value: []string{"p0", "critical"},
			assert: func(t *testing.T, cfg *Config) {
				t.Helper()
				assert.Equal(t, []string{"p0", "critical"}, cfg.Jira.Filters.Labels)
			},
			assertError: func(t *testing.T, err error) { t.Helper(); assert.NoError(t, err) },
		},
		{
			key: "jira.filters.custom_args", value: []string{"--bar"},
			assert: func(t *testing.T, cfg *Config) {
				t.Helper()
				assert.Equal(t, []string{"--bar"}, cfg.Jira.Filters.CustomArgs)
			},
			assertError: func(t *testing.T, err error) { t.Helper(); assert.NoError(t, err) },
		},
		{
			key: "jira.filters.reverse", value: false,
			assert: func(t *testing.T, cfg *Config) {
				t.Helper()
				assert.False(t, cfg.Jira.Filters.Reverse)
			},
			assertError: func(t *testing.T, err error) { t.Helper(); assert.NoError(t, err) },
		},

		// JIRA Markdown
		{
			key: "jira.markdown.filename_pattern", value: "issue-{key}.md",
			assert: func(t *testing.T, cfg *Config) {
				t.Helper()
				assert.Equal(t, "issue-{key}.md", cfg.Jira.Markdown.FilenamePattern)
			},
			assertError: func(t *testing.T, err error) { t.Helper(); assert.NoError(t, err) },
		},
		{
			key: "jira.markdown.max_depth", value: 5,
			assert: func(t *testing.T, cfg *Config) {
				t.Helper()
				assert.Equal(t, 5, cfg.Jira.Markdown.MaxDepth)
			},
			assertError: func(t *testing.T, err error) { t.Helper(); assert.NoError(t, err) },
		},
		{
			key: "jira.markdown.include_comments", value: false,
			assert: func(t *testing.T, cfg *Config) {
				t.Helper()
				assert.False(t, cfg.Jira.Markdown.IncludeComments)
			},
			assertError: func(t *testing.T, err error) { t.Helper(); assert.NoError(t, err) },
		},
		{
			key: "jira.markdown.include_attachments", value: false,
			assert: func(t *testing.T, cfg *Config) {
				t.Helper()
				assert.False(t, cfg.Jira.Markdown.IncludeAttachments)
			},
			assertError: func(t *testing.T, err error) { t.Helper(); assert.NoError(t, err) },
		},
		{
			key: "jira.markdown.use_relative_links", value: false,
			assert: func(t *testing.T, cfg *Config) {
				t.Helper()
				assert.False(t, cfg.Jira.Markdown.UseRelativeLinks)
			},
			assertError: func(t *testing.T, err error) { t.Helper(); assert.NoError(t, err) },
		},
		{
			key: "jira.markdown.include_linked_issues", value: true,
			assert: func(t *testing.T, cfg *Config) {
				t.Helper()
				assert.True(t, cfg.Jira.Markdown.IncludeLinkedIssues)
			},
			assertError: func(t *testing.T, err error) { t.Helper(); assert.NoError(t, err) },
		},

		// JIRA Attachments
		{
			key: "jira.attachments.enabled", value: false,
			assert: func(t *testing.T, cfg *Config) {
				t.Helper()
				assert.False(t, cfg.Jira.Attachments.Enabled)
			},
			assertError: func(t *testing.T, err error) { t.Helper(); assert.NoError(t, err) },
		},
		{
			key: "jira.attachments.max_size_mb", value: int64(100),
			assert: func(t *testing.T, cfg *Config) {
				t.Helper()
				assert.Equal(t, int64(100), cfg.Jira.Attachments.MaxSizeMB)
			},
			assertError: func(t *testing.T, err error) { t.Helper(); assert.NoError(t, err) },
		},
		{
			key: "jira.attachments.max_size_mb", value: 25,
			assert: func(t *testing.T, cfg *Config) {
				t.Helper()
				assert.Equal(t, int64(25), cfg.Jira.Attachments.MaxSizeMB)
			},
			assertError: func(t *testing.T, err error) { t.Helper(); assert.NoError(t, err) },
		},
		{
			key: "jira.attachments.directory", value: "downloads",
			assert: func(t *testing.T, cfg *Config) {
				t.Helper()
				assert.Equal(t, "downloads", cfg.Jira.Attachments.Directory)
			},
			assertError: func(t *testing.T, err error) { t.Helper(); assert.NoError(t, err) },
		},
		{
			key: "jira.attachments.download_timeout_seconds", value: 60,
			assert: func(t *testing.T, cfg *Config) {
				t.Helper()
				assert.Equal(t, 60, cfg.Jira.Attachments.DownloadTimeoutSec)
			},
			assertError: func(t *testing.T, err error) { t.Helper(); assert.NoError(t, err) },
		},
		{
			key: "jira.attachments.retry_attempts", value: 5,
			assert: func(t *testing.T, cfg *Config) {
				t.Helper()
				assert.Equal(t, 5, cfg.Jira.Attachments.RetryAttempts)
			},
			assertError: func(t *testing.T, err error) { t.Helper(); assert.NoError(t, err) },
		},
		{
			key: "jira.attachments.retry_backoff_ms", value: 1000,
			assert: func(t *testing.T, cfg *Config) {
				t.Helper()
				assert.Equal(t, 1000, cfg.Jira.Attachments.RetryBackoffMs)
			},
			assertError: func(t *testing.T, err error) { t.Helper(); assert.NoError(t, err) },
		},

		// File Copy Auto
		{
			key: "file_copy.auto.enabled", value: false,
			assert: func(t *testing.T, cfg *Config) {
				t.Helper()
				assert.False(t, cfg.FileCopy.Auto.Enabled)
			},
			assertError: func(t *testing.T, err error) { t.Helper(); assert.NoError(t, err) },
		},
		{
			key: "file_copy.auto.source_worktree", value: "main",
			assert: func(t *testing.T, cfg *Config) {
				t.Helper()
				assert.Equal(t, "main", cfg.FileCopy.Auto.SourceWorktree)
			},
			assertError: func(t *testing.T, err error) { t.Helper(); assert.NoError(t, err) },
		},
		{
			key: "file_copy.auto.copy_ignored", value: false,
			assert: func(t *testing.T, cfg *Config) {
				t.Helper()
				assert.False(t, cfg.FileCopy.Auto.CopyIgnored)
			},
			assertError: func(t *testing.T, err error) { t.Helper(); assert.NoError(t, err) },
		},
		{
			key: "file_copy.auto.copy_untracked", value: true,
			assert: func(t *testing.T, cfg *Config) {
				t.Helper()
				assert.True(t, cfg.FileCopy.Auto.CopyUntracked)
			},
			assertError: func(t *testing.T, err error) { t.Helper(); assert.NoError(t, err) },
		},
		{
			key: "file_copy.auto.exclude", value: []string{"build/"},
			assert: func(t *testing.T, cfg *Config) {
				t.Helper()
				assert.Equal(t, []string{"build/"}, cfg.FileCopy.Auto.Exclude)
			},
			assertError: func(t *testing.T, err error) { t.Helper(); assert.NoError(t, err) },
		},

		// File Copy Rules
		{
			key: "file_copy.rules", value: []FileCopyRule{
				{SourceWorktree: "staging", Files: []string{"db.conf"}},
			},
			assert: func(t *testing.T, cfg *Config) {
				t.Helper()
				require.Len(t, cfg.FileCopy.Rules, 1)
				assert.Equal(t, "staging", cfg.FileCopy.Rules[0].SourceWorktree)
				assert.Equal(t, []string{"db.conf"}, cfg.FileCopy.Rules[0].Files)
			},
			assertError: func(t *testing.T, err error) { t.Helper(); assert.NoError(t, err) },
		},

		// Worktrees
		{
			key: "worktrees", value: map[string]WorktreeConfig{
				"prod": {Branch: "production", Description: "production worktree"},
			},
			assert: func(t *testing.T, cfg *Config) {
				t.Helper()
				require.Len(t, cfg.Worktrees, 1)
				assert.Equal(t, "production", cfg.Worktrees["prod"].Branch)
				assert.Equal(t, "production worktree", cfg.Worktrees["prod"].Description)
			},
			assertError: func(t *testing.T, err error) { t.Helper(); assert.NoError(t, err) },
		},
	}
	for _, tc := range tests {
		t.Run(tc.key, func(t *testing.T) {
			cfg := newTestConfig()
			a := NewConfigAdapter(cfg)
			err := a.SetValue(tc.key, tc.value)
			tc.assertError(t, err)
			tc.assert(t, cfg)
		})
	}
}

func TestConfigAdapter_SetValue_unknown_key(t *testing.T) {
	cfg := &Config{}
	a := NewConfigAdapter(cfg)
	err := a.SetValue("nonexistent.key", "val")
	require.ErrorIs(t, err, tuiconfig.ErrUnknownKey)
	require.ErrorContains(t, err, "nonexistent.key")
}

func TestConfigAdapter_SetValue_type_mismatch(t *testing.T) {
	tests := []struct {
		value       any
		assertError func(t *testing.T, err error)
		name        string
		key         string
	}{
		{
			name: "string field given int",
			key:  "default_branch", value: 42,
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.ErrorContains(t, err, "expected string")
			},
		},
		{
			name: "bool field given string",
			key:  "jira.filters.reverse", value: "true",
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.ErrorContains(t, err, "expected bool")
			},
		},
		{
			name: "int field given string",
			key:  "jira.markdown.max_depth", value: "3",
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.ErrorContains(t, err, "expected int")
			},
		},
		{
			name: "string slice field given string",
			key:  "jira.filters.status", value: "Open",
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.ErrorContains(t, err, "expected []string")
			},
		},
		{
			name: "file_copy.rules given wrong type",
			key:  "file_copy.rules", value: "wrong type",
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.ErrorContains(t, err, "expected []FileCopyRule")
			},
		},
		{
			name: "worktrees given wrong type",
			key:  "worktrees", value: 42,
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.ErrorContains(t, err, "expected map[string]WorktreeConfig")
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := newTestConfig()
			a := NewConfigAdapter(cfg)
			err := a.SetValue(tc.key, tc.value)
			tc.assertError(t, err)
		})
	}
}

func TestConfigAdapter_GetValue_SetValue_roundtrip(t *testing.T) {
	cfg := &Config{DefaultBranch: "main", WorktreesDir: "worktrees"}
	a := NewConfigAdapter(cfg)

	// Verify initial value
	assert.Equal(t, "main", a.GetValue("default_branch"))

	// Set new value
	err := a.SetValue("default_branch", "develop")
	require.NoError(t, err)

	// Verify updated value
	assert.Equal(t, "develop", a.GetValue("default_branch"))
}

func TestConfigAdapter_ReloadFromFile(t *testing.T) {
	testCases := []struct {
		assert      func(t *testing.T, cfg *Config)
		assertError func(t *testing.T, err error)
		name        string
		yamlContent string
	}{
		{
			name:        "reload updates config fields",
			yamlContent: "default_branch: develop\nworktrees_dir: wt\n",
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
			assert: func(t *testing.T, cfg *Config) {
				t.Helper()
				assert.Equal(t, "develop", cfg.DefaultBranch)
				assert.Equal(t, "wt", cfg.WorktreesDir)
			},
		},
		{
			name:        "reload with invalid YAML returns error",
			yamlContent: "{{bad yaml",
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.ErrorContains(t, err, "unmarshal config for reload")
			},
			assert: func(t *testing.T, cfg *Config) {
				t.Helper()
				// Config should remain unchanged on error.
				assert.Equal(t, "main", cfg.DefaultBranch)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			path := tmpDir + "/config.yaml"
			require.NoError(t, os.WriteFile(path, []byte(tc.yamlContent), 0o644))

			cfg := &Config{DefaultBranch: "main", WorktreesDir: "worktrees"}
			a := NewConfigAdapter(cfg)

			err := a.ReloadFromFile(path)
			tc.assertError(t, err)
			tc.assert(t, cfg)
		})
	}
}

func TestConfigAdapter_ReloadFromFile_nonexistent(t *testing.T) {
	cfg := &Config{}
	a := NewConfigAdapter(cfg)
	err := a.ReloadFromFile("/nonexistent/path/config.yaml")
	assert.ErrorContains(t, err, "read config file for reload")
}

func TestConfigAdapter_GetValue_covers_all_section_keys(t *testing.T) {
	// Ensure every key declared in sections.go is handled by GetValue.
	// This test will fail at compile time if a new key is added to sections
	// but not to the adapter switch.
	cfg := newTestConfig()
	a := NewConfigAdapter(cfg)

	allKeys := []string{
		// General
		"default_branch", "worktrees_dir",
		// JIRA Connection
		"jira.host", "jira.me",
		// JIRA Filters
		"jira.filters.priority", "jira.filters.type", "jira.filters.component",
		"jira.filters.reporter", "jira.filters.assignee", "jira.filters.order_by",
		"jira.filters.status", "jira.filters.labels", "jira.filters.custom_args",
		"jira.filters.reverse",
		// JIRA Markdown
		"jira.markdown.filename_pattern", "jira.markdown.max_depth",
		"jira.markdown.include_comments", "jira.markdown.include_attachments",
		"jira.markdown.use_relative_links", "jira.markdown.include_linked_issues",
		// JIRA Attachments
		"jira.attachments.enabled", "jira.attachments.max_size_mb",
		"jira.attachments.directory", "jira.attachments.download_timeout_seconds",
		"jira.attachments.retry_attempts", "jira.attachments.retry_backoff_ms",
		// File Copy Rules
		"file_copy.rules",
		// File Copy Auto
		"file_copy.auto.enabled", "file_copy.auto.source_worktree",
		"file_copy.auto.copy_ignored", "file_copy.auto.copy_untracked",
		"file_copy.auto.exclude",
		// Worktrees
		"worktrees",
	}
	for _, key := range allKeys {
		t.Run(key, func(t *testing.T) {
			got := a.GetValue(key)
			assert.NotNil(t, got, "GetValue(%q) should return non-nil for a populated config", key)
		})
	}
}

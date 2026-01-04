package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateConfig_Valid(t *testing.T) {
	cfg := Config{
		DefaultBranch: "main",
		WorktreesDir:  "worktrees",
	}
	err := validateConfig(&cfg)
	require.NoError(t, err)
}

func TestValidateConfig_MissingDefaultBranch(t *testing.T) {
	cfg := Config{
		WorktreesDir: "worktrees",
	}
	err := validateConfig(&cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "default_branch")
	assert.Contains(t, err.Error(), "required")
}

func TestValidateConfig_MissingWorktreesDir(t *testing.T) {
	cfg := Config{
		DefaultBranch: "main",
	}
	err := validateConfig(&cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "worktrees_dir")
	assert.Contains(t, err.Error(), "required")
}

func TestValidateConfig_EmptyDefaultBranch(t *testing.T) {
	cfg := Config{
		DefaultBranch: "",
		WorktreesDir:  "worktrees",
	}
	err := validateConfig(&cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "default_branch")
}

func TestValidateConfig_EmptyWorktreesDir(t *testing.T) {
	cfg := Config{
		DefaultBranch: "main",
		WorktreesDir:  "",
	}
	err := validateConfig(&cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "worktrees_dir")
}

func TestValidateConfig_WithJira(t *testing.T) {
	cfg := Config{
		DefaultBranch: "main",
		WorktreesDir:  "worktrees",
		Jira:          JiraConfig{
			// Empty JIRA config should still validate
		},
	}
	err := validateConfig(&cfg)
	require.NoError(t, err)
}

func TestValidateConfig_WithFileCopy(t *testing.T) {
	cfg := Config{
		DefaultBranch: "main",
		WorktreesDir:  "worktrees",
		FileCopy: FileCopyConfig{
			Auto: AutoFileCopyConfig{
				Enabled:        true,
				SourceWorktree: "{default}",
			},
		},
	}
	err := validateConfig(&cfg)
	require.NoError(t, err)
}

func TestValidateTemplateVars_Valid(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"no variables", "worktrees"},
		{"gitroot variable", "../{gitroot}-worktrees"},
		{"branch variable", "../../{branch}/worktrees"},
		{"issue variable", "{issue}/worktrees"},
		{"multiple variables", "../{gitroot}-{branch}-{issue}"},
		{"with home", "~/dev/{gitroot}/worktrees"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTemplateVars(tt.path)
			require.NoError(t, err, "valid template vars should not error")
		})
	}
}

func TestValidateTemplateVars_Invalid(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		expectErr string
	}{
		{"invalid variable", "{invalid}", "invalid template variable"},
		{"unclosed brace", "{gitroot", "unclosed template variable"},
		{"unknown variable", "{foo}", "invalid template variable"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTemplateVars(tt.path)
			require.Error(t, err, "invalid template vars should error")
			assert.Contains(t, err.Error(), tt.expectErr)
		})
	}
}

func TestTranslateValidationTag(t *testing.T) {
	tests := []struct {
		tag      string
		expected string
	}{
		{"required", "required field is empty"},
		{"url", "must be valid URL"},
		{"email", "must be valid email address"},
		{"min", "value too short"},
		{"max", "value too long"},
		{"unknown", "validation failed (unknown)"},
	}

	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			result := translateValidationTag(tt.tag)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatValidationError(t *testing.T) {
	cfg := Config{
		WorktreesDir: "worktrees",
		// Missing DefaultBranch
	}

	err := validateConfig(&cfg)
	require.Error(t, err)

	// Error should be formatted nicely
	errMsg := err.Error()
	assert.Contains(t, errMsg, "invalid config")
	assert.Contains(t, errMsg, "default_branch")
	assert.Contains(t, errMsg, "required field is empty")
}

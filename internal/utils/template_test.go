package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpandTemplate(t *testing.T) {
	tests := []struct {
		name     string
		template string
		vars     map[string]string
		want     string
	}{
		{
			name:     "single variable",
			template: "../{gitroot}-worktrees",
			vars:     map[string]string{"gitroot": "gbm"},
			want:     "../gbm-worktrees",
		},
		{
			name:     "multiple variables",
			template: "~/dev/{gitroot}/branches/{branch}",
			vars:     map[string]string{"gitroot": "gbm", "branch": "feature"},
			want:     "~/dev/gbm/branches/feature",
		},
		{
			name:     "missing variable not replaced",
			template: "~/dev/{gitroot}/{branch}",
			vars:     map[string]string{"gitroot": "gbm"}, // branch missing
			want:     "~/dev/gbm/{branch}",
		},
		{
			name:     "repeated variable",
			template: "{gitroot}-{gitroot}",
			vars:     map[string]string{"gitroot": "gbm"},
			want:     "gbm-gbm",
		},
		{
			name:     "no variables",
			template: "worktrees",
			vars:     map[string]string{"gitroot": "gbm"},
			want:     "worktrees",
		},
		{
			name:     "empty map",
			template: "worktrees/{gitroot}",
			vars:     map[string]string{},
			want:     "worktrees/{gitroot}",
		},
		{
			name:     "issue template",
			template: "issues/{issue}",
			vars:     map[string]string{"issue": "PROJ-123"},
			want:     "issues/PROJ-123",
		},
		{
			name:     "complex path",
			template: "../{gitroot}/worktrees/{branch}/src",
			vars:     map[string]string{"gitroot": "myrepo", "branch": "feature/test"},
			want:     "../myrepo/worktrees/feature/test/src",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExpandTemplate(tt.template, tt.vars)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestGetTemplateVars(t *testing.T) {
	// Use a real path for testing
	repoRoot := "/path/to/gbm"
	vars := GetTemplateVars(repoRoot)

	require.NotNil(t, vars)
	assert.Equal(t, "gbm", vars["gitroot"])
	assert.Equal(t, 1, len(vars))
}

func TestGetTemplateVars_VariousRepoPaths(t *testing.T) {
	tests := []struct {
		name     string
		repoRoot string
		want     string
	}{
		{
			name:     "simple repo name",
			repoRoot: "/home/user/projects/myrepo",
			want:     "myrepo",
		},
		{
			name:     "repo with dashes",
			repoRoot: "/home/user/git-branch-manager",
			want:     "git-branch-manager",
		},
		{
			name:     "nested path",
			repoRoot: "/very/long/nested/path/to/repo",
			want:     "repo",
		},
		{
			name:     "single level",
			repoRoot: "/repo",
			want:     "repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vars := GetTemplateVars(tt.repoRoot)
			assert.Equal(t, tt.want, vars["gitroot"])
		})
	}
}

func TestExpandPath(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	repoRoot := "/path/to/repo"

	tests := []struct {
		name     string
		path     string
		repoRoot string
		want     string
	}{
		{
			name:     "expand home",
			path:     "~/dev/worktrees",
			repoRoot: repoRoot,
			want:     filepath.Join(home, "dev/worktrees"),
		},
		{
			name:     "expand only tilde",
			path:     "~",
			repoRoot: repoRoot,
			want:     home,
		},
		{
			name:     "relative path",
			path:     "worktrees",
			repoRoot: repoRoot,
			want:     "/path/to/repo/worktrees",
		},
		{
			name:     "relative parent path",
			path:     "../worktrees",
			repoRoot: repoRoot,
			want:     "/path/to/worktrees",
		},
		{
			name:     "absolute path unchanged",
			path:     "/absolute/path",
			repoRoot: repoRoot,
			want:     "/absolute/path",
		},
		{
			name:     "complex relative path",
			path:     "../../parent/worktrees",
			repoRoot: repoRoot,
			want:     "/path/parent/worktrees",
		},
		{
			name:     "dot relative path",
			path:     "./worktrees",
			repoRoot: repoRoot,
			want:     "/path/to/repo/worktrees",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExpandPath(tt.path, tt.repoRoot)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestExpandPath_InvalidHome(t *testing.T) {
	// Test that we handle home expansion gracefully
	// even if UserHomeDir() fails (shouldn't happen in practice)
	repoRoot := "/path/to/repo"

	// Test with a path that doesn't start with ~
	result := ExpandPath("worktrees", repoRoot)
	assert.Equal(t, "/path/to/repo/worktrees", result)
}

func TestExpandPath_CleansPaths(t *testing.T) {
	// Test that paths are cleaned (double slashes, etc.)
	repoRoot := "/path/to/repo"

	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "removes double slashes",
			path: "worktrees//branches",
			want: "/path/to/repo/worktrees/branches",
		},
		{
			name: "handles trailing slashes",
			path: "worktrees/",
			want: "/path/to/repo/worktrees",
		},
		{
			name: "multiple dots",
			path: "../../worktrees",
			want: "/path/worktrees",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExpandPath(tt.path, repoRoot)
			assert.Equal(t, tt.want, result)
		})
	}
}

// Integration test: template expansion + path expansion together
func TestTemplateAndPathExpansion(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	repoRoot := "/home/user/projects/gbm"
	vars := GetTemplateVars(repoRoot)

	// Scenario 1: dynamic path with template variable
	template := "~/{gitroot}-worktrees"
	expanded := ExpandTemplate(template, vars)
	// expanded should be "~/gbm-worktrees"
	assert.Equal(t, "~/gbm-worktrees", expanded)

	// Now expand the path part
	final := ExpandPath(expanded, repoRoot)
	expected := filepath.Join(home, "gbm-worktrees")
	assert.Equal(t, expected, final)

	// Scenario 2: relative path with template
	template2 := "../{gitroot}-worktrees"
	expanded2 := ExpandTemplate(template2, vars)
	assert.Equal(t, "../gbm-worktrees", expanded2)

	final2 := ExpandPath(expanded2, repoRoot)
	assert.Equal(t, "/home/user/projects/gbm-worktrees", final2)
}

// Test edge cases
func TestExpandTemplate_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		template string
		vars     map[string]string
		want     string
	}{
		{
			name:     "braces in string that aren't variables",
			template: "path {with} {multiple} {items}",
			vars:     map[string]string{},
			want:     "path {with} {multiple} {items}",
		},
		{
			name:     "empty variable value",
			template: "path/{name}",
			vars:     map[string]string{"name": ""},
			want:     "path/",
		},
		{
			name:     "variable with special characters",
			template: "{name}",
			vars:     map[string]string{"name": "feature/test-123"},
			want:     "feature/test-123",
		},
		{
			name:     "case sensitivity",
			template: "{gitroot}/{GITROOT}",
			vars:     map[string]string{"gitroot": "lower", "GITROOT": "upper"},
			want:     "lower/upper",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExpandTemplate(tt.template, tt.vars)
			assert.Equal(t, tt.want, result)
		})
	}
}

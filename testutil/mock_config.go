package testutil

import (
	"gbm/pkg/tui"
)

// MockWorktreeConfig implements tui.WorktreeConfig for testing.
type MockWorktreeConfig struct {
	branch    string
	mergeInto string
}

// GetBranch returns the branch name.
func (m *MockWorktreeConfig) GetBranch() string {
	return m.branch
}

// GetMergeInto returns the merge target branch.
func (m *MockWorktreeConfig) GetMergeInto() string {
	return m.mergeInto
}

// MockRepoConfig implements tui.RepoConfig for testing.
type MockRepoConfig struct {
	worktrees map[string]tui.WorktreeConfig
}

// NewMockRepoConfig creates a new empty MockRepoConfig.
func NewMockRepoConfig() *MockRepoConfig {
	return &MockRepoConfig{
		worktrees: make(map[string]tui.WorktreeConfig),
	}
}

// WithWorktree adds a worktree configuration and returns the MockRepoConfig for chaining.
func (m *MockRepoConfig) WithWorktree(name, branch, mergeInto string) *MockRepoConfig {
	m.worktrees[name] = &MockWorktreeConfig{
		branch:    branch,
		mergeInto: mergeInto,
	}
	return m
}

// GetWorktrees returns the worktrees configuration.
func (m *MockRepoConfig) GetWorktrees() map[string]tui.WorktreeConfig {
	if m.worktrees == nil {
		return make(map[string]tui.WorktreeConfig)
	}
	return m.worktrees
}

// Verify that MockRepoConfig implements tui.RepoConfig
var _ tui.RepoConfig = (*MockRepoConfig)(nil)

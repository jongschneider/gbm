package service

import (
	"gbm/pkg/tui"
)

// ConfigAdapter wraps the service Config to implement tui.RepoConfig interface.
type ConfigAdapter struct {
	config *Config
}

// NewConfigAdapter creates a new ConfigAdapter wrapping the given Config.
func NewConfigAdapter(cfg *Config) *ConfigAdapter {
	return &ConfigAdapter{config: cfg}
}

// GetWorktrees returns a map of worktree names to WorktreeConfigAdapter instances.
// Implements tui.RepoConfig interface.
func (ca *ConfigAdapter) GetWorktrees() map[string]tui.WorktreeConfig {
	if ca.config == nil || ca.config.Worktrees == nil {
		return make(map[string]tui.WorktreeConfig)
	}

	result := make(map[string]tui.WorktreeConfig, len(ca.config.Worktrees))
	for name, wtCfg := range ca.config.Worktrees {
		result[name] = NewWorktreeConfigAdapter(&wtCfg)
	}
	return result
}

// WorktreeConfigAdapter wraps the service WorktreeConfig to implement tui.WorktreeConfig interface.
type WorktreeConfigAdapter struct {
	config *WorktreeConfig
}

// NewWorktreeConfigAdapter creates a new WorktreeConfigAdapter wrapping the given WorktreeConfig.
func NewWorktreeConfigAdapter(cfg *WorktreeConfig) *WorktreeConfigAdapter {
	return &WorktreeConfigAdapter{config: cfg}
}

// GetBranch returns the branch name configured for this worktree.
// Implements tui.WorktreeConfig interface.
func (wca *WorktreeConfigAdapter) GetBranch() string {
	if wca.config == nil {
		return ""
	}
	return wca.config.Branch
}

// GetMergeInto returns the target branch for merging this worktree back.
// Implements tui.WorktreeConfig interface.
func (wca *WorktreeConfigAdapter) GetMergeInto() string {
	if wca.config == nil {
		return ""
	}
	return wca.config.MergeInto
}

// MockRepoConfig is a mock implementation of tui.RepoConfig for testing without real config.
type MockRepoConfig struct {
	worktrees map[string]tui.WorktreeConfig
}

// NewMockRepoConfig creates a new MockRepoConfig with an empty worktree map.
func NewMockRepoConfig() *MockRepoConfig {
	return &MockRepoConfig{
		worktrees: make(map[string]tui.WorktreeConfig),
	}
}

// WithWorktree adds a worktree configuration to the mock.
func (mrc *MockRepoConfig) WithWorktree(name, branch, mergeInto string) *MockRepoConfig {
	mrc.worktrees[name] = NewWorktreeConfigAdapter(&WorktreeConfig{
		Branch:    branch,
		MergeInto: mergeInto,
	})
	return mrc
}

// GetWorktrees returns the mock worktree configurations.
func (mrc *MockRepoConfig) GetWorktrees() map[string]tui.WorktreeConfig {
	return mrc.worktrees
}

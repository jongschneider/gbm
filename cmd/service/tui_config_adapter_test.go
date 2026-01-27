package service

import (
	"gbm/pkg/tui"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigAdapter_GetWorktrees(t *testing.T) {
	testCases := []struct {
		name      string
		config    *Config
		expect    func(t *testing.T, got map[string]tui.WorktreeConfig)
		expectErr func(t *testing.T, err error)
	}{
		{
			name:   "nil config returns empty map",
			config: nil,
			expect: func(t *testing.T, got map[string]tui.WorktreeConfig) {
				assert.NotNil(t, got)
				assert.Len(t, got, 0)
			},
			expectErr: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name: "config with no worktrees returns empty map",
			config: &Config{
				DefaultBranch: "main",
				WorktreesDir:  "worktrees",
				Worktrees:     nil,
			},
			expect: func(t *testing.T, got map[string]tui.WorktreeConfig) {
				assert.NotNil(t, got)
				assert.Len(t, got, 0)
			},
			expectErr: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name: "config with multiple worktrees returns all",
			config: &Config{
				DefaultBranch: "main",
				WorktreesDir:  "worktrees",
				Worktrees: map[string]WorktreeConfig{
					"feature-x": {
						Branch:      "feature/feature-x",
						MergeInto:   "develop",
						Description: "Feature X branch",
					},
					"hotfix-y": {
						Branch:      "hotfix/hotfix-y",
						MergeInto:   "main",
						Description: "Hotfix Y branch",
					},
				},
			},
			expect: func(t *testing.T, got map[string]tui.WorktreeConfig) {
				assert.Len(t, got, 2)

				// Check feature-x
				featureX, ok := got["feature-x"]
				assert.True(t, ok)
				assert.Equal(t, "feature/feature-x", featureX.GetBranch())
				assert.Equal(t, "develop", featureX.GetMergeInto())

				// Check hotfix-y
				hotfixY, ok := got["hotfix-y"]
				assert.True(t, ok)
				assert.Equal(t, "hotfix/hotfix-y", hotfixY.GetBranch())
				assert.Equal(t, "main", hotfixY.GetMergeInto())
			},
			expectErr: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			adapter := NewConfigAdapter(tc.config)
			got := adapter.GetWorktrees()
			tc.expect(t, got)
			tc.expectErr(t, nil)
		})
	}
}

func TestWorktreeConfigAdapter_GetBranch(t *testing.T) {
	testCases := []struct {
		name      string
		config    *WorktreeConfig
		expect    func(t *testing.T, got string)
		expectErr func(t *testing.T, err error)
	}{
		{
			name:   "nil config returns empty string",
			config: nil,
			expect: func(t *testing.T, got string) {
				assert.Equal(t, "", got)
			},
			expectErr: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name: "config with branch returns branch",
			config: &WorktreeConfig{
				Branch:    "feature/new-feature",
				MergeInto: "develop",
			},
			expect: func(t *testing.T, got string) {
				assert.Equal(t, "feature/new-feature", got)
			},
			expectErr: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name: "config with empty branch returns empty string",
			config: &WorktreeConfig{
				Branch:    "",
				MergeInto: "main",
			},
			expect: func(t *testing.T, got string) {
				assert.Equal(t, "", got)
			},
			expectErr: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			adapter := NewWorktreeConfigAdapter(tc.config)
			got := adapter.GetBranch()
			tc.expect(t, got)
			tc.expectErr(t, nil)
		})
	}
}

func TestWorktreeConfigAdapter_GetMergeInto(t *testing.T) {
	testCases := []struct {
		name      string
		config    *WorktreeConfig
		expect    func(t *testing.T, got string)
		expectErr func(t *testing.T, err error)
	}{
		{
			name:   "nil config returns empty string",
			config: nil,
			expect: func(t *testing.T, got string) {
				assert.Equal(t, "", got)
			},
			expectErr: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name: "config with merge_into returns merge_into",
			config: &WorktreeConfig{
				Branch:    "feature/new-feature",
				MergeInto: "develop",
			},
			expect: func(t *testing.T, got string) {
				assert.Equal(t, "develop", got)
			},
			expectErr: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name: "config with empty merge_into returns empty string",
			config: &WorktreeConfig{
				Branch:    "feature/new-feature",
				MergeInto: "",
			},
			expect: func(t *testing.T, got string) {
				assert.Equal(t, "", got)
			},
			expectErr: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			adapter := NewWorktreeConfigAdapter(tc.config)
			got := adapter.GetMergeInto()
			tc.expect(t, got)
			tc.expectErr(t, nil)
		})
	}
}

func TestMockRepoConfig(t *testing.T) {
	testCases := []struct {
		name      string
		setup     func() *MockRepoConfig
		expect    func(t *testing.T, got map[string]tui.WorktreeConfig)
		expectErr func(t *testing.T, err error)
	}{
		{
			name: "new mock config is empty",
			setup: func() *MockRepoConfig {
				return NewMockRepoConfig()
			},
			expect: func(t *testing.T, got map[string]tui.WorktreeConfig) {
				assert.Len(t, got, 0)
			},
			expectErr: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name: "with worktree adds configuration",
			setup: func() *MockRepoConfig {
				return NewMockRepoConfig().
					WithWorktree("feature-a", "feature/feature-a", "develop").
					WithWorktree("hotfix-b", "hotfix/hotfix-b", "main")
			},
			expect: func(t *testing.T, got map[string]tui.WorktreeConfig) {
				assert.Len(t, got, 2)

				featureA, ok := got["feature-a"]
				assert.True(t, ok)
				assert.Equal(t, "feature/feature-a", featureA.GetBranch())
				assert.Equal(t, "develop", featureA.GetMergeInto())

				hotfixB, ok := got["hotfix-b"]
				assert.True(t, ok)
				assert.Equal(t, "hotfix/hotfix-b", hotfixB.GetBranch())
				assert.Equal(t, "main", hotfixB.GetMergeInto())
			},
			expectErr: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mock := tc.setup()
			got := mock.GetWorktrees()
			tc.expect(t, got)
			tc.expectErr(t, nil)
		})
	}
}

func TestConfigAdapter_ImplementsInterface(t *testing.T) {
	// Compile-time check that ConfigAdapter implements tui.RepoConfig
	var _ tui.RepoConfig = (*ConfigAdapter)(nil)

	// Compile-time check that WorktreeConfigAdapter implements tui.WorktreeConfig
	var _ tui.WorktreeConfig = (*WorktreeConfigAdapter)(nil)

	// Compile-time check that MockRepoConfig implements tui.RepoConfig
	var _ tui.RepoConfig = (*MockRepoConfig)(nil)
}

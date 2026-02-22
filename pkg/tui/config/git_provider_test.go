package config

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockGitProvider implements GitDataProvider for testing.
type mockGitProvider struct {
	branchErr     error
	worktreeErr   error
	branches      []string
	worktreeNames []string
}

func (m *mockGitProvider) ListBranches() ([]string, error) {
	return m.branches, m.branchErr
}

func (m *mockGitProvider) ListWorktreeNames() ([]string, error) {
	return m.worktreeNames, m.worktreeErr
}

func TestConfigModel_InitWithGitProvider(t *testing.T) {
	testCases := []struct {
		name   string
		assert func(t *testing.T, cmd tea.Cmd)
		opts   []ConfigModelOption
	}{
		{
			name: "nil provider returns nil cmd",
			opts: nil,
			assert: func(t *testing.T, cmd tea.Cmd) {
				t.Helper()
				assert.Nil(t, cmd)
			},
		},
		{
			name: "with provider returns batch cmd",
			opts: []ConfigModelOption{WithGitProvider(&mockGitProvider{})},
			assert: func(t *testing.T, cmd tea.Cmd) {
				t.Helper()
				require.NotNil(t, cmd)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := NewConfigModel(tc.opts...)
			cmd := m.Init()
			tc.assert(t, cmd)
		})
	}
}

func TestConfigModel_GitBranchesMsg(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, m *ConfigModel)
		name   string
		msg    gitBranchesMsg
	}{
		{
			name: "populates cache on success",
			msg:  gitBranchesMsg{branches: []string{"main", "develop", "feature/x"}},
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.Equal(t, []string{"main", "develop", "feature/x"}, m.gitBranches)
			},
		},
		{
			name: "error leaves cache nil",
			msg:  gitBranchesMsg{err: errors.New("git error")},
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.Nil(t, m.gitBranches)
			},
		},
		{
			name: "empty slice leaves cache nil",
			msg:  gitBranchesMsg{branches: []string{}},
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.Nil(t, m.gitBranches)
			},
		},
		{
			name: "nil branches with no error leaves cache nil",
			msg:  gitBranchesMsg{},
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.Nil(t, m.gitBranches)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := NewConfigModel()
			result, cmd := m.Update(tc.msg)
			updated := result.(*ConfigModel)
			assert.Nil(t, cmd)
			tc.assert(t, updated)
		})
	}
}

func TestConfigModel_GitWorktreesMsg(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, m *ConfigModel)
		name   string
		msg    gitWorktreesMsg
	}{
		{
			name: "populates cache on success",
			msg:  gitWorktreesMsg{names: []string{"feat-a", "feat-b"}},
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.Equal(t, []string{"feat-a", "feat-b"}, m.gitWorktreeNames)
			},
		},
		{
			name: "error leaves cache nil",
			msg:  gitWorktreesMsg{err: errors.New("git error")},
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.Nil(t, m.gitWorktreeNames)
			},
		},
		{
			name: "empty slice leaves cache nil",
			msg:  gitWorktreesMsg{names: []string{}},
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.Nil(t, m.gitWorktreeNames)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := NewConfigModel()
			result, cmd := m.Update(tc.msg)
			updated := result.(*ConfigModel)
			assert.Nil(t, cmd)
			tc.assert(t, updated)
		})
	}
}

func TestConfigModel_FetchGitBranches(t *testing.T) {
	provider := &mockGitProvider{
		branches: []string{"main", "develop"},
	}
	m := NewConfigModel(WithGitProvider(provider))

	cmd := m.fetchGitBranches()
	require.NotNil(t, cmd)

	msg := cmd()
	bMsg, ok := msg.(gitBranchesMsg)
	require.True(t, ok)
	assert.Equal(t, []string{"main", "develop"}, bMsg.branches)
	assert.NoError(t, bMsg.err)
}

func TestConfigModel_FetchGitBranchesError(t *testing.T) {
	provider := &mockGitProvider{
		branchErr: errors.New("not a git repo"),
	}
	m := NewConfigModel(WithGitProvider(provider))

	cmd := m.fetchGitBranches()
	msg := cmd()
	bMsg := msg.(gitBranchesMsg)
	require.Error(t, bMsg.err)
	assert.Nil(t, bMsg.branches)
}

func TestConfigModel_FetchGitWorktrees(t *testing.T) {
	provider := &mockGitProvider{
		worktreeNames: []string{"feat-a", "feat-b"},
	}
	m := NewConfigModel(WithGitProvider(provider))

	cmd := m.fetchGitWorktrees()
	require.NotNil(t, cmd)

	msg := cmd()
	wMsg, ok := msg.(gitWorktreesMsg)
	require.True(t, ok)
	assert.Equal(t, []string{"feat-a", "feat-b"}, wMsg.names)
	assert.NoError(t, wMsg.err)
}

func TestConfigModel_DefaultBranchSuggestionsFallback(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, suggestions []string)
		setup  func(m *ConfigModel)
		name   string
	}{
		{
			name:  "static fallback when cache empty",
			setup: func(_ *ConfigModel) {},
			assert: func(t *testing.T, suggestions []string) {
				t.Helper()
				assert.Equal(t, []string{"main", "master", "develop", "development"}, suggestions)
			},
		},
		{
			name: "uses cached branches when available",
			setup: func(m *ConfigModel) {
				m.gitBranches = []string{"main", "release/v1", "feature/x"}
			},
			assert: func(t *testing.T, suggestions []string) {
				t.Helper()
				assert.Equal(t, []string{"main", "release/v1", "feature/x"}, suggestions)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := newEditTestModel(map[string]any{
				"default_branch": "main",
				"worktrees_dir":  "worktrees",
			})
			tc.setup(m)

			// Find the default_branch field row.
			var fr *FieldRow
			for _, row := range m.fieldRows[TabGeneral] {
				if row.Meta().Key == "default_branch" {
					fr = row
					break
				}
			}
			require.NotNil(t, fr, "default_branch field row not found")
			require.NotNil(t, fr.meta.Suggestions, "suggestions function should be wired")

			suggestions := fr.meta.Suggestions()
			tc.assert(t, suggestions)
		})
	}
}

func TestConfigModel_SourceWorktreeSuggestions(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, suggestions []string)
		setup  func(m *ConfigModel)
		name   string
	}{
		{
			name:  "prepends {default} with no git cache",
			setup: func(_ *ConfigModel) {},
			assert: func(t *testing.T, suggestions []string) {
				t.Helper()
				require.NotEmpty(t, suggestions)
				assert.Equal(t, "{default}", suggestions[0])
			},
		},
		{
			name: "uses git worktree names when cached",
			setup: func(m *ConfigModel) {
				m.gitWorktreeNames = []string{"wt-a", "wt-b"}
			},
			assert: func(t *testing.T, suggestions []string) {
				t.Helper()
				assert.Equal(t, []string{"{default}", "wt-a", "wt-b"}, suggestions)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := newEditTestModel(map[string]any{
				"default_branch":                 "main",
				"worktrees_dir":                  "worktrees",
				"file_copy.auto.enabled":         true,
				"file_copy.auto.source_worktree": "{default}",
				"file_copy.auto.copy_ignored":    false,
				"file_copy.auto.copy_untracked":  false,
				"file_copy.auto.exclude":         []string{},
			})
			tc.setup(m)

			// Find the source_worktree field row.
			var fr *FieldRow
			for _, row := range m.fieldRows[TabFileCopy] {
				if row.Meta().Key == "file_copy.auto.source_worktree" {
					fr = row
					break
				}
			}
			require.NotNil(t, fr, "source_worktree field row not found")
			require.NotNil(t, fr.meta.Suggestions, "suggestions function should be wired")

			suggestions := fr.meta.Suggestions()
			tc.assert(t, suggestions)
		})
	}
}

func TestConfigModel_WorktreeNamesForSuggestions(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, names []string)
		setup  func(m *ConfigModel)
		name   string
	}{
		{
			name:  "falls back to config worktree names when git cache empty",
			setup: func(_ *ConfigModel) {},
			assert: func(t *testing.T, names []string) {
				t.Helper()
				// No accessor means no config worktree names either.
				assert.Nil(t, names)
			},
		},
		{
			name: "prefers git cache when populated",
			setup: func(m *ConfigModel) {
				m.gitWorktreeNames = []string{"from-git-a", "from-git-b"}
			},
			assert: func(t *testing.T, names []string) {
				t.Helper()
				assert.Equal(t, []string{"from-git-a", "from-git-b"}, names)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := NewConfigModel()
			tc.setup(m)
			names := m.worktreeNamesForSuggestions()
			tc.assert(t, names)
		})
	}
}

func TestWithGitProvider(t *testing.T) {
	provider := &mockGitProvider{}
	m := NewConfigModel(WithGitProvider(provider))
	assert.Equal(t, provider, m.gitProvider)
}

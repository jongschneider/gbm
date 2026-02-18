package git

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseWorktreesPorcelain(t *testing.T) {
	testCases := []struct {
		name   string
		input  string
		assert func(t *testing.T, got []Worktree)
	}{
		{
			name:  "empty input",
			input: "",
			assert: func(t *testing.T, got []Worktree) {
				assert.Empty(t, got)
			},
		},
		{
			name: "bare repo with two worktrees",
			input: `worktree /path/to/repo
bare

worktree /path/to/repo/worktrees/main
HEAD 96a59d5208c63b492becd200405b27f1682d0ec1
branch refs/heads/main

worktree /path/to/repo/worktrees/feature
HEAD abc1234def5678901234567890abcdef12345678
branch refs/heads/feature/foo
`,
			assert: func(t *testing.T, got []Worktree) {
				assert.Len(t, got, 3)

				assert.Equal(t, "repo", got[0].Name)
				assert.Equal(t, "/path/to/repo", got[0].Path)
				assert.True(t, got[0].IsBare)
				assert.Empty(t, got[0].Branch)
				assert.Empty(t, got[0].Commit)

				assert.Equal(t, "main", got[1].Name)
				assert.Equal(t, "/path/to/repo/worktrees/main", got[1].Path)
				assert.Equal(t, "main", got[1].Branch)
				assert.Equal(t, "96a59d5", got[1].Commit)
				assert.False(t, got[1].IsBare)

				assert.Equal(t, "feature", got[2].Name)
				assert.Equal(t, "/path/to/repo/worktrees/feature", got[2].Path)
				assert.Equal(t, "feature/foo", got[2].Branch)
				assert.Equal(t, "abc1234", got[2].Commit)
				assert.False(t, got[2].IsBare)
			},
		},
		{
			name: "path with spaces",
			input: `worktree /Volumes/OWC Express 1M2/Dev/gbm
bare

worktree /Volumes/OWC Express 1M2/Dev/gbm/worktrees/main
HEAD 96a59d5208c63b492becd200405b27f1682d0ec1
branch refs/heads/main
`,
			assert: func(t *testing.T, got []Worktree) {
				assert.Len(t, got, 2)

				assert.Equal(t, "gbm", got[0].Name)
				assert.Equal(t, "/Volumes/OWC Express 1M2/Dev/gbm", got[0].Path)
				assert.True(t, got[0].IsBare)

				assert.Equal(t, "main", got[1].Name)
				assert.Equal(t, "/Volumes/OWC Express 1M2/Dev/gbm/worktrees/main", got[1].Path)
				assert.Equal(t, "main", got[1].Branch)
				assert.Equal(t, "96a59d5", got[1].Commit)
			},
		},
		{
			name: "detached HEAD",
			input: `worktree /path/to/repo/worktrees/detached
HEAD abc1234def5678901234567890abcdef12345678
detached
`,
			assert: func(t *testing.T, got []Worktree) {
				assert.Len(t, got, 1)
				assert.Equal(t, "detached", got[0].Name)
				assert.Equal(t, "abc1234", got[0].Commit)
				assert.Empty(t, got[0].Branch)
				assert.False(t, got[0].IsBare)
			},
		},
		{
			name: "single worktree no trailing newline",
			input: `worktree /path/to/repo/worktrees/main
HEAD 1234567890abcdef1234567890abcdef12345678
branch refs/heads/main`,
			assert: func(t *testing.T, got []Worktree) {
				assert.Len(t, got, 1)
				assert.Equal(t, "main", got[0].Name)
				assert.Equal(t, "main", got[0].Branch)
				assert.Equal(t, "1234567", got[0].Commit)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseWorktreesPorcelain(tc.input)
			tc.assert(t, got)
		})
	}
}

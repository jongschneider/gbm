package git

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitError_Error(t *testing.T) {
	err := &GitError{
		Op:       "branch delete",
		ExitCode: 1,
		Stderr:   "error: branch 'nonexistent' not found",
		Err:      ErrBranchNotFound,
	}

	expected := "git branch delete: branch not found (error: branch 'nonexistent' not found)"
	assert.Equal(t, expected, err.Error())
}

func TestGitError_Error_NoStderr(t *testing.T) {
	err := &GitError{
		Op:       "branch delete",
		ExitCode: 1,
		Stderr:   "",
		Err:      ErrBranchNotFound,
	}

	expected := "git branch delete: branch not found"
	assert.Equal(t, expected, err.Error())
}

func TestGitError_Unwrap(t *testing.T) {
	innerErr := ErrBranchNotFound
	err := &GitError{
		Op:  "branch delete",
		Err: innerErr,
	}

	assert.Equal(t, innerErr, err.Unwrap())
	assert.ErrorIs(t, err, ErrBranchNotFound)
}

func TestGitError_IsExitCode(t *testing.T) {
	err := &GitError{
		ExitCode: 1,
		Err:      ErrBranchNotFound,
	}

	assert.True(t, err.IsExitCode(1))
	assert.False(t, err.IsExitCode(0))
	assert.False(t, err.IsExitCode(128))
}

func TestNewGitError(t *testing.T) {
	err := NewGitError("branch delete", ErrBranchNotFound, 1, "  error: not found  \n")

	// Verify it's wrapped properly
	var gitErr *GitError
	require.ErrorAs(t, err, &gitErr)
	assert.Equal(t, "branch delete", gitErr.Op)
	assert.Equal(t, 1, gitErr.ExitCode)
	assert.Equal(t, "error: not found", gitErr.Stderr) // Trimmed
	assert.ErrorIs(t, err, ErrBranchNotFound)
}

func TestNewGitError_NilError(t *testing.T) {
	err := NewGitError("branch delete", nil, 1, "")
	assert.NoError(t, err)
}

func TestClassifyError_BranchNotFound(t *testing.T) {
	tests := []struct {
		name   string
		op     string
		output string
	}{
		{
			name:   "branch not found",
			op:     "branch delete",
			output: "error: branch 'test' not found",
		},
		{
			name:   "no such branch",
			op:     "branch delete",
			output: "fatal: no such branch 'test'",
		},
		{
			name:   "case insensitive",
			op:     "branch delete",
			output: "ERROR: BRANCH NOT FOUND",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ClassifyError(tt.op, errors.New("exit 1"), tt.output)

			assert.ErrorIs(t, err, ErrBranchNotFound)
		})
	}
}

func TestClassifyError_WorktreeNotFound(t *testing.T) {
	tests := []struct {
		name   string
		op     string
		output string
	}{
		{
			name:   "worktree not found",
			op:     "worktree remove",
			output: "fatal: 'test' is not a working tree",
		},
		{
			name:   "no such worktree",
			op:     "worktree remove",
			output: "fatal: no such worktree 'test'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ClassifyError(tt.op, errors.New("exit 1"), tt.output)

			assert.ErrorIs(t, err, ErrWorktreeNotFound)
		})
	}
}

func TestClassifyError_BranchExists(t *testing.T) {
	output := "fatal: A branch named 'test' already exists"
	err := ClassifyError("branch create", errors.New("exit 1"), output)

	assert.ErrorIs(t, err, ErrBranchExists)
}

func TestClassifyError_WorktreeExists(t *testing.T) {
	output := "fatal: 'test' already exists"
	err := ClassifyError("worktree add", errors.New("exit 1"), output)

	assert.ErrorIs(t, err, ErrWorktreeExists)
}

func TestClassifyError_NotMerged(t *testing.T) {
	tests := []struct {
		name   string
		output string
	}{
		{
			name:   "not fully merged",
			output: "error: The branch 'test' is not fully merged",
		},
		{
			name:   "not merged",
			output: "fatal: The branch is not merged",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ClassifyError("branch delete", errors.New("exit 1"), tt.output)

			assert.ErrorIs(t, err, ErrNotMerged)
		})
	}
}

func TestClassifyError_DirtyWorktree(t *testing.T) {
	tests := []struct {
		name   string
		output string
	}{
		{
			name:   "dirty worktree",
			output: "error: working tree is dirty",
		},
		{
			name:   "uncommitted changes",
			output: "error: uncommitted changes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ClassifyError("worktree remove", errors.New("exit 1"), tt.output)

			assert.ErrorIs(t, err, ErrDirtyWorktree)
		})
	}
}

func TestClassifyError_NoUpstream(t *testing.T) {
	tests := []struct {
		name   string
		output string
	}{
		{
			name:   "no upstream",
			output: "fatal: no upstream",
		},
		{
			name:   "no tracking",
			output: "error: no tracking information for the current branch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ClassifyError("push", errors.New("exit 1"), tt.output)

			assert.ErrorIs(t, err, ErrNoUpstream)
		})
	}
}

func TestClassifyError_NoRemoteTrackingBranch(t *testing.T) {
	output := "fatal: no remote tracking branch"
	err := ClassifyError("pull", errors.New("exit 1"), output)

	assert.ErrorIs(t, err, ErrNoRemoteTrackingBranch)
}

func TestClassifyError_RemoteNotFound(t *testing.T) {
	tests := []struct {
		name   string
		output string
	}{
		{
			name:   "no such remote",
			output: "fatal: no such remote 'origin'",
		},
		{
			name:   "unknown remote",
			output: "fatal: unknown remote",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ClassifyError("fetch", errors.New("exit 1"), tt.output)

			assert.ErrorIs(t, err, ErrRemoteNotFound)
		})
	}
}

func TestClassifyError_NotAGitRepository(t *testing.T) {
	output := "fatal: not a git repository"
	err := ClassifyError("status", errors.New("exit 128"), output)

	assert.ErrorIs(t, err, ErrNotAGitRepository)
}

func TestClassifyError_FallbackToWrap(t *testing.T) {
	// Unknown error - should fallback to wrapping
	output := "some unknown error"
	err := ClassifyError("custom operation", errors.New("exit 1"), output)

	// Should still be wrapped with context
	var gitErr *GitError
	require.ErrorAs(t, err, &gitErr)
	assert.Equal(t, "custom operation", gitErr.Op)
}

func TestClassifyError_NilError(t *testing.T) {
	err := ClassifyError("branch delete", nil, "")
	assert.NoError(t, err)
}

func TestWrap(t *testing.T) {
	innerErr := ErrBranchNotFound
	err := Wrap("branch operations", innerErr)

	var gitErr *GitError
	require.ErrorAs(t, err, &gitErr)
	assert.Equal(t, "branch operations", gitErr.Op)
	assert.ErrorIs(t, err, ErrBranchNotFound)
}

func TestWrap_NilError(t *testing.T) {
	err := Wrap("branch operations", nil)
	assert.NoError(t, err)
}

// TestErrorChaining demonstrates the error wrapping chain.
func TestErrorChaining(t *testing.T) {
	// Simulate a git error with multiple layers
	classifiedErr := ClassifyError("branch delete", errors.New("exit 1"), "fatal: branch 'test' not found")

	// Should be able to use errors.Is to check the underlying type
	require.ErrorIs(t, classifiedErr, ErrBranchNotFound)

	// Should be able to unwrap and inspect
	var gErr *GitError
	require.ErrorAs(t, classifiedErr, &gErr)
	assert.Equal(t, "branch delete", gErr.Op)
}

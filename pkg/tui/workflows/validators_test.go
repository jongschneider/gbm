package workflows

import (
	"gbm/pkg/tui"
	"testing"

	"github.com/stretchr/testify/assert"
)

// mockValidatorGitService implements tui.GitService for validator tests.
type mockValidatorGitService struct {
	branches  map[string]bool
	worktrees []tui.WorktreeInfo
}

func (m *mockValidatorGitService) BranchExists(branch string) (bool, error) {
	return m.branches[branch], nil
}

func (m *mockValidatorGitService) ListBranches(_ bool) ([]string, error) {
	return nil, nil
}

func (m *mockValidatorGitService) ListWorktrees(_ bool) ([]tui.WorktreeInfo, error) {
	return m.worktrees, nil
}

func TestValidateWorktreeName(t *testing.T) {
	t.Parallel()

	svc := &mockValidatorGitService{
		worktrees: []tui.WorktreeInfo{
			{Name: "existing-wt", Branch: "feature/existing"},
			{Name: "another-wt", Branch: "feature/another"},
		},
	}
	validate := validateWorktreeName(svc)

	testCases := []struct {
		assertError func(t *testing.T, err error)
		name        string
		input       string
	}{
		{
			name:  "rejects existing worktree name",
			input: "existing-wt",
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.ErrorContains(t, err, "already exists")
			},
		},
		{
			name:  "accepts new worktree name",
			input: "brand-new-wt",
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name:  "rejects second existing name",
			input: "another-wt",
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.ErrorContains(t, err, "already exists")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validate(tc.input)
			tc.assertError(t, err)
		})
	}
}

func TestValidateWorktreeName_NilService(t *testing.T) {
	t.Parallel()

	validate := validateWorktreeName(nil)
	err := validate("anything")
	assert.NoError(t, err)
}

func TestValidateBranchNameOnReview(t *testing.T) {
	t.Parallel()

	svc := &mockValidatorGitService{
		branches: map[string]bool{
			"feature/existing":    true,
			"feature/no-worktree": true,
		},
		worktrees: []tui.WorktreeInfo{
			{Name: "existing-wt", Branch: "feature/existing"},
		},
	}
	validate := validateBranchNameOnReview(svc)

	testCases := []struct {
		assertError func(t *testing.T, err error)
		name        string
		input       string
	}{
		{
			name:  "rejects branch checked out in worktree",
			input: "feature/existing",
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.ErrorContains(t, err, "already checked out")
				assert.ErrorContains(t, err, "existing-wt")
			},
		},
		{
			name:  "accepts existing branch not in any worktree",
			input: "feature/no-worktree",
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name:  "accepts new branch that does not exist",
			input: "feature/brand-new",
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validate(tc.input)
			tc.assertError(t, err)
		})
	}
}

func TestValidateBranchNameOnReview_NilService(t *testing.T) {
	t.Parallel()

	validate := validateBranchNameOnReview(nil)
	err := validate("anything")
	assert.NoError(t, err)
}

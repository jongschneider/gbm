package testing

import (
	"gbm/pkg/tui"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockGitService(t *testing.T) {
	testCases := []struct {
		setup     func(*MockGitService)
		name      string
		expectErr bool
	}{
		{
			name: "default branches",
			setup: func(m *MockGitService) {
				// No setup needed
			},
			expectErr: false,
		},
		{
			name: "custom branches",
			setup: func(m *MockGitService) {
				m.WithBranches([]string{"main", "develop", "feature/test"})
			},
			expectErr: false,
		},
		{
			name: "with delay",
			setup: func(m *MockGitService) {
				m.WithDelay(10 * time.Millisecond)
			},
			expectErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			svc := NewMockGitService()
			tc.setup(svc)

			branches, err := svc.ListBranches(false)

			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, branches)
			}
		})
	}
}

func TestMockGitServiceBranchExists(t *testing.T) {
	testCases := []struct {
		name      string
		setup     func(*MockGitService)
		branch    string
		expectErr bool
		expect    bool
	}{
		{
			name: "branch does not exist by default",
			setup: func(m *MockGitService) {
				// No setup
			},
			branch:    "feature/new",
			expectErr: false,
			expect:    false,
		},
		{
			name: "branch exists with custom function",
			setup: func(m *MockGitService) {
				m.WithBranchExists(func(name string) bool {
					return name == "main"
				})
			},
			branch:    "main",
			expectErr: false,
			expect:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			svc := NewMockGitService()
			tc.setup(svc)

			exists, err := svc.BranchExists(tc.branch)

			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expect, exists)
			}
		})
	}
}

func TestMockJiraService(t *testing.T) {
	testCases := []struct {
		setup     func(*MockJiraService)
		name      string
		expectErr bool
	}{
		{
			name: "default issues",
			setup: func(m *MockJiraService) {
				// No setup needed
			},
			expectErr: false,
		},
		{
			name: "custom issues",
			setup: func(m *MockJiraService) {
				m.WithIssues([]tui.JiraIssue{
					{Key: "PROJ-1", Summary: "Test issue"},
				})
			},
			expectErr: false,
		},
		{
			name: "with delay",
			setup: func(m *MockJiraService) {
				m.WithDelay(10 * time.Millisecond)
			},
			expectErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			svc := NewMockJiraService()
			tc.setup(svc)

			issues, err := svc.FetchIssues()

			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, issues)
			}
		})
	}
}

func TestMockJiraServiceCopyProtection(t *testing.T) {
	svc := NewMockJiraService()

	issues1, _ := svc.FetchIssues() //nolint:errcheck // Test intentionally ignores error
	issues2, _ := svc.FetchIssues() //nolint:errcheck // Test intentionally ignores error

	// Modify first result
	if len(issues1) > 0 {
		issues1[0].Summary = "Modified"
	}

	// Verify second result is not modified (proof of copy)
	if len(issues2) > 0 {
		assert.NotEqual(t, "Modified", issues2[0].Summary)
	}
}

func TestErrorMockGitService(t *testing.T) {
	svc := NewErrorMockGitService(nil)

	branches, err := svc.ListBranches(false)

	require.Error(t, err)
	assert.Nil(t, branches)
}

func TestErrorMockJiraService(t *testing.T) {
	svc := NewErrorMockJiraService(nil)

	issues, err := svc.FetchIssues()

	require.Error(t, err)
	assert.Nil(t, issues)
}

// Verify interface implementations.
func TestMockImplementsInterfaces(t *testing.T) {
	var _ tui.GitService = (*MockGitService)(nil)
	var _ tui.JiraService = (*MockJiraService)(nil)
	var _ tui.GitService = (*ErrorMockGitService)(nil)
	var _ tui.JiraService = (*ErrorMockJiraService)(nil)
}

// Package testing provides mock implementations of services for testing and development.
package testing

import (
	"fmt"
	"time"

	"gbm/pkg/tui"
)

// MockGitService implements tui.GitService for testing.
type MockGitService struct {
	branches []string
	delay    time.Duration

	// configurable result for BranchExists (default false)
	existsFunc func(string) bool
}

// NewMockGitService creates a new MockGitService with default branches.
func NewMockGitService() *MockGitService {
	return &MockGitService{
		branches: []string{
			"main",
			"master",
			"develop",
			"staging",
			"production-v1",
			"release/v1.0",
			"release/v1.1",
		},
		delay: 0,
		existsFunc: func(string) bool {
			return false
		},
	}
}

// WithDelay adds a simulated network delay.
func (m *MockGitService) WithDelay(d time.Duration) *MockGitService {
	m.delay = d
	return m
}

// WithBranches sets the list of branches.
func (m *MockGitService) WithBranches(branches []string) *MockGitService {
	m.branches = branches
	return m
}

// WithBranchExists sets a custom function for BranchExists.
func (m *MockGitService) WithBranchExists(fn func(string) bool) *MockGitService {
	m.existsFunc = fn
	return m
}

// ListBranches returns the list of branches with optional delay.
func (m *MockGitService) ListBranches() ([]string, error) {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	return m.branches, nil
}

// BranchExists checks if a branch exists.
func (m *MockGitService) BranchExists(name string) (bool, error) {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	return m.existsFunc(name), nil
}

// MockJiraService implements tui.JiraService for testing.
type MockJiraService struct {
	issues []tui.JiraIssue
	delay  time.Duration
}

// NewMockJiraService creates a new MockJiraService with default issues.
func NewMockJiraService() *MockJiraService {
	return &MockJiraService{
		issues: []tui.JiraIssue{
			{Key: "INGSVC-6468", Summary: "Add authentication middleware"},
			{Key: "INGSVC-6469", Summary: "Fix database connection pooling"},
			{Key: "INGSVC-6470", Summary: "Implement rate limiting"},
			{Key: "INGSVC-6471", Summary: "Add comprehensive logging"},
			{Key: "INGSVC-6472", Summary: "Optimize query performance"},
			{Key: "INGSVC-6473", Summary: "Security audit and patching"},
			{Key: "INGSVC-6474", Summary: "API documentation improvements"},
			{Key: "INGSVC-6475", Summary: "Add integration tests"},
		},
		delay: 0,
	}
}

// WithDelay adds a simulated network delay.
func (m *MockJiraService) WithDelay(d time.Duration) *MockJiraService {
	m.delay = d
	return m
}

// WithIssues sets the list of issues.
func (m *MockJiraService) WithIssues(issues []tui.JiraIssue) *MockJiraService {
	m.issues = issues
	return m
}

// FetchIssues returns the list of issues with optional delay.
func (m *MockJiraService) FetchIssues() ([]tui.JiraIssue, error) {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	// Return a copy to prevent external modifications
	result := make([]tui.JiraIssue, len(m.issues))
	copy(result, m.issues)
	return result, nil
}

// ErrorMockGitService always returns an error.
type ErrorMockGitService struct {
	err error
}

// NewErrorMockGitService creates a git service that always errors.
func NewErrorMockGitService(err error) *ErrorMockGitService {
	if err == nil {
		err = fmt.Errorf("git service error")
	}
	return &ErrorMockGitService{err: err}
}

// ListBranches returns an error.
func (m *ErrorMockGitService) ListBranches() ([]string, error) {
	return nil, m.err
}

// BranchExists returns an error.
func (m *ErrorMockGitService) BranchExists(name string) (bool, error) {
	return false, m.err
}

// ErrorMockJiraService always returns an error.
type ErrorMockJiraService struct {
	err error
}

// NewErrorMockJiraService creates a JIRA service that always errors.
func NewErrorMockJiraService(err error) *ErrorMockJiraService {
	if err == nil {
		err = fmt.Errorf("jira service error")
	}
	return &ErrorMockJiraService{err: err}
}

// FetchIssues returns an error.
func (m *ErrorMockJiraService) FetchIssues() ([]tui.JiraIssue, error) {
	return nil, m.err
}

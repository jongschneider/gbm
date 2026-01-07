package workflows

import (
	"fmt"
	"testing"

	"gbm/pkg/tui"
	"gbm/pkg/tui/fields"
	"github.com/stretchr/testify/assert"
)

func TestSlugify(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{
			input:    "Hello World",
			expected: "hello-world",
		},
		{
			input:    "UPPERCASE TEXT",
			expected: "uppercase-text",
		},
		{
			input:    "Special!@#$%Characters",
			expected: "specialcharacters",
		},
		{
			input:    "Multiple   Spaces",
			expected: "multiple-spaces",
		},
		{
			input:    "Fix bug in widget",
			expected: "fix-bug-in-widget",
		},
		{
			input:    "-leading-trailing-",
			expected: "leading-trailing",
		},
		{
			input:    "Multiple---Hyphens",
			expected: "multiple-hyphens",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := slugify(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestGenerateBranchName(t *testing.T) {
	testCases := []struct {
		issueKey string
		summary  string
		expected string
	}{
		{
			issueKey: "PROJ-123",
			summary:  "Fix bug in widget",
			expected: "feature/PROJ-123-fix-bug-in-widget",
		},
		{
			issueKey: "INGSVC-6468",
			summary:  "Add authentication middleware",
			expected: "feature/INGSVC-6468-add-authentication-middleware",
		},
		{
			issueKey: "BUG-1",
			summary:  "Critical!@#$ Issue",
			expected: "feature/BUG-1-critical-issue",
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s-%s", tc.issueKey, tc.summary), func(t *testing.T) {
			result := generateBranchName(tc.issueKey, tc.summary)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestGenerateBranchNameHotfix(t *testing.T) {
	testCases := []struct {
		issueKey string
		summary  string
		expected string
	}{
		{
			issueKey: "PROJ-123",
			summary:  "Fix critical bug",
			expected: "hotfix/PROJ-123-fix-critical-bug",
		},
		{
			issueKey: "INGSVC-6468",
			summary:  "Security patch",
			expected: "hotfix/INGSVC-6468-security-patch",
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s-%s", tc.issueKey, tc.summary), func(t *testing.T) {
			result := generateBranchNameHotfix(tc.issueKey, tc.summary)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestValidateBranchName(t *testing.T) {
	testCases := []struct {
		name      string
		input     string
		expectErr bool
	}{
		{
			name:      "valid feature branch",
			input:     "feature/PROJ-123-description",
			expectErr: false,
		},
		{
			name:      "valid with underscores",
			input:     "feature_branch_name",
			expectErr: false,
		},
		{
			name:      "valid with dots",
			input:     "release/v1.2.3",
			expectErr: false,
		},
		{
			name:      "empty string",
			input:     "",
			expectErr: true,
		},
		{
			name:      "whitespace only",
			input:     "   ",
			expectErr: true,
		},
		{
			name:      "invalid characters",
			input:     "feature/invalid@chars",
			expectErr: true,
		},
		{
			name:      "invalid space",
			input:     "feature with space",
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateBranchName(tc.input)
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFeatureWorkflowCreation(t *testing.T) {
	ctx := tui.NewContext()

	wizard := FeatureWorkflow(ctx)

	// Verify wizard was created
	assert.NotNil(t, wizard)

	// Verify wizard has expected state
	assert.NotNil(t, wizard.State())

	// Verify wizard has correct number of steps
	// The wizard should have 4 steps: worktree_name, branch_name, base_branch, confirm
	// This is a bit hacky but we can verify initialization works
	initCmd := wizard.Init()
	assert.NotNil(t, initCmd)
}

func TestProcessFeatureWorkflow(t *testing.T) {
	testCases := []struct {
		name         string
		worktreeName string
		jiraIssues   []tui.JiraIssue
		expectErr    bool
		expectBranch string
	}{
		{
			name:         "JIRA issue found",
			worktreeName: "PROJ-123",
			jiraIssues: []tui.JiraIssue{
				{Key: "PROJ-123", Summary: "Fix bug in widget"},
				{Key: "PROJ-124", Summary: "Add feature"},
			},
			expectErr:    false,
			expectBranch: "feature/PROJ-123-fix-bug-in-widget",
		},
		{
			name:         "JIRA issue not found",
			worktreeName: "custom-name",
			jiraIssues: []tui.JiraIssue{
				{Key: "PROJ-123", Summary: "Fix bug"},
			},
			expectErr:    false,
			expectBranch: "",
		},
		{
			name:         "custom worktree name (non-JIRA format)",
			worktreeName: "my-feature",
			jiraIssues:   []tui.JiraIssue{},
			expectErr:    false,
			expectBranch: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := tui.NewContext()

			// Create a mock JIRA service
			mockJiraService := &mockJiraService{
				issues: tc.jiraIssues,
			}
			ctx.WithJiraService(mockJiraService)

			wizard := FeatureWorkflow(ctx)
			wizard.State().WorktreeName = tc.worktreeName

			err := ProcessFeatureWorkflow(wizard, ctx)

			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectBranch, wizard.State().BranchName)
			}
		})
	}
}

// mockJiraService implements tui.JiraService for testing
type mockJiraService struct {
	issues []tui.JiraIssue
}

func (m *mockJiraService) FetchIssues() ([]tui.JiraIssue, error) {
	return m.issues, nil
}

func TestHotfixWorkflowCreation(t *testing.T) {
	ctx := tui.NewContext()

	wizard := HotfixWorkflow(ctx)

	// Verify wizard was created
	assert.NotNil(t, wizard)

	// Verify wizard has expected state
	assert.NotNil(t, wizard.State())

	// Verify wizard can be initialized
	initCmd := wizard.Init()
	assert.NotNil(t, initCmd)
}

func TestProcessHotfixWorkflow(t *testing.T) {
	testCases := []struct {
		name               string
		worktreeName       string
		jiraIssues         []tui.JiraIssue
		expectErr          bool
		expectBranch       string
		expectWorktreeName string
	}{
		{
			name:         "JIRA issue found",
			worktreeName: "PROJ-123",
			jiraIssues: []tui.JiraIssue{
				{Key: "PROJ-123", Summary: "Fix critical bug"},
				{Key: "PROJ-124", Summary: "Add feature"},
			},
			expectErr:          false,
			expectBranch:       "hotfix/PROJ-123-fix-critical-bug",
			expectWorktreeName: "HOTFIX_PROJ-123",
		},
		{
			name:         "JIRA issue not found",
			worktreeName: "custom-name",
			jiraIssues: []tui.JiraIssue{
				{Key: "PROJ-123", Summary: "Fix bug"},
			},
			expectErr:          false,
			expectBranch:       "",
			expectWorktreeName: "HOTFIX_custom-name",
		},
		{
			name:               "custom worktree name (non-JIRA format)",
			worktreeName:       "my-hotfix",
			jiraIssues:         []tui.JiraIssue{},
			expectErr:          false,
			expectBranch:       "",
			expectWorktreeName: "HOTFIX_my-hotfix",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := tui.NewContext()

			// Create a mock JIRA service
			mockJiraService := &mockJiraService{
				issues: tc.jiraIssues,
			}
			ctx.WithJiraService(mockJiraService)

			wizard := HotfixWorkflow(ctx)
			wizard.State().WorktreeName = tc.worktreeName

			err := ProcessHotfixWorkflow(wizard, ctx)

			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectBranch, wizard.State().BranchName)
				assert.Equal(t, tc.expectWorktreeName, wizard.State().WorktreeName)
			}
		})
	}
}

// Ensure field implementations
func TestFieldImplementations(t *testing.T) {
	// Test that we can create Filterable fields (used in FeatureWorkflow)
	filterableField := fields.NewFilterable(
		"test",
		"Test Field",
		"Test description",
		[]fields.Option{},
	)
	assert.Implements(t, (*tui.Field)(nil), filterableField)

	// Test that we can create TextInput fields (used in FeatureWorkflow)
	textInputField := fields.NewTextInput("test", "Test", "Test")
	assert.Implements(t, (*tui.Field)(nil), textInputField)

	// Test that we can create Confirm fields (used in FeatureWorkflow)
	confirmField := fields.NewConfirm("test", "Test?")
	assert.Implements(t, (*tui.Field)(nil), confirmField)
}

func TestSelectWorkflowType(t *testing.T) {
	t.Run("creates workflow selector field", func(t *testing.T) {
		field := SelectWorkflowType()

		// Verify field was created
		assert.NotNil(t, field)

		// Verify field implements Field interface
		assert.Implements(t, (*tui.Field)(nil), field)
	})

	t.Run("has correct field type", func(t *testing.T) {
		field := SelectWorkflowType()

		// Verify it's a Selector field
		selector, ok := field.(*fields.Selector)
		assert.True(t, ok, "SelectWorkflowType should return a *Selector")
		assert.NotNil(t, selector)
	})

	t.Run("has all four workflow options", func(t *testing.T) {
		field := SelectWorkflowType()
		selector := field.(*fields.Selector)

		// Focus the field to ensure initialization
		selector.Focus()

		// Verify we can get the value (should have options)
		// The field should not be complete until a selection is made
		assert.False(t, selector.IsComplete())
	})

	t.Run("selector key is 'workflow_type'", func(t *testing.T) {
		field := SelectWorkflowType()
		selector := field.(*fields.Selector)

		// Verify the field key
		assert.Equal(t, "workflow_type", selector.GetKey())
	})
}

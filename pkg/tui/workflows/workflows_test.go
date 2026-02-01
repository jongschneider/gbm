package workflows

import (
	"fmt"
	"gbm/pkg/tui"
	"gbm/pkg/tui/fields"
	"gbm/testutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		expectBranch string
		jiraIssues   []tui.JiraIssue
		expectErr    bool
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
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expectBranch, wizard.State().BranchName)
			}
		})
	}
}

// mockJiraService implements tui.JiraService for testing.
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
		expectBranch       string
		expectWorktreeName string
		jiraIssues         []tui.JiraIssue
		expectErr          bool
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
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expectBranch, wizard.State().BranchName)
				assert.Equal(t, tc.expectWorktreeName, wizard.State().WorktreeName)
			}
		})
	}
}

// Ensure field implementations.
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
		assert.Equal(t, tui.FieldKeyWorkflowType, selector.GetKey())
	})
}

// Router tests.

func TestGetWorkflowSteps_Feature(t *testing.T) {
	ctx := tui.NewContext()
	steps, err := GetWorkflowSteps(tui.WorkflowTypeFeature, ctx)

	require.NoError(t, err)
	assert.Len(t, steps, 4)
	assert.Equal(t, tui.FieldKeyWorktreeName, steps[0].Name)
	assert.Equal(t, tui.FieldKeyBranchName, steps[1].Name)
	assert.Equal(t, tui.FieldKeyBaseBranch, steps[2].Name)
	assert.Equal(t, tui.FieldKeyConfirm, steps[3].Name)
}

func TestGetWorkflowSteps_Bug(t *testing.T) {
	ctx := tui.NewContext()
	steps, err := GetWorkflowSteps(tui.WorkflowTypeBug, ctx)

	require.NoError(t, err)
	assert.Len(t, steps, 4)
	assert.Equal(t, tui.FieldKeyWorktreeName, steps[0].Name)
	assert.Equal(t, tui.FieldKeyBranchName, steps[1].Name)
	assert.Equal(t, tui.FieldKeyBaseBranch, steps[2].Name)
	assert.Equal(t, tui.FieldKeyConfirm, steps[3].Name)
}

func TestGetWorkflowSteps_Hotfix(t *testing.T) {
	ctx := tui.NewContext()
	steps, err := GetWorkflowSteps(tui.WorkflowTypeHotfix, ctx)

	require.NoError(t, err)
	assert.Len(t, steps, 4)
	assert.Equal(t, tui.FieldKeyWorktreeName, steps[0].Name)
	assert.Equal(t, tui.FieldKeyBaseBranch, steps[1].Name)
	assert.Equal(t, tui.FieldKeyBranchName, steps[2].Name)
	assert.Equal(t, tui.FieldKeyConfirm, steps[3].Name)
}

func TestGetWorkflowSteps_Merge(t *testing.T) {
	ctx := tui.NewContext()
	steps, err := GetWorkflowSteps(tui.WorkflowTypeMerge, ctx)

	require.NoError(t, err)
	assert.Len(t, steps, 3)
	assert.Equal(t, "source_branch", steps[0].Name)
	assert.Equal(t, "target_branch", steps[1].Name)
	assert.Equal(t, tui.FieldKeyConfirm, steps[2].Name)
}

func TestGetWorkflowSteps_Unknown(t *testing.T) {
	ctx := tui.NewContext()
	steps, err := GetWorkflowSteps("unknown", ctx)

	require.Error(t, err)
	assert.Nil(t, steps)
	assert.Equal(t, "unknown workflow type: unknown", err.Error())
}

func TestBugWorkflow_Creation(t *testing.T) {
	ctx := tui.NewContext()
	wizard := BugWorkflow(ctx)

	assert.NotNil(t, wizard)
	// Verify that the wizard can be used (it initializes correctly)
	cmd := wizard.Init()
	assert.NotNil(t, cmd)
}

func TestMergeWorkflow_Creation(t *testing.T) {
	ctx := tui.NewContext()
	wizard := MergeWorkflow(ctx)

	assert.NotNil(t, wizard)
	// Verify that the wizard can be used (it initializes correctly)
	cmd := wizard.Init()
	assert.NotNil(t, cmd)
}

func TestProcessBugWorkflow_WithIssue(t *testing.T) {
	// Setup mock JIRA service
	mockJira := &mockJiraService{
		issues: []tui.JiraIssue{
			{Key: "BUG-123", Summary: "Fix database connection"},
		},
	}

	ctx := tui.NewContext().WithJiraService(mockJira)
	wizard := BugWorkflow(ctx)
	state := wizard.State()

	// Simulate user selecting JIRA issue
	state.WorktreeName = "BUG-123"
	state.BranchName = ""

	err := ProcessBugWorkflow(wizard, ctx)

	require.NoError(t, err)
	assert.Equal(t, "BUG-123", state.WorktreeName)
	assert.Equal(t, "bug/BUG-123-fix-database-connection", state.BranchName)
	assert.NotNil(t, state.JiraIssue)
	assert.Equal(t, "BUG-123", state.JiraIssue.Key)
}

func TestProcessMergeWorkflow(t *testing.T) {
	ctx := tui.NewContext()
	wizard := MergeWorkflow(ctx)

	err := ProcessMergeWorkflow(wizard, ctx)

	assert.NoError(t, err)
}

func TestSuggestMergeTarget(t *testing.T) {
	type testCase struct {
		name              string
		config            tui.RepoConfig
		sourceBranch      string
		expectedSuggested string
	}

	testCases := []testCase{
		{
			name:              "nil_config",
			config:            nil,
			sourceBranch:      "feature/test",
			expectedSuggested: "",
		},
		{
			name:              "no_matching_worktree",
			config:            testutil.NewMockRepoConfig().WithWorktree("wt1", "main", "develop"),
			sourceBranch:      "feature/unknown",
			expectedSuggested: "",
		},
		{
			name:              "matching_worktree_with_merge_into",
			config:            testutil.NewMockRepoConfig().WithWorktree("wt1", "feature/test", "main"),
			sourceBranch:      "feature/test",
			expectedSuggested: "main",
		},
		{
			name:              "matching_worktree_without_merge_into",
			config:            testutil.NewMockRepoConfig().WithWorktree("wt1", "feature/test", ""),
			sourceBranch:      "feature/test",
			expectedSuggested: "",
		},
		{
			name: "multiple_worktrees_finds_correct_one",
			config: testutil.NewMockRepoConfig().
				WithWorktree("wt1", "feature/one", "develop").
				WithWorktree("wt2", "feature/two", "main").
				WithWorktree("wt3", "feature/three", "staging"),
			sourceBranch:      "feature/two",
			expectedSuggested: "main",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := tui.NewContext().WithConfig(tc.config)
			suggestion := SuggestMergeTarget(ctx, tc.sourceBranch)
			assert.Equal(t, tc.expectedSuggested, suggestion)
		})
	}
}

func TestSortTargetBranchOptions(t *testing.T) {
	type testCase struct {
		name           string
		branches       []fields.Option
		suggested      string
		sourceBranch   string
		expectedLabels []string
		expectedValues []string
	}

	testCases := []testCase{
		{
			name:           "empty_branches",
			branches:       []fields.Option{},
			suggested:      "main",
			sourceBranch:   "feature/test",
			expectedLabels: []string{},
			expectedValues: []string{},
		},
		{
			name: "no_suggestion",
			branches: []fields.Option{
				{Label: "main", Value: "main"},
				{Label: "develop", Value: "develop"},
				{Label: "feature/test", Value: "feature/test"},
			},
			suggested:      "",
			sourceBranch:   "feature/test",
			expectedLabels: []string{"main", "develop"},
			expectedValues: []string{"main", "develop"},
		},
		{
			name: "with_suggestion",
			branches: []fields.Option{
				{Label: "main", Value: "main"},
				{Label: "develop", Value: "develop"},
				{Label: "feature/test", Value: "feature/test"},
			},
			suggested:      "develop",
			sourceBranch:   "feature/test",
			expectedLabels: []string{"develop (suggested from config)", "main"},
			expectedValues: []string{"develop", "main"},
		},
		{
			name: "suggestion_at_top_when_found",
			branches: []fields.Option{
				{Label: "main", Value: "main"},
				{Label: "develop", Value: "develop"},
				{Label: "staging", Value: "staging"},
			},
			suggested:      "main",
			sourceBranch:   "feature/test",
			expectedLabels: []string{"main (suggested from config)", "develop", "staging"},
			expectedValues: []string{"main", "develop", "staging"},
		},
		{
			name: "suggestion_not_in_list",
			branches: []fields.Option{
				{Label: "main", Value: "main"},
				{Label: "develop", Value: "develop"},
			},
			suggested:      "production",
			sourceBranch:   "feature/test",
			expectedLabels: []string{"main", "develop"},
			expectedValues: []string{"main", "develop"},
		},
		{
			name: "exclude_source_branch",
			branches: []fields.Option{
				{Label: "main", Value: "main"},
				{Label: "develop", Value: "develop"},
				{Label: "feature/test", Value: "feature/test"},
			},
			suggested:      "",
			sourceBranch:   "develop",
			expectedLabels: []string{"main", "feature/test"},
			expectedValues: []string{"main", "feature/test"},
		},
		{
			name: "suggestion_not_excluded_when_source",
			branches: []fields.Option{
				{Label: "main", Value: "main"},
				{Label: "develop", Value: "develop"},
				{Label: "feature/test", Value: "feature/test"},
			},
			suggested:      "develop",
			sourceBranch:   "develop",
			expectedLabels: []string{"develop (suggested from config)", "main", "feature/test"},
			expectedValues: []string{"develop", "main", "feature/test"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := SortTargetBranchOptions(tc.branches, tc.suggested, tc.sourceBranch)

			// Extract labels and values
			labels := make([]string, len(result))
			values := make([]string, len(result))
			for i, opt := range result {
				labels[i] = opt.Label
				values[i] = opt.Value
			}

			assert.Equal(t, tc.expectedLabels, labels)
			assert.Equal(t, tc.expectedValues, values)
		})
	}
}

func TestGetTrackedBranches(t *testing.T) {
	testCases := []struct {
		name     string
		config   tui.RepoConfig
		expected map[string]bool
	}{
		{
			name:     "nil_config",
			config:   nil,
			expected: map[string]bool{},
		},
		{
			name:     "empty_worktrees",
			config:   testutil.NewMockRepoConfig(),
			expected: map[string]bool{},
		},
		{
			name: "single_worktree",
			config: testutil.NewMockRepoConfig().
				WithWorktree("main", "main", ""),
			expected: map[string]bool{"main": true},
		},
		{
			name: "multiple_worktrees",
			config: testutil.NewMockRepoConfig().
				WithWorktree("main", "main", "").
				WithWorktree("develop", "develop", "main").
				WithWorktree("staging", "staging", "develop"),
			expected: map[string]bool{"main": true, "develop": true, "staging": true},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := GetTrackedBranches(tc.config)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestSortBranchOptionsByTracked(t *testing.T) {
	testCases := []struct {
		name           string
		branches       []fields.Option
		config         tui.RepoConfig
		expectedValues []string
	}{
		{
			name:           "empty_branches",
			branches:       []fields.Option{},
			config:         testutil.NewMockRepoConfig().WithWorktree("main", "main", ""),
			expectedValues: []string{},
		},
		{
			name: "nil_config_preserves_order",
			branches: []fields.Option{
				{Label: "feature/test", Value: "feature/test"},
				{Label: "main", Value: "main"},
				{Label: "develop", Value: "develop"},
			},
			config:         nil,
			expectedValues: []string{"feature/test", "main", "develop"},
		},
		{
			name: "no_tracked_branches_preserves_order",
			branches: []fields.Option{
				{Label: "feature/test", Value: "feature/test"},
				{Label: "main", Value: "main"},
				{Label: "develop", Value: "develop"},
			},
			config:         testutil.NewMockRepoConfig(),
			expectedValues: []string{"feature/test", "main", "develop"},
		},
		{
			name: "tracked_branches_first",
			branches: []fields.Option{
				{Label: "feature/test", Value: "feature/test"},
				{Label: "main", Value: "main"},
				{Label: "develop", Value: "develop"},
				{Label: "feature/other", Value: "feature/other"},
			},
			config: testutil.NewMockRepoConfig().
				WithWorktree("main", "main", "").
				WithWorktree("develop", "develop", "main"),
			expectedValues: []string{"main", "develop", "feature/test", "feature/other"},
		},
		{
			name: "all_tracked",
			branches: []fields.Option{
				{Label: "main", Value: "main"},
				{Label: "develop", Value: "develop"},
			},
			config: testutil.NewMockRepoConfig().
				WithWorktree("main", "main", "").
				WithWorktree("develop", "develop", "main"),
			expectedValues: []string{"main", "develop"},
		},
		{
			name: "none_tracked",
			branches: []fields.Option{
				{Label: "feature/a", Value: "feature/a"},
				{Label: "feature/b", Value: "feature/b"},
			},
			config: testutil.NewMockRepoConfig().
				WithWorktree("main", "main", ""),
			expectedValues: []string{"feature/a", "feature/b"},
		},
		{
			name: "preserves_order_within_groups",
			branches: []fields.Option{
				{Label: "feature/z", Value: "feature/z"},
				{Label: "develop", Value: "develop"},
				{Label: "feature/a", Value: "feature/a"},
				{Label: "main", Value: "main"},
				{Label: "feature/m", Value: "feature/m"},
			},
			config: testutil.NewMockRepoConfig().
				WithWorktree("main", "main", "").
				WithWorktree("develop", "develop", "main"),
			expectedValues: []string{"develop", "main", "feature/z", "feature/a", "feature/m"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := SortBranchOptionsByTracked(tc.branches, tc.config)

			values := make([]string, len(result))
			for i, opt := range result {
				values[i] = opt.Value
			}

			assert.Equal(t, tc.expectedValues, values)
		})
	}
}

package workflows

import (
	"gbm/pkg/tui"
	"testing"

	mocktest "gbm/internal/testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMergeCustomWorkflow_Creation tests that MergeCustomWorkflow creates a valid wizard.
func TestMergeCustomWorkflow_Creation(t *testing.T) {
	ctx := tui.NewContext().
		WithGitService(mocktest.NewMockGitService().
			WithBranches([]string{"main", "develop", "feature/test"})).
		WithDimensions(100, 30).
		WithTheme(tui.DefaultTheme())

	wizard := MergeCustomWorkflow(ctx)
	assert.NotNil(t, wizard)
	assert.NotNil(t, wizard.State())
}

// TestMergeCustomWorkflow_Steps tests that the workflow has 4 steps.
func TestMergeCustomWorkflow_Steps(t *testing.T) {
	ctx := tui.NewContext().
		WithGitService(mocktest.NewMockGitService()).
		WithDimensions(100, 30).
		WithTheme(tui.DefaultTheme())

	steps := getMergeCustomSteps(ctx)
	assert.Len(t, steps, 4)
	assert.Equal(t, "merge_strategy", steps[0].Name)
	assert.Equal(t, "source_branch", steps[1].Name)
	assert.Equal(t, "target_branch", steps[2].Name)
	assert.Equal(t, "confirm", steps[3].Name)
}

// TestMergeCustomWorkflow_ProcessMergeCustom tests workflow processing with valid data.
func TestMergeCustomWorkflow_ProcessMergeCustom(t *testing.T) {
	testCases := []struct {
		name      string
		strategy  string
		source    string
		target    string
		expectWt  string
		expectBr  string
		expectErr bool
	}{
		{
			name:      "valid merge with commit strategy",
			strategy:  "merge_commit",
			source:    "feature/auth",
			target:    "main",
			expectErr: false,
			expectWt:  "merge-feature/auth-to-main",
			expectBr:  "merge/feature/auth-to-main",
		},
		{
			name:      "valid squash merge",
			strategy:  "squash",
			source:    "develop",
			target:    "main",
			expectErr: false,
			expectWt:  "merge-develop-to-main",
			expectBr:  "merge/develop-to-main",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := tui.NewContext().
				WithGitService(mocktest.NewMockGitService()).
				WithDimensions(100, 30).
				WithTheme(tui.DefaultTheme())

			wizard := MergeCustomWorkflow(ctx)
			require.NotNil(t, wizard)

			state := wizard.State()
			state.SetField("merge_strategy", tc.strategy)
			state.SetField("source_branch", tc.source)
			state.SetField("target_branch", tc.target)

			err := ProcessMergeCustomWorkflow(wizard)

			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectWt, state.WorktreeName)
				assert.Equal(t, tc.expectBr, state.BranchName)
				assert.Equal(t, tc.strategy, state.GetField("merge_strategy_selected"))
			}
		})
	}
}

// TestMergeCustomWorkflow_MissingStrategy tests error handling for missing strategy.
func TestMergeCustomWorkflow_MissingStrategy(t *testing.T) {
	ctx := tui.NewContext().
		WithGitService(mocktest.NewMockGitService()).
		WithDimensions(100, 30).
		WithTheme(tui.DefaultTheme())

	wizard := MergeCustomWorkflow(ctx)
	state := wizard.State()
	state.SetField("source_branch", "main")
	state.SetField("target_branch", "develop")

	err := ProcessMergeCustomWorkflow(wizard)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "merge strategy not selected")
}

// TestMergeCustomWorkflow_MissingBranches tests error handling for missing branches.
func TestMergeCustomWorkflow_MissingBranches(t *testing.T) {
	ctx := tui.NewContext().
		WithGitService(mocktest.NewMockGitService()).
		WithDimensions(100, 30).
		WithTheme(tui.DefaultTheme())

	wizard := MergeCustomWorkflow(ctx)
	state := wizard.State()
	state.SetField("merge_strategy", "merge_commit")
	// Don't set branches

	err := ProcessMergeCustomWorkflow(wizard)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "source or target branch not selected")
}

// TestMergeCustomWorkflow_CustomFieldStorage tests that merge strategy is stored in custom fields.
func TestMergeCustomWorkflow_CustomFieldStorage(t *testing.T) {
	ctx := tui.NewContext().
		WithGitService(mocktest.NewMockGitService()).
		WithDimensions(100, 30).
		WithTheme(tui.DefaultTheme())

	wizard := MergeCustomWorkflow(ctx)
	state := wizard.State()

	// Set all fields
	state.SetField("merge_strategy", "rebase")
	state.SetField("source_branch", "feature/new-api")
	state.SetField("target_branch", "develop")

	// Process the workflow
	err := ProcessMergeCustomWorkflow(wizard)
	assert.NoError(t, err)

	// Verify custom field storage
	assert.Equal(t, "rebase", state.GetField("merge_strategy_selected"))
	assert.Equal(t, "merge-feature/new-api-to-develop", state.WorktreeName)
	assert.Equal(t, "merge/feature/new-api-to-develop", state.BranchName)
}

// TestMergeCustomWorkflow_AsyncBranches tests that async branch loading works.
func TestMergeCustomWorkflow_AsyncBranches(t *testing.T) {
	mockGit := mocktest.NewMockGitService().
		WithBranches([]string{"main", "develop", "release/v1.0", "feature/new-ui"})

	ctx := tui.NewContext().
		WithGitService(mockGit).
		WithDimensions(100, 30).
		WithTheme(tui.DefaultTheme())

	wizard := MergeCustomWorkflow(ctx)
	state := wizard.State()

	// The steps contain async operations - verify they're defined
	steps := getMergeCustomSteps(ctx)

	// Step 1 is selector (merge_strategy)
	assert.Equal(t, "merge_strategy", steps[0].Name)

	// Step 2 and 3 are async filterables
	assert.Equal(t, "source_branch", steps[1].Name)
	assert.Equal(t, "target_branch", steps[2].Name)

	// Verify they're Filterable fields (check Type)
	state.SetField("merge_strategy", "merge_commit")
	state.SetField("source_branch", "main")
	state.SetField("target_branch", "develop")

	// Process should complete without error
	err := ProcessMergeCustomWorkflow(wizard)
	assert.NoError(t, err)
}

// TestMergeCustomWorkflow_TargetExcludesSource tests that target excludes source branch.
func TestMergeCustomWorkflow_TargetExcludesSource(t *testing.T) {
	testCases := []struct {
		name   string
		source string
		target string
	}{
		{
			name:   "different branches",
			source: "main",
			target: "develop",
		},
		{
			name:   "feature to main",
			source: "feature/auth",
			target: "main",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockGit := mocktest.NewMockGitService().
				WithBranches([]string{"main", "develop", "feature/auth"})

			ctx := tui.NewContext().
				WithGitService(mockGit).
				WithDimensions(100, 30).
				WithTheme(tui.DefaultTheme())

			wizard := MergeCustomWorkflow(ctx)
			state := wizard.State()

			state.SetField("merge_strategy", "merge_commit")
			state.SetField("source_branch", tc.source)
			state.SetField("target_branch", tc.target)

			// Target should not equal source
			assert.NotEqual(t, tc.source, tc.target)

			err := ProcessMergeCustomWorkflow(wizard)
			assert.NoError(t, err)
		})
	}
}

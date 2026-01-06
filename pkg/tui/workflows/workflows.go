// Package workflows provides pre-configured Wizard workflows for common git operations.
package workflows

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"gbm/pkg/tui"
	"gbm/pkg/tui/fields"
)

// slugify converts a string to a URL-friendly slug.
// Converts to lowercase, replaces spaces with hyphens, and removes special characters.
func slugify(s string) string {
	// Convert to lowercase
	s = strings.ToLower(s)

	// Replace spaces with hyphens
	s = strings.ReplaceAll(s, " ", "-")

	// Remove special characters, keep only alphanumeric, hyphens, and underscores
	s = regexp.MustCompile(`[^\w-]`).ReplaceAllString(s, "")

	// Remove multiple consecutive hyphens
	s = regexp.MustCompile(`-+`).ReplaceAllString(s, "-")

	// Trim hyphens from start and end
	s = strings.Trim(s, "-")

	return s
}

// generateBranchName generates a branch name from a JIRA issue key and summary.
// Format: feature/{issue-key}-{slugified-summary}
func generateBranchName(issueKey, summary string) string {
	slug := slugify(summary)
	return fmt.Sprintf("feature/%s-%s", issueKey, slug)
}

// generateBranchNameHotfix generates a hotfix branch name from a JIRA issue key and summary.
// Format: hotfix/{issue-key}-{slugified-summary}
// Used by HotfixWorkflow.
//
//nolint:unused
func generateBranchNameHotfix(issueKey, summary string) string {
	slug := slugify(summary)
	return fmt.Sprintf("hotfix/%s-%s", issueKey, slug)
}

// FeatureWorkflow creates and returns a Wizard configured for creating feature branches.
// Steps:
// 1. Filterable for JIRA issue selection or custom worktree name
// 2. TextInput for branch name (with auto-generated default from JIRA issue key)
// 3. Filterable for base branch (with Skip logic - skipped if branch exists)
// 4. Confirm step with summary
//
// After the wizard completes, call ProcessFeatureWorkflow to handle the special logic
// of generating branch names and setting worktree names from JIRA issues.
func FeatureWorkflow(ctx *tui.Context) *tui.Wizard {
	steps := []tui.Step{
		// Step 1: JIRA issue selection or custom worktree name
		{
			Name: "worktree_name",
			Field: fields.NewFilterable(
				"worktree_name",
				"Select JIRA Issue or Enter Worktree Name",
				"Search JIRA issues or enter a custom name",
				[]fields.Option{},
			).WithOptionsFunc(func() ([]fields.Option, error) {
				if ctx.JiraService == nil {
					return []fields.Option{}, nil
				}
				issues, err := ctx.JiraService.FetchIssues()
				if err != nil {
					return nil, err
				}

				options := make([]fields.Option, len(issues))
				for i, issue := range issues {
					options[i] = fields.Option{
						Label: fmt.Sprintf("%s - %s", issue.Key, issue.Summary),
						Value: issue.Key, // Store issue key as value
					}
				}
				return options, nil
			}),
		},

		// Step 2: Branch name (with auto-generated default from JIRA issue)
		{
			Name: "branch_name",
			Field: fields.NewTextInput("branch_name", "Enter Branch Name", "Name for the new branch").
				WithPlaceholder("feature/KEY-description").
				WithValidator(validateBranchName),
		},

		// Step 3: Base branch selection (skipped if branch name already exists)
		{
			Name: "base_branch",
			Field: fields.NewFilterable(
				"base_branch",
				"Select Base Branch",
				"Choose the branch to base this feature on",
				[]fields.Option{},
			).WithOptionsFunc(func() ([]fields.Option, error) {
				if ctx.GitService == nil {
					return []fields.Option{}, nil
				}
				branches, err := ctx.GitService.ListBranches()
				if err != nil {
					return nil, err
				}
				options := make([]fields.Option, len(branches))
				for i, branch := range branches {
					options[i] = fields.Option{
						Label: branch,
						Value: branch,
					}
				}
				return options, nil
			}),

			// Skip this step if the branch name already exists
			Skip: func(state *tui.WorkflowState) bool {
				if ctx.GitService == nil || state.BranchName == "" {
					return false
				}
				exists, err := ctx.GitService.BranchExists(state.BranchName)
				return err == nil && exists
			},
		},

		// Step 4: Confirmation
		{
			Name:  "confirm",
			Field: fields.NewConfirm("confirm", "Create Feature Branch?"),
		},
	}

	return tui.NewWizard(steps, ctx)
}

// ProcessFeatureWorkflow handles post-wizard processing for feature workflows.
// This function:
// 1. Looks up the full JIRA issue details to get the summary
// 2. Generates a default branch name if not provided (format: feature/{key}-{slugified-summary})
// 3. Stores the issue key as the worktree name
func ProcessFeatureWorkflow(wizard *tui.Wizard, ctx *tui.Context) error {
	state := wizard.State()

	// If a worktree name was entered, it might be a JIRA issue key
	// Try to look it up and generate a branch name
	if state.WorktreeName != "" && ctx.JiraService != nil {
		issues, err := ctx.JiraService.FetchIssues()
		if err != nil {
			// If we can't fetch issues, that's OK - use the worktree name as-is
			return nil
		}

		// Look for a matching JIRA issue
		for _, issue := range issues {
			if issue.Key == state.WorktreeName {
				// Found the issue - generate branch name if not already set
				if state.BranchName == "" {
					state.BranchName = generateBranchName(issue.Key, issue.Summary)
					// Also store the full issue info for reference
					state.JiraIssue = &issue
				}
				break
			}
		}
	}

	return nil
}

// HotfixWorkflow creates and returns a Wizard configured for creating hotfix branches.
// Unlike FeatureWorkflow, HotfixWorkflow:
// - Always requires base branch selection (no skip logic)
// - Prefixes worktree names with "HOTFIX_"
// - Uses "hotfix/" prefix for branch names
//
// Steps:
// 1. Filterable for JIRA issue selection or custom worktree name
// 2. Filterable for base branch (always shown - mandatory for hotfixes)
// 3. TextInput for branch name (with auto-generated default from JIRA issue key)
// 4. Confirm step with summary
//
// After the wizard completes, call ProcessHotfixWorkflow to handle the special logic
// of generating branch names, prefixing worktree names, and setting worktree names from JIRA issues.
func HotfixWorkflow(ctx *tui.Context) *tui.Wizard {
	steps := []tui.Step{
		// Step 1: JIRA issue selection or custom worktree name
		{
			Name: "worktree_name",
			Field: fields.NewFilterable(
				"worktree_name",
				"Select JIRA Issue or Enter Worktree Name",
				"Search JIRA issues or enter a custom name",
				[]fields.Option{},
			).WithOptionsFunc(func() ([]fields.Option, error) {
				if ctx.JiraService == nil {
					return []fields.Option{}, nil
				}
				issues, err := ctx.JiraService.FetchIssues()
				if err != nil {
					return nil, err
				}

				options := make([]fields.Option, len(issues))
				for i, issue := range issues {
					options[i] = fields.Option{
						Label: fmt.Sprintf("%s - %s", issue.Key, issue.Summary),
						Value: issue.Key, // Store issue key as value
					}
				}
				return options, nil
			}),
		},

		// Step 2: Base branch selection (mandatory - NOT skipped)
		{
			Name: "base_branch",
			Field: fields.NewFilterable(
				"base_branch",
				"Select Base Branch",
				"Choose the production or release branch to base this hotfix on",
				[]fields.Option{},
			).WithOptionsFunc(func() ([]fields.Option, error) {
				if ctx.GitService == nil {
					return []fields.Option{}, nil
				}
				branches, err := ctx.GitService.ListBranches()
				if err != nil {
					return nil, err
				}
				options := make([]fields.Option, len(branches))
				for i, branch := range branches {
					options[i] = fields.Option{
						Label: branch,
						Value: branch,
					}
				}
				return options, nil
			}),
			// No Skip function - base branch is always required for hotfixes
		},

		// Step 3: Branch name (with auto-generated default from JIRA issue)
		{
			Name: "branch_name",
			Field: fields.NewTextInput("branch_name", "Enter Branch Name", "Name for the hotfix branch").
				WithPlaceholder("hotfix/KEY-description").
				WithValidator(validateBranchName),
		},

		// Step 4: Confirmation
		{
			Name:  "confirm",
			Field: fields.NewConfirm("confirm", "Create Hotfix Branch?"),
		},
	}

	return tui.NewWizard(steps, ctx)
}

// ProcessHotfixWorkflow handles post-wizard processing for hotfix workflows.
// This function:
// 1. Looks up the full JIRA issue details to get the summary
// 2. Generates a default branch name if not provided (format: hotfix/{key}-{slugified-summary})
// 3. Prefixes the worktree name with "HOTFIX_"
// 4. Stores the issue key (with prefix) as the worktree name
func ProcessHotfixWorkflow(wizard *tui.Wizard, ctx *tui.Context) error {
	state := wizard.State()

	// If a worktree name was entered, it might be a JIRA issue key
	// Try to look it up and generate a branch name
	if state.WorktreeName != "" && ctx.JiraService != nil {
		issues, err := ctx.JiraService.FetchIssues()
		if err != nil {
			// If we can't fetch issues, that's OK - use the worktree name as-is
			return nil
		}

		// Look for a matching JIRA issue
		for _, issue := range issues {
			if issue.Key == state.WorktreeName {
				// Found the issue - generate branch name if not already set
				if state.BranchName == "" {
					state.BranchName = generateBranchNameHotfix(issue.Key, issue.Summary)
					// Also store the full issue info for reference
					state.JiraIssue = &issue
				}
				break
			}
		}
	}

	// Prefix the worktree name with HOTFIX_
	state.WorktreeName = fmt.Sprintf("HOTFIX_%s", state.WorktreeName)

	return nil
}

// validateBranchName validates a branch name.
// Returns an error if the branch name is empty or contains invalid characters.
func validateBranchName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("branch name cannot be empty")
	}

	// Check for invalid characters (basic validation)
	// Valid: alphanumeric, hyphens, underscores, slashes, dots
	for _, ch := range name {
		if !isValidBranchChar(ch) {
			return fmt.Errorf("branch name contains invalid character: %c", ch)
		}
	}

	return nil
}

// isValidBranchChar checks if a character is valid in a branch name.
func isValidBranchChar(ch rune) bool {
	return unicode.IsLetter(ch) || unicode.IsDigit(ch) ||
		ch == '-' || ch == '_' || ch == '/' || ch == '.'
}

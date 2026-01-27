// Package workflows provides pre-configured Wizard workflows for common git operations.
package workflows

import (
	"fmt"
	"gbm/pkg/tui"
	"regexp"
	"strings"
	"unicode"
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

// generateBranchNameBug generates a bug branch name from a JIRA issue key and summary.
// Format: bug/{issue-key}-{slugified-summary}
// Used by BugWorkflow.
//
//nolint:unused
func generateBranchNameBug(issueKey, summary string) string {
	slug := slugify(summary)
	return fmt.Sprintf("bug/%s-%s", issueKey, slug)
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
	steps := getFeatureSteps(ctx)
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
	steps := getHotfixSteps(ctx)
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

// BugWorkflow creates and returns a Wizard configured for creating bug fix branches.
// BugWorkflow has the same step sequence as FeatureWorkflow but uses "bug/" prefix for branches.
//
// Steps:
// 1. Filterable for JIRA issue selection or custom worktree name
// 2. TextInput for branch name (with auto-generated default from JIRA issue key)
// 3. Filterable for base branch (with Skip logic - skipped if branch exists)
// 4. Confirm step with summary
//
// After the wizard completes, call ProcessBugWorkflow to handle the special logic
// of generating branch names and setting worktree names from JIRA issues.
func BugWorkflow(ctx *tui.Context) *tui.Wizard {
	steps := getBugSteps(ctx)
	return tui.NewWizard(steps, ctx)
}

// ProcessBugWorkflow handles post-wizard processing for bug workflows.
// This function:
// 1. Looks up the full JIRA issue details to get the summary
// 2. Generates a default branch name if not provided (format: bug/{key}-{slugified-summary})
// 3. Stores the issue key as the worktree name
func ProcessBugWorkflow(wizard *tui.Wizard, ctx *tui.Context) error {
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
					state.BranchName = generateBranchNameBug(issue.Key, issue.Summary)
					// Also store the full issue info for reference
					state.JiraIssue = &issue
				}
				break
			}
		}
	}

	return nil
}

// MergeWorkflow creates and returns a Wizard configured for merging branches.
// Unlike feature/bug/hotfix workflows, merge workflows don't involve JIRA issues.
//
// Steps:
// 1. Filterable for source branch (branch to merge FROM)
// 2. Filterable for target branch (branch to merge INTO)
// 3. Confirm step with merge details (worktree and branch names auto-generated)
//
// After the wizard completes, call ProcessMergeWorkflow to handle the special logic
// of generating worktree and branch names from the selected branches.
func MergeWorkflow(ctx *tui.Context) *tui.Wizard {
	steps := getMergeSteps(ctx)
	return tui.NewWizard(steps, ctx)
}

// ProcessMergeWorkflow handles post-wizard processing for merge workflows.
// This function:
// 1. Generates the worktree name: MERGE_{source-to-target}
// 2. Generates the branch name: merge/{source-to-target}
func ProcessMergeWorkflow(wizard *tui.Wizard, ctx *tui.Context) error {
	state := wizard.State()

	// Get source and target branches from state
	// These are stored as custom fields in the workflow
	sourceBranch := getStateField(state, "source_branch")
	targetBranch := getStateField(state, "target_branch")

	if sourceBranch == "" || targetBranch == "" {
		return nil
	}

	// Generate worktree name: MERGE_{source-to-target}
	// Sanitize branch names for use in worktree name (remove slashes and special chars)
	sourceSanitized := sanitizeForWorktreeName(sourceBranch)
	targetSanitized := sanitizeForWorktreeName(targetBranch)
	state.WorktreeName = fmt.Sprintf("MERGE_%s-to-%s", sourceSanitized, targetSanitized)

	// Generate branch name: merge/{source-to-target}
	state.BranchName = fmt.Sprintf("merge/%s-to-%s", sourceSanitized, targetSanitized)

	// Set BaseBranch to target branch (merge branch is based off target)
	state.BaseBranch = targetBranch

	return nil
}

// sanitizeForWorktreeName sanitizes a branch name for use in a worktree name.
// Removes slashes and other special characters.
func sanitizeForWorktreeName(name string) string {
	// Replace slashes with underscores
	name = strings.ReplaceAll(name, "/", "_")
	// Remove any characters that aren't alphanumeric, underscores, or hyphens
	name = regexp.MustCompile(`[^\w-]`).ReplaceAllString(name, "")
	return name
}

// getStateField retrieves a custom field from WorkflowState.
// This is needed because merge workflows store custom fields (source_branch, target_branch)
// instead of the standard WorktreeName/BranchName fields.
func getStateField(state *tui.WorkflowState, fieldName string) string {
	value := state.GetField(fieldName)
	if str, ok := value.(string); ok {
		return str
	}
	return ""
}

package service

import (
	"fmt"
	"regexp"
	"strings"
)

// createSortedBranchItems creates sorted and labeled branch items for selection
// Sorting order: 1) Tracked by config, 2) Ad hoc worktree branches, 3) Other branches
func createSortedBranchItems(svc *Service) ([]FilterableItem, error) {
	// Get all branches
	branches, err := svc.Git.ListBranches(false)
	if err != nil {
		return nil, fmt.Errorf("failed to list branches: %w", err)
	}

	// Get config to see tracked worktrees
	config := svc.GetConfig()

	// Get worktrees to see which branches have worktrees
	worktrees, err := svc.Git.ListWorktrees(false)
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	// Create maps for quick lookup
	branchToWorktreeName := make(map[string]string) // branch -> worktree name
	trackedBranches := make(map[string]bool)        // branches tracked in config

	// Build worktree lookup map
	for _, wt := range worktrees {
		if !wt.IsBare {
			branchToWorktreeName[wt.Branch] = wt.Name
		}
	}

	// Build tracked branches map
	for wtName, wtConfig := range config.Worktrees {
		trackedBranches[wtConfig.Branch] = true
		// Also track the worktree name -> branch mapping
		if _, exists := branchToWorktreeName[wtConfig.Branch]; !exists {
			branchToWorktreeName[wtConfig.Branch] = wtName
		}
	}

	// Categorize branches
	var trackedItems []FilterableItem
	var adHocItems []FilterableItem
	var otherItems []FilterableItem

	for _, branch := range branches {
		wtName, hasWorktree := branchToWorktreeName[branch]
		isTracked := trackedBranches[branch]

		var label string
		if isTracked && hasWorktree {
			label = fmt.Sprintf("%s (tracked: %s)", branch, wtName)
		} else if isTracked {
			label = fmt.Sprintf("%s (tracked, no worktree)", branch)
		} else if hasWorktree {
			label = fmt.Sprintf("%s (worktree: %s)", branch, wtName)
		} else {
			label = branch
		}

		item := FilterableItem{
			Label: label,
			Value: branch,
		}

		if isTracked {
			trackedItems = append(trackedItems, item)
		} else if hasWorktree {
			adHocItems = append(adHocItems, item)
		} else {
			otherItems = append(otherItems, item)
		}
	}

	// Combine in priority order
	result := make([]FilterableItem, 0, len(branches))
	result = append(result, trackedItems...)
	result = append(result, adHocItems...)
	result = append(result, otherItems...)

	return result, nil
}

// sanitizeBranchName converts branch name to valid directory name
func sanitizeBranchName(branch string) string {
	// Remove common prefixes
	branch = strings.TrimPrefix(branch, "origin/")
	branch = strings.TrimPrefix(branch, "refs/heads/")
	// Replace invalid directory chars
	replacer := strings.NewReplacer("/", "-", "\\", "-")
	return replacer.Replace(branch)
}

// sanitizeSummaryForBranch converts JIRA summary to branch-name-friendly format
func sanitizeSummaryForBranch(summary string) string {
	// Convert summary to lowercase
	s := strings.ToLower(summary)

	// Replace spaces and special chars with hyphens
	reg := regexp.MustCompile(`[^a-z0-9]+`)
	s = reg.ReplaceAllString(s, "-")

	// Remove leading/trailing hyphens
	s = strings.Trim(s, "-")

	// Limit length to reasonable size
	if len(s) > 50 {
		s = s[:50]
	}

	// Remove trailing hyphen if we truncated mid-word
	s = strings.TrimRight(s, "-")

	return s
}

// generateMergeCommitMessage creates a commit message for a merge
// If both branches are tracked in config, uses worktree names and descriptions
// Otherwise uses branch names
func generateMergeCommitMessage(svc *Service, sourceBranch, targetBranch string) string {
	config := svc.GetConfig()

	// Find config entries for source and target branches
	var sourceWorktreeName, sourceDescription string
	var targetWorktreeName, targetDescription string
	var sourceTracked, targetTracked bool

	for wtName, wtConfig := range config.Worktrees {
		if wtConfig.Branch == sourceBranch {
			sourceWorktreeName = wtName
			sourceDescription = wtConfig.Description
			sourceTracked = true
		}
		if wtConfig.Branch == targetBranch {
			targetWorktreeName = wtName
			targetDescription = wtConfig.Description
			targetTracked = true
		}
	}

	// If both are tracked, use worktree names and descriptions
	if sourceTracked && targetTracked {
		fromPart := sourceWorktreeName
		if sourceDescription != "" {
			fromPart = fmt.Sprintf("%s - %s", sourceWorktreeName, sourceDescription)
		}

		toPart := targetWorktreeName
		if targetDescription != "" {
			toPart = fmt.Sprintf("%s - %s", targetWorktreeName, targetDescription)
		}

		return fmt.Sprintf("merge: FROM [%s], TO [%s]", fromPart, toPart)
	}

	// Otherwise use branch names
	return fmt.Sprintf("merge: FROM [%s], TO [%s]", sourceBranch, targetBranch)
}

// fetchJiraItems retrieves JIRA issues and converts them to FilterableItem format
func fetchJiraItems(svc *Service) []FilterableItem {
	filters := svc.GetJiraFilters()
	issues, err := svc.Jira.GetJiraIssues(filters, false)
	if err != nil || len(issues) == 0 {
		return []FilterableItem{}
	}

	items := make([]FilterableItem, len(issues))
	for i, issue := range issues {
		items[i] = FilterableItem{
			Label: fmt.Sprintf("%s: %s", issue.Key, issue.Summary),
			Value: issue.Key,
		}
	}
	return items
}

// generateWorktreeCompletions creates shell completion entries for worktrees
// Returns a slice of formatted completions in the format "name\tbranch"
// Excludes bare repositories from the completions
func generateWorktreeCompletions(svc *Service) ([]string, error) {
	worktrees, err := svc.Git.ListWorktrees(false)
	if err != nil {
		return nil, err
	}

	var completions []string
	for _, wt := range worktrees {
		// Exclude bare repository
		if wt.IsBare {
			continue
		}
		// Add completion with branch context
		completion := wt.Name
		if wt.Branch != "" {
			completion = fmt.Sprintf("%s\t%s", wt.Name, wt.Branch)
		}
		completions = append(completions, completion)
	}

	return completions, nil
}

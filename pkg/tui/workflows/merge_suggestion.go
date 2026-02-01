package workflows

import (
	"gbm/pkg/tui"
	"gbm/pkg/tui/fields"
)

// SuggestMergeTarget suggests a merge target branch based on the source branch and configuration.
// It looks up the source branch in the repository configuration and returns the configured
// merge_into target if available.
//
// Parameters:
// - ctx: Context containing the repository configuration
// - sourceBranch: The source branch name to look up
//
// Returns:
// - The merge_into target branch if configured, or empty string if not found or not configured.
func SuggestMergeTarget(ctx *tui.Context, sourceBranch string) string {
	// If no config available, cannot suggest
	if ctx.Config == nil {
		return ""
	}

	// Get the worktrees configuration
	worktrees := ctx.Config.GetWorktrees()
	if worktrees == nil {
		return ""
	}

	// Look for a worktree configuration that matches the source branch
	for _, wt := range worktrees {
		if wt == nil {
			continue
		}
		if wt.GetBranch() == sourceBranch {
			// Found the worktree - return its merge_into target
			mergeInto := wt.GetMergeInto()
			if mergeInto != "" {
				return mergeInto
			}
		}
	}

	return ""
}

// SortTargetBranchOptions sorts branch options, placing suggested branch at top if found.
// If a suggested branch is provided and exists in the list, it's moved to the top
// with an updated label indicating it's suggested from config.
//
// Parameters:
// - branches: List of available branch options
// - suggested: The suggested merge target branch (empty string means no suggestion)
// - sourceBranch: The source branch to exclude from the options
//
// Returns:
// - Sorted list of options with suggested branch at top (if applicable).
func SortTargetBranchOptions(branches []fields.Option, suggested, sourceBranch string) []fields.Option {
	if len(branches) == 0 {
		return branches
	}

	// If no suggestion or suggestion is empty, just exclude source branch and return
	if suggested == "" {
		return excludeSourceBranch(branches, sourceBranch)
	}

	// Build result with suggested branch at top
	result := make([]fields.Option, 0, len(branches))
	foundSuggested := false

	// First, add the suggested branch if it exists in the list
	for _, opt := range branches {
		if opt.Value == suggested {
			// Add suggested option with updated label
			result = append(result, fields.Option{
				Label: opt.Label + " (suggested from config)",
				Value: opt.Value,
			})
			foundSuggested = true
			break
		}
	}

	// Then add all other branches (except source)
	for _, opt := range branches {
		// Skip source branch
		if opt.Value == sourceBranch {
			continue
		}
		// Skip the suggested branch (already added above)
		if foundSuggested && opt.Value == suggested {
			continue
		}
		result = append(result, opt)
	}

	return result
}

// excludeSourceBranch removes the source branch from the options list.
// Used when there's no suggestion available.
func excludeSourceBranch(branches []fields.Option, sourceBranch string) []fields.Option {
	result := make([]fields.Option, 0, len(branches))
	for _, opt := range branches {
		if opt.Value != sourceBranch {
			result = append(result, opt)
		}
	}
	return result
}

// GetTrackedBranches builds a set of tracked branch names from the config.
// Returns an empty map if config is nil or has no worktrees.
func GetTrackedBranches(cfg tui.RepoConfig) map[string]bool {
	tracked := make(map[string]bool)
	if cfg == nil {
		return tracked
	}

	worktrees := cfg.GetWorktrees()
	if worktrees == nil {
		return tracked
	}

	for _, wt := range worktrees {
		if wt == nil {
			continue
		}
		if branch := wt.GetBranch(); branch != "" {
			tracked[branch] = true
		}
	}
	return tracked
}

// SortBranchOptionsByTracked sorts branch options, placing tracked branches first.
// Tracked branches are those that have a worktree entry in the config.
// Within each group (tracked and non-tracked), the original order is preserved.
//
// Parameters:
// - branches: List of available branch options
// - cfg: Repository configuration containing worktree definitions
//
// Returns:
// - Sorted list with tracked branches first, then non-tracked branches.
func SortBranchOptionsByTracked(branches []fields.Option, cfg tui.RepoConfig) []fields.Option {
	if len(branches) == 0 {
		return branches
	}

	tracked := GetTrackedBranches(cfg)
	if len(tracked) == 0 {
		return branches
	}

	// Partition into tracked and non-tracked
	trackedOpts := make([]fields.Option, 0)
	otherOpts := make([]fields.Option, 0)

	for _, opt := range branches {
		if tracked[opt.Value] {
			trackedOpts = append(trackedOpts, opt)
		} else {
			otherOpts = append(otherOpts, opt)
		}
	}

	// Combine: tracked first, then others
	result := make([]fields.Option, 0, len(branches))
	result = append(result, trackedOpts...)
	result = append(result, otherOpts...)
	return result
}

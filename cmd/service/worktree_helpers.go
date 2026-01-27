package service

import (
	"fmt"
)

// generateWorktreeCompletions creates shell completion entries for worktrees
// Returns a slice of formatted completions in the format "name\tbranch"
// Excludes bare repositories from the completions.
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

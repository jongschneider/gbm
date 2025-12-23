package service

import (
	"fmt"
	"strings"
)

// validateWorktreeName ensures name is valid directory name
func validateWorktreeName(name string) error {
	if name == "" {
		return fmt.Errorf("worktree name cannot be empty")
	}
	if strings.ContainsAny(name, "/\\:*?\"<>|") {
		return fmt.Errorf("invalid characters in worktree name")
	}
	return nil
}

// createWorktreeNameValidator creates a validator that checks both format and existence
func createWorktreeNameValidator(svc *Service) func(string) error {
	return func(name string) error {
		// First check basic validation
		if err := validateWorktreeName(name); err != nil {
			return err
		}

		// Check if worktree already exists
		worktrees, err := svc.Git.ListWorktrees(false)
		if err != nil {
			// Don't fail validation on list error, just skip existence check
			return nil
		}

		for _, wt := range worktrees {
			if wt.Name == name {
				return fmt.Errorf("worktree '%s' already exists", name)
			}
		}

		return nil
	}
}

// validateBranchName ensures branch name is valid
func validateBranchName(name string) error {
	if name == "" {
		return fmt.Errorf("branch name cannot be empty")
	}
	// Git branch name rules
	if strings.HasPrefix(name, "-") || strings.HasSuffix(name, ".") {
		return fmt.Errorf("invalid branch name format")
	}
	return nil
}

// createBranchNameValidator creates a validator that checks both format and existence
func createBranchNameValidator(svc *Service) func(string) error {
	return func(name string) error {
		// First check basic validation
		if err := validateBranchName(name); err != nil {
			return err
		}

		// Check if branch already exists
		exists, err := svc.Git.BranchExists(name)
		if err != nil {
			// Don't fail validation on check error, just skip existence check
			return nil
		}

		if exists {
			return fmt.Errorf("branch '%s' already exists", name)
		}

		return nil
	}
}

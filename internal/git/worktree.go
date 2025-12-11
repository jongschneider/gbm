package git

import (
	"cmp"
	"fmt"
	"os/exec"
	"path/filepath"
)

// AddWorktree creates a new git worktree in the specified directory
func (s *Service) AddWorktree(worktreesDir, worktreeName, branchName string, createBranch bool, baseBranch string, dryRun bool) error {
	if worktreesDir == "" {
		return fmt.Errorf("worktrees directory cannot be empty")
	}
	if worktreeName == "" {
		return fmt.Errorf("worktree name cannot be empty")
	}
	if branchName == "" {
		return fmt.Errorf("branch name cannot be empty")
	}

	// Construct the full worktree path
	worktreePath := filepath.Join(worktreesDir, worktreeName)
	args := []string{"worktree", "add", worktreePath, branchName}

	// Build git worktree add command
	if createBranch {
		// Use default base branch if not specified
		baseBranch = cmp.Or(baseBranch, "main")

		args = []string{"worktree", "add", "-b", branchName, worktreePath, baseBranch}
	}

	cmd := exec.Command("git", args...)
	output, err := s.runCommand(cmd, dryRun)
	if err != nil {
		return fmt.Errorf("failed to add worktree: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// ListWorktrees lists all git worktrees in the repository
func (s *Service) ListWorktrees(dryRun bool) (string, error) {
	args := []string{"worktree", "list"}

	cmd := exec.Command("git", args...)
	output, err := s.runCommand(cmd, dryRun)
	if err != nil {
		return "", fmt.Errorf("failed to list worktrees: %w\nOutput: %s", err, string(output))
	}

	return string(output), nil
}

// RemoveWorktree removes a git worktree from the specified directory
func (s *Service) RemoveWorktree(worktreesDir, worktreeName string, force bool, dryRun bool) error {
	if worktreesDir == "" {
		return fmt.Errorf("worktrees directory cannot be empty")
	}
	if worktreeName == "" {
		return fmt.Errorf("worktree name cannot be empty")
	}

	// Construct the full worktree path
	worktreePath := filepath.Join(worktreesDir, worktreeName)
	args := []string{"worktree", "remove", worktreePath}

	if force {
		args = []string{"worktree", "remove", "--force", worktreePath}
	}

	cmd := exec.Command("git", args...)
	output, err := s.runCommand(cmd, dryRun)
	if err != nil {
		return fmt.Errorf("failed to remove worktree: %w\nOutput: %s", err, string(output))
	}

	return nil
}

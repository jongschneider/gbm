package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// BranchExists checks if a branch exists locally
func (s *Service) BranchExists(branchName string) (bool, error) {
	if branchName == "" {
		return false, ErrBranchNameEmpty
	}

	cmd := exec.Command("git", "rev-parse", "--verify", branchName)
	err := cmd.Run()
	if err != nil {
		// Branch doesn't exist
		return false, nil
	}

	return true, nil
}

// BranchExistsInPath checks if a branch exists in a specific worktree path
func (s *Service) BranchExistsInPath(worktreePath, branchName string) (bool, error) {
	if worktreePath == "" {
		return false, ErrWorktreePathEmpty
	}
	if branchName == "" {
		return false, ErrBranchNameEmpty
	}

	cmd := exec.Command("git", "-C", worktreePath, "rev-parse", "--verify", branchName)
	err := cmd.Run()
	if err != nil {
		// Branch doesn't exist
		return false, nil
	}

	return true, nil
}

// DeleteBranch deletes a git branch
func (s *Service) DeleteBranch(branchName string, force bool, dryRun bool) error {
	if branchName == "" {
		return ErrBranchNameEmpty
	}

	args := []string{"branch", "-d", branchName}
	if force {
		args = []string{"branch", "-D", branchName}
	}

	cmd := exec.Command("git", args...)
	if dryRun {
		printDryRun(cmd)
		return nil
	}

	_, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to delete branch: %w", err)
	}

	return nil
}

// ListBranches returns all local and remote branches
func (s *Service) ListBranches(dryRun bool) ([]string, error) {
	cmd := exec.Command("git", "branch", "-a", "--format=%(refname:short)")

	if dryRun {
		printDryRun(cmd)
		return []string{}, nil
	}

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list branches: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	// Filter out empty lines
	branches := []string{}
	for _, line := range lines {
		if line != "" {
			branches = append(branches, line)
		}
	}
	return branches, nil
}

// MergeBranchWithCommit merges a branch and creates a commit with the specified message
func (s *Service) MergeBranchWithCommit(worktreePath, sourceBranch, commitMessage string, dryRun bool) error {
	cmd := exec.Command("git", "-C", worktreePath, "merge", "-m", commitMessage, sourceBranch)

	if dryRun {
		printDryRun(cmd)
		return nil
	}

	_, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to merge branch: %w", err)
	}
	return nil
}

// GetUpstreamBranch gets the upstream tracking branch for a worktree
// Returns empty string if no upstream is configured (not an error)
func (s *Service) GetUpstreamBranch(worktreePath string) (string, error) {
	if worktreePath == "" {
		return "", ErrWorktreePathEmpty
	}

	cmd := exec.Command("git", "-C", worktreePath, "rev-parse", "--abbrev-ref", "@{upstream}")
	output, err := cmd.Output()
	if err != nil {
		// No upstream configured - this is not an error, just return empty string
		return "", nil
	}

	return strings.TrimSpace(string(output)), nil
}

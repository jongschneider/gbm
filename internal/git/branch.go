package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// BranchExists checks if a branch exists locally.
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

// BranchExistsInPath checks if a branch exists in a specific worktree path.
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

// DeleteBranch deletes a git branch.
func (s *Service) DeleteBranch(branchName string, force, dryRun bool) error {
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

	output, err := cmd.CombinedOutput()
	if err != nil {
		return ClassifyError("branch delete", err, string(output))
	}

	return nil
}

// ListBranches returns all local and remote branches.
func (s *Service) ListBranches(dryRun bool) ([]string, error) {
	cmd := exec.Command("git", "branch", "-a", "--format=%(refname:short)")

	if dryRun {
		printDryRun(cmd)
		return []string{}, nil
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, ClassifyError("branch list", err, string(output))
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

// MergeBranchWithCommit merges a branch into the worktree with --no-ff to always create a merge commit.
func (s *Service) MergeBranchWithCommit(worktreePath, sourceBranch, commitMessage string, dryRun bool) error {
	cmd := exec.Command("git", "-C", worktreePath, "merge", "--no-ff", "-m", commitMessage, sourceBranch)

	if dryRun {
		printDryRun(cmd)
		return nil
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return ClassifyError("branch merge", err, string(output))
	}
	return nil
}

// GetUpstreamBranch gets the upstream tracking branch for a worktree
// Returns empty string if no upstream is configured (not an error).
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

// SetGbmBase records the base branch a worktree was created from, in the
// worktree's git config under `branch.<branchName>.gbmBase`. This survives
// upstream changes (e.g. `git push -u`), so `gbm wt info` can still report
// the original base after the branch is published.
func (s *Service) SetGbmBase(worktreePath, branchName, baseRef string, dryRun bool) error {
	if worktreePath == "" {
		return ErrWorktreePathEmpty
	}
	if branchName == "" {
		return ErrBranchNameEmpty
	}

	key := "branch." + branchName + ".gbmBase"
	cmd := exec.Command("git", "-C", worktreePath, "config", "--local", key, baseRef)

	if dryRun {
		printDryRun(cmd)
		return nil
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return ClassifyError("config set gbmBase", err, string(output))
	}
	return nil
}

// UnsetGbmBase removes the recorded base branch config for the given branch.
// Idempotent: missing keys are treated as success.
func (s *Service) UnsetGbmBase(worktreePath, branchName string, dryRun bool) error {
	if worktreePath == "" {
		return ErrWorktreePathEmpty
	}
	if branchName == "" {
		return ErrBranchNameEmpty
	}

	key := "branch." + branchName + ".gbmBase"
	cmd := exec.Command("git", "-C", worktreePath, "config", "--local", "--unset", key)

	if dryRun {
		printDryRun(cmd)
		return nil
	}

	output, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}
	// `git config --unset` exits 5 when the key/section does not exist; treat as success.
	if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 5 {
		return nil
	}
	return ClassifyError("config unset gbmBase", err, string(output))
}

// GetGbmBase returns the recorded base branch for the given worktree's current
// branch. Returns empty string if no base was recorded (not an error).
func (s *Service) GetGbmBase(worktreePath, branchName string) (string, error) {
	if worktreePath == "" {
		return "", ErrWorktreePathEmpty
	}
	if branchName == "" {
		return "", ErrBranchNameEmpty
	}

	key := "branch." + branchName + ".gbmBase"
	cmd := exec.Command("git", "-C", worktreePath, "config", "--local", "--get", key)
	output, err := cmd.Output()
	if err != nil {
		// `git config --get` exits 1 when the key is missing; treat as "no base recorded".
		return "", nil
	}

	return strings.TrimSpace(string(output)), nil
}

// CountAheadBehind returns how many commits HEAD is ahead/behind of ref.
// ref must be a resolvable revision (branch, tag, SHA, etc.).
func (s *Service) CountAheadBehind(worktreePath, ref string) (ahead, behind int, err error) {
	if worktreePath == "" {
		return 0, 0, ErrWorktreePathEmpty
	}

	cmd := exec.Command("git", "-C", worktreePath, "rev-list", "--left-right", "--count", ref+"...HEAD")
	output, err := cmd.Output()
	if err != nil {
		return 0, 0, ClassifyError("rev-list count", err, "")
	}

	counts := strings.Fields(strings.TrimSpace(string(output)))
	if len(counts) != 2 {
		return 0, 0, nil
	}
	if _, err := fmt.Sscanf(counts[0], "%d", &behind); err != nil {
		behind = 0
	}
	if _, err := fmt.Sscanf(counts[1], "%d", &ahead); err != nil {
		ahead = 0
	}
	return ahead, behind, nil
}

// GetHeadCommit returns the short SHA of HEAD in the given worktree.
func (s *Service) GetHeadCommit(worktreePath string) (string, error) {
	if worktreePath == "" {
		return "", ErrWorktreePathEmpty
	}
	cmd := exec.Command("git", "-C", worktreePath, "rev-parse", "--short", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", ClassifyError("rev-parse HEAD", err, "")
	}
	return strings.TrimSpace(string(output)), nil
}

// IsWorktreeClean reports whether the worktree has no uncommitted changes
// (no staged, unstaged, or untracked changes per `git status --porcelain`).
func (s *Service) IsWorktreeClean(worktreePath string) (bool, error) {
	if worktreePath == "" {
		return false, ErrWorktreePathEmpty
	}
	cmd := exec.Command("git", "-C", worktreePath, "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return false, ClassifyError("status porcelain", err, "")
	}
	return strings.TrimSpace(string(output)) == "", nil
}

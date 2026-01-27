package git

import (
	"fmt"
	"gbm/internal/utils"
	"os/exec"
	"path/filepath"
)

// Init creates a new git repository with worktree structure:
// - <name>/.git (bare repository)
// - <name>/worktrees/<defaultBranch>/ (main worktree)
// - <name>/.gbm/config.yaml (configuration file).
func (s *Service) Init(name, defaultBranchName string, dryRun bool) error {
	if name == "" {
		name = "."
	}
	if defaultBranchName == "" {
		defaultBranchName = "main"
	}

	absPath, err := filepath.Abs(name)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	gitDir := filepath.Join(absPath, ".git")

	// Initialize bare repository
	if err := initBareRepo(absPath, gitDir, dryRun); err != nil {
		return err
	}
	if err := configureDefaultBranch(gitDir, defaultBranchName, dryRun); err != nil {
		return err
	}

	// Create worktree structure
	if err := initWorktreeStructure(absPath, gitDir, defaultBranchName, dryRun); err != nil {
		return err
	}

	return createGbmConfig(absPath, defaultBranchName, dryRun)
}

// initBareRepo initializes a bare git repository.
func initBareRepo(absPath, gitDir string, dryRun bool) error {
	err := utils.MkdirAll(absPath, dryRun)
	if err != nil {
		return err
	}
	err = utils.MkdirAll(gitDir, dryRun)
	if err != nil {
		return err
	}
	cmd := exec.Command("git", "init", "--bare", gitDir)
	if dryRun {
		printDryRun(cmd)
		return nil
	}
	if output, err := cmd.CombinedOutput(); err != nil {
		return ClassifyError("init bare", err, string(output))
	}
	return nil
}

// configureDefaultBranch sets the default branch name in git config.
func configureDefaultBranch(gitDir, branchName string, dryRun bool) error {
	cmd := exec.Command("git", "--git-dir", gitDir, "config", "init.defaultBranch", branchName)
	if dryRun {
		printDryRun(cmd)
		return nil
	}
	if output, err := cmd.CombinedOutput(); err != nil {
		return ClassifyError("config init.defaultBranch", err, string(output))
	}
	return nil
}

// initWorktreeStructure creates the worktrees directory, main worktree, and initial commit.
func initWorktreeStructure(absPath, gitDir, branchName string, dryRun bool) error {
	worktreesDir := filepath.Join(absPath, "worktrees")
	err := utils.MkdirAll(worktreesDir, dryRun)
	if err != nil {
		return err
	}
	mainWorktreePath := filepath.Join(worktreesDir, branchName)
	cmd := exec.Command("git", "--git-dir", gitDir, "worktree", "add", mainWorktreePath, "-b", branchName)
	if dryRun {
		printDryRun(cmd)
	} else if output, err := cmd.CombinedOutput(); err != nil {
		return ClassifyError("worktree add", err, string(output))
	}
	cmd = exec.Command("git", "-C", mainWorktreePath, "commit", "--allow-empty", "-m", "Initial commit")
	if dryRun {
		printDryRun(cmd)
		return nil
	}
	if output, err := cmd.CombinedOutput(); err != nil {
		return ClassifyError("commit", err, string(output))
	}
	return nil
}

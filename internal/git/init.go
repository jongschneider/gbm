package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Init creates a new git repository with worktree structure:
// - <name>/.git (bare repository)
// - <name>/worktrees/<defaultBranch>/ (main worktree)
// - <name>/.gbm/config.yaml (configuration file)
func (s *Service) Init(name, defaultBranchName string, dryRun bool) error {
	// Use current directory if name is empty
	if name == "" {
		name = "."
	}

	// Use "main" as default branch if not specified
	if defaultBranchName == "" {
		defaultBranchName = "main"
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(name)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	// Create root directory if it doesn't exist
	if dryRun {
		fmt.Printf("[DRY RUN] mkdir -p %s\n", absPath)
	} else {
		if err := os.MkdirAll(absPath, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", absPath, err)
		}
	}

	// Create .git directory for bare repository
	gitDir := filepath.Join(absPath, ".git")
	if dryRun {
		fmt.Printf("[DRY RUN] mkdir -p %s\n", gitDir)
	} else {
		if err := os.MkdirAll(gitDir, 0755); err != nil {
			return fmt.Errorf("failed to create .git directory: %w", err)
		}
	}

	// Initialize bare repository
	cmd := exec.Command("git", "init", "--bare", gitDir)
	if output, err := s.runCommand(cmd, dryRun); err != nil {
		return fmt.Errorf("failed to initialize bare repository: %w\nOutput: %s", err, output)
	}

	// Set the default branch name
	cmd = exec.Command("git", "--git-dir", gitDir, "config", "init.defaultBranch", defaultBranchName)
	if output, err := s.runCommand(cmd, dryRun); err != nil {
		return fmt.Errorf("failed to set default branch name: %w\nOutput: %s", err, output)
	}

	// Create worktrees directory structure
	worktreesDir := filepath.Join(absPath, "worktrees")
	if dryRun {
		fmt.Printf("[DRY RUN] mkdir -p %s\n", worktreesDir)
	} else {
		if err := os.MkdirAll(worktreesDir, 0755); err != nil {
			return fmt.Errorf("failed to create worktrees directory: %w", err)
		}
	}

	// Create main worktree path
	mainWorktreePath := filepath.Join(worktreesDir, defaultBranchName)

	// Add worktree for the default branch
	cmd = exec.Command("git", "--git-dir", gitDir, "worktree", "add", mainWorktreePath, "-b", defaultBranchName)
	if output, err := s.runCommand(cmd, dryRun); err != nil {
		return fmt.Errorf("failed to create main worktree: %w\nOutput: %s", err, output)
	}

	// Create initial empty commit in the worktree
	cmd = exec.Command("git", "-C", mainWorktreePath, "commit", "--allow-empty", "-m", "Initial commit")
	if output, err := s.runCommand(cmd, dryRun); err != nil {
		return fmt.Errorf("failed to create initial commit: %w\nOutput: %s", err, output)
	}

	// Create .gbm directory and config.yaml
	gbmDir := filepath.Join(absPath, ".gbm")
	if dryRun {
		fmt.Printf("[DRY RUN] mkdir -p %s\n", gbmDir)
	} else {
		if err := os.MkdirAll(gbmDir, 0755); err != nil {
			return fmt.Errorf("failed to create .gbm directory: %w", err)
		}
	}

	configPath := filepath.Join(gbmDir, "config.yaml")
	configContent := fmt.Sprintf(`# Git Branch Manager Configuration
default_branch: %s
worktrees_dir: worktrees
`, defaultBranchName)

	if dryRun {
		fmt.Printf("[DRY RUN] write file %s:\n%s\n", configPath, configContent)
	} else {
		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			return fmt.Errorf("failed to create config.yaml: %w", err)
		}
	}

	return nil
}

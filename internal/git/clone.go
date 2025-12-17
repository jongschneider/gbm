package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gbm/internal/utils"
)

// Clone clones a remote git repository with worktree structure:
// - <name>/.git (bare repository)
// - <name>/worktrees/<defaultBranch>/ (main worktree)
// - <name>/.gbm/config.yaml (configuration file)
func (s *Service) Clone(repoURL, name string, dryRun bool) error {
	// Extract repository name from URL if name is not provided
	if name == "" {
		name = extractRepoName(repoURL)
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(name)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	// Create root directory if it doesn't exist
	if err := utils.MkdirAll(absPath, dryRun); err != nil {
		return err
	}

	// Create .git directory path
	gitDir := filepath.Join(absPath, ".git")

	// Clone bare repository to .git
	cmd := exec.Command("git", "clone", "--bare", repoURL, gitDir)
	if output, err := s.runCommand(cmd, dryRun); err != nil {
		// Clean up the directory if cloning fails
		if !dryRun {
			_ = os.RemoveAll(absPath)
		}
		return fmt.Errorf("failed to clone bare repository: %w\nOutput: %s", err, output)
	}

	// Set remote origin fetch configuration
	cmd = exec.Command("git", "--git-dir", gitDir, "config", "remote.origin.fetch", "+refs/heads/*:refs/remotes/origin/*")
	if output, err := s.runCommand(cmd, dryRun); err != nil {
		return fmt.Errorf("failed to configure remote origin fetch: %w\nOutput: %s", err, output)
	}

	// Fetch all branches from remote
	cmd = exec.Command("git", "--git-dir", gitDir, "fetch", "origin")
	if output, err := s.runCommand(cmd, dryRun); err != nil {
		return fmt.Errorf("failed to fetch from origin: %w\nOutput: %s", err, output)
	}

	// Get the default branch from remote
	defaultBranch, err := s.getDefaultBranch(gitDir, dryRun)
	if err != nil {
		return fmt.Errorf("failed to determine default branch: %w", err)
	}

	// Create worktrees directory structure
	worktreesDir := filepath.Join(absPath, "worktrees")
	if err := utils.MkdirAll(worktreesDir, dryRun); err != nil {
		return err
	}

	// Create main worktree path
	mainWorktreePath := filepath.Join(worktreesDir, defaultBranch)

	// Add worktree for the default branch
	cmd = exec.Command("git", "--git-dir", gitDir, "worktree", "add", mainWorktreePath, defaultBranch)
	if output, err := s.runCommand(cmd, dryRun); err != nil {
		return fmt.Errorf("failed to create main worktree: %w\nOutput: %s", err, output)
	}

	// Create .gbm directory and config.yaml
	gbmDir := filepath.Join(absPath, ".gbm")
	if err := utils.MkdirAll(gbmDir, dryRun); err != nil {
		return err
	}

	configPath := filepath.Join(gbmDir, "config.yaml")
	configContent := fmt.Sprintf(`# Git Branch Manager Configuration
default_branch: %s
worktrees_dir: worktrees

# File copying rules - automatically copy files from source worktrees to new worktrees
# Useful for configuration files, .env files, etc.
# file_copy:
#   rules:
#     - source_worktree: %s
#       files:
#         - .env
#         - config/
#         - .vscode/settings.json
`, defaultBranch, defaultBranch)

	if dryRun {
		fmt.Printf("[DRY RUN] write file %s:\n%s\n", configPath, configContent)
	} else {
		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			return fmt.Errorf("failed to create config.yaml: %w", err)
		}
	}

	return nil
}

// extractRepoName extracts the repository name from a Git URL
func extractRepoName(repoURL string) string {
	// Remove .git suffix if present
	url := strings.TrimSuffix(repoURL, ".git")

	// Extract the last part of the URL (repository name)
	parts := strings.Split(url, "/")
	if len(parts) > 0 && parts[len(parts)-1] != "" {
		return parts[len(parts)-1]
	}

	return "repository"
}

// getDefaultBranch determines the default branch from the remote repository
func (s *Service) getDefaultBranch(gitDir string, dryRun bool) (string, error) {
	// First, try to set the remote HEAD reference
	cmd := exec.Command("git", "--git-dir", gitDir, "remote", "set-head", "origin", "-a")
	if _, err := s.runCommand(cmd, dryRun); err != nil {
		// If that fails, try to get the remote HEAD manually using ls-remote
		cmd = exec.Command("git", "--git-dir", gitDir, "ls-remote", "--symref", "origin", "HEAD")
		output, err := s.runCommand(cmd, dryRun)
		if err != nil {
			return "", fmt.Errorf("failed to get default branch: %w", err)
		}

		// In dry run mode, return a sensible default
		if dryRun {
			return "", nil
		}

		// Parse the output to extract branch name
		// Output format: ref: refs/heads/main    HEAD
		lines := strings.SplitSeq(string(output), "\n")
		for line := range lines {
			if strings.HasPrefix(line, "ref: refs/heads/") {
				parts := strings.Split(line, "\t")
				if len(parts) > 0 {
					refPath := parts[0]
					branchName := strings.TrimPrefix(refPath, "ref: refs/heads/")
					return branchName, nil
				}
			}
		}

		return "", fmt.Errorf("could not determine default branch from remote")
	}

	// Now try to get the symbolic ref
	cmd = exec.Command("git", "--git-dir", gitDir, "symbolic-ref", "refs/remotes/origin/HEAD")
	output, err := s.runCommand(cmd, dryRun)
	if err != nil {
		return "", fmt.Errorf("failed to get default branch: %w", err)
	}

	// In dry run mode, return a sensible default
	if dryRun {
		return "", nil
	}

	// Parse the output to extract branch name
	// Output format: refs/remotes/origin/main
	refPath := strings.TrimSpace(string(output))
	parts := strings.Split(refPath, "/")
	if len(parts) < 4 {
		return "", fmt.Errorf("unexpected symbolic-ref output format: %s", refPath)
	}

	return parts[len(parts)-1], nil
}

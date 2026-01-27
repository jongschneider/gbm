package git

import (
	"fmt"
	"gbm/internal/utils"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Clone clones a remote git repository with worktree structure:
// - <name>/.git (bare repository)
// - <name>/worktrees/<defaultBranch>/ (main worktree)
// - <name>/.gbm/config.yaml (configuration file).
func (s *Service) Clone(repoURL, name string, dryRun bool) error {
	if name == "" {
		name = extractRepoName(repoURL)
	}

	absPath, err := filepath.Abs(name)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	if err := utils.MkdirAll(absPath, dryRun); err != nil {
		return err
	}

	gitDir := filepath.Join(absPath, ".git")

	// Clone and configure bare repository
	if err := cloneBareRepo(repoURL, gitDir, absPath, dryRun); err != nil {
		return err
	}
	if err := configureRemoteFetch(gitDir, dryRun); err != nil {
		return err
	}
	if err := fetchFromOrigin(gitDir, dryRun); err != nil {
		return err
	}

	// Get the default branch and create worktree structure
	defaultBranch, err := s.getDefaultBranch(gitDir, dryRun)
	if err != nil {
		return fmt.Errorf("failed to determine default branch: %w", err)
	}

	if err := setupWorktreeStructure(absPath, gitDir, defaultBranch, dryRun); err != nil {
		return err
	}

	return createGbmConfig(absPath, defaultBranch, dryRun)
}

// cloneBareRepo clones a bare repository to the specified directory.
func cloneBareRepo(repoURL, gitDir, absPath string, dryRun bool) error {
	cmd := exec.Command("git", "clone", "--bare", repoURL, gitDir)
	if dryRun {
		printDryRun(cmd)
		return nil
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		_ = os.RemoveAll(absPath)
		return fmt.Errorf("failed to clone bare repository: %w", err)
	}
	return nil
}

// configureRemoteFetch sets the remote origin fetch configuration.
func configureRemoteFetch(gitDir string, dryRun bool) error {
	cmd := exec.Command("git", "--git-dir", gitDir, "config", "remote.origin.fetch", "+refs/heads/*:refs/remotes/origin/*")
	if dryRun {
		printDryRun(cmd)
		return nil
	}
	if output, err := cmd.CombinedOutput(); err != nil {
		return ClassifyError("config remote.origin.fetch", err, string(output))
	}
	return nil
}

// fetchFromOrigin fetches all branches from the origin remote.
func fetchFromOrigin(gitDir string, dryRun bool) error {
	cmd := exec.Command("git", "--git-dir", gitDir, "fetch", "origin")
	if dryRun {
		printDryRun(cmd)
		return nil
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to fetch from origin: %w", err)
	}
	return nil
}

// setupWorktreeStructure creates the worktrees directory and main worktree.
func setupWorktreeStructure(absPath, gitDir, defaultBranch string, dryRun bool) error {
	worktreesDir := filepath.Join(absPath, "worktrees")
	err := utils.MkdirAll(worktreesDir, dryRun)
	if err != nil {
		return err
	}

	mainWorktreePath := filepath.Join(worktreesDir, defaultBranch)
	cmd := exec.Command("git", "--git-dir", gitDir, "worktree", "add", mainWorktreePath, defaultBranch)
	if dryRun {
		printDryRun(cmd)
		return nil
	}
	if output, err := cmd.CombinedOutput(); err != nil {
		return ClassifyError("worktree add", err, string(output))
	}
	return nil
}

// createGbmConfig creates the .gbm directory and config.yaml file.
func createGbmConfig(absPath, defaultBranch string, dryRun bool) error {
	gbmDir := filepath.Join(absPath, ".gbm")
	err := utils.MkdirAll(gbmDir, dryRun)
	if err != nil {
		return err
	}

	configPath := filepath.Join(gbmDir, "config.yaml")
	configContent := generateConfigContent(defaultBranch)

	if dryRun {
		fmt.Printf("[DRY RUN] write file %s:\n%s\n", configPath, configContent)
		return nil
	}

	err = os.WriteFile(configPath, []byte(configContent), 0o644)
	if err != nil {
		return fmt.Errorf("failed to create config.yaml: %w", err)
	}
	return nil
}

// generateConfigContent generates the default config.yaml content.
func generateConfigContent(defaultBranch string) string {
	return fmt.Sprintf(`# Git Branch Manager Configuration
default_branch: %s
worktrees_dir: worktrees

# JIRA Integration (optional)
jira:
  # Attachment download settings
  attachments:
    enabled: true                    # Enable attachment downloads
    max_size_mb: 50                  # Skip files larger than this (MB)
    directory: ".jira/attachments"   # Directory relative to worktree root
    download_timeout_seconds: 30     # HTTP download timeout
    retry_attempts: 3                # Number of retry attempts for failed downloads
    retry_backoff_ms: 1000          # Initial retry backoff in milliseconds

  # Markdown generation settings
  markdown:
    include_comments: true           # Include all comments in markdown
    include_attachments: true        # Include attachments section
    use_relative_links: true         # Use relative paths for attachment links
    filename_pattern: "{key}.md"     # Output filename pattern

  # Issue list filters (optional)
  # Configure filters for issue fetching when browsing in TUI
  # filters:
  #   # Filter by multiple statuses
  #   status:
  #     - "In Dev."
  #     - "Open"
  #     - "To Do"
  #   # Filter by priority, type, labels
  #   priority: "High"
  #   type: "Bug"
  #   labels:
  #     - "backend"
  #   # Additional filters: component, reporter, order_by, custom_args
  #   # Run 'jira issue list --help' for all available options

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
}

// extractRepoName extracts the repository name from a Git URL.
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

// getDefaultBranch determines the default branch from the remote repository.
func (s *Service) getDefaultBranch(gitDir string, dryRun bool) (string, error) {
	// First, try to set the remote HEAD reference
	cmd := exec.Command("git", "--git-dir", gitDir, "remote", "set-head", "origin", "-a")

	if dryRun {
		printDryRun(cmd)
		return "main", nil // Return sensible default for dry-run
	}

	if _, err := cmd.CombinedOutput(); err != nil {
		// If that fails, try to get the remote HEAD manually using ls-remote
		cmd = exec.Command("git", "--git-dir", gitDir, "ls-remote", "--symref", "origin", "HEAD")
		output, err := cmd.CombinedOutput()
		if err != nil {
			return "", ClassifyError("ls-remote", err, string(output))
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

		return "", ErrCouldNotDetermineDefaultBranch
	}

	// Now try to get the symbolic ref
	cmd = exec.Command("git", "--git-dir", gitDir, "symbolic-ref", "refs/remotes/origin/HEAD")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", ClassifyError("symbolic-ref", err, string(output))
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

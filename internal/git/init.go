package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"gbm/internal/utils"
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
	if err := utils.MkdirAll(absPath, dryRun); err != nil {
		return err
	}

	// Create .git directory for bare repository
	gitDir := filepath.Join(absPath, ".git")
	if err := utils.MkdirAll(gitDir, dryRun); err != nil {
		return err
	}

	// Initialize bare repository
	cmd := exec.Command("git", "init", "--bare", gitDir)
	if dryRun {
		printDryRun(cmd)
	} else {
		if output, err := cmd.CombinedOutput(); err != nil {
			return ClassifyError("init bare", err, string(output))
		}
	}

	// Set the default branch name
	cmd = exec.Command("git", "--git-dir", gitDir, "config", "init.defaultBranch", defaultBranchName)
	if dryRun {
		printDryRun(cmd)
	} else {
		if output, err := cmd.CombinedOutput(); err != nil {
			return ClassifyError("config init.defaultBranch", err, string(output))
		}
	}

	// Create worktrees directory structure
	worktreesDir := filepath.Join(absPath, "worktrees")
	if err := utils.MkdirAll(worktreesDir, dryRun); err != nil {
		return err
	}

	// Create main worktree path
	mainWorktreePath := filepath.Join(worktreesDir, defaultBranchName)

	// Add worktree for the default branch
	cmd = exec.Command("git", "--git-dir", gitDir, "worktree", "add", mainWorktreePath, "-b", defaultBranchName)
	if dryRun {
		printDryRun(cmd)
	} else {
		if output, err := cmd.CombinedOutput(); err != nil {
			return ClassifyError("worktree add", err, string(output))
		}
	}

	// Create initial empty commit in the worktree
	cmd = exec.Command("git", "-C", mainWorktreePath, "commit", "--allow-empty", "-m", "Initial commit")
	if dryRun {
		printDryRun(cmd)
	} else {
		if output, err := cmd.CombinedOutput(); err != nil {
			return ClassifyError("commit", err, string(output))
		}
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
`, defaultBranchName, defaultBranchName)

	if dryRun {
		fmt.Printf("[DRY RUN] write file %s:\n%s\n", configPath, configContent)
	} else {
		if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
			return fmt.Errorf("failed to create config.yaml: %w", err)
		}
	}

	return nil
}

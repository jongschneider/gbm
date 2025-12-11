package git

import (
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"strings"
)

type Service struct{}

func NewService() *Service {
	if _, err := exec.LookPath("git"); err != nil {
		log.Fatal("git command not found in PATH - please install git")
	}
	return &Service{}
}

// FindGitRoot finds the root directory of the git repository.
// This works correctly whether you're in a worktree, the main repo, or a subdirectory.
//
// Our repository structure (created by Init/Clone):
//
//	repo/
//	  .git/           # bare repository
//	  worktrees/
//	    main/         # worktree
//	    feature/      # worktree
//
// How this function handles different scenarios:
//
// Scenario 1: Running from INSIDE a worktree (repo/worktrees/main/)
//   - git rev-parse --git-dir returns: /path/to/repo/.git/worktrees/main
//   - Contains /.git/worktrees/ → YES
//   - Returns early, NEVER checks --is-bare-repository
//
// Scenario 2: Running from repo root (repo/)
//   - git rev-parse --git-dir returns: .git or /path/to/repo/.git
//   - Contains /.git/worktrees/ → NO
//   - Proceeds to --is-bare-repository check
//   - Returns true (because .git is bare)
//   - Uses bare repository logic
//
// Scenario 3: Running from worktrees directory (repo/worktrees/)
//   - Git searches upward and finds repo/.git
//   - git rev-parse --git-dir returns: ../.git or /path/to/repo/.git
//   - Contains /.git/worktrees/ → NO (path points to .git, not .git/worktrees/something)
//   - Proceeds to --is-bare-repository check
//   - Returns true
//   - Uses bare repository logic
//
// The --show-toplevel fallback is for regular (non-bare) repositories, which our
// Init/Clone commands never create, but is included for edge cases.
func (s *Service) FindGitRoot(startPath string) (string, error) {
	// Get the git directory path
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = startPath
	gitDirOutput, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not in a git repository: %w", err)
	}

	gitDir := strings.TrimSpace(string(gitDirOutput))

	// If we're in a worktree, the git-dir will contain "/.git/worktrees/"
	// Example: /path/to/repo/.git/worktrees/main
	// We need to extract /path/to/repo
	if strings.Contains(gitDir, "/.git/worktrees/") {
		parts := strings.Split(gitDir, "/.git/worktrees/")
		if len(parts) >= 2 {
			return parts[0], nil
		}
	}

	// Check if this is a bare repository
	cmd = exec.Command("git", "rev-parse", "--is-bare-repository")
	cmd.Dir = startPath
	output, err := cmd.Output()
	if err == nil && strings.TrimSpace(string(output)) == "true" {
		// For bare repositories, the git directory is the repository root
		if filepath.IsAbs(gitDir) {
			// gitDir is something like /path/to/repo/.git
			// We want /path/to/repo
			return filepath.Dir(gitDir), nil
		}
		// gitDir is relative (e.g., ".git"), so repository root is startPath
		return startPath, nil
	}

	// For regular repositories, use --show-toplevel
	cmd = exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = startPath
	output, err = cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to find repository root: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// GetCurrentWorktree returns the name of the current worktree if we're in one.
// Returns an error if not in a worktree or if git commands fail.
//
// Example: If in /path/to/repo/worktrees/feature-x, returns "feature-x".
func (s *Service) GetCurrentWorktree() (string, error) {
	// Get the git directory path
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	gitDirOutput, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not in a git repository: %w", err)
	}

	gitDir := strings.TrimSpace(string(gitDirOutput))

	// If we're in a worktree, the git-dir will contain "/.git/worktrees/"
	// Example: /path/to/repo/.git/worktrees/feature-x
	const worktreesSegment = "/.git/worktrees/"
	if !strings.Contains(gitDir, worktreesSegment) {
		return "", fmt.Errorf("not in a worktree")
	}

	parts := strings.Split(gitDir, worktreesSegment)
	if len(parts) < 2 || parts[1] == "" {
		return "", fmt.Errorf("not in a worktree")
	}

	// Extract the worktree name (last component after /.git/worktrees/)
	worktreeName := parts[1]
	return worktreeName, nil
}

// runCommand executes a command or prints it if in dry-run mode
func (s *Service) runCommand(cmd *exec.Cmd, dryRun bool) ([]byte, error) {
	cmdStr := formatCommand(cmd)

	if dryRun {
		fmt.Printf("[DRY RUN] %s\n", cmdStr)
		return nil, nil
	}

	return cmd.CombinedOutput()
}

// formatCommand formats a command for display
func formatCommand(cmd *exec.Cmd) string {
	parts := []string{cmd.Path}
	parts = append(parts, cmd.Args[1:]...)

	// Add working directory if set
	if cmd.Dir != "" {
		return fmt.Sprintf("(cd %s && %s)", cmd.Dir, strings.Join(parts, " "))
	}

	// Add git-dir if set in env
	for _, env := range cmd.Env {
		if after, ok := strings.CutPrefix(env, "GIT_DIR="); ok {
			return fmt.Sprintf("GIT_DIR=%s %s", after, strings.Join(parts, " "))
		}
	}

	return strings.Join(parts, " ")
}

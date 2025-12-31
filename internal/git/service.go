package git

import (
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"strings"

	"gbm/internal/utils"
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

// GetCurrentWorktree returns the Worktree we're currently in.
// Returns an error if not in a worktree or if git commands fail.
//
// Example: If in /path/to/repo/worktrees/feature-x, returns the Worktree for feature-x.
func (s *Service) GetCurrentWorktree() (*Worktree, error) {
	// Get the git directory path
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	gitDirOutput, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("not in a git repository: %w", err)
	}

	gitDir := strings.TrimSpace(string(gitDirOutput))

	// If we're in a worktree, the git-dir will contain "/.git/worktrees/"
	// Example: /path/to/repo/.git/worktrees/feature-x
	const worktreesSegment = "/.git/worktrees/"
	if !strings.Contains(gitDir, worktreesSegment) {
		return nil, ErrNotInWorktree
	}

	parts := strings.Split(gitDir, worktreesSegment)
	if len(parts) < 2 || parts[1] == "" {
		return nil, ErrNotInWorktree
	}

	// Extract the worktree name (last component after /.git/worktrees/)
	worktreeName := parts[1]

	// List all worktrees to find the current one
	worktrees, err := s.ListWorktrees(false)
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	// Find the worktree by name
	for _, wt := range worktrees {
		if wt.Name == worktreeName {
			return &wt, nil
		}
	}

	return nil, fmt.Errorf("current worktree '%s' not found in worktree list", worktreeName)
}

// BranchStatus represents the sync status of a branch with its remote
type BranchStatus struct {
	Ahead    int
	Behind   int
	UpToDate bool
	NoRemote bool
}

// GetBranchStatus returns the sync status of a worktree's branch
// comparing it to its remote tracking branch. Does NOT fetch - assumes
// local tracking branch is recent. Use FetchAllWorktrees first if you need
// fresh remote info.
func (s *Service) GetBranchStatus(worktreePath string) (*BranchStatus, error) {
	status := &BranchStatus{}

	// Get current branch name
	cmd := exec.Command("git", "-C", worktreePath, "rev-parse", "--abbrev-ref", "HEAD")
	branchOutput, err := cmd.Output()
	if err != nil {
		return status, fmt.Errorf("failed to get branch: %w", err)
	}
	branch := strings.TrimSpace(string(branchOutput))

	// Check if branch has a remote tracking branch
	cmd = exec.Command("git", "-C", worktreePath, "rev-parse", "--abbrev-ref", fmt.Sprintf("%s@{u}", branch))
	remoteOutput, err := cmd.Output()
	if err != nil {
		status.NoRemote = true
		status.UpToDate = true // No remote = nothing to sync
		return status, nil
	}

	remoteBranch := strings.TrimSpace(string(remoteOutput))
	if remoteBranch == "" || strings.Contains(remoteBranch, "fatal") {
		status.NoRemote = true
		status.UpToDate = true
		return status, nil
	}

	// Count commits ahead and behind
	cmd = exec.Command("git", "-C", worktreePath, "rev-list", "--left-right", "--count", fmt.Sprintf("%s...HEAD", remoteBranch))
	output, err := cmd.Output()
	if err != nil {
		return status, fmt.Errorf("failed to count commits: %w", err)
	}

	counts := strings.Fields(strings.TrimSpace(string(output)))
	if len(counts) == 2 {
		// Parse commit counts, falling back to 0 on error
		if _, err := fmt.Sscanf(counts[0], "%d", &status.Behind); err != nil {
			status.Behind = 0
		}
		if _, err := fmt.Sscanf(counts[1], "%d", &status.Ahead); err != nil {
			status.Ahead = 0
		}
	}

	status.UpToDate = status.Ahead == 0 && status.Behind == 0

	return status, nil
}

// runCommand executes a command or prints it if in dry-run mode
func (s *Service) runCommand(cmd *exec.Cmd, dryRun bool) ([]byte, error) {
	cmdStr := utils.FormatCommand(cmd)

	if dryRun {
		fmt.Printf("[DRY RUN] %s\n", cmdStr)
		return nil, nil
	}

	return cmd.CombinedOutput()
}

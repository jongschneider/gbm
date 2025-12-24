package git

import (
	"cmp"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// Worktree represents a git worktree with its metadata
type Worktree struct {
	Name   string // Worktree name (e.g., "feature-x")
	Path   string // Absolute path to the worktree
	Branch string // Branch name (e.g., "feature/ABC-123")
	Commit string // Commit hash (short form)
	IsBare bool   // True if this is the bare repository worktree
}

// parseWorktrees parses the output of 'git worktree list' into Worktree structs
func parseWorktrees(output string) []Worktree {
	var worktrees []Worktree

	// Regex to parse:
	//   /path/to/worktree  abcd1234 [branch-name]
	//   /path/to/repo (bare)  <- Note: bare repos may not have commit hash
	re := regexp.MustCompile(`^(\S+)\s+(?:([a-f0-9]+)\s+)?(?:\[(.*?)\]|\((.*?)\))`)

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		matches := re.FindStringSubmatch(line)
		if len(matches) < 2 {
			continue
		}

		path := matches[1]
		commit := ""
		if len(matches) > 2 && matches[2] != "" {
			commit = matches[2]
		}
		branch := ""
		isBare := false

		// Check if it's a branch [branch-name] or (bare)/(detached)
		if len(matches) > 3 && matches[3] != "" {
			branch = matches[3]
		} else if len(matches) > 4 && matches[4] == "bare" {
			isBare = true
		}

		worktree := Worktree{
			Name:   filepath.Base(path),
			Path:   path,
			Branch: branch,
			Commit: commit,
			IsBare: isBare,
		}

		worktrees = append(worktrees, worktree)
	}

	return worktrees
}

// AddWorktree creates a new git worktree in the specified directory
func (s *Service) AddWorktree(worktreesDir, worktreeName, branchName string, createBranch bool, baseBranch string, dryRun bool) (*Worktree, error) {
	if worktreesDir == "" {
		return nil, ErrWorktreesDirectoryEmpty
	}
	if worktreeName == "" {
		return nil, ErrWorktreeNameEmpty
	}
	if branchName == "" {
		return nil, ErrBranchNameEmpty
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
		return nil, fmt.Errorf("failed to add worktree: %w\nOutput: %s", err, string(output))
	}

	// If dry-run, return a mock Worktree
	if dryRun {
		return &Worktree{
			Name:   worktreeName,
			Path:   worktreePath,
			Branch: branchName,
			Commit: "",
			IsBare: false,
		}, nil
	}

	// Get the newly created worktree info
	worktrees, err := s.ListWorktrees(false)
	if err != nil {
		return nil, fmt.Errorf("worktree created but failed to get info: %w", err)
	}

	// Resolve canonical path for comparison (handles symlinks like /tmp -> /private/tmp)
	canonicalPath, err := filepath.EvalSymlinks(worktreePath)
	if err != nil {
		canonicalPath = worktreePath // Fallback if EvalSymlinks fails
	}

	// Find the worktree we just created
	for _, wt := range worktrees {
		wtCanonicalPath, err := filepath.EvalSymlinks(wt.Path)
		if err != nil {
			wtCanonicalPath = wt.Path
		}
		if wtCanonicalPath == canonicalPath {
			return &wt, nil
		}
	}

	// If we can't find it, something went wrong
	return nil, fmt.Errorf("worktree created at %s but not found in worktree list", worktreePath)
}

// ListWorktrees lists all git worktrees in the repository
func (s *Service) ListWorktrees(dryRun bool) ([]Worktree, error) {
	args := []string{"worktree", "list"}

	cmd := exec.Command("git", args...)
	output, err := s.runCommand(cmd, dryRun)
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w\nOutput: %s", err, string(output))
	}

	// In dry-run mode, git command prints but doesn't execute, so we can't parse output
	if dryRun {
		return nil, nil
	}

	return parseWorktrees(string(output)), nil
}

// GetWorktreeBranch returns the branch associated with a worktree
func (s *Service) GetWorktreeBranch(worktreePath string) (string, error) {
	if worktreePath == "" {
		return "", ErrWorktreePathEmpty
	}

	// Use git -C to run command in the worktree directory
	cmd := exec.Command("git", "-C", worktreePath, "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get branch for worktree: %w", err)
	}

	// Trim whitespace and newlines from the output
	branchName := strings.TrimSpace(string(output))

	return branchName, nil
}

// Fetch fetches from the remote repository
func (s *Service) Fetch(dryRun bool) error {
	args := []string{"fetch", "--all"}

	cmd := exec.Command("git", args...)
	output, err := s.runCommand(cmd, dryRun)
	if err != nil {
		return fmt.Errorf("failed to fetch: %w\nOutput: %s", err, string(output))
	}

	return nil
}

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
	output, err := s.runCommand(cmd, dryRun)
	if err != nil {
		return fmt.Errorf("failed to delete branch: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// MoveWorktree moves a git worktree to a new location
func (s *Service) MoveWorktree(oldName, newName string, dryRun bool) error {
	if oldName == "" {
		return ErrOldWorktreeNameEmpty
	}
	if newName == "" {
		return ErrNewWorktreeNameEmpty
	}

	// List all worktrees to find the source
	worktrees, err := s.ListWorktrees(false)
	if err != nil {
		return fmt.Errorf("failed to list worktrees: %w", err)
	}

	var oldWorktree *Worktree
	for _, wt := range worktrees {
		if wt.Name == oldName {
			oldWorktree = &wt
			break
		}
	}

	if oldWorktree == nil {
		return fmt.Errorf("worktree '%s' not found", oldName)
	}

	// Calculate new path (same parent directory, different name)
	oldPath := oldWorktree.Path
	parentDir := filepath.Dir(oldPath)
	newPath := filepath.Join(parentDir, newName)

	args := []string{"worktree", "move", oldPath, newPath}

	cmd := exec.Command("git", args...)
	output, err := s.runCommand(cmd, dryRun)
	if err != nil {
		return fmt.Errorf("failed to move worktree: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// RemoveWorktree removes a git worktree by name and returns the removed worktree info
func (s *Service) RemoveWorktree(worktreeName string, force bool, dryRun bool) (*Worktree, error) {
	if worktreeName == "" {
		return nil, ErrWorktreeNameEmpty
	}

	// List all worktrees to find the one to remove
	worktrees, err := s.ListWorktrees(false)
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	// Find the worktree by name
	var targetWorktree *Worktree
	for _, wt := range worktrees {
		if wt.Name == worktreeName {
			targetWorktree = &wt
			break
		}
	}

	if targetWorktree == nil {
		return nil, fmt.Errorf("worktree '%s' not found", worktreeName)
	}

	// Use the full path from the worktree list
	args := []string{"worktree", "remove", targetWorktree.Path}

	if force {
		args = []string{"worktree", "remove", "--force", targetWorktree.Path}
	}

	cmd := exec.Command("git", args...)
	output, err := s.runCommand(cmd, dryRun)
	if err != nil {
		return nil, fmt.Errorf("failed to remove worktree: %w\nOutput: %s", err, string(output))
	}

	return targetWorktree, nil
}

// ListBranches returns all local and remote branches
func (s *Service) ListBranches(dryRun bool) ([]string, error) {
	cmd := exec.Command("git", "branch", "-a", "--format=%(refname:short)")
	output, err := s.runCommand(cmd, dryRun)
	if err != nil {
		return nil, fmt.Errorf("failed to list branches: %w\nOutput: %s", err, output)
	}

	if dryRun {
		return []string{"main", "develop", "origin/feature/example"}, nil
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

// MergeBranch initiates a merge in the specified worktree
func (s *Service) MergeBranch(worktreePath, sourceBranch string, dryRun bool) error {
	cmd := exec.Command("git", "-C", worktreePath, "merge", "--no-commit", sourceBranch)
	output, err := s.runCommand(cmd, dryRun)
	if err != nil {
		return fmt.Errorf("failed to merge branch: %w\nOutput: %s", err, output)
	}
	return nil
}

// MergeBranchWithCommit merges a branch and creates a commit with the specified message
func (s *Service) MergeBranchWithCommit(worktreePath, sourceBranch, commitMessage string, dryRun bool) error {
	cmd := exec.Command("git", "-C", worktreePath, "merge", "-m", commitMessage, sourceBranch)
	output, err := s.runCommand(cmd, dryRun)
	if err != nil {
		return fmt.Errorf("failed to merge branch: %w\nOutput: %s", err, output)
	}
	return nil
}

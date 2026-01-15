package git

import (
	"cmp"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/Kei-K23/trashbox"
)

// Worktree represents a git worktree with its metadata.
//
// A worktree is a separate working directory that shares the same git
// repository but has its own branch and commit state. This allows
// parallel development across multiple branches.
//
// Fields:
//   - Name: Short name of the worktree (usually directory name)
//   - Path: Absolute filesystem path to the worktree
//   - Branch: Git branch name this worktree is on
//   - Commit: Short commit hash (first 7 characters)
//   - IsBare: True only for the bare repository (.git directory)
//   - BaseBranch: The base branch used when creating this worktree
type Worktree struct {
	Name       string // Worktree name (e.g., "feature-x")
	Path       string // Absolute path to the worktree
	Branch     string // Branch name (e.g., "feature/ABC-123")
	Commit     string // Commit hash (short form)
	IsBare     bool   // True if this is the bare repository worktree
	BaseBranch string // Base branch used to create this worktree (e.g., "main")
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

// AddWorktree creates a new git worktree in the specified directory.
//
// This function creates a new worktree that can be checked out on a
// specified branch. It optionally creates the branch first if it doesn't
// exist, using a base branch as the starting point.
//
// Parameters:
//   - worktreesDir: Directory where worktrees should be created (usually repo/worktrees/)
//   - worktreeName: Name of the worktree (becomes the directory name)
//   - branchName: Git branch to check out in the new worktree
//   - createBranch: If true, create the branch first; if false, branch must exist
//   - baseBranch: Base branch for creating new branch (only used if createBranch=true)
//   - dryRun: If true, print the command without executing it
//
// Returns:
//   - *Worktree: The created worktree with metadata, or nil on error
//   - error: Non-nil if worktree creation failed
//
// Errors:
//   - ErrWorktreesDirectoryEmpty: If worktreesDir is empty
//   - ErrWorktreeNameEmpty: If worktreeName is empty
//   - ErrBranchNameEmpty: If branchName is empty
//   - Other git errors from command execution
//
// Example:
//
//	worktree, err := gitSvc.AddWorktree("./worktrees", "feature-x", "feature/x", true, "main", false)
//	if err != nil {
//	    return err // "failed to add worktree: ..."
//	}
//	fmt.Println("Created worktree at:", worktree.Path)
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
		// baseBranch should be provided by caller (from config.default_branch)
		// Fallback to "master" if somehow empty (should not happen in normal usage)
		baseBranch = cmp.Or(baseBranch, "master")

		args = []string{"worktree", "add", "-b", branchName, worktreePath, baseBranch}
	}

	cmd := exec.Command("git", args...)

	if dryRun {
		printDryRun(cmd)
		return &Worktree{
			Name:       worktreeName,
			Path:       worktreePath,
			Branch:     branchName,
			Commit:     "",
			IsBare:     false,
			BaseBranch: baseBranch,
		}, nil
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, ClassifyError("worktree add", err, string(output))
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
			// Set the base branch since git worktree list doesn't include it
			wt.BaseBranch = baseBranch
			return &wt, nil
		}
	}

	// If we can't find it, something went wrong
	return nil, fmt.Errorf("worktree created at %s but not found in worktree list", worktreePath)
}

// ListWorktrees lists all git worktrees in the repository.
//
// This returns all worktrees associated with the repository, including
// the bare repository entry. The list includes both active worktrees and
// broken worktree references (which should be cleaned up with RemoveWorktree).
//
// Parameters:
//   - dryRun: If true, print the command without executing it
//
// Returns:
//   - []Worktree: Slice of worktrees, sorted by path
//   - error: Non-nil if the command failed
//
// Example:
//
//	worktrees, err := gitSvc.ListWorktrees(false)
//	if err != nil {
//	    return err
//	}
//	for _, wt := range worktrees {
//	    fmt.Printf("Worktree: %s on branch %s\n", wt.Name, wt.Branch)
//	}
func (s *Service) ListWorktrees(dryRun bool) ([]Worktree, error) {
	args := []string{"worktree", "list"}

	cmd := exec.Command("git", args...)

	if dryRun {
		printDryRun(cmd)
		return []Worktree{}, nil
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, ClassifyError("worktree list", err, string(output))
	}

	return parseWorktrees(string(output)), nil
}

// GetWorktreeBranch returns the branch name associated with a worktree.
//
// This function queries the specified worktree's git directory to determine
// which branch it's currently on. It's useful for verifying the state of
// a worktree or displaying branch information.
//
// Parameters:
//   - worktreePath: Absolute path to the worktree directory
//
// Returns:
//   - string: Branch name (e.g., "feature/ABC-123")
//   - error: Non-nil if the path is invalid or git fails
//
// Errors:
//   - ErrWorktreePathEmpty: If worktreePath is empty
//   - Git command errors if the path is not a valid worktree
//
// Example:
//
//	branch, err := gitSvc.GetWorktreeBranch("/path/to/repo/worktrees/feature")
//	if err != nil {
//	    return err
//	}
//	fmt.Println("Branch:", branch)
func (s *Service) GetWorktreeBranch(worktreePath string) (string, error) {
	if worktreePath == "" {
		return "", ErrWorktreePathEmpty
	}

	// Use git -C to run command in the worktree directory
	cmd := exec.Command("git", "-C", worktreePath, "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", ClassifyError("rev-parse branch", err, string(output))
	}

	// Trim whitespace and newlines from the output
	branchName := strings.TrimSpace(string(output))

	return branchName, nil
}

// Fetch fetches from the remote repository
func (s *Service) Fetch(dryRun bool) error {
	args := []string{"fetch", "--all"}

	cmd := exec.Command("git", args...)

	if dryRun {
		printDryRun(cmd)
		return nil
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to fetch: %w", err)
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

	if dryRun {
		printDryRun(cmd)
		return nil
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return ClassifyError("worktree move", err, string(output))
	}

	return nil
}

// RemoveWorktree removes a git worktree by name and returns the removed worktree info.
//
// This function safely removes a worktree by first moving it to the trash
// (if available) before deleting the git reference. This provides a safety
// mechanism to recover accidentally deleted worktrees.
//
// The worktree is renamed with a timestamp before trashing to avoid
// conflicts if the same worktree name is reused immediately.
//
// Parameters:
//   - worktreeName: Name of the worktree to remove (not the path)
//   - force: If true, force removal even if the worktree is in an inconsistent state
//   - dryRun: If true, print what would happen without actually removing
//
// Returns:
//   - *Worktree: The removed worktree metadata
//   - error: Non-nil if the worktree wasn't found or removal failed
//
// Errors:
//   - ErrWorktreeNameEmpty: If worktreeName is empty
//   - Returns error if worktree not found
//
// Example:
//
//	removed, err := gitSvc.RemoveWorktree("feature-x", false, false)
//	if err != nil {
//	    return err // "worktree 'feature-x' not found"
//	}
//	fmt.Printf("Removed worktree: %s\n", removed.Name)
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

	// Move to Trash before git worktree remove (safety mechanism)
	timestamp := time.Now().Format("20060102-150405")
	baseName := filepath.Base(targetWorktree.Path)
	parentDir := filepath.Dir(targetWorktree.Path)
	renamedPath := filepath.Join(parentDir, fmt.Sprintf("%s-%s", baseName, timestamp))

	if dryRun {
		fmt.Printf("[DRY RUN] mv %s %s\n", targetWorktree.Path, renamedPath)
		fmt.Printf("[DRY RUN] trash %s\n", renamedPath)
	} else {
		// Rename directory to add timestamp
		if err := os.Rename(targetWorktree.Path, renamedPath); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Could not rename worktree for Trash: %v\n", err)
			fmt.Fprintf(os.Stderr, "Proceeding with removal...\n")
		} else {
			// Move to Trash using trashbox
			if err := trashbox.MoveToTrash(renamedPath); err != nil {
				// Failed to trash - rename back and warn (best effort, ignore errors)
				_ = os.Rename(renamedPath, targetWorktree.Path)
				fmt.Fprintf(os.Stderr, "Warning: Could not move worktree to Trash: %v\n", err)
				fmt.Fprintf(os.Stderr, "Proceeding with removal...\n")
			} else {
				fmt.Printf("Moved worktree to Trash: %s\n", fmt.Sprintf("%s-%s", baseName, timestamp))
			}
		}
	}

	// Use the full path from the worktree list
	args := []string{"worktree", "remove", targetWorktree.Path}

	if force {
		args = []string{"worktree", "remove", "--force", targetWorktree.Path}
	}

	cmd := exec.Command("git", args...)

	if dryRun {
		printDryRun(cmd)
		return targetWorktree, nil
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, ClassifyError("worktree remove", err, string(output))
	}

	return targetWorktree, nil
}

// IsInWorktree checks if the current directory is inside a worktree.
//
// This function determines whether the given path is within a worktree
// by comparing it against all known worktrees in the repository. It's
// useful for commands that need to validate they're being run from within
// a worktree.
//
// Parameters:
//   - currentPath: Path to check (usually the current working directory)
//
// Returns:
//   - bool: True if currentPath is inside a worktree
//   - string: Worktree name if in a worktree, empty string otherwise
//   - error: Non-nil if listing worktrees failed
//
// Example:
//
//	inWorktree, name, err := gitSvc.IsInWorktree(pwd)
//	if err != nil {
//	    return err
//	}
//	if inWorktree {
//	    fmt.Printf("You're in worktree: %s\n", name)
//	} else {
//	    fmt.Println("You're in the bare repository")
//	}
func (s *Service) IsInWorktree(currentPath string) (bool, string, error) {
	// Get all worktrees
	worktrees, err := s.ListWorktrees(false)
	if err != nil {
		return false, "", fmt.Errorf("failed to list worktrees: %w", err)
	}

	// Resolve symlinks for current path (handles /var -> /private/var on macOS)
	resolvedCurrentPath, err := filepath.EvalSymlinks(currentPath)
	if err != nil {
		// If we can't resolve symlinks, use the original path
		resolvedCurrentPath = currentPath
	}
	// Ensure path ends with separator for accurate prefix matching
	resolvedCurrentPath = filepath.Clean(resolvedCurrentPath)

	// Check if current path is within any worktree path
	for _, wt := range worktrees {
		// Skip bare repository
		if wt.IsBare {
			continue
		}

		// Resolve symlinks for worktree path
		resolvedWtPath, err := filepath.EvalSymlinks(wt.Path)
		if err != nil {
			resolvedWtPath = wt.Path
		}
		resolvedWtPath = filepath.Clean(resolvedWtPath)

		// Check if current path is the worktree path or a subdirectory
		if resolvedCurrentPath == resolvedWtPath || strings.HasPrefix(resolvedCurrentPath, resolvedWtPath+string(filepath.Separator)) {
			return true, wt.Name, nil
		}
	}

	return false, "", nil
}

// PullWorktree pulls changes from remote for a specific worktree
func (s *Service) PullWorktree(worktreePath string, dryRun bool) error {
	if worktreePath == "" {
		return ErrWorktreePathEmpty
	}

	// Check if worktree path exists
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		return fmt.Errorf("worktree path does not exist: %s", worktreePath)
	}

	// Get the current branch
	currentBranch, err := s.GetWorktreeBranch(worktreePath)
	if err != nil {
		return err
	}

	// Check if upstream is set
	upstream, err := s.GetUpstreamBranch(worktreePath)
	if err != nil {
		return fmt.Errorf("failed to check upstream branch: %w", err)
	}

	args := []string{"-C", worktreePath, "pull"}

	if upstream == "" {
		// No upstream set, try to pull with explicit remote and branch
		remoteBranch := fmt.Sprintf("origin/%s", currentBranch)

		// Check if remote branch exists
		remoteBranchExists, err := s.BranchExistsInPath(worktreePath, remoteBranch)
		if err != nil {
			return fmt.Errorf("failed to check if remote branch exists: %w", err)
		}

		if remoteBranchExists {
			// Remote branch exists, set upstream and pull
			setUpstreamCmd := exec.Command("git", "-C", worktreePath, "branch", "--set-upstream-to", remoteBranch)
			if dryRun {
				printDryRun(setUpstreamCmd)
			} else {
				if _, err := setUpstreamCmd.Output(); err != nil {
					return fmt.Errorf("failed to set upstream: %w", err)
				}
			}
		} else {
			// Remote branch doesn't exist, try to pull with explicit remote and branch
			args = append(args, "origin", currentBranch)
		}
	}

	cmd := exec.Command("git", args...)

	if dryRun {
		printDryRun(cmd)
		return nil
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to pull worktree: %w", err)
	}

	return nil
}

// PushWorktree pushes changes to remote for a specific worktree
func (s *Service) PushWorktree(worktreePath string, dryRun bool) error {
	if worktreePath == "" {
		return ErrWorktreePathEmpty
	}

	// Check if worktree path exists
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		return fmt.Errorf("worktree path does not exist: %s", worktreePath)
	}

	// Get the current branch
	currentBranch, err := s.GetWorktreeBranch(worktreePath)
	if err != nil {
		return err
	}

	// Check if upstream is set
	upstream, err := s.GetUpstreamBranch(worktreePath)
	if err != nil {
		return fmt.Errorf("failed to check upstream branch: %w", err)
	}

	var args []string
	if upstream == "" {
		// No upstream set, push with -u flag to set it
		args = []string{"-C", worktreePath, "push", "-u", "origin", currentBranch}
	} else {
		// Upstream already set, just push
		args = []string{"-C", worktreePath, "push"}
	}

	cmd := exec.Command("git", args...)

	if dryRun {
		printDryRun(cmd)
		return nil
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to push worktree: %w", err)
	}

	return nil
}

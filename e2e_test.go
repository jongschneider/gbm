package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildBinary builds gbm binary for testing and returns the path.
func buildBinary(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "gbm")

	cmd := exec.Command("go", "build", "-o", binPath, "./cmd")
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	require.NoError(t, err, "failed to build binary")

	return binPath
}

// runGBM runs gbm command and returns combined output (stdout + stderr).
func runGBM(t *testing.T, binPath, dir string, args ...string) (string, error) {
	t.Helper()

	cmd := exec.Command(binPath, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// runGBMStdout runs gbm command and returns stdout and stderr separately.
// This is important for shell integration tests where only stdout is captured.
func runGBMStdout(t *testing.T, binPath, dir string, args ...string) (stdout string, stderr string, err error) {
	t.Helper()

	cmd := exec.Command(binPath, args...)
	cmd.Dir = dir
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	err = cmd.Run()
	return strings.TrimSpace(stdoutBuf.String()), strings.TrimSpace(stderrBuf.String()), err
}

// testRepo holds test repository information
type testRepo struct {
	Root string
}

// setupGBMRepo creates a GBM repository with initial commit
func setupGBMRepo(t *testing.T) (*testRepo, string) {
	t.Helper()

	binPath := buildBinary(t)

	// Create a parent temp directory
	parentDir, err := os.MkdirTemp("", "gbm-e2e-test-*")
	require.NoError(t, err, "failed to create temp dir")
	t.Cleanup(func() {
		_ = os.RemoveAll(parentDir)
	})

	// Resolve symlinks (macOS /var -> /private/var issue)
	parentDir, err = filepath.EvalSymlinks(parentDir)
	require.NoError(t, err, "failed to resolve symlinks")

	repoDir := filepath.Join(parentDir, "repo")
	err = os.Mkdir(repoDir, 0755)
	require.NoError(t, err, "failed to create repo dir")

	repo := &testRepo{Root: repoDir}

	// Run gbm init to create the bare repo + main worktree
	out, err := runGBM(t, binPath, repoDir, "init")
	require.NoError(t, err, "gbm init failed\noutput: %s", out)

	// Create initial commit in the main worktree
	mainWorktreePath := filepath.Join(repoDir, "worktrees", "main")

	cmd := exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = mainWorktreePath
	err = cmd.Run()
	require.NoError(t, err, "failed to set git user.email")

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = mainWorktreePath
	err = cmd.Run()
	require.NoError(t, err, "failed to set git user.name")

	// Create a file in main worktree
	readmePath := filepath.Join(mainWorktreePath, "README.md")
	err = os.WriteFile(readmePath, []byte("# Test"), 0600)
	require.NoError(t, err, "failed to create README")

	// Commit in main worktree
	cmd = exec.Command("git", "add", "-A")
	cmd.Dir = mainWorktreePath
	err = cmd.Run()
	require.NoError(t, err, "failed to git add")

	cmd = exec.Command("git", "commit", "-m", "initial commit")
	cmd.Dir = mainWorktreePath
	err = cmd.Run()
	require.NoError(t, err, "failed to git commit")

	return repo, binPath
}

func TestE2E_WorktreeAdd_CLI(t *testing.T) {
	repo, binPath := setupGBMRepo(t)

	// Create worktree with new branch - use require (critical operation)
	out, err := runGBM(t, binPath, repo.Root, "worktree", "add", "feature-x", "feature-x", "-b")
	require.NoError(t, err, "worktree add failed\noutput: %s", out)

	// Verify worktree structure - use assert (want to see all failures)
	wtPath := filepath.Join(repo.Root, "worktrees", "feature-x")
	assert.DirExists(t, wtPath, "worktree directory should exist")

	gitDir := filepath.Join(wtPath, ".git")
	assert.FileExists(t, gitDir, "worktree .git should exist")

	// Verify branch was created
	cmd := exec.Command("git", "branch", "--list", "feature-x")
	cmd.Dir = repo.Root
	branchesBytes, err := cmd.Output()
	require.NoError(t, err, "failed to list branches")
	branches := string(branchesBytes)
	assert.Contains(t, branches, "feature-x", "branch should exist")
}

func TestE2E_WorktreeAdd_ExistingBranch(t *testing.T) {
	repo, binPath := setupGBMRepo(t)

	// Create a branch first
	cmd := exec.Command("git", "branch", "existing-branch")
	cmd.Dir = repo.Root
	err := cmd.Run()
	require.NoError(t, err, "failed to create branch")

	// Create worktree for existing branch (without -b)
	out, err := runGBM(t, binPath, repo.Root, "worktree", "add", "existing-wt", "existing-branch")
	require.NoError(t, err, "worktree add failed\noutput: %s", out)

	// Verify worktree exists
	wtPath := filepath.Join(repo.Root, "worktrees", "existing-wt")
	require.DirExists(t, wtPath, "worktree directory should exist")
}

func TestE2E_WorktreeList(t *testing.T) {
	t.Skip("TUI testing requires interactive terminal - tested separately")

	// Note: worktree list opens TUI which requires /dev/tty
	// This is tested manually or with a different testing approach
}

func TestE2E_WorktreeSwitch_StdoutOutput(t *testing.T) {
	repo, binPath := setupGBMRepo(t)

	// Create a worktree
	_, err := runGBM(t, binPath, repo.Root, "wt", "add", "feature-x", "feature-x", "-b")
	require.NoError(t, err, "failed to create worktree")

	// Test stdout/stderr separation - require (need output for validation)
	stdout, stderr, err := runGBMStdout(t, binPath, repo.Root, "worktree", "switch", "feature-x")
	require.NoError(t, err, "worktree switch should succeed\nstdout: %s\nstderr: %s", stdout, stderr)

	// Validate stdout properties - use assert (want to see all issues)
	expectedPath := filepath.Join(repo.Root, "worktrees", "feature-x")
	assert.Contains(t, stdout, expectedPath, "stdout should contain worktree path")
	assert.NotContains(t, stdout, "Switched to", "stdout should not contain messages")
	assert.NotContains(t, stdout, "✓", "stdout should not contain checkmark")

	// Validate stderr - use assert
	assert.True(t,
		strings.Contains(stderr, "Switched to") || strings.Contains(stderr, "✓"),
		"stderr should contain success message, got: %q", stderr)

	// Verify stdout format - use assert
	stdoutLines := strings.Split(strings.TrimSpace(stdout), "\n")
	assert.Len(t, stdoutLines, 1, "stdout should contain exactly 1 line (the path)")
}

func TestE2E_WorktreeAdd_StdoutOutput(t *testing.T) {
	repo, binPath := setupGBMRepo(t)

	// Test stdout/stderr separation for worktree add - require (need output)
	stdout, stderr, err := runGBMStdout(t, binPath, repo.Root, "wt", "add", "feature-y", "feature-y", "-b")
	require.NoError(t, err, "worktree add should succeed\nstdout: %s\nstderr: %s", stdout, stderr)

	// Validate stdout properties - use assert (want to see all issues)
	expectedPath := filepath.Join(repo.Root, "worktrees", "feature-y")
	assert.Contains(t, stdout, expectedPath, "stdout should contain worktree path")
	assert.NotContains(t, stdout, "Created", "stdout should not contain messages")
	assert.NotContains(t, stdout, "✓", "stdout should not contain checkmark")

	// Validate stderr - use assert
	assert.True(t,
		strings.Contains(stderr, "Created") || strings.Contains(stderr, "✓"),
		"stderr should contain success message, got: %q", stderr)

	// Verify stdout format - use assert
	stdoutLines := strings.Split(strings.TrimSpace(stdout), "\n")
	assert.Len(t, stdoutLines, 1, "stdout should contain exactly 1 line (the path)")
}

func TestE2E_WorktreeSwitch_Aliases(t *testing.T) {
	repo, binPath := setupGBMRepo(t)

	// Create worktrees for testing different aliases - require (setup)
	_, err := runGBM(t, binPath, repo.Root, "wt", "add", "test-sw", "test-sw", "-b")
	require.NoError(t, err, "failed to create test-sw worktree")

	_, err = runGBM(t, binPath, repo.Root, "wt", "add", "test-s", "test-s", "-b")
	require.NoError(t, err, "failed to create test-s worktree")

	// Test 'sw' alias - require (need output), assert (validation)
	stdout, _, err := runGBMStdout(t, binPath, repo.Root, "wt", "sw", "test-sw")
	require.NoError(t, err, "wt sw should succeed")
	expectedPath := filepath.Join(repo.Root, "worktrees", "test-sw")
	assert.Contains(t, stdout, expectedPath, "'wt sw' should output correct path")

	// Test 's' alias
	stdout, _, err = runGBMStdout(t, binPath, repo.Root, "wt", "s", "test-s")
	require.NoError(t, err, "wt s should succeed")
	expectedPath = filepath.Join(repo.Root, "worktrees", "test-s")
	assert.Contains(t, stdout, expectedPath, "'wt s' should output correct path")
}

func TestE2E_WorktreeAdd_Aliases(t *testing.T) {
	repo, binPath := setupGBMRepo(t)

	// Test 'a' alias - require (need output)
	stdout, _, err := runGBMStdout(t, binPath, repo.Root, "wt", "a", "test-a", "test-a", "-b")
	require.NoError(t, err, "wt a should succeed")

	// Validate output and worktree - use assert
	expectedPath := filepath.Join(repo.Root, "worktrees", "test-a")
	assert.Contains(t, stdout, expectedPath, "'wt a' should output correct path")
	assert.DirExists(t, expectedPath, "worktree directory should exist")
}

func TestE2E_WorktreeSwitch_NonExistent(t *testing.T) {
	repo, binPath := setupGBMRepo(t)

	// Try to switch to non-existent worktree
	_, err := runGBM(t, binPath, repo.Root, "wt", "switch", "does-not-exist")
	require.Error(t, err, "should error when switching to non-existent worktree")
}

func TestE2E_Init_CreatesStructure(t *testing.T) {
	binPath := buildBinary(t)

	// Create a temp directory for the test - require (setup)
	tempDir := t.TempDir()
	repoDir := filepath.Join(tempDir, "test-repo")
	err := os.Mkdir(repoDir, 0755)
	require.NoError(t, err, "failed to create repo dir")

	// Run gbm init - require (critical)
	out, err := runGBM(t, binPath, repoDir, "init")
	require.NoError(t, err, "gbm init should succeed\noutput: %s", out)

	// Verify directory structure - use assert (want to see all failures)
	bareGit := filepath.Join(repoDir, ".git")
	assert.DirExists(t, bareGit, ".git directory should exist")

	worktreesDir := filepath.Join(repoDir, "worktrees")
	assert.DirExists(t, worktreesDir, "worktrees directory should exist")
}

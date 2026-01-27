package testutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTestRepo(t *testing.T) {
	repo := NewTestRepo(t)

	// Verify repository structure - use assert (want to see all issues)
	assert.DirExists(t, repo.Root, "repository directory should exist")

	gitDir := filepath.Join(repo.Root, ".git")
	assert.DirExists(t, gitDir, ".git directory should exist")

	// Verify git config was set - use assert
	email := strings.TrimSpace(repo.Git("config", "user.email"))
	assert.Equal(t, "test@example.com", email, "user.email should be set correctly")

	name := strings.TrimSpace(repo.Git("config", "user.name"))
	assert.Equal(t, "Test User", name, "user.name should be set correctly")

	// Verify default branch is main
	branch := strings.TrimSpace(repo.Git("branch", "--show-current"))
	assert.Equal(t, "main", branch, "default branch should be main")
}

func TestGit(t *testing.T) {
	repo := NewTestRepo(t)

	// Test successful git command - empty repo should have no output
	out := repo.Git("status", "--short")
	assert.Empty(t, out, "empty repo should have no status output")

	// Create a file and test git status shows it
	repo.CreateFile("test.txt", "hello")
	out = repo.Git("status", "--short")
	assert.Contains(t, out, "test.txt", "status should show test.txt")
}

func TestGitE(t *testing.T) {
	repo := NewTestRepo(t)

	// Test successful command - use assert
	out, err := repo.GitE("status")
	assert.NoError(t, err, "status command should succeed")
	assert.Contains(t, out, "On branch main", "status should show current branch")

	// Test failing command - use assert
	_, err = repo.GitE("invalidcommand")
	assert.Error(t, err, "invalid command should return error")
}

func TestCreateFile(t *testing.T) {
	repo := NewTestRepo(t)

	// Test creating file in repo root
	repo.CreateFile("test.txt", "hello world")

	content, err := os.ReadFile(filepath.Join(repo.Root, "test.txt"))
	require.NoError(t, err, "should be able to read created file")
	assert.Equal(t, "hello world", string(content), "file content should match")

	// Test creating file in subdirectory
	repo.CreateFile("subdir/nested.txt", "nested content")

	content, err = os.ReadFile(filepath.Join(repo.Root, "subdir/nested.txt"))
	require.NoError(t, err, "should be able to read nested file")
	assert.Equal(t, "nested content", string(content), "nested file content should match")
}

func TestCommit(t *testing.T) {
	repo := NewTestRepo(t)

	// Create and commit a file
	repo.CreateFile("test.txt", "hello")
	repo.Commit("initial commit")

	// Verify commit was created - use assert
	log := repo.Git("log", "--oneline")
	assert.Contains(t, log, "initial commit", "log should contain commit message")

	// Verify working tree is clean
	status := repo.Git("status", "--short")
	assert.Empty(t, status, "working tree should be clean after commit")
}

func TestChdir(t *testing.T) {
	repo := NewTestRepo(t)

	// Get original directory - require (critical for test)
	origDir, err := os.Getwd()
	require.NoError(t, err, "should be able to get working directory")

	// Change to repo directory
	restore := repo.Chdir()

	// Verify we're in the repo directory - require (need this for restore test)
	currentDir, err := os.Getwd()
	require.NoError(t, err, "should be able to get working directory")
	assert.Equal(t, repo.Root, currentDir, "should be in repo directory")

	// Restore original directory
	restore()

	// Verify we're back in original directory - assert
	currentDir, err = os.Getwd()
	require.NoError(t, err, "should be able to get working directory")
	assert.Equal(t, origDir, currentDir, "should be back in original directory")
}

func TestPath(t *testing.T) {
	repo := NewTestRepo(t)

	// Test Path method - use assert
	testPath := repo.Path("test.txt")
	expected := filepath.Join(repo.Root, "test.txt")
	assert.Equal(t, expected, testPath, "Path should return correct absolute path")

	// Test with nested path
	nestedPath := repo.Path("subdir/nested.txt")
	expected = filepath.Join(repo.Root, "subdir/nested.txt")
	assert.Equal(t, expected, nestedPath, "Path should handle nested paths")
}

func TestParentDir(t *testing.T) {
	repo := NewTestRepo(t)

	parentDir := repo.ParentDir()
	expected := filepath.Dir(repo.Root)
	assert.Equal(t, expected, parentDir, "ParentDir should return parent directory")

	// Verify parent directory exists - use assert
	assert.DirExists(t, parentDir, "parent directory should exist")
}

func TestCleanup(t *testing.T) {
	var repoRoot string
	var parentDir string

	// Create repo in a subtest so cleanup runs at end of subtest
	t.Run("create", func(t *testing.T) {
		repo := NewTestRepo(t)
		repoRoot = repo.Root
		parentDir = repo.ParentDir()

		// Verify repo exists - use assert
		assert.DirExists(t, repoRoot, "repository directory should be created")
	})

	// After subtest ends, t.Cleanup should have run and removed the directory
	_, err := os.Stat(parentDir)
	assert.True(t, os.IsNotExist(err), "parent directory should be cleaned up after subtest")
}

// TestIntegration tests a complete workflow.
func TestIntegration(t *testing.T) {
	repo := NewTestRepo(t)

	// Create initial commit
	repo.CreateFile("README.md", "# Test Project")
	repo.Commit("initial commit")

	// Create feature branch
	repo.Git("checkout", "-b", "feature-x")

	// Add feature
	repo.CreateFile("feature.txt", "new feature")
	repo.Commit("add feature")

	// Verify we have 2 commits - use assert
	logLines := strings.Split(repo.Git("log", "--oneline"), "\n")
	assert.Len(t, logLines, 2, "should have 2 commits")

	// Switch back to main
	repo.Git("checkout", "main")

	// Verify feature.txt doesn't exist on main - use assert
	featurePath := filepath.Join(repo.Root, "feature.txt")
	_, err := os.Stat(featurePath)
	assert.True(t, os.IsNotExist(err), "feature.txt should not exist on main branch")
}

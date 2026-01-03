package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListIgnoredFiles(t *testing.T) {
	tempDir := t.TempDir()
	
	// Initialize a git repo
	service := NewService()
	err := service.Init(tempDir, "main", false)
	require.NoError(t, err)
	
	repoPath := filepath.Join(tempDir, "worktrees", "main")
	
	// Create a .gitignore file
	gitignoreContent := "*.log\nnode_modules/\n.env\n"
	gitignorePath := filepath.Join(repoPath, ".gitignore")
	err = os.WriteFile(gitignorePath, []byte(gitignoreContent), 0644)
	require.NoError(t, err)
	
	// Create some ignored files
	err = os.WriteFile(filepath.Join(repoPath, "app.log"), []byte("log content"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(repoPath, ".env"), []byte("SECRET=123"), 0644)
	require.NoError(t, err)
	
	// Add gitignore to git
	cmd := exec.Command("git", "add", ".gitignore")
	cmd.Dir = repoPath
	_, err = cmd.Output()
	require.NoError(t, err)
	
	cmd = exec.Command("git", "commit", "-m", "Add gitignore")
	cmd.Dir = repoPath
	_, err = cmd.Output()
	require.NoError(t, err)
	
	// Test ListIgnoredFiles
	ignored, err := service.ListIgnoredFiles(repoPath)
	require.NoError(t, err)
	
	assert.Contains(t, ignored, "app.log", "should contain ignored log file")
	assert.Contains(t, ignored, ".env", "should contain ignored env file")
}

func TestListUntrackedFiles(t *testing.T) {
	tempDir := t.TempDir()
	
	// Initialize a git repo
	service := NewService()
	err := service.Init(tempDir, "main", false)
	require.NoError(t, err)
	
	repoPath := filepath.Join(tempDir, "worktrees", "main")
	
	// Create some untracked files
	err = os.WriteFile(filepath.Join(repoPath, "newfile.txt"), []byte("content"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(repoPath, "another.md"), []byte("# Title"), 0644)
	require.NoError(t, err)
	
	// Test ListUntrackedFiles
	untracked, err := service.ListUntrackedFiles(repoPath)
	require.NoError(t, err)
	
	assert.Contains(t, untracked, "newfile.txt", "should contain untracked file")
	assert.Contains(t, untracked, "another.md", "should contain untracked file")
}

func TestMatchesPattern(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		patterns []string
		want     bool
	}{
		{
			name:     "exact match",
			path:     ".env",
			patterns: []string{".env"},
			want:     true,
		},
		{
			name:     "suffix wildcard",
			path:     "file.log",
			patterns: []string{"*.log"},
			want:     true,
		},
		{
			name:     "suffix wildcard no match",
			path:     "file.txt",
			patterns: []string{"*.log"},
			want:     false,
		},
		{
			name:     "directory pattern",
			path:     "node_modules/package.json",
			patterns: []string{"node_modules/"},
			want:     true,
		},
		{
			name:     "multiple patterns",
			path:     "app.log",
			patterns: []string{".env", "*.log", "*.md"},
			want:     true,
		},
		{
			name:     "no patterns match",
			path:     "main.go",
			patterns: []string{".env", "*.log"},
			want:     false,
		},
		{
			name:     "empty patterns",
			path:     "file.txt",
			patterns: []string{},
			want:     false,
		},
		{
			name:     "prefix wildcard",
			path:     "test_file.txt",
			patterns: []string{"test_*"},
			want:     true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MatchesPattern(tt.path, tt.patterns)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestParseFileList(t *testing.T) {
	tests := []struct {
		name string
		out  string
		want []string
	}{
		{
			name: "single file",
			out:  "file.txt\n",
			want: []string{"file.txt"},
		},
		{
			name: "multiple files",
			out:  "file1.txt\nfile2.md\ndir/file3.go\n",
			want: []string{"file1.txt", "file2.md", "dir/file3.go"},
		},
		{
			name: "empty output",
			out:  "",
			want: nil,
		},
		{
			name: "with empty lines",
			out:  "file1.txt\n\nfile2.txt\n",
			want: []string{"file1.txt", "file2.txt"},
		},
		{
			name: "with spaces",
			out:  "  file1.txt  \nfile2.txt\n",
			want: []string{"file1.txt", "file2.txt"},
		},
		{
			name: "skip .git directory",
			out:  ".git/objects\nfile.txt\n.git/refs\n",
			want: []string{"file.txt"},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseFileList(tt.out)
			assert.Equal(t, tt.want, result)
		})
	}
}

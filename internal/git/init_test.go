package git

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInit(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	testRepo := filepath.Join(tempDir, "test-repo")

	service := NewService()

	// Test Init with custom name and branch
	err := service.Init(testRepo, "main", false)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Verify directory structure
	tests := []struct {
		path        string
		description string
	}{
		{filepath.Join(testRepo, ".git"), "bare git repository"},
		{filepath.Join(testRepo, "worktrees"), "worktrees directory"},
		{filepath.Join(testRepo, "worktrees", "main"), "main worktree"},
		{filepath.Join(testRepo, ".gbm"), ".gbm directory"},
		{filepath.Join(testRepo, ".gbm", "config.yaml"), "config file"},
	}

	for _, tt := range tests {
		if _, err := os.Stat(tt.path); os.IsNotExist(err) {
			t.Errorf("%s does not exist: %s", tt.description, tt.path)
		}
	}

	// Verify config content
	configPath := filepath.Join(testRepo, ".gbm", "config.yaml")
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	expectedContent := "default_branch: main"
	if !contains(string(content), expectedContent) {
		t.Errorf("Config file doesn't contain expected content. Got: %s", content)
	}
}

func TestInitWithDefaults(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Change to temp directory
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer func() { _ = os.Chdir(oldWd) }()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	service := NewService()

	// Test Init with empty strings (should use defaults)
	err = service.Init("", "", false)
	if err != nil {
		t.Fatalf("Init with defaults failed: %v", err)
	}

	// Verify default branch "main" was created
	mainWorktree := filepath.Join(tempDir, "worktrees", "main")
	if _, err := os.Stat(mainWorktree); os.IsNotExist(err) {
		t.Errorf("Default main worktree was not created")
	}

	// Verify config has default branch
	configPath := filepath.Join(tempDir, ".gbm", "config.yaml")
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	if !contains(string(content), "default_branch: main") {
		t.Errorf("Config doesn't have default branch 'main'")
	}
}

func TestInitWithCustomBranch(t *testing.T) {
	tempDir := t.TempDir()
	testRepo := filepath.Join(tempDir, "custom-branch-repo")

	service := NewService()

	// Test Init with custom branch name
	err := service.Init(testRepo, "develop", false)
	if err != nil {
		t.Fatalf("Init with custom branch failed: %v", err)
	}

	// Verify develop worktree was created
	developWorktree := filepath.Join(testRepo, "worktrees", "develop")
	if _, err := os.Stat(developWorktree); os.IsNotExist(err) {
		t.Errorf("Custom 'develop' worktree was not created")
	}

	// Verify config has correct branch
	configPath := filepath.Join(testRepo, ".gbm", "config.yaml")
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	if !contains(string(content), "default_branch: develop") {
		t.Errorf("Config doesn't have custom branch 'develop'")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestInitDryRun(t *testing.T) {
	tempDir := t.TempDir()
	testRepo := filepath.Join(tempDir, "dryrun-test")

	service := NewService()

	// Test Init in dry-run mode
	err := service.Init(testRepo, "main", true)
	if err != nil {
		t.Fatalf("Init in dry-run mode failed: %v", err)
	}

	// Verify that NO directories were created
	tests := []struct {
		path        string
		description string
	}{
		{filepath.Join(testRepo, ".git"), "bare git repository"},
		{filepath.Join(testRepo, "worktrees"), "worktrees directory"},
		{filepath.Join(testRepo, "worktrees", "main"), "main worktree"},
		{filepath.Join(testRepo, ".gbm"), ".gbm directory"},
		{filepath.Join(testRepo, ".gbm", "config.yaml"), "config file"},
	}

	for _, tt := range tests {
		if _, err := os.Stat(tt.path); !os.IsNotExist(err) {
			t.Errorf("In dry-run mode, %s should not exist: %s", tt.description, tt.path)
		}
	}
}

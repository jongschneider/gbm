package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gbm/internal/git"

	"gopkg.in/yaml.v3"
)

// Config represents the .gbm/config.yaml structure
type Config struct {
	DefaultBranch string `yaml:"default_branch"`
	WorktreesDir  string `yaml:"worktrees_dir"`
}

// Service wraps the git service and provides configuration
type Service struct {
	Git         *git.Service
	WorktreeDir string
	RepoRoot    string
	config      *Config
}

// NewService creates a new service instance with loaded configuration.
// If not in a git repository or config doesn't exist, uses defaults.
func NewService() *Service {
	gitSvc := git.NewService()

	svc := &Service{
		Git:         gitSvc,
		WorktreeDir: "worktrees", // default
	}

	// Try to load config from .gbm/config.yaml
	if err := svc.loadConfig(); err != nil {
		// Not in a repo or config doesn't exist - use defaults
		// This is fine for commands like `init` and `clone`
		return svc
	}

	return svc
}

// loadConfig attempts to load configuration from .gbm/config.yaml
func (s *Service) loadConfig() error {
	// Get repository root
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("not in a git repository")
	}
	s.RepoRoot = strings.TrimSpace(string(output))

	// Try to read config file
	configPath := filepath.Join(s.RepoRoot, ".gbm", "config.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("config file not found: %w", err)
	}

	// Parse YAML
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	s.config = &config

	// Set worktree directory from config
	if config.WorktreesDir != "" {
		s.WorktreeDir = config.WorktreesDir
	}

	return nil
}

// GetWorktreesPath returns the absolute path to the worktrees directory
func (s *Service) GetWorktreesPath() (string, error) {
	if s.RepoRoot == "" {
		return "", fmt.Errorf("not in a git repository")
	}
	return filepath.Join(s.RepoRoot, s.WorktreeDir), nil
}

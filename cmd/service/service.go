package service

import (
	"fmt"
	"os"
	"path/filepath"

	"gbm/internal/git"
	"gbm/internal/jira"

	"gopkg.in/yaml.v3"
)

// JiraConfig represents JIRA-specific configuration
type JiraConfig struct {
	Me string `yaml:"me,omitempty"` // Cached JIRA username
}

// Config represents the .gbm/config.yaml structure
type Config struct {
	DefaultBranch string     `yaml:"default_branch"`
	WorktreesDir  string     `yaml:"worktrees_dir"`
	Jira          JiraConfig `yaml:"jira,omitempty"`
}

// Service wraps the git and jira services and provides configuration
type Service struct {
	Git         *git.Service
	Jira        *jira.Service
	WorktreeDir string
	RepoRoot    string
	config      *Config
}

// NewService creates a new service instance with loaded configuration.
// If not in a git repository or config doesn't exist, uses defaults.
func NewService() *Service {
	gitSvc := git.NewService()

	// Check if debug mode is enabled
	debug := os.Getenv("GBM_DEBUG") != ""
	jiraSvc := jira.NewService(debug)

	svc := &Service{
		Git:         gitSvc,
		Jira:        jiraSvc,
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
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Find repository root (works from worktrees too!)
	repoRoot, err := s.Git.FindGitRoot(cwd)
	if err != nil {
		return err // Not in a git repository
	}
	s.RepoRoot = repoRoot

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

// GetConfig returns the current configuration
// Returns a default config if not loaded
func (s *Service) GetConfig() *Config {
	if s.config == nil {
		return &Config{
			DefaultBranch: "main",
			WorktreesDir:  "worktrees",
			Jira:          JiraConfig{},
		}
	}
	return s.config
}

// SaveConfig writes the current configuration to .gbm/config.yaml
func (s *Service) SaveConfig() error {
	if s.RepoRoot == "" {
		return fmt.Errorf("not in a git repository")
	}

	configPath := filepath.Join(s.RepoRoot, ".gbm", "config.yaml")
	data, err := yaml.Marshal(s.config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

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

// State represents the .gbm/state.yaml structure for cached data
type State struct {
	Jira             jira.IssuesCache `yaml:"jira"`
	CurrentWorktree  string           `yaml:"current_worktree,omitempty"`  // Last worktree we switched to
	PreviousWorktree string           `yaml:"previous_worktree,omitempty"` // Worktree before current
}

// Service wraps the git and jira services and provides configuration
type Service struct {
	Git         *git.Service
	Jira        *jira.Service
	WorktreeDir string
	RepoRoot    string
	config      *Config
	state       *State
}

// cacheStore implements jira.CacheStore using the CLI's state file
type cacheStore struct {
	svc *Service
}

// Load returns the cached issues and user from state
func (c *cacheStore) Load() (*jira.IssuesCache, string, error) {
	state := c.svc.GetState()
	config := c.svc.GetConfig()
	return &state.Jira, config.Jira.Me, nil
}

// Save persists the cache and user to state
func (c *cacheStore) Save(cache *jira.IssuesCache, user string) error {
	if cache != nil {
		state := c.svc.GetState()
		state.Jira = *cache
		_ = c.svc.SaveState() // Ignore errors - caching is optional
	}

	if user != "" {
		config := c.svc.GetConfig()
		config.Jira.Me = user
		_ = c.svc.SaveConfig() // Ignore errors - caching is optional
	}

	return nil
}

// NewService creates a new service instance with loaded configuration.
// If not in a git repository or config doesn't exist, uses defaults.
func NewService() *Service {
	gitSvc := git.NewService()

	// Check if debug mode is enabled
	debug := os.Getenv("GBM_DEBUG") != ""

	// Create a temporary service to load config and state
	svc := &Service{
		Git:         gitSvc,
		WorktreeDir: "worktrees", // default
	}

	// Try to load config from .gbm/config.yaml
	if err := svc.loadConfig(); err != nil {
		// Not in a repo or config doesn't exist - use defaults
		// This is fine for commands like `init` and `clone`
		// Create jira service with no cache store
		svc.Jira = jira.NewService(debug, nil)
		return svc
	}

	// Load state to get cached jira data
	_ = svc.loadState() // Ignore errors - state might not exist yet

	// Create cache store that wraps this service
	store := &cacheStore{svc: svc}

	// Create jira service with cache store
	svc.Jira = jira.NewService(debug, store)

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

// loadState attempts to load state from .gbm/state.yaml
func (s *Service) loadState() error {
	if s.RepoRoot == "" {
		return fmt.Errorf("not in a git repository")
	}

	statePath := filepath.Join(s.RepoRoot, ".gbm", "state.yaml")
	data, err := os.ReadFile(statePath)
	if err != nil {
		// State file doesn't exist yet - that's okay
		s.state = &State{}
		return nil
	}

	var state State
	if err := yaml.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("failed to parse state: %w", err)
	}

	s.state = &state
	return nil
}

// SaveState writes the current state to .gbm/state.yaml
func (s *Service) SaveState() error {
	if s.RepoRoot == "" {
		return fmt.Errorf("not in a git repository")
	}

	statePath := filepath.Join(s.RepoRoot, ".gbm", "state.yaml")
	data, err := yaml.Marshal(s.state)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	if err := os.WriteFile(statePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write state: %w", err)
	}

	return nil
}

// GetState returns the current state (loads it if not already loaded)
func (s *Service) GetState() *State {
	if s.state == nil {
		_ = s.loadState() // Ignore errors, return empty state
		if s.state == nil {
			s.state = &State{}
		}
	}
	return s.state
}

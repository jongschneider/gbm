// Package service implements the gbm CLI commands and business logic.
package service

import (
	"fmt"
	"gbm/internal/git"
	"gbm/internal/jira"
	"gbm/internal/utils"
	"gbm/pkg/tui"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// JiraConfig represents JIRA-specific configuration.
// This holds all JIRA integration settings including filters, attachment downloads, and markdown generation.
type JiraConfig struct {
	Me          string           `yaml:"me,omitempty"`
	Filters     jira.JiraFilters `yaml:"filters,omitempty"`
	Markdown    MarkdownConfig   `yaml:"markdown,omitempty"`
	Attachments AttachmentConfig `yaml:"attachments,omitempty"`
}

// AttachmentConfig holds configuration for JIRA attachment downloads.
// These settings control how attachments are downloaded when creating worktrees from JIRA issues.
type AttachmentConfig struct {
	Directory          string `yaml:"directory"`
	MaxSizeMB          int64  `yaml:"max_size_mb"`
	DownloadTimeoutSec int    `yaml:"download_timeout_seconds"`
	RetryAttempts      int    `yaml:"retry_attempts"`
	RetryBackoffMs     int    `yaml:"retry_backoff_ms"`
	Enabled            bool   `yaml:"enabled"`
}

// MarkdownConfig holds configuration for JIRA markdown generation.
// Controls how JIRA issue information is formatted and exported to markdown files.
type MarkdownConfig struct {
	FilenamePattern     string `yaml:"filename_pattern"`
	MaxDepth            int    `yaml:"max_depth"`
	IncludeComments     bool   `yaml:"include_comments"`
	IncludeAttachments  bool   `yaml:"include_attachments"`
	UseRelativeLinks    bool   `yaml:"use_relative_links"`
	IncludeLinkedIssues bool   `yaml:"include_linked_issues"`
}

// FileCopyRule defines files to copy from a source worktree.
// Rules-based file copying allows selective copy of specific files and directories.
type FileCopyRule struct {
	SourceWorktree string   `yaml:"source_worktree"`
	Files          []string `yaml:"files"`
}

// AutoFileCopyConfig holds configuration for automatic file copying.
// Enables automatic copying of .gitignore'd and untracked files to new worktrees.
//
// Example:
//
//	auto:
//	  enabled: true
//	  source_worktree: "{default}"
//	  copy_ignored: true
//	  exclude:
//	    - "*.log"
//	    - "node_modules/"
type AutoFileCopyConfig struct {
	SourceWorktree string   `yaml:"source_worktree"`
	Exclude        []string `yaml:"exclude"`
	Enabled        bool     `yaml:"enabled"`
	CopyIgnored    bool     `yaml:"copy_ignored"`
	CopyUntracked  bool     `yaml:"copy_untracked"`
}

// FileCopyConfig holds file copying rules for new worktrees.
// Supports both rule-based (explicit file lists) and automatic (gitignore-based) copying.
type FileCopyConfig struct {
	Rules []FileCopyRule     `yaml:"rules,omitempty"`
	Auto  AutoFileCopyConfig `yaml:"auto,omitempty"`
}

// WorktreeConfig defines a persistent worktree configuration.
// Stores metadata about a worktree such as its branch and merge strategy.
type WorktreeConfig struct {
	Branch      string `yaml:"branch"`
	MergeInto   string `yaml:"merge_into,omitempty"`
	Description string `yaml:"description,omitempty"`
}

// GetBranch implements tui.WorktreeConfig.GetBranch.
func (wc *WorktreeConfig) GetBranch() string {
	return wc.Branch
}

// GetMergeInto implements tui.WorktreeConfig.GetMergeInto.
func (wc *WorktreeConfig) GetMergeInto() string {
	return wc.MergeInto
}

// Config represents the .gbm/config.yaml structure.
// This is the main configuration file for GBM, stored at .gbm/config.yaml in the repository root.
//
// Example:
//
//	default_branch: main
//	worktrees_dir: worktrees
//	jira:
//	  enabled: true
//	  host: https://jira.example.com
//	file_copy:
//	  auto:
//	    enabled: true
//	    source_worktree: "{default}"
type Config struct {
	Worktrees     map[string]WorktreeConfig `yaml:"worktrees,omitempty"`
	DefaultBranch string                    `yaml:"default_branch"      validate:"required,min=1"`
	WorktreesDir  string                    `yaml:"worktrees_dir"       validate:"required,min=1"`
	FileCopy      FileCopyConfig            `yaml:"file_copy,omitempty"`
	Jira          JiraConfig                `yaml:"jira,omitempty"`
}

// GetWorktrees implements tui.RepoConfig.GetWorktrees.
// Returns the configured worktrees as a map of name to WorktreeConfig.
func (c *Config) GetWorktrees() map[string]tui.WorktreeConfig {
	if c.Worktrees == nil {
		return make(map[string]tui.WorktreeConfig)
	}
	result := make(map[string]tui.WorktreeConfig)
	for name, wt := range c.Worktrees {
		result[name] = &wt
	}
	return result
}

// State represents the .gbm/state.yaml structure for cached data.
// This file stores runtime state like the current worktree and cached JIRA issues.
// It's automatically managed and can be safely deleted.
type State struct {
	CurrentWorktree  string           `yaml:"current_worktree,omitempty"`
	PreviousWorktree string           `yaml:"previous_worktree,omitempty"`
	Jira             jira.IssuesCache `yaml:"jira"`
}

// Service wraps the git and jira services and provides configuration management.
//
// This is the main service for CLI commands. It manages configuration files,
// state caching, and coordinates between git and JIRA services.
//
// Fields:
//   - Git: Git service for repository operations
//   - Jira: JIRA service for issue integration
//   - WorktreeDir: Configured worktrees directory (from config or default)
//   - RepoRoot: Absolute path to repository root
type Service struct {
	Git         *git.Service
	Jira        *jira.Service
	config      *Config
	state       *State
	WorktreeDir string
	RepoRoot    string
}

// cacheStore implements jira.CacheStore using the CLI's state file.
type cacheStore struct {
	svc *Service
}

// Load returns the cached issues and user from state.
func (c *cacheStore) Load() (*jira.IssuesCache, string, error) {
	state := c.svc.GetState()
	config := c.svc.GetConfig()
	return &state.Jira, config.Jira.Me, nil
}

// Save persists the cache and user to state.
func (c *cacheStore) Save(cache *jira.IssuesCache, user string) error {
	if cache != nil {
		state := c.svc.GetState()
		state.Jira = *cache
		//nolint:errcheck // Caching is optional - errors are non-fatal
		c.svc.SaveState()
	}

	if user != "" {
		config := c.svc.GetConfig()
		config.Jira.Me = user
		//nolint:errcheck // Caching is optional - errors are non-fatal
		c.svc.SaveConfig()
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
	err := svc.loadConfig()
	if err != nil {
		// Not in a repo or config doesn't exist - use defaults
		// This is fine for commands like `init` and `clone`
		// Create jira service with no cache store
		svc.Jira = jira.NewService(debug, nil)
		return svc
	}

	// Load state to get cached jira data
	//nolint:errcheck // State might not exist yet - this is expected
	svc.loadState()

	// Create cache store that wraps this service
	store := &cacheStore{svc: svc}

	// Create jira service with cache store
	svc.Jira = jira.NewService(debug, store)

	return svc
}

// loadConfig attempts to load configuration from .gbm/config.yaml.
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

	// Validate config structure
	if err := validateConfig(&config); err != nil {
		return err
	}

	s.config = &config

	// Set worktree directory from config
	if config.WorktreesDir != "" {
		s.WorktreeDir = config.WorktreesDir
	}

	return nil
}

// GetWorktreesPath returns the absolute path to the worktrees directory.
// Supports template variables like {gitroot} in the configured worktrees_dir.
//
// Example:
//
//	Config: worktrees_dir = "../{gitroot}-worktrees"
//	Repo: /home/user/projects/gbm
//	Returns: /home/user/projects/gbm-worktrees
func (s *Service) GetWorktreesPath() (string, error) {
	if s.RepoRoot == "" {
		return "", ErrNotInGitRepository
	}

	// Expand template variables
	vars := utils.GetTemplateVars(s.RepoRoot)
	expandedDir := utils.ExpandTemplate(s.WorktreeDir, vars)

	// Expand ~ and resolve relative paths
	expandedDir = utils.ExpandPath(expandedDir, s.RepoRoot)

	return expandedDir, nil
}

// GetConfig returns the current configuration.
// Returns a default config if not loaded from file.
//
// The returned config is read from .gbm/config.yaml if the service
// successfully loaded it, otherwise returns a sensible default.
//
// Returns:
//   - *Config: The loaded or default configuration
//
// Example:
//
//	config := svc.GetConfig()
//	fmt.Println("Default branch:", config.DefaultBranch)
//	fmt.Println("Worktrees directory:", config.WorktreesDir)
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

// GetJiraFilters returns the configured JIRA filters with sensible defaults.
func (s *Service) GetJiraFilters() jira.JiraFilters {
	config := s.GetConfig()
	filters := config.Jira.Filters

	// Apply default assignee if not specified
	if filters.Assignee == "" {
		filters.Assignee = "me"
	}

	return filters
}

// GetJiraAttachmentConfig returns the attachment configuration with defaults.
func (s *Service) GetJiraAttachmentConfig() jira.AttachmentConfig {
	config := s.GetConfig()
	svcConfig := config.Jira.Attachments

	// Create jira.AttachmentConfig with defaults
	jiraConfig := jira.DefaultAttachmentConfig()

	// Override with user configuration if provided
	if svcConfig.MaxSizeMB > 0 {
		jiraConfig.MaxSizeMB = svcConfig.MaxSizeMB
	}
	if svcConfig.DownloadTimeoutSec > 0 {
		jiraConfig.Timeout = time.Duration(svcConfig.DownloadTimeoutSec) * time.Second
	}
	if svcConfig.RetryAttempts > 0 {
		jiraConfig.RetryAttempts = svcConfig.RetryAttempts
	}
	if svcConfig.RetryBackoffMs > 0 {
		jiraConfig.RetryBackoffMs = svcConfig.RetryBackoffMs
	}

	return jiraConfig
}

// GetJiraMarkdownConfig returns the markdown configuration with defaults.
func (s *Service) GetJiraMarkdownConfig() (includeComments, includeAttachments, includeLinkedIssues bool, maxDepth int) {
	config := s.GetConfig()
	mdConfig := config.Jira.Markdown

	// Default to true if not explicitly configured
	includeComments = mdConfig.IncludeComments
	includeAttachments = mdConfig.IncludeAttachments
	includeLinkedIssues = mdConfig.IncludeLinkedIssues
	maxDepth = mdConfig.MaxDepth

	// If no config provided, default to true
	if mdConfig == (MarkdownConfig{}) {
		includeComments = true
		includeAttachments = true
		includeLinkedIssues = true
		maxDepth = 2
	}

	// If maxDepth is 0 (not configured), default to 2
	if maxDepth == 0 {
		maxDepth = 2
	}

	// If attachments are disabled in config, don't include them
	if !config.Jira.Attachments.Enabled && config.Jira.Attachments != (AttachmentConfig{}) {
		includeAttachments = false
	}

	return includeComments, includeAttachments, includeLinkedIssues, maxDepth
}

// SaveConfig writes the current configuration to .gbm/config.yaml.
// Creates the .gbm directory if it doesn't exist.
//
// The configuration is serialized to YAML format. Errors are returned if
// the file cannot be written or if not in a git repository.
//
// Returns:
//   - error: Non-nil if the config directory/file cannot be created or written
//
// Example:
//
//	config := svc.GetConfig()
//	config.DefaultBranch = "develop"
//	if err := svc.SaveConfig(); err != nil {
//	    return err
//	}
func (s *Service) SaveConfig() error {
	if s.RepoRoot == "" {
		return ErrNotInGitRepository
	}

	configPath := filepath.Join(s.RepoRoot, ".gbm", "config.yaml")
	data, err := yaml.Marshal(s.config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// loadState attempts to load state from .gbm/state.yaml.
func (s *Service) loadState() error {
	if s.RepoRoot == "" {
		return ErrNotInGitRepository
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

// SaveState writes the current state to .gbm/state.yaml.
// Creates the .gbm directory if it doesn't exist.
//
// State files are automatically managed and can be safely deleted without
// affecting repository functionality. They cache JIRA issues and worktree history.
//
// Returns:
//   - error: Non-nil if the state file cannot be written
//
// Example:
//
//	state := svc.GetState()
//	state.CurrentWorktree = "feature-x"
//	if err := svc.SaveState(); err != nil {
//	    return err
//	}
func (s *Service) SaveState() error {
	if s.RepoRoot == "" {
		return ErrNotInGitRepository
	}

	statePath := filepath.Join(s.RepoRoot, ".gbm", "state.yaml")
	data, err := yaml.Marshal(s.state)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	if err := os.WriteFile(statePath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write state: %w", err)
	}

	return nil
}

// GetState returns the current state (loads it if not already loaded).
// Never returns nil - returns empty state if loading fails.
//
// The state is lazily loaded from .gbm/state.yaml on first access.
// If the file doesn't exist or can't be read, an empty state is returned.
//
// Returns:
//   - *State: Always non-nil state (loaded or empty)
//
// Example:
//
//	state := svc.GetState()
//	fmt.Println("Current worktree:", state.CurrentWorktree)
//	fmt.Println("Previous worktree:", state.PreviousWorktree)
func (s *Service) GetState() *State {
	if s.state == nil {
		//nolint:errcheck // Missing state file is expected - return empty state
		s.loadState()
		if s.state == nil {
			s.state = &State{}
		}
	}
	return s.state
}

// CreateJiraMarkdownFile creates a markdown file with JIRA ticket information
// in the .jira/ directory of the worktree. Only creates the file if worktreeName
// is a valid JIRA key. All errors return nil to avoid failing worktree creation.
func (s *Service) CreateJiraMarkdownFile(worktreeName string) error {
	// Check if worktree name is a JIRA key
	if !jira.IsJiraKey(worktreeName) {
		return nil // Silently skip non-JIRA worktrees
	}

	// Check if JIRA CLI is available
	if !s.Jira.IsJiraCliAvailable() {
		return nil // Silently skip if JIRA CLI not installed
	}

	// Get worktree path
	if s.RepoRoot == "" {
		return nil // Not in a git repo
	}
	worktreePath := filepath.Join(s.RepoRoot, s.WorktreeDir, worktreeName)

	// Check if worktree exists
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		fmt.Printf("Warning: worktree '%s' does not exist, skipping JIRA markdown creation\n", worktreeName)
		return nil
	}

	// Load configuration
	attachmentConfig := s.GetJiraAttachmentConfig()
	includeComments, includeAttachments, includeLinkedIssues, maxDepth := s.GetJiraMarkdownConfig()

	// Use the enhanced markdown generation with configuration
	opts := jira.DefaultIssueMarkdownOptions(worktreePath)
	opts.AttachmentConfig = attachmentConfig
	opts.DownloadAttachments = includeAttachments
	opts.IncludeComments = includeComments
	opts.IncludeLinkedIssues = includeLinkedIssues
	opts.MaxDepth = maxDepth
	opts.Filename = fmt.Sprintf(".jira/%s.md", worktreeName) // Place in .jira directory

	// Generate markdown with attachments
	result, err := s.Jira.GenerateIssueMarkdownFile(
		worktreeName,
		opts,
		false, // not dry-run
	)
	if err != nil {
		fmt.Printf("Warning: failed to generate JIRA markdown for %s: %v\n", worktreeName, err)
		return nil
	}

	// Print success message with details
	fmt.Printf("✓ Created JIRA markdown: %s\n", result.MarkdownPath)

	// Report attachment results if any
	if len(result.AttachmentResults) > 0 {
		fmt.Printf("  📎 Attachments: %d downloaded, %d skipped, %d failed\n",
			result.AttachmentsDownloaded,
			result.AttachmentsSkipped,
			result.AttachmentsFailed,
		)

		// Show details for skipped and failed attachments
		for _, ar := range result.AttachmentResults {
			if ar.Skipped {
				fmt.Printf("  ⚠️  %s - skipped (%s)\n", ar.Attachment.Filename, ar.SkipReason)
			} else if ar.Error != nil {
				fmt.Printf("  ✗ %s - failed: %v\n", ar.Attachment.Filename, ar.Error)
			}
		}
	}

	return nil
}

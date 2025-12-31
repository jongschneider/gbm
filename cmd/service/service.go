package service

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gbm/internal/git"
	"gbm/internal/jira"

	"gopkg.in/yaml.v3"
)

// JiraConfig represents JIRA-specific configuration
type JiraConfig struct {
	Me          string           `yaml:"me,omitempty"`          // Cached JIRA username
	Filters     jira.JiraFilters `yaml:"filters,omitempty"`     // Issue list filters
	Attachments AttachmentConfig `yaml:"attachments,omitempty"` // Attachment download settings
	Markdown    MarkdownConfig   `yaml:"markdown,omitempty"`    // Markdown generation settings
}

// AttachmentConfig holds configuration for JIRA attachment downloads
type AttachmentConfig struct {
	Enabled            bool   `yaml:"enabled"`                  // Enable attachment downloads
	MaxSizeMB          int64  `yaml:"max_size_mb"`              // Maximum file size in MB
	Directory          string `yaml:"directory"`                // Directory relative to worktree root
	DownloadTimeoutSec int    `yaml:"download_timeout_seconds"` // HTTP timeout in seconds
	RetryAttempts      int    `yaml:"retry_attempts"`           // Number of retry attempts
	RetryBackoffMs     int    `yaml:"retry_backoff_ms"`         // Initial retry backoff in milliseconds
}

// MarkdownConfig holds configuration for JIRA markdown generation
type MarkdownConfig struct {
	IncludeComments    bool   `yaml:"include_comments"`    // Include comments in markdown
	IncludeAttachments bool   `yaml:"include_attachments"` // Include attachments section
	UseRelativeLinks   bool   `yaml:"use_relative_links"`  // Use relative paths for attachments
	FilenamePattern    string `yaml:"filename_pattern"`    // Output filename pattern
}

// FileCopyRule defines files to copy from a source worktree
type FileCopyRule struct {
	SourceWorktree string   `yaml:"source_worktree"`
	Files          []string `yaml:"files"`
}

// FileCopyConfig holds file copying rules for new worktrees
type FileCopyConfig struct {
	Rules []FileCopyRule `yaml:"rules,omitempty"`
}

// WorktreeConfig defines a persistent worktree configuration
type WorktreeConfig struct {
	Branch      string `yaml:"branch"`
	MergeInto   string `yaml:"merge_into,omitempty"`
	Description string `yaml:"description,omitempty"`
}

// Config represents the .gbm/config.yaml structure
type Config struct {
	DefaultBranch string                    `yaml:"default_branch"`
	WorktreesDir  string                    `yaml:"worktrees_dir"`
	Worktrees     map[string]WorktreeConfig `yaml:"worktrees,omitempty"`
	Jira          JiraConfig                `yaml:"jira,omitempty"`
	FileCopy      FileCopyConfig            `yaml:"file_copy,omitempty"`
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
		return "", ErrNotInGitRepository
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

// GetJiraFilters returns the configured JIRA filters with sensible defaults
func (s *Service) GetJiraFilters() jira.JiraFilters {
	config := s.GetConfig()
	filters := config.Jira.Filters

	// Apply default assignee if not specified
	if filters.Assignee == "" {
		filters.Assignee = "me"
	}

	return filters
}

// GetJiraAttachmentConfig returns the attachment configuration with defaults
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

// GetJiraMarkdownConfig returns the markdown configuration with defaults
func (s *Service) GetJiraMarkdownConfig() (includeComments, includeAttachments bool) {
	config := s.GetConfig()
	mdConfig := config.Jira.Markdown

	// Default to true if not explicitly configured
	includeComments = mdConfig.IncludeComments
	includeAttachments = mdConfig.IncludeAttachments

	// If no config provided, default to true
	if mdConfig == (MarkdownConfig{}) {
		includeComments = true
		includeAttachments = true
	}

	// If attachments are disabled in config, don't include them
	if !config.Jira.Attachments.Enabled && config.Jira.Attachments != (AttachmentConfig{}) {
		includeAttachments = false
	}

	return includeComments, includeAttachments
}

// SaveConfig writes the current configuration to .gbm/config.yaml
func (s *Service) SaveConfig() error {
	if s.RepoRoot == "" {
		return ErrNotInGitRepository
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

// SaveState writes the current state to .gbm/state.yaml
func (s *Service) SaveState() error {
	if s.RepoRoot == "" {
		return ErrNotInGitRepository
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

// CopyFilesToWorktree copies files from source worktrees to the target worktree
// based on the file copy rules in the config
func (s *Service) CopyFilesToWorktree(targetWorktreeName string) error {
	config := s.GetConfig()
	if len(config.FileCopy.Rules) == 0 {
		return nil // No rules configured
	}

	if s.RepoRoot == "" {
		return ErrNotInGitRepository
	}

	targetWorktreePath := filepath.Join(s.RepoRoot, s.WorktreeDir, targetWorktreeName)

	for _, rule := range config.FileCopy.Rules {
		sourceWorktreePath := filepath.Join(s.RepoRoot, s.WorktreeDir, rule.SourceWorktree)

		// Check if source worktree exists
		if _, err := os.Stat(sourceWorktreePath); os.IsNotExist(err) {
			fmt.Printf("Warning: source worktree '%s' does not exist, skipping file copy rule\n", rule.SourceWorktree)
			continue
		}

		for _, filePattern := range rule.Files {
			if err := s.copyFileOrDirectory(sourceWorktreePath, targetWorktreePath, filePattern); err != nil {
				fmt.Printf("Warning: failed to copy '%s' from '%s': %v\n", filePattern, rule.SourceWorktree, err)
			}
		}
	}

	return nil
}

// copyFileOrDirectory copies a file or directory from source to target
func (s *Service) copyFileOrDirectory(sourceWorktreePath, targetWorktreePath, filePattern string) error {
	sourcePath := filepath.Join(sourceWorktreePath, filePattern)
	targetPath := filepath.Join(targetWorktreePath, filePattern)

	sourceInfo, err := os.Stat(sourcePath)
	if os.IsNotExist(err) {
		return fmt.Errorf("source file/directory '%s' does not exist", sourcePath)
	}
	if err != nil {
		return fmt.Errorf("failed to stat source path: %w", err)
	}

	if sourceInfo.IsDir() {
		return s.copyDirectory(sourcePath, targetPath)
	}
	return s.copyFile(sourcePath, targetPath)
}

// copyFile copies a single file from source to target
func (s *Service) copyFile(sourcePath, targetPath string) error {
	// Create target directory if it doesn't exist
	targetDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Check if target file already exists
	if _, err := os.Stat(targetPath); err == nil {
		fmt.Printf("File '%s' already exists in target worktree, skipping\n", filepath.Base(targetPath))
		return nil
	}

	// Get source file info for permissions
	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to get source file info: %w", err)
	}

	// Read source content
	content, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to read source file: %w", err)
	}

	// Write to target with same permissions
	if err := os.WriteFile(targetPath, content, sourceInfo.Mode()); err != nil {
		return fmt.Errorf("failed to write target file: %w", err)
	}

	return nil
}

// copyDirectory recursively copies a directory from source to target
func (s *Service) copyDirectory(sourcePath, targetPath string) error {
	// Create target directory
	if err := os.MkdirAll(targetPath, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Read source directory
	entries, err := os.ReadDir(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to read source directory: %w", err)
	}

	for _, entry := range entries {
		sourceEntryPath := filepath.Join(sourcePath, entry.Name())
		targetEntryPath := filepath.Join(targetPath, entry.Name())

		if entry.IsDir() {
			if err := s.copyDirectory(sourceEntryPath, targetEntryPath); err != nil {
				return err
			}
		} else {
			if err := s.copyFile(sourceEntryPath, targetEntryPath); err != nil {
				return err
			}
		}
	}

	return nil
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
	includeComments, includeAttachments := s.GetJiraMarkdownConfig()

	// Use the enhanced markdown generation with configuration
	opts := jira.DefaultIssueMarkdownOptions(worktreePath)
	opts.AttachmentConfig = attachmentConfig
	opts.DownloadAttachments = includeAttachments
	opts.IncludeComments = includeComments
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

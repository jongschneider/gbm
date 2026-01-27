package service

import (
	"fmt"
	"gbm/internal/git"
	"os"
	"path/filepath"
	"strings"
)

// CopyFilesToWorktree copies files from source worktrees to the target worktree.
// Uses both rule-based and automatic file copying based on config.
//
// This function applies two copying strategies in order:
// 1. Automatic copying: If enabled, copies .gitignore'd and untracked files
// 2. Rule-based copying: Copies files/directories specified in file_copy rules
//
// Parameters:
//   - targetWorktreeName: Name of the worktree where files will be copied
//
// Returns:
//   - error: Non-nil if any file operations fail
//
// Example:
//
//	// After creating a worktree, copy .env and config files
//	if err := svc.CopyFilesToWorktree("feature-x"); err != nil {
//	    return err
//	}
func (s *Service) CopyFilesToWorktree(targetWorktreeName string) error {
	config := s.GetConfig()

	if s.RepoRoot == "" {
		return ErrNotInGitRepository
	}

	targetWorktreePath := filepath.Join(s.RepoRoot, s.WorktreeDir, targetWorktreeName)

	// Phase 1: Automatic copying (if enabled)
	if config.FileCopy.Auto.Enabled {
		err := s.autoCopyFiles(targetWorktreeName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: automatic file copy failed: %v\n", err)
		}
	}

	// Phase 2: Explicit rules (existing behavior)
	if len(config.FileCopy.Rules) > 0 {
		for _, rule := range config.FileCopy.Rules {
			sourceWorktreePath := filepath.Join(s.RepoRoot, s.WorktreeDir, rule.SourceWorktree)

			// Check if source worktree exists
			if _, err := os.Stat(sourceWorktreePath); os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "Warning: source worktree '%s' does not exist, skipping file copy rule\n", rule.SourceWorktree)
				continue
			}

			for _, filePattern := range rule.Files {
				err := s.copyFileOrDirectory(sourceWorktreePath, targetWorktreePath, filePattern)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to copy '%s' from '%s': %v\n", filePattern, rule.SourceWorktree, err)
				}
			}
		}
	}

	return nil
}

// resolveSourceWorktree determines the actual source worktree based on the config specification
// Template values: "{default}" or "" → worktree with DefaultBranch
//
//	"{current}" → current worktree
//	other → literal worktree name
func (s *Service) resolveSourceWorktree(sourceSpec string) (*git.Worktree, error) {
	config := s.GetConfig()

	// Determine what worktree to use
	switch sourceSpec {
	case "", "{default}":
		// Find worktree associated with DefaultBranch
		worktrees, err := s.Git.ListWorktrees(false)
		if err != nil {
			return nil, err
		}

		defaultBranch := config.DefaultBranch
		if defaultBranch == "" {
			defaultBranch = "master" // Ultimate fallback
		}

		for _, wt := range worktrees {
			if wt.Branch == defaultBranch {
				return &wt, nil
			}
		}

		// Fallback: use current worktree with warning
		fmt.Fprintf(os.Stderr, "Warning: No worktree found for default branch '%s', using current worktree\n", defaultBranch)
		return s.Git.GetCurrentWorktree()

	case "{current}":
		return s.Git.GetCurrentWorktree()

	default:
		// Literal worktree name
		worktrees, err := s.Git.ListWorktrees(false)
		if err != nil {
			return nil, err
		}

		for _, wt := range worktrees {
			if wt.Name == sourceSpec {
				return &wt, nil
			}
		}

		return nil, fmt.Errorf("worktree '%s' not found", sourceSpec)
	}
}

// autoCopyFiles automatically copies ignored and untracked files from source to target worktree.
func (s *Service) autoCopyFiles(targetWorktreeName string) error {
	config := s.GetConfig()

	// Resolve source worktree using template expansion
	sourceWorktree, err := s.resolveSourceWorktree(config.FileCopy.Auto.SourceWorktree)
	if err != nil {
		return err
	}

	// Use map to collect unique files (avoid duplicates if file appears in both ignored and untracked)
	fileMap := make(map[string]struct{})

	if config.FileCopy.Auto.CopyIgnored {
		ignored, err := s.Git.ListIgnoredFiles(sourceWorktree.Path)
		if err == nil {
			for _, f := range ignored {
				fileMap[f] = struct{}{}
			}
		}
	}
	if config.FileCopy.Auto.CopyUntracked {
		untracked, err := s.Git.ListUntrackedFiles(sourceWorktree.Path)
		if err == nil {
			for _, f := range untracked {
				fileMap[f] = struct{}{}
			}
		}
	}

	// Convert map keys to slice
	var files []string
	for f := range fileMap {
		files = append(files, f)
	}

	// Filter by exclude patterns
	filtered := filterFiles(files, config.FileCopy.Auto.Exclude)

	// Copy files
	targetPath := filepath.Join(s.RepoRoot, s.WorktreeDir, targetWorktreeName)
	for _, file := range filtered {
		err := s.copyFile(
			filepath.Join(sourceWorktree.Path, file),
			filepath.Join(targetPath, file),
		)
		if err != nil {
			// Skip files that fail to copy (e.g., permission issues)
			continue
		}
	}

	return nil
}

// filterFiles removes files that match any of the exclude patterns.
func filterFiles(files, excludePatterns []string) []string {
	if len(excludePatterns) == 0 {
		return files
	}

	var filtered []string
	for _, file := range files {
		if !matchesAnyPattern(file, excludePatterns) {
			filtered = append(filtered, file)
		}
	}
	return filtered
}

// matchesAnyPattern checks if a path matches any of the exclude patterns.
func matchesAnyPattern(path string, patterns []string) bool {
	for _, pattern := range patterns {
		if matchGlob(path, pattern) {
			return true
		}
	}
	return false
}

// matchGlob implements simple glob matching (supports * and /).
func matchGlob(path, pattern string) bool {
	// Simple glob matching: * matches everything in a path component
	parts := filepath.SplitList(path)
	patternParts := filepath.SplitList(pattern)

	// Handle ** (matches anything)
	if pattern == "**" || pattern == "*" {
		return true
	}

	// Check if pattern matches the file name
	if len(patternParts) == 1 && len(parts) >= 1 {
		return globMatch(filepath.Base(path), patternParts[0])
	}

	return false
}

// globMatch implements simple glob matching for a single component.
func globMatch(name, pattern string) bool {
	// Handle exact match
	if pattern == name {
		return true
	}

	// Handle * wildcard
	if pattern == "*" {
		return true
	}

	// Handle patterns like "*.log" or "node_modules"
	if strings.Contains(pattern, "*") {
		// Simple wildcard matching
		parts := strings.Split(pattern, "*")
		if len(parts) == 2 {
			prefix, suffix := parts[0], parts[1]
			return strings.HasPrefix(name, prefix) && strings.HasSuffix(name, suffix)
		}
	}

	// Handle directory patterns like "node_modules/"
	if before, ok := strings.CutSuffix(pattern, "/"); ok {
		return strings.HasSuffix(name, before)
	}

	return false
}

// copyFileOrDirectory copies a file or directory from source to target.
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

// copyFile copies a single file from source to target.
func (s *Service) copyFile(sourcePath, targetPath string) error {
	// Create target directory if it doesn't exist
	targetDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
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

// copyDirectory recursively copies a directory from source to target.
func (s *Service) copyDirectory(sourcePath, targetPath string) error {
	// Create target directory
	if err := os.MkdirAll(targetPath, 0o755); err != nil {
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
			err := s.copyDirectory(sourceEntryPath, targetEntryPath)
			if err != nil {
				return err
			}
		} else {
			err := s.copyFile(sourceEntryPath, targetEntryPath)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

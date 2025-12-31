package jira

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// AttachmentConfig holds configuration for attachment downloads
type AttachmentConfig struct {
	MaxSizeMB      int64         // Maximum file size in MB
	Timeout        time.Duration // HTTP timeout
	RetryAttempts  int           // Number of retry attempts
	RetryBackoffMs int           // Initial backoff in milliseconds
}

// DefaultAttachmentConfig returns default attachment configuration
func DefaultAttachmentConfig() AttachmentConfig {
	return AttachmentConfig{
		MaxSizeMB:      50,
		Timeout:        30 * time.Second,
		RetryAttempts:  3,
		RetryBackoffMs: 1000,
	}
}

// AttachmentService handles downloading JIRA attachments
type AttachmentService struct {
	config AttachmentConfig
	client *http.Client
}

// NewAttachmentService creates a new attachment service
func NewAttachmentService(config AttachmentConfig) *AttachmentService {
	return &AttachmentService{
		config: config,
		client: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// DownloadResult represents the result of a download operation
type DownloadResult struct {
	Attachment Attachment
	LocalPath  string
	Skipped    bool
	SkipReason string
	Error      error
}

// DownloadAttachment downloads a single attachment to the specified directory
// Returns the local path relative to the base directory
func (s *AttachmentService) DownloadAttachment(
	attachment Attachment,
	destDir string,
	baseDir string, // For calculating relative paths
) (*DownloadResult, error) {
	result := &DownloadResult{
		Attachment: attachment,
	}

	// Check file size limit
	maxBytes := s.config.MaxSizeMB * 1024 * 1024
	if attachment.Size > maxBytes {
		result.Skipped = true
		result.SkipReason = fmt.Sprintf("exceeds size limit (%d MB > %d MB)",
			attachment.Size/(1024*1024), s.config.MaxSizeMB)
		return result, nil
	}

	// Create destination directory
	if err := os.MkdirAll(destDir, 0755); err != nil {
		result.Error = fmt.Errorf("failed to create directory: %w", err)
		return result, result.Error
	}

	// Sanitize filename
	sanitized := sanitizeFilename(attachment.Filename)

	// Handle filename collisions
	destPath := filepath.Join(destDir, sanitized)
	destPath = s.resolveFilenameCollision(destPath)

	// Download with retries
	err := s.downloadWithRetry(attachment.Content, destPath)
	if err != nil {
		result.Error = fmt.Errorf("failed to download %s: %w", attachment.Filename, err)
		return result, result.Error
	}

	// Calculate relative path from base directory
	relPath, err := filepath.Rel(baseDir, destPath)
	if err != nil {
		// Fallback to absolute path if relative calculation fails
		relPath = destPath
	}

	result.LocalPath = relPath
	return result, nil
}

// DownloadAllAttachments downloads all attachments for a ticket
// Returns successful downloads, skipped files, and any errors
func (s *AttachmentService) DownloadAllAttachments(
	attachments []Attachment,
	destDir string,
	baseDir string,
) ([]DownloadResult, error) {
	results := make([]DownloadResult, 0, len(attachments))
	var firstError error

	for _, attachment := range attachments {
		result, err := s.DownloadAttachment(attachment, destDir, baseDir)
		if result != nil {
			results = append(results, *result)
		}

		// Track first error but continue processing
		if err != nil && firstError == nil {
			firstError = err
		}
	}

	return results, firstError
}

// downloadWithRetry downloads a file with exponential backoff retry
func (s *AttachmentService) downloadWithRetry(url, destPath string) error {
	var lastErr error
	backoff := time.Duration(s.config.RetryBackoffMs) * time.Millisecond

	for attempt := 0; attempt < s.config.RetryAttempts; attempt++ {
		if attempt > 0 {
			time.Sleep(backoff)
			backoff *= 2 // Exponential backoff
		}

		err := s.downloadFile(url, destPath)
		if err == nil {
			return nil
		}

		lastErr = err
	}

	return fmt.Errorf("failed after %d attempts: %w", s.config.RetryAttempts, lastErr)
}

// downloadFile downloads a single file from URL to destination path
func (s *AttachmentService) downloadFile(url, destPath string) error {
	// Create HTTP request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Note: JIRA CLI handles authentication through session/cookies
	// The attachment URLs are authenticated via the JIRA session
	// If authentication is needed, it would be handled by the jira CLI setup

	// Execute request
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("failed to close response body: %w", closeErr)
		}
	}()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Create destination file
	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() {
		if closeErr := out.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("failed to close output file: %w", closeErr)
		}
	}()

	// Stream the download to avoid loading large files into memory
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		// Clean up partial file on error
		_ = os.Remove(destPath) // Ignore error on cleanup
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// resolveFilenameCollision appends a counter if file already exists
func (s *AttachmentService) resolveFilenameCollision(path string) string {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return path
	}

	// File exists, find a unique name
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	nameWithoutExt := strings.TrimSuffix(base, ext)

	counter := 1
	for {
		newName := fmt.Sprintf("%s-%d%s", nameWithoutExt, counter, ext)
		newPath := filepath.Join(dir, newName)
		if _, err := os.Stat(newPath); os.IsNotExist(err) {
			return newPath
		}
		counter++
	}
}

// sanitizeFilename removes or replaces invalid characters in filenames
func sanitizeFilename(filename string) string {
	// Replace path separators and other dangerous characters
	// Keep alphanumeric, spaces, dots, dashes, and underscores
	reg := regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)
	sanitized := reg.ReplaceAllString(filename, "_")

	// Remove leading/trailing dots and spaces
	sanitized = strings.Trim(sanitized, ". ")

	// Ensure filename is not empty
	if sanitized == "" {
		sanitized = "attachment"
	}

	// Limit length (max 255 characters is typical filesystem limit)
	if len(sanitized) > 255 {
		ext := filepath.Ext(sanitized)
		nameLen := 255 - len(ext)
		if nameLen > 0 {
			sanitized = sanitized[:nameLen] + ext
		} else {
			sanitized = sanitized[:255]
		}
	}

	return sanitized
}

// FormatAttachmentSize formats bytes into human-readable size
func FormatAttachmentSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

package jira

import (
	"fmt"
	"os"
	"path/filepath"
)

// IssueMarkdownOptions configures how issue markdown is generated and saved
type IssueMarkdownOptions struct {
	// WorktreeRoot is the root directory of the worktree
	WorktreeRoot string

	// DownloadAttachments enables attachment downloads
	DownloadAttachments bool

	// AttachmentConfig configures attachment downloads
	AttachmentConfig AttachmentConfig

	// IncludeComments includes all comments in markdown
	IncludeComments bool

	// Filename is the output markdown filename (default: {key}.md)
	Filename string
}

// IssueMarkdownResult contains the results of generating issue markdown
type IssueMarkdownResult struct {
	// MarkdownPath is the path to the generated markdown file
	MarkdownPath string

	// AttachmentResults contains results for each attachment download
	AttachmentResults []DownloadResult

	// AttachmentsDownloaded is the count of successfully downloaded attachments
	AttachmentsDownloaded int

	// AttachmentsSkipped is the count of skipped attachments
	AttachmentsSkipped int

	// AttachmentsFailed is the count of failed attachment downloads
	AttachmentsFailed int
}

// DefaultIssueMarkdownOptions returns default options for markdown generation
func DefaultIssueMarkdownOptions(worktreeRoot string) IssueMarkdownOptions {
	return IssueMarkdownOptions{
		WorktreeRoot:        worktreeRoot,
		DownloadAttachments: true,
		AttachmentConfig:    DefaultAttachmentConfig(),
		IncludeComments:     true,
		Filename:            "", // Will use {key}.md by default
	}
}

// GenerateIssueMarkdownFile is a high-level function that:
// 1. Fetches full issue details from JIRA
// 2. Downloads attachments (if enabled)
// 3. Generates comprehensive markdown
// 4. Saves markdown file to worktree
//
// This is the main integration point for worktree creation flows.
func (s *Service) GenerateIssueMarkdownFile(
	issueKey string,
	opts IssueMarkdownOptions,
	dryRun bool,
) (*IssueMarkdownResult, error) {
	result := &IssueMarkdownResult{}

	// Step 1: Fetch full issue details
	details, err := s.GetJiraIssue(issueKey, dryRun)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch issue %s: %w", issueKey, err)
	}

	// Step 2: Download attachments if enabled
	var attachmentResults []DownloadResult
	if opts.DownloadAttachments && len(details.Attachments) > 0 {
		attachmentDir := GenerateAttachmentPath(opts.WorktreeRoot, issueKey)

		if !dryRun {
			attachmentService := NewAttachmentService(opts.AttachmentConfig)
			attachmentResults, err = attachmentService.DownloadAllAttachments(
				details.Attachments,
				attachmentDir,
				opts.WorktreeRoot,
			)
			if err != nil {
				// Log warning but continue - attachment downloads are not critical
				fmt.Fprintf(os.Stderr, "Warning: some attachments failed to download: %v\n", err)
			}

			// Count results
			for _, ar := range attachmentResults {
				if ar.Skipped {
					result.AttachmentsSkipped++
				} else if ar.Error != nil {
					result.AttachmentsFailed++
				} else {
					result.AttachmentsDownloaded++
				}
			}
		} else {
			fmt.Printf("[DRY RUN] Would download %d attachments to %s\n",
				len(details.Attachments), attachmentDir)
		}
	}

	result.AttachmentResults = attachmentResults

	// Step 3: Generate markdown
	generator := NewMarkdownGenerator()
	markdown, err := generator.GenerateIssueMarkdown(details, MarkdownOptions{
		IncludeComments:    opts.IncludeComments,
		IncludeAttachments: opts.DownloadAttachments,
		AttachmentBaseDir:  opts.WorktreeRoot,
		AttachmentResults:  attachmentResults,
		UseRelativeLinks:   true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate markdown: %w", err)
	}

	// Step 4: Determine output filename
	filename := opts.Filename
	if filename == "" {
		filename = fmt.Sprintf("%s.md", issueKey)
	}
	markdownPath := filepath.Join(opts.WorktreeRoot, filename)

	// Step 5: Write markdown file
	if !dryRun {
		if err := os.WriteFile(markdownPath, []byte(markdown), 0644); err != nil {
			return nil, fmt.Errorf("failed to write markdown file: %w", err)
		}
	} else {
		fmt.Printf("[DRY RUN] Would write markdown to %s\n", markdownPath)
	}

	result.MarkdownPath = markdownPath
	return result, nil
}

// PrintMarkdownResult prints a summary of the markdown generation results
func PrintMarkdownResult(result *IssueMarkdownResult) {
	fmt.Printf("✓ Markdown generated: %s\n", result.MarkdownPath)

	if len(result.AttachmentResults) > 0 {
		fmt.Printf("  Attachments: %d downloaded, %d skipped, %d failed\n",
			result.AttachmentsDownloaded,
			result.AttachmentsSkipped,
			result.AttachmentsFailed,
		)

		// Show details for skipped and failed attachments
		for _, ar := range result.AttachmentResults {
			if ar.Skipped {
				fmt.Printf("  ⚠ %s - skipped (%s)\n", ar.Attachment.Filename, ar.SkipReason)
			} else if ar.Error != nil {
				fmt.Printf("  ✗ %s - failed: %v\n", ar.Attachment.Filename, ar.Error)
			}
		}
	}
}

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

	// IncludeLinkedIssues enables processing of linked issues
	IncludeLinkedIssues bool

	// MaxDepth controls how deep to traverse linked issues
	// Depth 1 = main ticket only
	// Depth 2 = main ticket + its direct linked issues (default)
	// Depth 3 = main ticket + linked issues + their linked issues
	// etc.
	MaxDepth int
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

	// LinkedIssueResults contains results for linked issues
	LinkedIssueResults map[string]*IssueMarkdownResult
}

// DefaultIssueMarkdownOptions returns default options for markdown generation
func DefaultIssueMarkdownOptions(worktreeRoot string) IssueMarkdownOptions {
	return IssueMarkdownOptions{
		WorktreeRoot:        worktreeRoot,
		DownloadAttachments: true,
		AttachmentConfig:    DefaultAttachmentConfig(),
		IncludeComments:     true,
		Filename:            "", // Will use {key}.md by default
		IncludeLinkedIssues: true,
		MaxDepth:            2, // Default: main ticket + direct linked issues
	}
}

// GenerateIssueMarkdownFile is a high-level function that:
// 1. Fetches full issue details from JIRA
// 2. Downloads attachments (if enabled)
// 3. Generates comprehensive markdown
// 4. Saves markdown file to worktree
// 5. Recursively processes linked issues (up to MaxDepth)
//
// This is the main integration point for worktree creation flows.
func (s *Service) GenerateIssueMarkdownFile(
	issueKey string,
	opts IssueMarkdownOptions,
	dryRun bool,
) (*IssueMarkdownResult, error) {
	// Initialize tracking map to prevent circular dependencies and duplicate processing
	processedIssues := make(map[string]bool)

	// Start at depth 1 for the main ticket
	return s.generateIssueMarkdownFileWithDepth(issueKey, opts, dryRun, 1, processedIssues)
}

// generateIssueMarkdownFileWithDepth is the internal implementation with depth tracking
// and circular dependency prevention
func (s *Service) generateIssueMarkdownFileWithDepth(
	issueKey string,
	opts IssueMarkdownOptions,
	dryRun bool,
	currentDepth int,
	processedIssues map[string]bool,
) (*IssueMarkdownResult, error) {
	result := &IssueMarkdownResult{}

	// Check if we've already processed this issue (circular dependency prevention)
	if processedIssues[issueKey] {
		if !dryRun {
			fmt.Fprintf(os.Stderr, "  Skipping %s (already processed - circular dependency detected)\n", issueKey)
		}
		return result, nil
	}

	// Mark this issue as being processed
	processedIssues[issueKey] = true

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
		// Ensure parent directory exists
		markdownDir := filepath.Dir(markdownPath)
		if err := os.MkdirAll(markdownDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create markdown directory: %w", err)
		}

		if err := os.WriteFile(markdownPath, []byte(markdown), 0644); err != nil {
			return nil, fmt.Errorf("failed to write markdown file: %w", err)
		}
	} else {
		fmt.Printf("[DRY RUN] Would write markdown to %s\n", markdownPath)
	}

	result.MarkdownPath = markdownPath

	// Step 6: Process linked issues if enabled and within depth limit

	// Early return if linked issues disabled
	if !opts.IncludeLinkedIssues {
		return result, nil
	}

	// Early return if no linked issues
	if len(details.IssueLinks) == 0 {
		return result, nil
	}

	// Early return if depth limit reached
	if currentDepth >= opts.MaxDepth {
		if !dryRun {
			fmt.Fprintf(os.Stderr, "  Skipping %d linked issues (depth limit %d reached)\n",
				len(details.IssueLinks), opts.MaxDepth)
		}
		return result, nil
	}

	// Process linked issues
	result.LinkedIssueResults = make(map[string]*IssueMarkdownResult)

	if !dryRun {
		fmt.Fprintf(os.Stderr, "  Processing %d linked issues at depth %d/%d\n",
			len(details.IssueLinks), currentDepth+1, opts.MaxDepth)
	}

	for _, link := range details.IssueLinks {
		// Determine which linked issue to process
		var linkedKey string
		if link.InwardIssue != nil {
			linkedKey = link.InwardIssue.Key
		} else if link.OutwardIssue != nil {
			linkedKey = link.OutwardIssue.Key
		}

		if linkedKey == "" {
			continue
		}

		// Skip if already processed (prevents circular dependencies and duplicates)
		if processedIssues[linkedKey] {
			if !dryRun {
				fmt.Fprintf(os.Stderr, "  Skipping %s (already processed)\n", linkedKey)
			}
			continue
		}

		// Create options for linked issue
		linkedOpts := IssueMarkdownOptions{
			WorktreeRoot:        opts.WorktreeRoot,
			DownloadAttachments: opts.DownloadAttachments,
			AttachmentConfig:    opts.AttachmentConfig,
			IncludeComments:     opts.IncludeComments,
			Filename:            fmt.Sprintf(".jira/%s.md", linkedKey),
			IncludeLinkedIssues: opts.IncludeLinkedIssues,
			MaxDepth:            opts.MaxDepth,
		}

		// Process linked issue at next depth level
		linkedResult, err := s.generateIssueMarkdownFileWithDepth(
			linkedKey,
			linkedOpts,
			dryRun,
			currentDepth+1,
			processedIssues, // Pass the tracking map to prevent circular dependencies
		)
		if err != nil {
			// Log warning but continue - linked issue failures are not critical
			fmt.Fprintf(os.Stderr, "Warning: failed to process linked issue %s at depth %d: %v\n",
				linkedKey, currentDepth+1, err)
			continue
		}

		result.LinkedIssueResults[linkedKey] = linkedResult
	}

	// Step 7: Process parent issue if exists and within depth limit
	if details.Parent != nil && currentDepth < opts.MaxDepth {
		parentKey := details.Parent.Key

		// Skip if already processed (prevents circular dependencies)
		if processedIssues[parentKey] {
			if !dryRun {
				fmt.Fprintf(os.Stderr, "  Skipping parent %s (already processed)\n", parentKey)
			}
		} else {
			if !dryRun {
				fmt.Fprintf(os.Stderr, "  Processing parent issue %s at depth %d/%d\n",
					parentKey, currentDepth+1, opts.MaxDepth)
			}

			// Create options for parent issue
			parentOpts := IssueMarkdownOptions{
				WorktreeRoot:        opts.WorktreeRoot,
				DownloadAttachments: opts.DownloadAttachments,
				AttachmentConfig:    opts.AttachmentConfig,
				IncludeComments:     opts.IncludeComments,
				Filename:            fmt.Sprintf(".jira/%s.md", parentKey),
				IncludeLinkedIssues: opts.IncludeLinkedIssues,
				MaxDepth:            opts.MaxDepth,
			}

			// Process parent issue at next depth level
			parentResult, err := s.generateIssueMarkdownFileWithDepth(
				parentKey,
				parentOpts,
				dryRun,
				currentDepth+1,
				processedIssues, // Pass the tracking map to prevent circular dependencies
			)
			if err != nil {
				// Log warning but continue - parent issue failure is not critical
				fmt.Fprintf(os.Stderr, "Warning: failed to process parent issue %s at depth %d: %v\n",
					parentKey, currentDepth+1, err)
			} else {
				result.LinkedIssueResults[parentKey] = parentResult
			}
		}
	}

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

	// Show linked issues
	if len(result.LinkedIssueResults) > 0 {
		fmt.Printf("  Linked issues: %d processed\n", len(result.LinkedIssueResults))
		for key, linkedResult := range result.LinkedIssueResults {
			fmt.Printf("    ✓ %s → %s\n", key, linkedResult.MarkdownPath)
			if linkedResult.AttachmentsDownloaded > 0 || linkedResult.AttachmentsSkipped > 0 || linkedResult.AttachmentsFailed > 0 {
				fmt.Printf("      Attachments: %d downloaded, %d skipped, %d failed\n",
					linkedResult.AttachmentsDownloaded,
					linkedResult.AttachmentsSkipped,
					linkedResult.AttachmentsFailed,
				)
			}
		}
	}
}

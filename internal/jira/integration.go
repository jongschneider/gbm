package jira

import (
	"fmt"
	"os"
	"path/filepath"
)

// IssueMarkdownOptions configures how issue markdown is generated and saved.
type IssueMarkdownOptions struct {
	WorktreeRoot        string
	Filename            string
	AttachmentConfig    AttachmentConfig
	MaxDepth            int
	DownloadAttachments bool
	IncludeComments     bool
	IncludeLinkedIssues bool
}

// IssueMarkdownResult contains the results of generating issue markdown.
type IssueMarkdownResult struct {
	LinkedIssueResults    map[string]*IssueMarkdownResult
	MarkdownPath          string
	AttachmentResults     []DownloadResult
	AttachmentsDownloaded int
	AttachmentsSkipped    int
	AttachmentsFailed     int
}

// DefaultIssueMarkdownOptions returns default options for markdown generation.
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
// and circular dependency prevention.
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
	processedIssues[issueKey] = true

	// Fetch issue details
	details, err := s.GetJiraIssue(issueKey, dryRun)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch issue %s: %w", issueKey, err)
	}

	// Download attachments
	attachmentResults := s.downloadAttachments(issueKey, details, opts, dryRun, result)

	// Generate and write markdown
	if err := s.generateAndWriteMarkdown(issueKey, details, opts, attachmentResults, dryRun, result); err != nil {
		return nil, err
	}

	// Process related issues
	result.LinkedIssueResults = make(map[string]*IssueMarkdownResult)
	s.processRelatedIssues(details, opts, dryRun, currentDepth, processedIssues, result)

	return result, nil
}

// downloadAttachments downloads attachments for an issue if enabled.
func (s *Service) downloadAttachments(
	issueKey string,
	details *JiraTicketDetails,
	opts IssueMarkdownOptions,
	dryRun bool,
	result *IssueMarkdownResult,
) []DownloadResult {
	if !opts.DownloadAttachments || len(details.Attachments) == 0 {
		return nil
	}

	attachmentDir := GenerateAttachmentPath(opts.WorktreeRoot, issueKey)

	if dryRun {
		fmt.Printf("[DRY RUN] Would download %d attachments to %s\n", len(details.Attachments), attachmentDir)
		return nil
	}

	attachmentService := NewAttachmentService(opts.AttachmentConfig)
	attachmentResults, err := attachmentService.DownloadAllAttachments(
		details.Attachments, attachmentDir, opts.WorktreeRoot,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: some attachments failed to download: %v\n", err)
	}

	countAttachmentResults(attachmentResults, result)
	result.AttachmentResults = attachmentResults
	return attachmentResults
}

// countAttachmentResults counts downloaded, skipped, and failed attachments.
func countAttachmentResults(results []DownloadResult, result *IssueMarkdownResult) {
	for _, ar := range results {
		switch {
		case ar.Skipped:
			result.AttachmentsSkipped++
		case ar.Error != nil:
			result.AttachmentsFailed++
		default:
			result.AttachmentsDownloaded++
		}
	}
}

// generateAndWriteMarkdown generates the markdown content and writes it to a file.
func (s *Service) generateAndWriteMarkdown(
	issueKey string,
	details *JiraTicketDetails,
	opts IssueMarkdownOptions,
	attachmentResults []DownloadResult,
	dryRun bool,
	result *IssueMarkdownResult,
) error {
	generator := NewMarkdownGenerator()
	markdown, err := generator.GenerateIssueMarkdown(details, MarkdownOptions{
		IncludeComments:    opts.IncludeComments,
		IncludeAttachments: opts.DownloadAttachments,
		AttachmentBaseDir:  opts.WorktreeRoot,
		AttachmentResults:  attachmentResults,
		UseRelativeLinks:   true,
	})
	if err != nil {
		return fmt.Errorf("failed to generate markdown: %w", err)
	}

	filename := opts.Filename
	if filename == "" {
		filename = issueKey + ".md"
	}
	markdownPath := filepath.Join(opts.WorktreeRoot, filename)

	if dryRun {
		fmt.Printf("[DRY RUN] Would write markdown to %s\n", markdownPath)
	} else {
		err := os.MkdirAll(filepath.Dir(markdownPath), 0o755)
		if err != nil {
			return fmt.Errorf("failed to create markdown directory: %w", err)
		}
		err := os.WriteFile(markdownPath, []byte(markdown), 0o644)
		if err != nil {
			return fmt.Errorf("failed to write markdown file: %w", err)
		}
	}

	result.MarkdownPath = markdownPath
	return nil
}

// processRelatedIssues processes linked issues, parent, and children.
func (s *Service) processRelatedIssues(
	details *JiraTicketDetails,
	opts IssueMarkdownOptions,
	dryRun bool,
	currentDepth int,
	processedIssues map[string]bool,
	result *IssueMarkdownResult,
) {
	if currentDepth >= opts.MaxDepth {
		return
	}

	// Process linked issues
	if opts.IncludeLinkedIssues && len(details.IssueLinks) > 0 {
		s.processLinkedIssues(details.IssueLinks, opts, dryRun, currentDepth, processedIssues, result)
	}

	// Process parent
	if details.Parent != nil {
		s.processRelatedIssue(details.Parent.Key, "parent", opts, dryRun, currentDepth, processedIssues, result)
	}

	// Process children
	if len(details.Children) > 0 {
		if !dryRun {
			fmt.Fprintf(os.Stderr, "  Processing %d children at depth %d/%d\n",
				len(details.Children), currentDepth+1, opts.MaxDepth)
		}
		for _, child := range details.Children {
			s.processRelatedIssue(child.Key, "child", opts, dryRun, currentDepth, processedIssues, result)
		}
	}
}

// processLinkedIssues processes issue links.
func (s *Service) processLinkedIssues(
	issueLinks []IssueLink,
	opts IssueMarkdownOptions,
	dryRun bool,
	currentDepth int,
	processedIssues map[string]bool,
	result *IssueMarkdownResult,
) {
	if !dryRun {
		fmt.Fprintf(os.Stderr, "  Processing %d linked issues at depth %d/%d\n",
			len(issueLinks), currentDepth+1, opts.MaxDepth)
	}

	for _, link := range issueLinks {
		linkedKey := getLinkKeyFromIssueLink(link)
		if linkedKey == "" {
			continue
		}
		s.processRelatedIssue(linkedKey, "linked issue", opts, dryRun, currentDepth, processedIssues, result)
	}
}

// getLinkKeyFromIssueLink extracts the key from an issue link.
func getLinkKeyFromIssueLink(link IssueLink) string {
	if link.InwardIssue != nil {
		return link.InwardIssue.Key
	}
	if link.OutwardIssue != nil {
		return link.OutwardIssue.Key
	}
	return ""
}

// processRelatedIssue processes a single related issue (linked, parent, or child).
func (s *Service) processRelatedIssue(
	issueKey string,
	issueType string,
	opts IssueMarkdownOptions,
	dryRun bool,
	currentDepth int,
	processedIssues map[string]bool,
	result *IssueMarkdownResult,
) {
	if processedIssues[issueKey] {
		if !dryRun {
			fmt.Fprintf(os.Stderr, "  Skipping %s %s (already processed)\n", issueType, issueKey)
		}
		return
	}

	if !dryRun && issueType == "parent" {
		fmt.Fprintf(os.Stderr, "  Processing parent issue %s at depth %d/%d\n",
			issueKey, currentDepth+1, opts.MaxDepth)
	}

	childOpts := IssueMarkdownOptions{
		WorktreeRoot:        opts.WorktreeRoot,
		DownloadAttachments: opts.DownloadAttachments,
		AttachmentConfig:    opts.AttachmentConfig,
		IncludeComments:     opts.IncludeComments,
		Filename:            fmt.Sprintf(".jira/%s.md", issueKey),
		IncludeLinkedIssues: opts.IncludeLinkedIssues,
		MaxDepth:            opts.MaxDepth,
	}

	childResult, err := s.generateIssueMarkdownFileWithDepth(
		issueKey, childOpts, dryRun, currentDepth+1, processedIssues,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to process %s %s at depth %d: %v\n",
			issueType, issueKey, currentDepth+1, err)
		return
	}

	result.LinkedIssueResults[issueKey] = childResult
}

// PrintMarkdownResult prints a summary of the markdown generation results.
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

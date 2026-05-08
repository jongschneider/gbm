package jira

import (
	"fmt"
	"os"
	"path/filepath"
)

// IssueMarkdownOptions configures how issue markdown is generated and saved.
//
// Layout written under WorktreeRoot:
//
//	<WorktreeRoot>/.jira/<KEY>/<KEY>.md
//	<WorktreeRoot>/.jira/<KEY>/attachments/<file>
//	<WorktreeRoot>/.jira/<KEY>/linked/<LINKED_KEY>/<LINKED_KEY>.md
//	<WorktreeRoot>/.jira/<KEY>/linked/<LINKED_KEY>/attachments/<file>
type IssueMarkdownOptions struct {
	WorktreeRoot        string
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
	processedIssues := make(map[string]bool)
	ticketDir := RootTicketDir(opts.WorktreeRoot, issueKey)
	return s.generateIssueMarkdownFileWithDepth(issueKey, ticketDir, opts, dryRun, 1, processedIssues)
}

// generateIssueMarkdownFileWithDepth is the internal implementation with depth tracking
// and circular dependency prevention. ticketDir is the directory holding this ticket's
// bundle (markdown + attachments/ + linked/).
func (s *Service) generateIssueMarkdownFileWithDepth(
	issueKey string,
	ticketDir string,
	opts IssueMarkdownOptions,
	dryRun bool,
	currentDepth int,
	processedIssues map[string]bool,
) (*IssueMarkdownResult, error) {
	result := &IssueMarkdownResult{}

	if processedIssues[issueKey] {
		if !dryRun {
			fmt.Fprintf(os.Stderr, "  Skipping %s (already processed - circular dependency detected)\n", issueKey)
		}
		return result, nil
	}
	processedIssues[issueKey] = true

	details, err := s.GetJiraIssue(issueKey, dryRun)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch issue %s: %w", issueKey, err)
	}

	attachmentResults := s.downloadAttachments(ticketDir, details, opts, dryRun, result)

	if err := s.generateAndWriteMarkdown(issueKey, ticketDir, details, opts, attachmentResults, dryRun, result); err != nil {
		return nil, err
	}

	result.LinkedIssueResults = make(map[string]*IssueMarkdownResult)
	s.processRelatedIssues(details, ticketDir, opts, dryRun, currentDepth, processedIssues, result)

	return result, nil
}

// downloadAttachments downloads attachments for an issue if enabled. The
// destination is <ticketDir>/attachments and LocalPath in each result is
// relative to ticketDir, so the link works from <ticketDir>/<KEY>.md.
func (s *Service) downloadAttachments(
	ticketDir string,
	details *JiraTicketDetails,
	opts IssueMarkdownOptions,
	dryRun bool,
	result *IssueMarkdownResult,
) []DownloadResult {
	if !opts.DownloadAttachments || len(details.Attachments) == 0 {
		return nil
	}

	attachmentDir := filepath.Join(ticketDir, "attachments")

	if dryRun {
		fmt.Printf("[DRY RUN] Would download %d attachments to %s\n", len(details.Attachments), attachmentDir)
		return nil
	}

	attachmentService := NewAttachmentService(opts.AttachmentConfig)
	attachmentResults, err := attachmentService.DownloadAllAttachments(
		details.Attachments, attachmentDir, ticketDir,
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

// generateAndWriteMarkdown generates the markdown content and writes it to
// <ticketDir>/<issueKey>.md.
func (s *Service) generateAndWriteMarkdown(
	issueKey string,
	ticketDir string,
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
		AttachmentBaseDir:  ticketDir,
		AttachmentResults:  attachmentResults,
		UseRelativeLinks:   true,
	})
	if err != nil {
		return fmt.Errorf("failed to generate markdown: %w", err)
	}

	markdownPath := filepath.Join(ticketDir, issueKey+".md")

	if dryRun {
		fmt.Printf("[DRY RUN] Would write markdown to %s\n", markdownPath)
	} else {
		if err := os.MkdirAll(ticketDir, 0o755); err != nil {
			return fmt.Errorf("failed to create markdown directory: %w", err)
		}
		if err := os.WriteFile(markdownPath, []byte(markdown), 0o644); err != nil {
			return fmt.Errorf("failed to write markdown file: %w", err)
		}
	}

	result.MarkdownPath = markdownPath
	return nil
}

// processRelatedIssues processes linked issues, parent, and children. Each is
// written into <parentTicketDir>/linked/<KEY>/.
func (s *Service) processRelatedIssues(
	details *JiraTicketDetails,
	parentTicketDir string,
	opts IssueMarkdownOptions,
	dryRun bool,
	currentDepth int,
	processedIssues map[string]bool,
	result *IssueMarkdownResult,
) {
	if currentDepth >= opts.MaxDepth {
		return
	}

	if opts.IncludeLinkedIssues && len(details.IssueLinks) > 0 {
		s.processLinkedIssues(details.IssueLinks, parentTicketDir, opts, dryRun, currentDepth, processedIssues, result)
	}

	if details.Parent != nil {
		s.processRelatedIssue(details.Parent.Key, "parent", parentTicketDir, opts, dryRun, currentDepth, processedIssues, result)
	}

	if len(details.Children) > 0 {
		if !dryRun {
			fmt.Fprintf(os.Stderr, "  Processing %d children at depth %d/%d\n",
				len(details.Children), currentDepth+1, opts.MaxDepth)
		}
		for _, child := range details.Children {
			s.processRelatedIssue(child.Key, "child", parentTicketDir, opts, dryRun, currentDepth, processedIssues, result)
		}
	}
}

// processLinkedIssues processes issue links.
func (s *Service) processLinkedIssues(
	issueLinks []IssueLink,
	parentTicketDir string,
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
		s.processRelatedIssue(linkedKey, "linked issue", parentTicketDir, opts, dryRun, currentDepth, processedIssues, result)
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
// It places the child's bundle at <parentTicketDir>/linked/<issueKey>/.
func (s *Service) processRelatedIssue(
	issueKey string,
	issueType string,
	parentTicketDir string,
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

	childTicketDir := filepath.Join(parentTicketDir, "linked", issueKey)

	childResult, err := s.generateIssueMarkdownFileWithDepth(
		issueKey, childTicketDir, opts, dryRun, currentDepth+1, processedIssues,
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

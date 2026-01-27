package jira

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// MarkdownGenerator generates markdown documentation from JIRA issues.
type MarkdownGenerator struct {
	parser *ADFParser
}

// NewMarkdownGenerator creates a new markdown generator.
func NewMarkdownGenerator() *MarkdownGenerator {
	return &MarkdownGenerator{
		parser: NewADFParser(),
	}
}

// MarkdownOptions configures markdown generation.
type MarkdownOptions struct {
	AttachmentBaseDir  string
	AttachmentResults  []DownloadResult
	IncludeComments    bool
	IncludeAttachments bool
	UseRelativeLinks   bool
}

// GenerateIssueMarkdown creates a comprehensive markdown document for a JIRA issue.
func (g *MarkdownGenerator) GenerateIssueMarkdown(
	details *JiraTicketDetails,
	opts MarkdownOptions,
) (string, error) {
	var builder strings.Builder

	// Header with ticket key and summary
	builder.WriteString(fmt.Sprintf("# [%s] %s\n\n", details.Key, details.Summary))

	// Metadata section
	g.writeMetadata(&builder, details)

	// Description section
	if details.Description != "" {
		builder.WriteString("## Description\n\n")
		builder.WriteString(details.Description)
		builder.WriteString("\n\n")
	}

	// Epic section (if applicable)
	if details.Epic != "" {
		builder.WriteString("## Epic\n\n")
		builder.WriteString(fmt.Sprintf("**%s**\n\n", details.Epic))
	}

	// Labels section
	if len(details.Labels) > 0 {
		builder.WriteString("## Labels\n\n")
		for _, label := range details.Labels {
			builder.WriteString(fmt.Sprintf("- `%s`\n", label))
		}
		builder.WriteString("\n")
	}

	// Linked Issues section
	if len(details.IssueLinks) > 0 {
		g.writeLinkedIssues(&builder, details.IssueLinks)
	}

	// Attachments section
	if opts.IncludeAttachments && len(opts.AttachmentResults) > 0 {
		g.writeAttachments(&builder, opts.AttachmentResults, opts.AttachmentBaseDir)
	}

	// Comments section
	if opts.IncludeComments && len(details.Comments) > 0 {
		err := g.writeComments(&builder, details.Comments, opts)
		if err != nil {
			return "", fmt.Errorf("failed to write comments: %w", err)
		}
	}

	// Footer with JIRA link
	builder.WriteString("---\n\n")
	builder.WriteString(fmt.Sprintf("**JIRA Link**: [%s](%s)\n", details.Key, details.URL))

	return builder.String(), nil
}

// writeMetadata writes the metadata section.
func (g *MarkdownGenerator) writeMetadata(builder *strings.Builder, details *JiraTicketDetails) {
	builder.WriteString("## Metadata\n\n")
	builder.WriteString("| Field | Value |\n")
	builder.WriteString("|-------|-------|\n")
	fmt.Fprintf(builder, "| **Key** | %s |\n", details.Key)
	fmt.Fprintf(builder, "| **Status** | %s |\n", details.Status)
	fmt.Fprintf(builder, "| **Priority** | %s |\n", details.Priority)
	fmt.Fprintf(builder, "| **Assignee** | %s |\n", details.Assignee)
	fmt.Fprintf(builder, "| **Reporter** | %s |\n", details.Reporter)
	fmt.Fprintf(builder, "| **Created** | %s |\n", details.Created.Format("2006-01-02 15:04:05"))

	if details.DueDate != nil {
		fmt.Fprintf(builder, "| **Due Date** | %s |\n", details.DueDate.Format("2006-01-02"))
	}

	if details.Epic != "" {
		fmt.Fprintf(builder, "| **Epic** | %s |\n", details.Epic)
	}

	builder.WriteString("\n")
}

// writeLinkedIssues writes the linked issues section.
func (g *MarkdownGenerator) writeLinkedIssues(builder *strings.Builder, links []IssueLink) {
	builder.WriteString("## Linked Issues\n\n")

	for _, link := range links {
		var linkedIssue *LinkedIssue
		var relationship string

		// Determine the relationship and which issue to display
		if link.InwardIssue != nil {
			linkedIssue = link.InwardIssue
			relationship = link.Type.Inward // e.g., "is blocked by", "relates to"
		} else if link.OutwardIssue != nil {
			linkedIssue = link.OutwardIssue
			relationship = link.Type.Outward // e.g., "blocks", "is related to"
		}

		if linkedIssue == nil {
			continue
		}

		// Write the linked issue with relationship
		fmt.Fprintf(builder, "### %s [%s](./%s.md)\n\n",
			relationship,
			linkedIssue.Key,
			linkedIssue.Key,
		)

		// Create a metadata table for the linked issue
		builder.WriteString("| Field | Value |\n")
		builder.WriteString("|-------|-------|\n")
		fmt.Fprintf(builder, "| **Summary** | %s |\n", linkedIssue.Summary)
		fmt.Fprintf(builder, "| **Status** | %s |\n", linkedIssue.Status)
		fmt.Fprintf(builder, "| **Priority** | %s |\n", linkedIssue.Priority)
		fmt.Fprintf(builder, "| **Type** | %s |\n", linkedIssue.IssueType)

		builder.WriteString("\n")
	}
}

// writeAttachments writes the attachments section.
func (g *MarkdownGenerator) writeAttachments(
	builder *strings.Builder,
	results []DownloadResult,
	_ string, // baseDir - reserved for future use
) {
	builder.WriteString("## Attachments\n\n")

	for _, result := range results {
		attachment := result.Attachment

		if result.Skipped {
			// Show skipped attachments with reason
			fmt.Fprintf(builder, "- **%s** (%s) - ⚠️ Skipped: %s\n",
				attachment.Filename,
				FormatAttachmentSize(attachment.Size),
				result.SkipReason,
			)
			fmt.Fprintf(builder, "  - *Original URL*: <%s>\n", attachment.Content)
		} else if result.Error != nil {
			// Show failed downloads
			fmt.Fprintf(builder, "- **%s** (%s) - ❌ Failed: %s\n",
				attachment.Filename,
				FormatAttachmentSize(attachment.Size),
				result.Error.Error(),
			)
			fmt.Fprintf(builder, "  - *Original URL*: <%s>\n", attachment.Content)
		} else {
			// Show successful downloads with link
			sizeStr := FormatAttachmentSize(attachment.Size)
			uploadedBy := attachment.Author.DisplayName
			createdAt := formatDate(attachment.Created)

			// Use the local path from the download result
			localPath := result.LocalPath

			// For images, embed them; for other files, just link
			if g.isImageFile(attachment.Filename) {
				fmt.Fprintf(builder, "- ![%s](%s)\n",
					attachment.Filename,
					localPath,
				)
				fmt.Fprintf(builder, "  - **%s** (%s) - *Uploaded by %s on %s*\n",
					attachment.Filename,
					sizeStr,
					uploadedBy,
					createdAt,
				)
			} else {
				fmt.Fprintf(builder, "- [%s](%s) (%s) - *Uploaded by %s on %s*\n",
					attachment.Filename,
					localPath,
					sizeStr,
					uploadedBy,
					createdAt,
				)
			}
		}
	}

	builder.WriteString("\n")
}

// writeComments writes the comments section.
func (g *MarkdownGenerator) writeComments(
	builder *strings.Builder,
	comments []Comment,
	_ MarkdownOptions, // opts - reserved for future use
) error {
	builder.WriteString("## Comments\n\n")

	for i, comment := range comments {
		// Comment header with author and timestamp
		authorName := comment.Author.DisplayName
		if authorName == "" {
			authorName = "Unknown User"
		}

		timestamp := formatDate(comment.Created)
		fmt.Fprintf(builder, "### Comment by %s - %s\n\n", authorName, timestamp)

		// Parse ADF body to markdown
		if len(comment.Body.Content) > 0 {
			markdown, mediaIDs, err := g.parser.ParseToMarkdown(comment.Body)
			if err != nil {
				return fmt.Errorf("failed to parse comment %s: %w", comment.ID, err)
			}

			builder.WriteString(markdown)
			builder.WriteString("\n\n")

			// Include media attachments if any
			if len(mediaIDs) > 0 {
				builder.WriteString("**Media References**:\n")
				for _, mediaID := range mediaIDs {
					fmt.Fprintf(builder, "- Media ID: `%s`\n", mediaID)
				}
				builder.WriteString("\n")
			}
		} else if comment.Content != "" {
			// Fallback to legacy content field
			builder.WriteString(comment.Content)
			builder.WriteString("\n\n")
		}

		// Add separator between comments (except after the last one)
		if i < len(comments)-1 {
			builder.WriteString("---\n\n")
		}
	}

	return nil
}

// isImageFile checks if a filename is likely an image.
func (g *MarkdownGenerator) isImageFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	imageExtensions := map[string]bool{
		".png":  true,
		".jpg":  true,
		".jpeg": true,
		".gif":  true,
		".bmp":  true,
		".svg":  true,
		".webp": true,
	}
	return imageExtensions[ext]
}

// formatDate formats a date string to a readable format.
func formatDate(dateStr string) string {
	// Try parsing common JIRA date formats
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05.999-0700",
		"2006-01-02T15:04:05.999Z",
		"2006-01-02T15:04:05-0700",
		"2006-01-02 15:04:05",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t.Format("2006-01-02 15:04")
		}
	}

	// If parsing fails, return original string
	return dateStr
}

// GenerateAttachmentPath generates a path for an attachment
// Returns the directory path where attachments should be stored.
func GenerateAttachmentPath(worktreeRoot, ticketKey string) string {
	return filepath.Join(worktreeRoot, ".jira", "attachments", ticketKey)
}

package jira

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const (
	// issuesCacheTTL is how long we cache JIRA issues (5 minutes)
	issuesCacheTTL = 5 * time.Minute
)

// IssuesCache represents cached JIRA issues with timestamp
type IssuesCache struct {
	Issues    []JiraIssue `json:"issues"`
	Timestamp time.Time   `json:"timestamp"`
}

// buildIssueListArgs constructs jira issue list command arguments from filters
func buildIssueListArgs(filters JiraFilters, username string) []string {
	args := []string{"issue", "list"}

	// Assignee filter (defaults to current user)
	assignee := filters.Assignee
	if assignee == "" || assignee == "me" {
		args = append(args, "-a"+username)
	} else if assignee != "none" {
		args = append(args, "-a"+assignee)
	}

	// Status filters (can have multiple)
	for _, status := range filters.Status {
		args = append(args, "-s"+status)
	}

	// Priority filter
	if filters.Priority != "" {
		args = append(args, "-y"+filters.Priority)
	}

	// Type filter
	if filters.Type != "" {
		args = append(args, "-t"+filters.Type)
	}

	// Label filters (can have multiple)
	for _, label := range filters.Labels {
		args = append(args, "-l"+label)
	}

	// Component filter
	if filters.Component != "" {
		args = append(args, "-C"+filters.Component)
	}

	// Reporter filter
	if filters.Reporter != "" {
		args = append(args, "-r"+filters.Reporter)
	}

	// Order by
	if filters.OrderBy != "" {
		args = append(args, "--order-by", filters.OrderBy)
	}

	// Reverse order
	if filters.Reverse {
		args = append(args, "--reverse")
	}

	// Custom arguments
	args = append(args, filters.CustomArgs...)

	// Always request raw JSON output
	args = append(args, "--raw")

	return args
}

// GetJiraIssues fetches JIRA issues using configured filters
// Returns slice of typed JiraIssue structs
// Results are cached for 5 minutes to improve performance
func (s *Service) GetJiraIssues(filters JiraFilters, dryRun bool) ([]JiraIssue, error) {
	// Load cache and user from store
	var cache *IssuesCache
	var cachedUser string
	if s.store != nil {
		cache, cachedUser, _ = s.store.Load() // Ignore errors
	}

	// Try to use cache first if valid
	if cache != nil && len(cache.Issues) > 0 && time.Since(cache.Timestamp) < issuesCacheTTL {
		return cache.Issues, nil
	}

	// Get JIRA user (cached or fetch)
	user, _, err := s.GetJiraUser(cachedUser, dryRun)
	if err != nil {
		return nil, err
	}

	// Build command arguments from filters
	args := buildIssueListArgs(filters, user)

	// Execute command
	cmd := exec.Command("jira", args...)
	output, err := s.runCommand(cmd, dryRun)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JIRA issues: %w\nOutput: %s", err, string(output))
	}

	// Parse JSON response
	var rawIssues []jiraRawResponse
	if err := json.Unmarshal(output, &rawIssues); err != nil {
		return nil, fmt.Errorf("failed to parse JIRA issues JSON: %w", err)
	}

	// Convert to JiraIssue slice
	issues := make([]JiraIssue, 0, len(rawIssues))
	for _, raw := range rawIssues {
		issue := JiraIssue{
			Key:     raw.Key,
			Summary: raw.Fields.Summary,
			Status:  raw.Fields.Status.Name,
		}

		// Get issue type name
		if raw.Fields.IssueType.Name != "" {
			issue.Type = raw.Fields.IssueType.Name
		}

		issues = append(issues, issue)
	}

	// Create fresh cache with new data
	freshCache := &IssuesCache{
		Issues:    issues,
		Timestamp: time.Now(),
	}

	// Persist the cache through the store
	if s.store != nil {
		_ = s.store.Save(freshCache, user) // Ignore errors - caching is optional
	}

	return issues, nil
}

// GetJiraIssue fetches detailed information for a specific JIRA issue using --raw JSON output
// Returns fully populated JiraTicketDetails
func (s *Service) GetJiraIssue(key string, dryRun bool) (*JiraTicketDetails, error) {
	// Check if JIRA CLI is available
	if !s.IsJiraCliAvailable() {
		return nil, ErrJiraCliNotFound
	}

	// Get raw JSON data using jira CLI
	cmd := exec.Command("jira", "issue", "view", key, "--raw")
	output, err := s.runCommand(cmd, dryRun)
	if err != nil {
		return nil, fmt.Errorf("failed to get JIRA issue: %w\nOutput: %s", err, string(output))
	}

	// Parse the JSON response
	var jiraResponse jiraRawResponse
	if err := json.Unmarshal(output, &jiraResponse); err != nil {
		return nil, fmt.Errorf("failed to parse JIRA response: %w", err)
	}

	// Build the ticket details from parsed JSON
	ticket := &JiraTicketDetails{
		Key:     jiraResponse.Key,
		Summary: jiraResponse.Fields.Summary,
		Status:  jiraResponse.Fields.Status.Name,
		URL:     formatJiraURL(jiraResponse.Self, jiraResponse.Key),
	}

	// Parse created date
	if jiraResponse.Fields.Created != "" {
		if createdDate, err := time.Parse(time.RFC3339, jiraResponse.Fields.Created); err == nil {
			ticket.Created = createdDate
		}
	}

	// Add priority
	if jiraResponse.Fields.Priority.Name != "" {
		ticket.Priority = jiraResponse.Fields.Priority.Name
	}

	// Add reporter
	if jiraResponse.Fields.Reporter.DisplayName != "" {
		reporter := jiraResponse.Fields.Reporter.DisplayName
		if jiraResponse.Fields.Reporter.EmailAddress != "" {
			reporter += " (" + jiraResponse.Fields.Reporter.EmailAddress + ")"
		}
		ticket.Reporter = reporter
	}

	// Add assignee (can be null)
	if jiraResponse.Fields.Assignee != nil {
		assignee := jiraResponse.Fields.Assignee.DisplayName
		if jiraResponse.Fields.Assignee.EmailAddress != "" {
			assignee += " (" + jiraResponse.Fields.Assignee.EmailAddress + ")"
		}
		ticket.Assignee = assignee
	}

	// Add due date (can be null)
	if jiraResponse.Fields.DueDate != nil && *jiraResponse.Fields.DueDate != "" {
		if dueDate, err := time.Parse("2006-01-02", *jiraResponse.Fields.DueDate); err == nil {
			ticket.DueDate = &dueDate
		}
	}

	// Add epic information
	if jiraResponse.Fields.Parent != nil {
		ticket.Epic = jiraResponse.Fields.Parent.Key
	}

	// Add latest comment
	if len(jiraResponse.Fields.Comment.Comments) > 0 {
		latest := jiraResponse.Fields.Comment.Comments[0]

		// Extract comment text from nested structure
		var commentText strings.Builder
		for _, content := range latest.Body.Content {
			for _, textContent := range content.Content {
				if textContent.Text != "" {
					commentText.WriteString(textContent.Text)
				}
			}
		}

		if commentText.Len() > 0 {
			comment := &Comment{
				Author:  latest.Author.DisplayName,
				Content: commentText.String(),
			}

			// Parse comment timestamp
			if latest.Created != "" {
				if timestamp, err := time.Parse(time.RFC3339, latest.Created); err == nil {
					comment.Timestamp = timestamp
				}
			}

			ticket.LatestComment = comment
		}
	}

	// Parse description
	if jiraResponse.Fields.Description != nil {
		ticket.Description = parseDescription(jiraResponse.Fields.Description)
	}

	return ticket, nil
}

// parseDescription recursively traverses JIRA's nested content structure
// and converts it to clean markdown format
func parseDescription(desc *Description) string {
	if desc == nil || len(desc.Content) == 0 {
		return ""
	}

	var md strings.Builder

	for _, block := range desc.Content {
		parseContentBlock(&md, block)
	}

	return strings.TrimSpace(md.String())
}

// parseContentBlock handles different content block types
func parseContentBlock(md *strings.Builder, node ContentNode) {
	switch node.Type {
	case "paragraph":
		parseInlineContent(md, node.Content)
		md.WriteString("\n\n")

	case "codeBlock":
		md.WriteString("```")
		if node.Attrs != nil && node.Attrs.Language != "" {
			md.WriteString(node.Attrs.Language)
		}
		md.WriteString("\n")
		parseInlineContent(md, node.Content)
		md.WriteString("\n```\n\n")

	case "heading":
		// Default to h3 since the markdown file already has h1/h2
		level := 3
		md.WriteString(strings.Repeat("#", level) + " ")
		parseInlineContent(md, node.Content)
		md.WriteString("\n\n")

	case "bulletList":
		for _, item := range node.Content {
			if item.Type == "listItem" {
				md.WriteString("- ")
				parseInlineContent(md, item.Content)
				md.WriteString("\n")
			}
		}
		md.WriteString("\n")

	case "orderedList":
		for i, item := range node.Content {
			if item.Type == "listItem" {
				fmt.Fprintf(md, "%d. ", i+1)
				parseInlineContent(md, item.Content)
				md.WriteString("\n")
			}
		}
		md.WriteString("\n")

	case "listItem":
		// List items contain paragraphs, handle their content
		for _, childNode := range node.Content {
			if childNode.Type == "paragraph" {
				parseInlineContent(md, childNode.Content)
			} else {
				parseContentBlock(md, childNode)
			}
		}
	}
}

// parseInlineContent extracts text from inline content nodes
func parseInlineContent(md *strings.Builder, content []ContentNode) {
	for _, node := range content {
		if node.Text != "" {
			md.WriteString(node.Text)
		}
		if len(node.Content) > 0 {
			parseInlineContent(md, node.Content)
		}
	}
}

package jira

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const (
	// issuesCacheTTL is how long we cache JIRA issues (5 minutes).
	issuesCacheTTL = 5 * time.Minute
)

// IssuesCache represents cached JIRA issues with timestamp.
type IssuesCache struct {
	Timestamp time.Time   `json:"timestamp"`
	Issues    []JiraIssue `json:"issues"`
}

// buildIssueListArgs constructs jira issue list command arguments from filters.
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
// Results are cached for 5 minutes to improve performance.
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

	if dryRun {
		printDryRun(cmd)
		return []JiraIssue{}, nil
	}

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JIRA issues: %w", err)
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
// Returns fully populated JiraTicketDetails.
func (s *Service) GetJiraIssue(key string, dryRun bool) (*JiraTicketDetails, error) {
	// Check if JIRA CLI is available
	if !s.IsJiraCliAvailable() {
		return nil, ErrJiraCliNotFound
	}

	// Get raw JSON data using jira CLI
	cmd := exec.Command("jira", "issue", "view", key, "--raw")

	if dryRun {
		printDryRun(cmd)
		return &JiraTicketDetails{
			Key:     key,
			Summary: "Sample ticket",
			Status:  "Open",
		}, nil
	}

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get JIRA issue: %w", err)
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

	// Add epic information (backward compatibility)
	if jiraResponse.Fields.Parent != nil {
		ticket.Epic = jiraResponse.Fields.Parent.Key
	}

	// Parse parent issue
	if jiraResponse.Fields.Parent != nil {
		ticket.Parent = &LinkedIssue{
			ID:        jiraResponse.Fields.Parent.ID,
			Key:       jiraResponse.Fields.Parent.Key,
			Summary:   jiraResponse.Fields.Parent.Fields.Summary,
			Status:    jiraResponse.Fields.Parent.Fields.Status.Name,
			Priority:  jiraResponse.Fields.Parent.Fields.Priority.Name,
			IssueType: jiraResponse.Fields.Parent.Fields.IssueType.Name,
		}
	}

	// Parse children (subtasks)
	if len(jiraResponse.Fields.Subtasks) > 0 {
		ticket.Children = make([]LinkedIssue, 0, len(jiraResponse.Fields.Subtasks))
		for _, subtask := range jiraResponse.Fields.Subtasks {
			ticket.Children = append(ticket.Children, LinkedIssue{
				ID:        subtask.ID,
				Key:       subtask.Key,
				Summary:   subtask.Fields.Summary,
				Status:    subtask.Fields.Status.Name,
				Priority:  subtask.Fields.Priority.Name,
				IssueType: subtask.Fields.IssueType.Name,
			})
		}
	}

	// Parse labels
	ticket.Labels = jiraResponse.Fields.Labels

	// Parse attachments
	ticket.Attachments = make([]Attachment, 0, len(jiraResponse.Fields.Attachment))
	for _, rawAttachment := range jiraResponse.Fields.Attachment {
		attachment := Attachment{
			ID:       rawAttachment.ID,
			Filename: rawAttachment.Filename,
			Author: User{
				DisplayName: rawAttachment.Author.DisplayName,
				Email:       rawAttachment.Author.EmailAddress,
				AccountID:   rawAttachment.Author.AccountID,
				AvatarURL:   rawAttachment.Author.AvatarURLs.Px48,
			},
			Created:  rawAttachment.Created,
			Size:     rawAttachment.Size,
			MimeType: rawAttachment.MimeType,
			Content:  rawAttachment.Content,
		}
		ticket.Attachments = append(ticket.Attachments, attachment)
	}

	// Parse all comments (not just latest)
	ticket.Comments = make([]Comment, 0, len(jiraResponse.Fields.Comment.Comments))
	for _, rawComment := range jiraResponse.Fields.Comment.Comments {
		comment := Comment{
			ID: rawComment.ID,
			Author: User{
				DisplayName: rawComment.Author.DisplayName,
				Email:       rawComment.Author.EmailAddress,
				AccountID:   rawComment.Author.AccountID,
				AvatarURL:   rawComment.Author.AvatarURLs.Px48,
			},
			Body:    rawComment.Body,
			Created: rawComment.Created,
			Updated: rawComment.Updated,
		}

		// Parse timestamp for backward compatibility
		if rawComment.Created != "" {
			if timestamp, err := time.Parse(time.RFC3339, rawComment.Created); err == nil {
				comment.Timestamp = timestamp
			}
		}

		// Extract plain text content for backward compatibility
		var commentText strings.Builder
		for _, content := range rawComment.Body.Content {
			for _, textContent := range content.Content {
				if textContent.Text != "" {
					commentText.WriteString(textContent.Text)
				}
			}
		}
		comment.Content = commentText.String()

		// Extract media IDs from comment body
		parser := NewADFParser()
		_, mediaIDs, _ := parser.ParseToMarkdown(rawComment.Body)
		comment.Attachments = mediaIDs

		ticket.Comments = append(ticket.Comments, comment)
	}

	// Set latest comment for backward compatibility
	if len(ticket.Comments) > 0 {
		// Create a pointer copy of the last comment
		latest := ticket.Comments[len(ticket.Comments)-1]
		ticket.LatestComment = &Comment{
			ID:          latest.ID,
			Author:      latest.Author,
			Content:     latest.Content,
			Timestamp:   latest.Timestamp,
			Body:        latest.Body,
			Created:     latest.Created,
			Updated:     latest.Updated,
			Attachments: latest.Attachments,
		}
	}

	// Parse description
	if jiraResponse.Fields.Description != nil {
		ticket.Description = parseDescription(jiraResponse.Fields.Description)
	}

	// Build a set of child keys for deduplication
	childKeys := make(map[string]bool)
	for _, child := range ticket.Children {
		childKeys[child.Key] = true
	}

	// Parse issue links
	ticket.IssueLinks = make([]IssueLink, 0, len(jiraResponse.Fields.IssueLinks))
	for _, rawLink := range jiraResponse.Fields.IssueLinks {
		link := IssueLink{
			ID: rawLink.ID,
			Type: IssueLinkType{
				ID:      rawLink.Type.ID,
				Name:    rawLink.Type.Name,
				Inward:  rawLink.Type.Inward,
				Outward: rawLink.Type.Outward,
			},
		}

		// Parse inward issue (issue this ticket links to)
		if rawLink.InwardIssue != nil {
			link.InwardIssue = &LinkedIssue{
				ID:        rawLink.InwardIssue.ID,
				Key:       rawLink.InwardIssue.Key,
				Summary:   rawLink.InwardIssue.Fields.Summary,
				Status:    rawLink.InwardIssue.Fields.Status.Name,
				Priority:  rawLink.InwardIssue.Fields.Priority.Name,
				IssueType: rawLink.InwardIssue.Fields.IssueType.Name,
			}
		}

		// Parse outward issue (issue that links to this ticket)
		if rawLink.OutwardIssue != nil {
			link.OutwardIssue = &LinkedIssue{
				ID:        rawLink.OutwardIssue.ID,
				Key:       rawLink.OutwardIssue.Key,
				Summary:   rawLink.OutwardIssue.Fields.Summary,
				Status:    rawLink.OutwardIssue.Fields.Status.Name,
				Priority:  rawLink.OutwardIssue.Fields.Priority.Name,
				IssueType: rawLink.OutwardIssue.Fields.IssueType.Name,
			}
		}

		// Get the linked issue key for deduplication checks
		linkedKey := ""
		if link.InwardIssue != nil {
			linkedKey = link.InwardIssue.Key
		} else if link.OutwardIssue != nil {
			linkedKey = link.OutwardIssue.Key
		}

		// Skip links that reference the parent issue (deduplicate parent from linked issues)
		if ticket.Parent != nil && linkedKey == ticket.Parent.Key {
			continue
		}

		// Skip links that reference child issues (deduplicate children from linked issues)
		if childKeys[linkedKey] {
			continue
		}

		ticket.IssueLinks = append(ticket.IssueLinks, link)
	}

	return ticket, nil
}

// parseDescription converts a JIRA ADF document to clean markdown format
// using the ADFParser which supports tables, panels, and other rich content.
func parseDescription(desc *ADFDocument) string {
	if desc == nil || len(desc.Content) == 0 {
		return ""
	}

	parser := NewADFParser()
	markdown, _, _ := parser.ParseToMarkdown(*desc)
	return markdown
}

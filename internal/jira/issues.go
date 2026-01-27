// Package jira provides JIRA API integration for fetching issues and generating branch names.
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
		cache, cachedUser, _ = s.store.Load() //nolint:errcheck // Cache miss is expected
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
		//nolint:errcheck // Caching is optional - errors are non-fatal
		s.store.Save(freshCache, user)
	}

	return issues, nil
}

// GetJiraIssue fetches detailed information for a specific JIRA issue using --raw JSON output
// Returns fully populated JiraTicketDetails.
func (s *Service) GetJiraIssue(key string, dryRun bool) (*JiraTicketDetails, error) {
	if !s.IsJiraCliAvailable() {
		return nil, ErrJiraCliNotFound
	}

	cmd := exec.Command("jira", "issue", "view", key, "--raw")

	if dryRun {
		printDryRun(cmd)
		return &JiraTicketDetails{Key: key, Summary: "Sample ticket", Status: "Open"}, nil
	}

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get JIRA issue: %w", err)
	}

	var jiraResponse jiraRawResponse
	if err := json.Unmarshal(output, &jiraResponse); err != nil {
		return nil, fmt.Errorf("failed to parse JIRA response: %w", err)
	}

	return buildTicketDetails(&jiraResponse), nil
}

// buildTicketDetails constructs a JiraTicketDetails from the raw API response.
func buildTicketDetails(resp *jiraRawResponse) *JiraTicketDetails {
	ticket := &JiraTicketDetails{
		Key:     resp.Key,
		Summary: resp.Fields.Summary,
		Status:  resp.Fields.Status.Name,
		URL:     formatJiraURL(resp.Self, resp.Key),
		Labels:  resp.Fields.Labels,
	}

	parseBasicFields(ticket, resp)
	parseParentAndChildren(ticket, resp)
	ticket.Attachments = parseAttachments(resp.Fields.Attachment)
	ticket.Comments = parseComments(resp.Fields.Comment.Comments)
	setLatestComment(ticket)

	if resp.Fields.Description != nil {
		ticket.Description = parseDescription(resp.Fields.Description)
	}

	ticket.IssueLinks = parseIssueLinks(resp.Fields.IssueLinks, ticket.Parent, ticket.Children)
	return ticket
}

// parseBasicFields parses dates, priority, reporter, and assignee.
func parseBasicFields(ticket *JiraTicketDetails, resp *jiraRawResponse) {
	if resp.Fields.Created != "" {
		if createdDate, err := time.Parse(time.RFC3339, resp.Fields.Created); err == nil {
			ticket.Created = createdDate
		}
	}

	if resp.Fields.Priority.Name != "" {
		ticket.Priority = resp.Fields.Priority.Name
	}

	if resp.Fields.Reporter.DisplayName != "" {
		ticket.Reporter = formatUserFromRaw(resp.Fields.Reporter)
	}

	if resp.Fields.Assignee != nil {
		ticket.Assignee = formatUserFromRaw(*resp.Fields.Assignee)
	}

	if resp.Fields.DueDate != nil && *resp.Fields.DueDate != "" {
		if dueDate, err := time.Parse("2006-01-02", *resp.Fields.DueDate); err == nil {
			ticket.DueDate = &dueDate
		}
	}
}

// formatUserFromRaw formats a jiraRawAuthor to a display string.
func formatUserFromRaw(author jiraRawAuthor) string {
	return formatUserWithEmail(author.DisplayName, author.EmailAddress)
}

// formatUserWithEmail formats a display name with optional email.
func formatUserWithEmail(displayName, email string) string {
	if email != "" {
		return displayName + " (" + email + ")"
	}
	return displayName
}

// parseParentAndChildren parses parent issue and subtasks.
func parseParentAndChildren(ticket *JiraTicketDetails, resp *jiraRawResponse) {
	if resp.Fields.Parent != nil {
		ticket.Epic = resp.Fields.Parent.Key
		ticket.Parent = parseLinkedIssueFromParent(resp.Fields.Parent)
	}

	if len(resp.Fields.Subtasks) > 0 {
		ticket.Children = make([]LinkedIssue, 0, len(resp.Fields.Subtasks))
		for _, subtask := range resp.Fields.Subtasks {
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
}

// parseLinkedIssueFromParent converts a parent issue to LinkedIssue.
func parseLinkedIssueFromParent(parent *jiraParent) *LinkedIssue {
	return &LinkedIssue{
		ID:        parent.ID,
		Key:       parent.Key,
		Summary:   parent.Fields.Summary,
		Status:    parent.Fields.Status.Name,
		Priority:  parent.Fields.Priority.Name,
		IssueType: parent.Fields.IssueType.Name,
	}
}

// parseAttachments converts raw attachments to Attachment slice.
func parseAttachments(rawAttachments []jiraRawAttachment) []Attachment {
	attachments := make([]Attachment, 0, len(rawAttachments))
	for _, raw := range rawAttachments {
		attachments = append(attachments, Attachment{
			ID:       raw.ID,
			Filename: raw.Filename,
			Author:   userFromRawAuthor(raw.Author),
			Created:  raw.Created,
			Size:     raw.Size,
			MimeType: raw.MimeType,
			Content:  raw.Content,
		})
	}
	return attachments
}

// userFromRawAuthor converts a jiraRawAuthor to a User.
func userFromRawAuthor(author jiraRawAuthor) User {
	return User{
		DisplayName: author.DisplayName,
		Email:       author.EmailAddress,
		AccountID:   author.AccountID,
		AvatarURL:   author.AvatarURLs.Px48,
	}
}

// parseComments converts raw comments to Comment slice.
func parseComments(rawComments []jiraRawComment) []Comment {
	comments := make([]Comment, 0, len(rawComments))
	for _, raw := range rawComments {
		comments = append(comments, parseComment(raw))
	}
	return comments
}

// parseComment converts a single raw comment to Comment.
func parseComment(raw jiraRawComment) Comment {
	comment := Comment{
		ID:      raw.ID,
		Author:  userFromRawAuthor(raw.Author),
		Body:    raw.Body,
		Created: raw.Created,
		Updated: raw.Updated,
	}

	if raw.Created != "" {
		if timestamp, err := time.Parse(time.RFC3339, raw.Created); err == nil {
			comment.Timestamp = timestamp
		}
	}

	comment.Content = extractCommentText(raw.Body)

	parser := NewADFParser()
	_, mediaIDs, _ := parser.ParseToMarkdown(raw.Body) //nolint:errcheck // Best-effort markdown parsing
	comment.Attachments = mediaIDs

	return comment
}

// extractCommentText extracts plain text from ADF body.
func extractCommentText(body ADFDocument) string {
	var builder strings.Builder
	for _, content := range body.Content {
		for _, textContent := range content.Content {
			if textContent.Text != "" {
				builder.WriteString(textContent.Text)
			}
		}
	}
	return builder.String()
}

// setLatestComment sets the latest comment for backward compatibility.
func setLatestComment(ticket *JiraTicketDetails) {
	if len(ticket.Comments) == 0 {
		return
	}
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

// parseIssueLinks converts raw issue links with deduplication.
func parseIssueLinks(rawLinks []jiraRawIssueLink, parent *LinkedIssue, children []LinkedIssue) []IssueLink {
	childKeys := make(map[string]bool)
	for _, child := range children {
		childKeys[child.Key] = true
	}

	links := make([]IssueLink, 0, len(rawLinks))
	for _, raw := range rawLinks {
		link := parseIssueLink(raw)
		linkedKey := getLinkKey(link)

		// Skip duplicates (parent or children)
		if parent != nil && linkedKey == parent.Key {
			continue
		}
		if childKeys[linkedKey] {
			continue
		}

		links = append(links, link)
	}
	return links
}

// parseIssueLink converts a single raw issue link.
func parseIssueLink(raw jiraRawIssueLink) IssueLink {
	link := IssueLink{
		ID: raw.ID,
		Type: IssueLinkType{
			ID:      raw.Type.ID,
			Name:    raw.Type.Name,
			Inward:  raw.Type.Inward,
			Outward: raw.Type.Outward,
		},
	}

	if raw.InwardIssue != nil {
		link.InwardIssue = linkedIssueFromRaw(raw.InwardIssue)
	}

	if raw.OutwardIssue != nil {
		link.OutwardIssue = linkedIssueFromRaw(raw.OutwardIssue)
	}

	return link
}

// linkedIssueFromRaw converts a jiraRawLinkedIssue to a LinkedIssue.
func linkedIssueFromRaw(raw *jiraRawLinkedIssue) *LinkedIssue {
	return &LinkedIssue{
		ID:        raw.ID,
		Key:       raw.Key,
		Summary:   raw.Fields.Summary,
		Status:    raw.Fields.Status.Name,
		Priority:  raw.Fields.Priority.Name,
		IssueType: raw.Fields.IssueType.Name,
	}
}

// getLinkKey returns the key of the linked issue.
func getLinkKey(link IssueLink) string {
	if link.InwardIssue != nil {
		return link.InwardIssue.Key
	}
	if link.OutwardIssue != nil {
		return link.OutwardIssue.Key
	}
	return ""
}

// parseDescription converts a JIRA ADF document to clean markdown format
// using the ADFParser which supports tables, panels, and other rich content.
func parseDescription(desc *ADFDocument) string {
	if desc == nil || len(desc.Content) == 0 {
		return ""
	}

	parser := NewADFParser()
	markdown, _, _ := parser.ParseToMarkdown(*desc) //nolint:errcheck // Best-effort markdown parsing
	return markdown
}

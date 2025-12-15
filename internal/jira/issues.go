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

// GetJiraIssues fetches all JIRA issues for the current user using --raw JSON output
// Returns slice of typed JiraIssue structs
// Results are cached for 5 minutes to improve performance
func (s *Service) GetJiraIssues(dryRun bool) ([]JiraIssue, error) {
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

	// Fetch issues for the user using jira CLI with --raw for JSON output
	cmd := exec.Command("jira", "issue", "list", "-a"+user, "--raw")
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

	return ticket, nil
}


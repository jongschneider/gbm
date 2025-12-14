package jira

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// GetJiraIssues fetches all JIRA issues for the current user using --raw JSON output
// Returns slice of typed JiraIssue structs
func (s *Service) GetJiraIssues(cachedUser string, dryRun bool) ([]JiraIssue, error) {
	// Get JIRA user (cached or fetch)
	user, _, err := s.GetJiraUser(cachedUser, dryRun)
	if err != nil {
		return nil, err
	}

	// Fetch issues for the user using jira CLI
	// Note: jira issue list doesn't have a --raw flag, so we get plain text
	// We'll need to use jira issue view --raw for individual issues
	cmd := exec.Command("jira", "issue", "list", "-a"+user, "--plain")
	output, err := s.runCommand(cmd, dryRun)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JIRA issues: %w\nOutput: %s", err, string(output))
	}

	return parseJiraList(string(output)), nil
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

// parseJiraList parses the output of 'jira issue list --plain' command
// This is a helper function that will be called by GetJiraIssues
func parseJiraList(output string) []JiraIssue {
	var issues []JiraIssue
	lines := strings.Split(output, "\n")

	for _, line := range lines[1:] { // Skip header
		if line = strings.TrimSpace(line); line != "" {
			fields := strings.Split(line, "\t")
			if len(fields) >= 3 {
				// Find the JIRA key in this line
				var issueKey, issueType, summary, status string
				keyIndex := -1

				for i, field := range fields {
					trimmedField := strings.TrimSpace(field)
					if IsJiraKey(trimmedField) {
						issueKey = trimmedField
						keyIndex = i
						break
					}
				}

				if issueKey != "" {
					// Type is usually the first field
					issueType = strings.TrimSpace(fields[0])

					// Summary is usually the field after the key
					if keyIndex+1 < len(fields) {
						summary = strings.TrimSpace(fields[keyIndex+1])
					}

					// Status is the last non-empty field
					for i := len(fields) - 1; i >= 0; i-- {
						if trimmed := strings.TrimSpace(fields[i]); trimmed != "" {
							status = trimmed
							break
						}
					}

					issue := JiraIssue{
						Type:    issueType,
						Key:     issueKey,
						Summary: summary,
						Status:  status,
					}
					issues = append(issues, issue)
				}
			}
		}
	}
	return issues
}

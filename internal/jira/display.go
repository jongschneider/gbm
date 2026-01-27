package jira

import (
	"fmt"
	"regexp"
	"strings"
)

// IsJiraKey checks if a string matches the JIRA key pattern (PROJECT-NUMBER).
func IsJiraKey(s string) bool {
	matched, _ := regexp.MatchString(`^[A-Z]+-\d+$`, s)
	return matched
}

// ExtractJiraKey extracts a JIRA key from a string, handling prefixed worktree names
// For example: "HOTFIX_INGSVC-5638" returns "INGSVC-5638".
func ExtractJiraKey(s string) string {
	re := regexp.MustCompile(`[A-Z]+-\d+`)
	match := re.FindString(s)
	return match
}

// String returns a one-line summary of the JIRA issue.
func (j *JiraIssue) String() string {
	return fmt.Sprintf("[%s] %s: %s (%s)", j.Type, j.Key, j.Summary, j.Status)
}

// Display returns a multi-line formatted display of the JIRA issue.
func (j *JiraIssue) Display() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Key:     %s\n", j.Key))
	sb.WriteString(fmt.Sprintf("Type:    %s\n", j.Type))
	sb.WriteString(fmt.Sprintf("Status:  %s\n", j.Status))
	sb.WriteString(fmt.Sprintf("Summary: %s\n", j.Summary))
	return sb.String()
}

// String returns a one-line summary of the JIRA ticket.
func (j *JiraTicketDetails) String() string {
	return fmt.Sprintf("[%s] %s (%s)", j.Key, j.Summary, j.Status)
}

// Display returns a full formatted display of the JIRA ticket with all details.
func (j *JiraTicketDetails) Display() string {
	var sb strings.Builder

	// Header
	sb.WriteString(fmt.Sprintf("=== %s ===\n", j.Key))
	sb.WriteString(fmt.Sprintf("Summary:  %s\n", j.Summary))
	sb.WriteString(fmt.Sprintf("Status:   %s\n", j.Status))

	// URL
	if j.URL != "" {
		sb.WriteString(fmt.Sprintf("URL:      %s\n", j.URL))
	}

	// Priority
	if j.Priority != "" {
		sb.WriteString(fmt.Sprintf("Priority: %s\n", j.Priority))
	}

	// Reporter
	if j.Reporter != "" {
		sb.WriteString(fmt.Sprintf("Reporter: %s\n", j.Reporter))
	}

	// Assignee
	if j.Assignee != "" {
		sb.WriteString(fmt.Sprintf("Assignee: %s\n", j.Assignee))
	}

	// Created date
	if !j.Created.IsZero() {
		sb.WriteString(fmt.Sprintf("Created:  %s\n", j.Created.Format("2006-01-02 15:04:05")))
	}

	// Due date
	if j.DueDate != nil {
		sb.WriteString(fmt.Sprintf("Due Date: %s\n", j.DueDate.Format("2006-01-02")))
	}

	// Epic
	if j.Epic != "" {
		sb.WriteString(fmt.Sprintf("Epic:     %s\n", j.Epic))
	}

	// Latest comment
	if j.LatestComment != nil {
		sb.WriteString("\nLatest Comment:\n")
		sb.WriteString(fmt.Sprintf("  Author:  %s\n", j.LatestComment.Author))
		if !j.LatestComment.Timestamp.IsZero() {
			sb.WriteString(fmt.Sprintf("  Date:    %s\n", j.LatestComment.Timestamp.Format("2006-01-02 15:04:05")))
		}
		sb.WriteString(fmt.Sprintf("  Content: %s\n", j.LatestComment.Content))
	}

	return sb.String()
}

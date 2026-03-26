package jira

import (
	"fmt"
	"regexp"
	"strings"
)

// IsJiraKey checks if a string matches the JIRA key pattern (PROJECT-NUMBER).
func IsJiraKey(s string) bool {
	matched, err := regexp.MatchString(`^[A-Z]+-\d+$`, s)
	if err != nil {
		return false
	}
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
	fmt.Fprintf(&sb, "Key:     %s\n", j.Key)
	fmt.Fprintf(&sb, "Type:    %s\n", j.Type)
	fmt.Fprintf(&sb, "Status:  %s\n", j.Status)
	fmt.Fprintf(&sb, "Summary: %s\n", j.Summary)
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
	fmt.Fprintf(&sb, "=== %s ===\n", j.Key)
	fmt.Fprintf(&sb, "Summary:  %s\n", j.Summary)
	fmt.Fprintf(&sb, "Status:   %s\n", j.Status)

	// URL
	if j.URL != "" {
		fmt.Fprintf(&sb, "URL:      %s\n", j.URL)
	}

	// Priority
	if j.Priority != "" {
		fmt.Fprintf(&sb, "Priority: %s\n", j.Priority)
	}

	// Reporter
	if j.Reporter != "" {
		fmt.Fprintf(&sb, "Reporter: %s\n", j.Reporter)
	}

	// Assignee
	if j.Assignee != "" {
		fmt.Fprintf(&sb, "Assignee: %s\n", j.Assignee)
	}

	// Created date
	if !j.Created.IsZero() {
		fmt.Fprintf(&sb, "Created:  %s\n", j.Created.Format("2006-01-02 15:04:05"))
	}

	// Due date
	if j.DueDate != nil {
		fmt.Fprintf(&sb, "Due Date: %s\n", j.DueDate.Format("2006-01-02"))
	}

	// Epic
	if j.Epic != "" {
		fmt.Fprintf(&sb, "Epic:     %s\n", j.Epic)
	}

	// Latest comment
	if j.LatestComment != nil {
		sb.WriteString("\nLatest Comment:\n")
		fmt.Fprintf(&sb, "  Author:  %s\n", j.LatestComment.Author)
		if !j.LatestComment.Timestamp.IsZero() {
			fmt.Fprintf(&sb, "  Date:    %s\n", j.LatestComment.Timestamp.Format("2006-01-02 15:04:05"))
		}
		fmt.Fprintf(&sb, "  Content: %s\n", j.LatestComment.Content)
	}

	return sb.String()
}

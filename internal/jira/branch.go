package jira

import (
	"fmt"
	"regexp"
	"strings"
)

// BranchName generates a filesystem-safe branch name from a JIRA issue
// Format: <type>/<key>_<summary>
// Example: feature/PROJ-123_add_user_authentication
func (j *JiraIssue) BranchName() string {
	summary := strings.ReplaceAll(j.Summary, " ", "_")
	summary = strings.ReplaceAll(summary, "-", "_")
	// Remove special characters and make it filesystem-safe
	summary = regexp.MustCompile(`[^a-zA-Z0-9_]`).ReplaceAllString(summary, "_")
	// Clean up multiple underscores
	summary = regexp.MustCompile(`_+`).ReplaceAllString(summary, "_")
	summary = strings.Trim(summary, "_")

	issueType := strings.ToLower(j.Type)
	if issueType == "story" || issueType == "improvement" {
		issueType = "feature"
	}

	branchName := fmt.Sprintf("%s/%s_%s", issueType, j.Key, summary)
	return branchName
}

// BranchName generates a filesystem-safe branch name from a JIRA ticket
// This is a convenience method on JiraTicketDetails
// Format: <type>/<key>_<summary>
func (j *JiraTicketDetails) BranchName() string {
	summary := strings.ReplaceAll(j.Summary, " ", "_")
	summary = strings.ReplaceAll(summary, "-", "_")
	// Remove special characters and make it filesystem-safe
	summary = regexp.MustCompile(`[^a-zA-Z0-9_]`).ReplaceAllString(summary, "_")
	// Clean up multiple underscores
	summary = regexp.MustCompile(`_+`).ReplaceAllString(summary, "_")
	summary = strings.Trim(summary, "_")

	// Default to "feature" since we don't have type in JiraTicketDetails
	// Could be enhanced to infer from priority or other fields
	issueType := "feature"

	branchName := fmt.Sprintf("%s/%s_%s", issueType, j.Key, summary)
	return branchName
}

// GenerateBranchFromJira fetches a JIRA issue and generates a branch name
// This is a convenience method that combines GetJiraIssue + BranchName
func (s *Service) GenerateBranchFromJira(key string, dryRun bool) (string, error) {
	issue, err := s.GetJiraIssue(key, dryRun)
	if err != nil {
		return "", err
	}

	return issue.BranchName(), nil
}

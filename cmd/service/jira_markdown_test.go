package service

import (
	"flag"
	"gbm/internal/jira"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var update = flag.Bool("update", false, "update golden files")

func TestExtractAcceptanceCriteria(t *testing.T) {
	tests := []struct {
		name        string
		description string
		want        string
	}{
		{
			name: "with_acceptance_criteria",
			description: `# Description

This is a bug that needs to be fixed.

## Acceptance Criteria

- User can login successfully
- Error message is displayed on failure
- Session is maintained after login

## Technical Notes

Some technical details here.`,
			want: `- User can login successfully
- Error message is displayed on failure
- Session is maintained after login`,
		},
		{
			name: "with_acceptance_criteria_lowercase",
			description: `# Description

This is a feature request.

## acceptance criteria

- Feature A works
- Feature B works

## Other Section

More content.`,
			want: `- Feature A works
- Feature B works`,
		},
		{
			name: "with_acceptance_criteria_mixed_case",
			description: `Some description here.

Acceptance Criteria:

1. First criterion
2. Second criterion
3. Third criterion

# Next Section

Content here.`,
			want: `1. First criterion
2. Second criterion
3. Third criterion`,
		},
		{
			name: "without_acceptance_criteria",
			description: `# Description

This is a simple description without any AC section.

## Technical Details

Some technical notes.`,
			want: "",
		},
		{
			name:        "empty_description",
			description: "",
			want:        "",
		},
		{
			name: "acceptance_criteria_at_end",
			description: `# Summary

This ticket adds a new feature.

## Acceptance Criteria

- User can access the new feature
- Feature works as expected`,
			want: `- User can access the new feature
- Feature works as expected`,
		},
		{
			name: "acceptance_criteria_with_nested_content",
			description: `## Description

Bug fix needed.

### Acceptance Criteria

- Fix applied
- Tests pass
  - Unit tests
  - Integration tests
- Documentation updated

### Implementation Notes

Technical details.`,
			want: `- Fix applied
- Tests pass
  - Unit tests
  - Integration tests
- Documentation updated`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractAcceptanceCriteria(tt.description)
			if got != tt.want {
				t.Errorf("extractAcceptanceCriteria() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGenerateJiraMarkdown(t *testing.T) {
	// Fixed timestamp for consistent golden files
	fixedTime := time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC)
	dueDate := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		ticket *jira.JiraTicketDetails
		name   string
	}{
		{
			name: "full_ticket_with_ac",
			ticket: &jira.JiraTicketDetails{
				Key:      "PROJ-123",
				Summary:  "Fix login bug",
				Status:   "In Progress",
				Priority: "High",
				Assignee: "John Doe",
				Created:  fixedTime,
				DueDate:  &dueDate,
				Epic:     "User Authentication",
				URL:      "https://jira.company.com/browse/PROJ-123",
				Description: `# Description

Users are unable to login when using special characters in password.

## Acceptance Criteria

- Users can login with special characters in password
- Error handling is improved
- Security is not compromised

## Technical Details

Need to update password validation logic.`,
			},
		},
		{
			name: "minimal_ticket",
			ticket: &jira.JiraTicketDetails{
				Key:     "PROJ-456",
				Summary: "Update documentation",
				Status:  "Open",
				URL:     "https://jira.company.com/browse/PROJ-456",
				Created: fixedTime,
			},
		},
		{
			name: "ticket_without_ac_in_description",
			ticket: &jira.JiraTicketDetails{
				Key:      "PROJ-789",
				Summary:  "Add new feature",
				Status:   "In Dev.",
				Priority: "Medium",
				Assignee: "Jane Smith",
				Created:  fixedTime,
				URL:      "https://jira.company.com/browse/PROJ-789",
				Description: `This ticket adds a new reporting feature.

The feature should allow users to generate reports in PDF format.

Some additional context and requirements.`,
			},
		},
		{
			name: "ticket_with_multiline_ac",
			ticket: &jira.JiraTicketDetails{
				Key:      "PROJ-999",
				Summary:  "Refactor authentication module",
				Status:   "Code Review",
				Priority: "Critical",
				Assignee: "Alice Johnson",
				Created:  fixedTime,
				DueDate:  &dueDate,
				Epic:     "Tech Debt",
				URL:      "https://jira.company.com/browse/PROJ-999",
				Description: `## Overview

The authentication module needs refactoring for better maintainability.

## Acceptance Criteria

- [ ] Code is refactored following SOLID principles
- [ ] All existing tests pass
- [ ] New unit tests added for refactored code
- [ ] Performance is improved or maintained
- [ ] Documentation is updated
- [ ] Code review is completed

## Scope

- User login
- Session management
- Token refresh`,
			},
		},
		{
			name: "ticket_with_empty_description",
			ticket: &jira.JiraTicketDetails{
				Key:      "PROJ-111",
				Summary:  "Quick fix",
				Status:   "Done",
				Priority: "Low",
				Created:  fixedTime,
				URL:      "https://jira.company.com/browse/PROJ-111",
			},
		},
		{
			name: "ticket_with_parent",
			ticket: &jira.JiraTicketDetails{
				Key:      "PROJ-222",
				Summary:  "Implement login button",
				Status:   "In Progress",
				Priority: "Medium",
				Assignee: "John Doe",
				Created:  fixedTime,
				URL:      "https://jira.company.com/browse/PROJ-222",
				Parent: &jira.LinkedIssue{
					Key:     "PROJ-100",
					Summary: "User Authentication Epic",
				},
				Description: `Implement the login button on the home page.

## Acceptance Criteria

- Button is visible on home page
- Button redirects to login form`,
			},
		},
		{
			name: "ticket_without_parent",
			ticket: &jira.JiraTicketDetails{
				Key:      "PROJ-333",
				Summary:  "Top-level feature",
				Status:   "Open",
				Priority: "High",
				Created:  fixedTime,
				URL:      "https://jira.company.com/browse/PROJ-333",
				Parent:   nil,
				Description: `This is a top-level feature with no parent.

## Acceptance Criteria

- Feature works as expected`,
			},
		},
		{
			name: "ticket_with_children",
			ticket: &jira.JiraTicketDetails{
				Key:      "PROJ-444",
				Summary:  "Parent feature with children",
				Status:   "In Progress",
				Priority: "High",
				Assignee: "Jane Doe",
				Created:  fixedTime,
				URL:      "https://jira.company.com/browse/PROJ-444",
				Children: []jira.LinkedIssue{
					{Key: "PROJ-445", Summary: "First child task"},
					{Key: "PROJ-446", Summary: "Second child task"},
					{Key: "PROJ-447", Summary: "Third child task"},
				},
				Description: `This is a parent feature with multiple children.

## Acceptance Criteria

- All child tasks are completed
- Feature is fully functional`,
			},
		},
		{
			name: "ticket_with_parent_and_children",
			ticket: &jira.JiraTicketDetails{
				Key:      "PROJ-555",
				Summary:  "Middle-level feature",
				Status:   "In Progress",
				Priority: "Medium",
				Assignee: "Bob Smith",
				Created:  fixedTime,
				URL:      "https://jira.company.com/browse/PROJ-555",
				Parent: &jira.LinkedIssue{
					Key:     "PROJ-500",
					Summary: "Top-level epic",
				},
				Children: []jira.LinkedIssue{
					{Key: "PROJ-556", Summary: "Implementation task"},
					{Key: "PROJ-557", Summary: "Testing task"},
				},
				Description: `This feature has both a parent and children.

## Acceptance Criteria

- Implements part of parent epic
- Child tasks are completed`,
			},
		},
		{
			name: "ticket_without_children",
			ticket: &jira.JiraTicketDetails{
				Key:      "PROJ-666",
				Summary:  "Leaf task with no children",
				Status:   "Open",
				Priority: "Low",
				Created:  fixedTime,
				URL:      "https://jira.company.com/browse/PROJ-666",
				Children: nil,
				Description: `This is a leaf task with no children.

## Acceptance Criteria

- Task is completed`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Override time.Now() for consistent golden files
			// We'll work around this by using a fixed timestamp in the test data
			got := generateJiraMarkdown(tt.ticket)

			// Replace the timestamp line with a fixed one for golden file comparison
			// This is a simplified approach - in production you might use dependency injection
			got = replaceTimestamp(got, fixedTime)

			goldenFile := filepath.Join("testdata", tt.name+".golden.md")

			if *update {
				// Create directory if it doesn't exist
				dir := filepath.Dir(goldenFile)
				err := os.MkdirAll(dir, 0o755)
				if err != nil {
					t.Fatalf("failed to create testdata directory: %v", err)
				}

				// Write golden file
				err = os.WriteFile(goldenFile, []byte(got), 0o644)
				if err != nil {
					t.Fatalf("failed to write golden file: %v", err)
				}
			}

			// Read golden file
			want, err := os.ReadFile(goldenFile)
			if err != nil {
				t.Fatalf("failed to read golden file %s: %v\nRun with -update flag to create it", goldenFile, err)
			}

			if got != string(want) {
				t.Errorf("generateJiraMarkdown() output differs from golden file %s\n\nGot:\n%s\n\nWant:\n%s\n\nRun with -update flag to update golden files", goldenFile, got, string(want))
			}
		})
	}
}

// replaceTimestamp replaces the dynamic timestamp with a fixed one for testing.
func replaceTimestamp(markdown string, fixedTime time.Time) string {
	// Replace the last line which contains the timestamp
	lines := []string{}
	for _, line := range splitLines(markdown) {
		if line != "" && line[0] == '*' && contains(line, "Generated by gbm on") {
			lines = append(lines, "*Generated by gbm on "+fixedTime.Format("2006-01-02 15:04:05")+"*")
		} else {
			lines = append(lines, line)
		}
	}
	return joinLines(lines)
}

func splitLines(s string) []string {
	result := []string{}
	current := ""
	for _, ch := range s {
		if ch == '\n' {
			result = append(result, current)
			current = ""
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

func joinLines(lines []string) string {
	result := ""
	var resultSb421 strings.Builder
	for i, line := range lines {
		resultSb421.WriteString(line)
		if i < len(lines)-1 {
			resultSb421.WriteString("\n")
		}
	}
	result += resultSb421.String()
	return result
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr || len(s) > len(substr) && containsHelper(s, substr)
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

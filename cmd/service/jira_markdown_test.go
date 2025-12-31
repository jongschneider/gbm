package service

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gbm/internal/jira"
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
		name   string
		ticket *jira.JiraTicketDetails
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
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatalf("failed to create testdata directory: %v", err)
				}

				// Write golden file
				if err := os.WriteFile(goldenFile, []byte(got), 0644); err != nil {
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

// replaceTimestamp replaces the dynamic timestamp with a fixed one for testing
func replaceTimestamp(markdown string, fixedTime time.Time) string {
	// Replace the last line which contains the timestamp
	lines := []string{}
	for _, line := range splitLines(markdown) {
		if len(line) > 0 && line[0] == '*' && contains(line, "Generated by gbm on") {
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
	for i, line := range lines {
		result += line
		if i < len(lines)-1 {
			result += "\n"
		}
	}
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

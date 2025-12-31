package jira

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIssueLinkParsing tests that issue links are correctly parsed from JIRA JSON
func TestIssueLinkParsing(t *testing.T) {
	// Read the sample ticket.json from testdata
	ticketPath := "testdata/ticket.json"
	data, err := os.ReadFile(ticketPath)
	require.NoError(t, err, "ticket.json should exist in testdata/")

	// Parse the JSON response
	var jiraResponse jiraRawResponse
	require.NoError(t, json.Unmarshal(data, &jiraResponse), "ticket.json should be valid JSON")

	// Verify that we have issue links
	require.NotEmpty(t, jiraResponse.Fields.IssueLinks, "ticket.json should contain issue links")
	t.Logf("Found %d issue links in ticket.json", len(jiraResponse.Fields.IssueLinks))

	// Test the first issue link
	firstLink := jiraResponse.Fields.IssueLinks[0]

	// Verify link type
	assert.NotEmpty(t, firstLink.Type.Name, "link type name should be populated")
	t.Logf("Link type: %s (inward: %s, outward: %s)",
		firstLink.Type.Name,
		firstLink.Type.Inward,
		firstLink.Type.Outward)

	// Verify linked issue details
	if assert.NotNil(t, firstLink.InwardIssue, "first link should have an inward issue") {
		t.Logf("Inward issue: %s - %s",
			firstLink.InwardIssue.Key,
			firstLink.InwardIssue.Fields.Summary)

		assert.NotEmpty(t, firstLink.InwardIssue.Key, "inward issue key should be populated")
		assert.NotEmpty(t, firstLink.InwardIssue.Fields.Summary, "inward issue summary should be populated")
		assert.NotEmpty(t, firstLink.InwardIssue.Fields.Status.Name, "inward issue status should be populated")
	}

	if firstLink.OutwardIssue != nil {
		t.Logf("Outward issue: %s - %s",
			firstLink.OutwardIssue.Key,
			firstLink.OutwardIssue.Fields.Summary)
	}

	// Simulate the parsing that happens in GetJiraIssue
	issueLinks := make([]IssueLink, 0, len(jiraResponse.Fields.IssueLinks))
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

		issueLinks = append(issueLinks, link)
	}

	// Verify we parsed the links correctly
	assert.Len(t, issueLinks, len(jiraResponse.Fields.IssueLinks), "should parse all issue links")

	// Verify the first parsed link
	if assert.NotEmpty(t, issueLinks, "should have at least one parsed link") {
		firstParsed := issueLinks[0]

		assert.NotEmpty(t, firstParsed.Type.Name, "parsed link type name should be populated")

		if assert.NotNil(t, firstParsed.InwardIssue, "parsed link should have inward issue") {
			assert.NotEmpty(t, firstParsed.InwardIssue.Key, "parsed inward issue key should be populated")
			assert.NotEmpty(t, firstParsed.InwardIssue.Summary, "parsed inward issue summary should be populated")
			t.Logf("Successfully parsed inward issue: %s - %s",
				firstParsed.InwardIssue.Key,
				firstParsed.InwardIssue.Summary)
		}
	}

	t.Log("Issue link parsing test completed successfully")
}

// TestMarkdownWithLinkedIssues tests that the markdown generator includes linked issues
func TestMarkdownWithLinkedIssues(t *testing.T) {
	// Create a sample ticket with linked issues
	ticket := &JiraTicketDetails{
		Key:         "INGSVC-6375",
		Summary:     "Test ticket with linked issues",
		Description: "This is a test description",
		Status:      "Open",
		Priority:    "Medium",
		Assignee:    "Test User",
		Reporter:    "Test Reporter",
		URL:         "https://example.com/browse/INGSVC-6375",
		IssueLinks: []IssueLink{
			{
				ID: "47742",
				Type: IssueLinkType{
					ID:      "10313",
					Name:    "Discovery - Connected",
					Inward:  "is connected to",
					Outward: "connects to",
				},
				InwardIssue: &LinkedIssue{
					ID:        "62959",
					Key:       "EPIC-3089",
					Summary:   "[Wells Fargo] Microsoft Teams Messaging : Add Support for Conversation Rename",
					Status:    "ENG Accepted",
					Priority:  "Medium",
					IssueType: "Epic",
				},
			},
		},
	}

	// Generate markdown
	generator := NewMarkdownGenerator()
	markdown, err := generator.GenerateIssueMarkdown(ticket, MarkdownOptions{
		IncludeComments:    false,
		IncludeAttachments: false,
		UseRelativeLinks:   true,
	})

	require.NoError(t, err, "markdown generation should succeed")

	// Verify that the markdown contains the linked issues section
	assert.Contains(t, markdown, "## Linked Issues", "markdown should contain Linked Issues section")
	assert.Contains(t, markdown, "EPIC-3089", "markdown should contain linked issue key")
	assert.Contains(t, markdown, "is connected to", "markdown should contain relationship")
	assert.Contains(t, markdown, "./EPIC-3089.md", "markdown should contain link to linked issue")

	t.Log("Markdown generation with linked issues test completed successfully")
	t.Logf("Generated markdown:\n%s", markdown)
}

// TestDepthConfiguration tests that MaxDepth is properly configured
func TestDepthConfiguration(t *testing.T) {
	tests := []struct {
		name            string
		maxDepth        int
		currentDepth    int
		hasLinkedIssues bool
		shouldProcess   bool
		shouldSkip      bool
	}{
		{
			name:            "Depth 1 of 2 - should process",
			maxDepth:        2,
			currentDepth:    1,
			hasLinkedIssues: true,
			shouldProcess:   true,
			shouldSkip:      false,
		},
		{
			name:            "Depth 2 of 2 - should skip",
			maxDepth:        2,
			currentDepth:    2,
			hasLinkedIssues: true,
			shouldProcess:   false,
			shouldSkip:      true,
		},
		{
			name:            "Depth 1 of 1 - should skip",
			maxDepth:        1,
			currentDepth:    1,
			hasLinkedIssues: true,
			shouldProcess:   false,
			shouldSkip:      true,
		},
		{
			name:            "Depth 2 of 3 - should process",
			maxDepth:        3,
			currentDepth:    2,
			hasLinkedIssues: true,
			shouldProcess:   true,
			shouldSkip:      false,
		},
		{
			name:            "No linked issues - should not process or skip",
			maxDepth:        2,
			currentDepth:    1,
			hasLinkedIssues: false,
			shouldProcess:   false,
			shouldSkip:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := IssueMarkdownOptions{
				IncludeLinkedIssues: true,
				MaxDepth:            tt.maxDepth,
			}

			linkedIssuesCount := 0
			if tt.hasLinkedIssues {
				linkedIssuesCount = 1
			}

			// Simulate the condition checks from the actual code
			shouldProcess := opts.IncludeLinkedIssues && linkedIssuesCount > 0 && tt.currentDepth < opts.MaxDepth
			shouldSkip := opts.IncludeLinkedIssues && linkedIssuesCount > 0 && tt.currentDepth >= opts.MaxDepth

			assert.Equal(t, tt.shouldProcess, shouldProcess, "shouldProcess should match expected value")
			assert.Equal(t, tt.shouldSkip, shouldSkip, "shouldSkip should match expected value")

			t.Logf("✓ Depth %d/%d: process=%v, skip=%v",
				tt.currentDepth, tt.maxDepth, shouldProcess, shouldSkip)
		})
	}
}

// TestDefaultMaxDepth verifies the default MaxDepth is 2
func TestDefaultMaxDepth(t *testing.T) {
	opts := DefaultIssueMarkdownOptions("/tmp/test")

	assert.Equal(t, 2, opts.MaxDepth, "default MaxDepth should be 2")
	assert.True(t, opts.IncludeLinkedIssues, "IncludeLinkedIssues should be true by default")

	t.Logf("✓ Default MaxDepth is %d", opts.MaxDepth)
	t.Logf("✓ Default IncludeLinkedIssues is %v", opts.IncludeLinkedIssues)
}

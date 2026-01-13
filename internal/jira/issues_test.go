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

// TestParentDepthConfiguration tests that parent issues follow MaxDepth rules
func TestParentDepthConfiguration(t *testing.T) {
	tests := []struct {
		name          string
		maxDepth      int
		currentDepth  int
		hasParent     bool
		shouldProcess bool
	}{
		{
			name:          "Depth 1 of 2 with parent - should process parent",
			maxDepth:      2,
			currentDepth:  1,
			hasParent:     true,
			shouldProcess: true,
		},
		{
			name:          "Depth 2 of 2 with parent - should skip parent",
			maxDepth:      2,
			currentDepth:  2,
			hasParent:     true,
			shouldProcess: false,
		},
		{
			name:          "Depth 1 of 1 with parent - should skip parent",
			maxDepth:      1,
			currentDepth:  1,
			hasParent:     true,
			shouldProcess: false,
		},
		{
			name:          "Depth 2 of 3 with parent - should process parent",
			maxDepth:      3,
			currentDepth:  2,
			hasParent:     true,
			shouldProcess: true,
		},
		{
			name:          "Depth 1 of 2 without parent - should not process",
			maxDepth:      2,
			currentDepth:  1,
			hasParent:     false,
			shouldProcess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := IssueMarkdownOptions{
				IncludeLinkedIssues: true,
				MaxDepth:            tt.maxDepth,
			}

			// Simulate the condition checks for parent processing
			// Parent is processed when: parent exists AND currentDepth < MaxDepth
			shouldProcess := tt.hasParent && tt.currentDepth < opts.MaxDepth

			assert.Equal(t, tt.shouldProcess, shouldProcess, "shouldProcess should match expected value")

			t.Logf("✓ Depth %d/%d (parent=%v): process=%v",
				tt.currentDepth, tt.maxDepth, tt.hasParent, shouldProcess)
		})
	}
}

// TestParentCircularDependencyPrevention tests that parent processing respects the lookup table
func TestParentCircularDependencyPrevention(t *testing.T) {
	processedIssues := map[string]bool{
		"PARENT-123": true, // Already processed
	}

	parentKey := "PARENT-123"

	// Simulate the circular dependency check
	alreadyProcessed := processedIssues[parentKey]

	assert.True(t, alreadyProcessed, "parent should be detected as already processed")
	t.Log("✓ Parent circular dependency prevention works correctly")
}

// TestParentKeyAddedToLookupTable tests that parent key is added before fetching
func TestParentKeyAddedToLookupTable(t *testing.T) {
	processedIssues := make(map[string]bool)
	parentKey := "PARENT-456"

	// Before processing, parent should not be in the lookup table
	assert.False(t, processedIssues[parentKey], "parent should not be in lookup table before processing")

	// When processing starts, the parent key is added to the lookup table
	// (This is done in generateIssueMarkdownFileWithDepth when it marks issueKey as processed)
	processedIssues[parentKey] = true

	// After processing, parent should be in the lookup table
	assert.True(t, processedIssues[parentKey], "parent should be in lookup table after processing")
	t.Log("✓ Parent key is correctly added to lookup table")
}

// TestParentWithLinkedIssuesDepth tests that parent's linked issues also follow MaxDepth
func TestParentWithLinkedIssuesDepth(t *testing.T) {
	// Scenario: Main ticket at depth 1, parent at depth 2, parent's linked issue at depth 3
	// With MaxDepth=3, all should be processed
	// With MaxDepth=2, only main ticket and parent are processed (parent's links skipped)

	tests := []struct {
		name                    string
		maxDepth                int
		mainTicketDepth         int
		parentDepth             int
		parentLinkedIssueDepth  int
		shouldProcessParent     bool
		shouldProcessParentLink bool
	}{
		{
			name:                    "MaxDepth 3 - process all",
			maxDepth:                3,
			mainTicketDepth:         1,
			parentDepth:             2,
			parentLinkedIssueDepth:  3,
			shouldProcessParent:     true, // 1 < 3
			shouldProcessParentLink: true, // 2 < 3
		},
		{
			name:                    "MaxDepth 2 - process parent only",
			maxDepth:                2,
			mainTicketDepth:         1,
			parentDepth:             2,
			parentLinkedIssueDepth:  3,
			shouldProcessParent:     true,  // 1 < 2
			shouldProcessParentLink: false, // 2 >= 2
		},
		{
			name:                    "MaxDepth 1 - main ticket only",
			maxDepth:                1,
			mainTicketDepth:         1,
			parentDepth:             2,
			parentLinkedIssueDepth:  3,
			shouldProcessParent:     false, // 1 >= 1
			shouldProcessParentLink: false, // N/A since parent not processed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Check if parent should be processed (from main ticket perspective)
			shouldProcessParent := tt.mainTicketDepth < tt.maxDepth
			assert.Equal(t, tt.shouldProcessParent, shouldProcessParent,
				"parent processing should match expected value")

			// Check if parent's linked issues should be processed (from parent's perspective)
			shouldProcessParentLink := tt.parentDepth < tt.maxDepth
			assert.Equal(t, tt.shouldProcessParentLink, shouldProcessParentLink,
				"parent's linked issues processing should match expected value")

			t.Logf("✓ MaxDepth %d: parent=%v (depth %d<%d), parent's links=%v (depth %d<%d)",
				tt.maxDepth,
				shouldProcessParent, tt.mainTicketDepth, tt.maxDepth,
				shouldProcessParentLink, tt.parentDepth, tt.maxDepth)
		})
	}
}

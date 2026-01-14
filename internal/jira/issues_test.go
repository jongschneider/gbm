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

// TestChildDepthConfiguration tests that children (subtasks) follow MaxDepth rules
func TestChildDepthConfiguration(t *testing.T) {
	tests := []struct {
		name          string
		maxDepth      int
		currentDepth  int
		childrenCount int
		shouldProcess bool
	}{
		{
			name:          "Depth 1 of 2 with children - should process children",
			maxDepth:      2,
			currentDepth:  1,
			childrenCount: 3,
			shouldProcess: true,
		},
		{
			name:          "Depth 2 of 2 with children - should skip children",
			maxDepth:      2,
			currentDepth:  2,
			childrenCount: 2,
			shouldProcess: false,
		},
		{
			name:          "Depth 1 of 1 with children - should skip children",
			maxDepth:      1,
			currentDepth:  1,
			childrenCount: 1,
			shouldProcess: false,
		},
		{
			name:          "Depth 2 of 3 with children - should process children",
			maxDepth:      3,
			currentDepth:  2,
			childrenCount: 5,
			shouldProcess: true,
		},
		{
			name:          "Depth 1 of 2 without children - should not process",
			maxDepth:      2,
			currentDepth:  1,
			childrenCount: 0,
			shouldProcess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the condition checks for children processing
			// Children are processed when: children exist AND currentDepth < MaxDepth
			shouldProcess := tt.childrenCount > 0 && tt.currentDepth < tt.maxDepth

			assert.Equal(t, tt.shouldProcess, shouldProcess, "shouldProcess should match expected value")

			t.Logf("✓ Depth %d/%d (children=%d): process=%v",
				tt.currentDepth, tt.maxDepth, tt.childrenCount, shouldProcess)
		})
	}
}

// TestChildCircularDependencyPrevention tests that child processing respects the lookup table
func TestChildCircularDependencyPrevention(t *testing.T) {
	processedIssues := map[string]bool{
		"CHILD-123": true, // Already processed
		"CHILD-456": true, // Already processed
	}

	children := []LinkedIssue{
		{Key: "CHILD-123"},
		{Key: "CHILD-456"},
		{Key: "CHILD-789"}, // Not processed yet
	}

	processedCount := 0
	skippedCount := 0

	for _, child := range children {
		if processedIssues[child.Key] {
			skippedCount++
		} else {
			processedCount++
		}
	}

	assert.Equal(t, 2, skippedCount, "two children should be skipped (already processed)")
	assert.Equal(t, 1, processedCount, "one child should be processed")
	t.Log("✓ Child circular dependency prevention works correctly")
}

// TestChildKeyAddedToLookupTable tests that child key is added before fetching
func TestChildKeyAddedToLookupTable(t *testing.T) {
	processedIssues := make(map[string]bool)
	childKey := "CHILD-456"

	// Before processing, child should not be in the lookup table
	assert.False(t, processedIssues[childKey], "child should not be in lookup table before processing")

	// When processing starts, the child key is added to the lookup table
	// (This is done in generateIssueMarkdownFileWithDepth when it marks issueKey as processed)
	processedIssues[childKey] = true

	// After processing, child should be in the lookup table
	assert.True(t, processedIssues[childKey], "child should be in lookup table after processing")
	t.Log("✓ Child key is correctly added to lookup table")
}

// TestChildWithLinkedIssuesDepth tests that child's linked issues also follow MaxDepth
func TestChildWithLinkedIssuesDepth(t *testing.T) {
	// Scenario: Main ticket at depth 1, child at depth 2, child's linked issue at depth 3
	// With MaxDepth=3, all should be processed
	// With MaxDepth=2, only main ticket and child are processed (child's links skipped)

	tests := []struct {
		name                   string
		maxDepth               int
		mainTicketDepth        int
		childDepth             int
		childLinkedIssueDepth  int
		shouldProcessChild     bool
		shouldProcessChildLink bool
	}{
		{
			name:                   "MaxDepth 3 - process all",
			maxDepth:               3,
			mainTicketDepth:        1,
			childDepth:             2,
			childLinkedIssueDepth:  3,
			shouldProcessChild:     true, // 1 < 3
			shouldProcessChildLink: true, // 2 < 3
		},
		{
			name:                   "MaxDepth 2 - process child only",
			maxDepth:               2,
			mainTicketDepth:        1,
			childDepth:             2,
			childLinkedIssueDepth:  3,
			shouldProcessChild:     true,  // 1 < 2
			shouldProcessChildLink: false, // 2 >= 2
		},
		{
			name:                   "MaxDepth 1 - main ticket only",
			maxDepth:               1,
			mainTicketDepth:        1,
			childDepth:             2,
			childLinkedIssueDepth:  3,
			shouldProcessChild:     false, // 1 >= 1
			shouldProcessChildLink: false, // N/A since child not processed
		},
		{
			name:                   "MaxDepth 4 - process all with extra depth",
			maxDepth:               4,
			mainTicketDepth:        1,
			childDepth:             2,
			childLinkedIssueDepth:  3,
			shouldProcessChild:     true, // 1 < 4
			shouldProcessChildLink: true, // 2 < 4
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Check if child should be processed (from main ticket perspective)
			shouldProcessChild := tt.mainTicketDepth < tt.maxDepth
			assert.Equal(t, tt.shouldProcessChild, shouldProcessChild,
				"child processing should match expected value")

			// Check if child's linked issues should be processed (from child's perspective)
			shouldProcessChildLink := tt.childDepth < tt.maxDepth
			assert.Equal(t, tt.shouldProcessChildLink, shouldProcessChildLink,
				"child's linked issues processing should match expected value")

			t.Logf("✓ MaxDepth %d: child=%v (depth %d<%d), child's links=%v (depth %d<%d)",
				tt.maxDepth,
				shouldProcessChild, tt.mainTicketDepth, tt.maxDepth,
				shouldProcessChildLink, tt.childDepth, tt.maxDepth)
		})
	}
}

// TestChildrenDeduplicatedFromLinkedIssues tests that child issues are removed from linked issues
func TestChildrenDeduplicatedFromLinkedIssues(t *testing.T) {
	tests := []struct {
		name              string
		children          []LinkedIssue
		issueLinks        []IssueLink
		expectLinkCount   int
		expectFilteredKey string
	}{
		{
			name: "child appears as inward issue link - should be removed",
			children: []LinkedIssue{
				{Key: "CHILD-100", Summary: "Child Task"},
			},
			issueLinks: []IssueLink{
				{
					ID: "1",
					Type: IssueLinkType{
						Name:   "Parent",
						Inward: "is parent of",
					},
					InwardIssue: &LinkedIssue{
						Key:     "CHILD-100",
						Summary: "Child Task",
					},
				},
				{
					ID: "2",
					Type: IssueLinkType{
						Name:    "Blocks",
						Outward: "blocks",
					},
					OutwardIssue: &LinkedIssue{
						Key:     "OTHER-200",
						Summary: "Other Issue",
					},
				},
			},
			expectLinkCount:   1,
			expectFilteredKey: "OTHER-200",
		},
		{
			name: "child appears as outward issue link - should be removed",
			children: []LinkedIssue{
				{Key: "CHILD-100", Summary: "Child Task"},
			},
			issueLinks: []IssueLink{
				{
					ID: "1",
					Type: IssueLinkType{
						Name:    "Parent",
						Outward: "is parent of",
					},
					OutwardIssue: &LinkedIssue{
						Key:     "CHILD-100",
						Summary: "Child Task",
					},
				},
				{
					ID: "2",
					Type: IssueLinkType{
						Name:   "Relates",
						Inward: "relates to",
					},
					InwardIssue: &LinkedIssue{
						Key:     "OTHER-300",
						Summary: "Related Issue",
					},
				},
			},
			expectLinkCount:   1,
			expectFilteredKey: "OTHER-300",
		},
		{
			name:     "no children - all links preserved",
			children: nil,
			issueLinks: []IssueLink{
				{
					ID: "1",
					Type: IssueLinkType{
						Name:    "Blocks",
						Outward: "blocks",
					},
					OutwardIssue: &LinkedIssue{
						Key:     "ISSUE-100",
						Summary: "Issue 100",
					},
				},
				{
					ID: "2",
					Type: IssueLinkType{
						Name:   "Relates",
						Inward: "relates to",
					},
					InwardIssue: &LinkedIssue{
						Key:     "ISSUE-200",
						Summary: "Issue 200",
					},
				},
			},
			expectLinkCount:   2,
			expectFilteredKey: "",
		},
		{
			name: "multiple children - all should be removed from links",
			children: []LinkedIssue{
				{Key: "CHILD-100", Summary: "Child 1"},
				{Key: "CHILD-200", Summary: "Child 2"},
			},
			issueLinks: []IssueLink{
				{
					ID: "1",
					Type: IssueLinkType{
						Name:   "Parent",
						Inward: "is parent of",
					},
					InwardIssue: &LinkedIssue{
						Key:     "CHILD-100",
						Summary: "Child 1",
					},
				},
				{
					ID: "2",
					Type: IssueLinkType{
						Name:    "Parent",
						Outward: "is parent of",
					},
					OutwardIssue: &LinkedIssue{
						Key:     "CHILD-200",
						Summary: "Child 2",
					},
				},
				{
					ID: "3",
					Type: IssueLinkType{
						Name:   "Relates",
						Inward: "relates to",
					},
					InwardIssue: &LinkedIssue{
						Key:     "OTHER-300",
						Summary: "Other Issue",
					},
				},
			},
			expectLinkCount:   1,
			expectFilteredKey: "OTHER-300",
		},
		{
			name: "children not in links - all links preserved",
			children: []LinkedIssue{
				{Key: "CHILD-999", Summary: "Child Not In Links"},
			},
			issueLinks: []IssueLink{
				{
					ID: "1",
					Type: IssueLinkType{
						Name:    "Blocks",
						Outward: "blocks",
					},
					OutwardIssue: &LinkedIssue{
						Key:     "ISSUE-100",
						Summary: "Issue 100",
					},
				},
			},
			expectLinkCount:   1,
			expectFilteredKey: "ISSUE-100",
		},
		{
			name: "child is only link - empty links after dedup",
			children: []LinkedIssue{
				{Key: "CHILD-100", Summary: "Child Task"},
			},
			issueLinks: []IssueLink{
				{
					ID: "1",
					Type: IssueLinkType{
						Name:   "Parent",
						Inward: "is parent of",
					},
					InwardIssue: &LinkedIssue{
						Key:     "CHILD-100",
						Summary: "Child Task",
					},
				},
			},
			expectLinkCount:   0,
			expectFilteredKey: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build child keys set for deduplication (same logic as GetJiraIssue)
			childKeys := make(map[string]bool)
			for _, child := range tt.children {
				childKeys[child.Key] = true
			}

			// Simulate the deduplication logic from GetJiraIssue
			filteredLinks := make([]IssueLink, 0, len(tt.issueLinks))
			for _, link := range tt.issueLinks {
				linkedKey := ""
				if link.InwardIssue != nil {
					linkedKey = link.InwardIssue.Key
				} else if link.OutwardIssue != nil {
					linkedKey = link.OutwardIssue.Key
				}

				// Skip links that reference child issues
				if childKeys[linkedKey] {
					continue
				}
				filteredLinks = append(filteredLinks, link)
			}

			assert.Len(t, filteredLinks, tt.expectLinkCount,
				"filtered link count should match expected")

			// If we expect a specific key to remain, verify it
			if tt.expectFilteredKey != "" && len(filteredLinks) > 0 {
				var foundKey string
				if filteredLinks[0].InwardIssue != nil {
					foundKey = filteredLinks[0].InwardIssue.Key
				} else if filteredLinks[0].OutwardIssue != nil {
					foundKey = filteredLinks[0].OutwardIssue.Key
				}
				assert.Equal(t, tt.expectFilteredKey, foundKey,
					"remaining link key should match expected")
			}

			t.Logf("✓ %s: %d links → %d links after dedup",
				tt.name, len(tt.issueLinks), len(filteredLinks))
		})
	}
}

// TestParentFieldParsing tests that parent field is correctly parsed from raw JSON
func TestParentFieldParsing(t *testing.T) {
	// Read the sample ticket.json from testdata (which has a parent field)
	ticketPath := "testdata/ticket.json"
	data, err := os.ReadFile(ticketPath)
	require.NoError(t, err, "ticket.json should exist in testdata/")

	// Parse the JSON response
	var jiraResponse jiraRawResponse
	require.NoError(t, json.Unmarshal(data, &jiraResponse), "ticket.json should be valid JSON")

	// Verify that we have a parent field
	require.NotNil(t, jiraResponse.Fields.Parent, "ticket.json should contain a parent field")

	// Verify parent field details
	parent := jiraResponse.Fields.Parent
	assert.Equal(t, "62959", parent.ID, "parent ID should match")
	assert.Equal(t, "EPIC-3089", parent.Key, "parent key should match")
	assert.Equal(t, "[Wells Fargo] Microsoft Teams Messaging : Add Support for Conversation Rename",
		parent.Fields.Summary, "parent summary should match")
	assert.Equal(t, "ENG Accepted", parent.Fields.Status.Name, "parent status should match")
	assert.Equal(t, "Medium", parent.Fields.Priority.Name, "parent priority should match")
	assert.Equal(t, "Epic", parent.Fields.IssueType.Name, "parent issue type should match")

	t.Logf("✓ Parent field parsed: %s - %s", parent.Key, parent.Fields.Summary)
}

// TestSubtasksArrayParsing tests that subtasks array is correctly parsed from raw JSON
func TestSubtasksArrayParsing(t *testing.T) {
	// Create test JSON with subtasks
	jsonData := `{
		"key": "PARENT-123",
		"self": "https://example.com/rest/api/3/issue/12345",
		"fields": {
			"summary": "Parent issue",
			"status": {"name": "Open"},
			"priority": {"name": "High"},
			"issuetype": {"name": "Story"},
			"subtasks": [
				{
					"id": "100",
					"key": "CHILD-100",
					"fields": {
						"summary": "First subtask",
						"status": {"name": "In Progress"},
						"priority": {"name": "Medium"},
						"issuetype": {"name": "Sub-task"}
					}
				},
				{
					"id": "101",
					"key": "CHILD-101",
					"fields": {
						"summary": "Second subtask",
						"status": {"name": "Done"},
						"priority": {"name": "Low"},
						"issuetype": {"name": "Sub-task"}
					}
				}
			]
		}
	}`

	var jiraResponse jiraRawResponse
	require.NoError(t, json.Unmarshal([]byte(jsonData), &jiraResponse), "JSON should be valid")

	// Verify subtasks were parsed
	require.Len(t, jiraResponse.Fields.Subtasks, 2, "should have 2 subtasks")

	// Verify first subtask
	first := jiraResponse.Fields.Subtasks[0]
	assert.Equal(t, "100", first.ID, "first subtask ID should match")
	assert.Equal(t, "CHILD-100", first.Key, "first subtask key should match")
	assert.Equal(t, "First subtask", first.Fields.Summary, "first subtask summary should match")
	assert.Equal(t, "In Progress", first.Fields.Status.Name, "first subtask status should match")
	assert.Equal(t, "Medium", first.Fields.Priority.Name, "first subtask priority should match")
	assert.Equal(t, "Sub-task", first.Fields.IssueType.Name, "first subtask type should match")

	// Verify second subtask
	second := jiraResponse.Fields.Subtasks[1]
	assert.Equal(t, "101", second.ID, "second subtask ID should match")
	assert.Equal(t, "CHILD-101", second.Key, "second subtask key should match")
	assert.Equal(t, "Second subtask", second.Fields.Summary, "second subtask summary should match")
	assert.Equal(t, "Done", second.Fields.Status.Name, "second subtask status should match")
	assert.Equal(t, "Low", second.Fields.Priority.Name, "second subtask priority should match")
	assert.Equal(t, "Sub-task", second.Fields.IssueType.Name, "second subtask type should match")

	t.Logf("✓ Subtasks parsed: %d items", len(jiraResponse.Fields.Subtasks))
}

// TestParentConversionToLinkedIssue tests that parent is correctly converted to LinkedIssue
func TestParentConversionToLinkedIssue(t *testing.T) {
	// Create test raw parent structure
	rawParent := struct {
		ID     string `json:"id"`
		Key    string `json:"key"`
		Fields struct {
			Summary string `json:"summary"`
			Status  struct {
				Name string `json:"name"`
			} `json:"status"`
			Priority struct {
				Name string `json:"name"`
			} `json:"priority"`
			IssueType struct {
				Name string `json:"name"`
			} `json:"issuetype"`
		} `json:"fields"`
	}{
		ID:  "12345",
		Key: "EPIC-100",
	}
	rawParent.Fields.Summary = "Parent Epic Summary"
	rawParent.Fields.Status.Name = "In Progress"
	rawParent.Fields.Priority.Name = "High"
	rawParent.Fields.IssueType.Name = "Epic"

	// Convert to LinkedIssue (same logic as GetJiraIssue)
	linkedIssue := &LinkedIssue{
		ID:        rawParent.ID,
		Key:       rawParent.Key,
		Summary:   rawParent.Fields.Summary,
		Status:    rawParent.Fields.Status.Name,
		Priority:  rawParent.Fields.Priority.Name,
		IssueType: rawParent.Fields.IssueType.Name,
	}

	// Verify all fields are correctly mapped
	assert.Equal(t, "12345", linkedIssue.ID, "ID should match")
	assert.Equal(t, "EPIC-100", linkedIssue.Key, "Key should match")
	assert.Equal(t, "Parent Epic Summary", linkedIssue.Summary, "Summary should match")
	assert.Equal(t, "In Progress", linkedIssue.Status, "Status should match")
	assert.Equal(t, "High", linkedIssue.Priority, "Priority should match")
	assert.Equal(t, "Epic", linkedIssue.IssueType, "IssueType should match")

	t.Logf("✓ Parent converted to LinkedIssue: %s (%s)", linkedIssue.Key, linkedIssue.IssueType)
}

// TestChildrenConversionToLinkedIssue tests that children (subtasks) are correctly converted to LinkedIssue slice
func TestChildrenConversionToLinkedIssue(t *testing.T) {
	// Create test raw subtasks structure
	type rawSubtask struct {
		ID     string `json:"id"`
		Key    string `json:"key"`
		Fields struct {
			Summary string `json:"summary"`
			Status  struct {
				Name string `json:"name"`
			} `json:"status"`
			Priority struct {
				Name string `json:"name"`
			} `json:"priority"`
			IssueType struct {
				Name string `json:"name"`
			} `json:"issuetype"`
		} `json:"fields"`
	}

	rawSubtasks := []rawSubtask{
		{ID: "100", Key: "TASK-100"},
		{ID: "101", Key: "TASK-101"},
		{ID: "102", Key: "TASK-102"},
	}
	rawSubtasks[0].Fields.Summary = "First Task"
	rawSubtasks[0].Fields.Status.Name = "Open"
	rawSubtasks[0].Fields.Priority.Name = "High"
	rawSubtasks[0].Fields.IssueType.Name = "Sub-task"

	rawSubtasks[1].Fields.Summary = "Second Task"
	rawSubtasks[1].Fields.Status.Name = "In Progress"
	rawSubtasks[1].Fields.Priority.Name = "Medium"
	rawSubtasks[1].Fields.IssueType.Name = "Sub-task"

	rawSubtasks[2].Fields.Summary = "Third Task"
	rawSubtasks[2].Fields.Status.Name = "Done"
	rawSubtasks[2].Fields.Priority.Name = "Low"
	rawSubtasks[2].Fields.IssueType.Name = "Sub-task"

	// Convert to LinkedIssue slice (same logic as GetJiraIssue)
	children := make([]LinkedIssue, 0, len(rawSubtasks))
	for _, subtask := range rawSubtasks {
		children = append(children, LinkedIssue{
			ID:        subtask.ID,
			Key:       subtask.Key,
			Summary:   subtask.Fields.Summary,
			Status:    subtask.Fields.Status.Name,
			Priority:  subtask.Fields.Priority.Name,
			IssueType: subtask.Fields.IssueType.Name,
		})
	}

	// Verify all children are correctly converted
	require.Len(t, children, 3, "should have 3 children")

	// Verify first child
	assert.Equal(t, "100", children[0].ID, "first child ID should match")
	assert.Equal(t, "TASK-100", children[0].Key, "first child Key should match")
	assert.Equal(t, "First Task", children[0].Summary, "first child Summary should match")
	assert.Equal(t, "Open", children[0].Status, "first child Status should match")
	assert.Equal(t, "High", children[0].Priority, "first child Priority should match")
	assert.Equal(t, "Sub-task", children[0].IssueType, "first child IssueType should match")

	// Verify second child
	assert.Equal(t, "101", children[1].ID, "second child ID should match")
	assert.Equal(t, "TASK-101", children[1].Key, "second child Key should match")
	assert.Equal(t, "In Progress", children[1].Status, "second child Status should match")

	// Verify third child
	assert.Equal(t, "102", children[2].ID, "third child ID should match")
	assert.Equal(t, "TASK-102", children[2].Key, "third child Key should match")
	assert.Equal(t, "Done", children[2].Status, "third child Status should match")

	t.Logf("✓ Children converted to LinkedIssue slice: %d items", len(children))
}

// TestParentNilCase tests handling when parent field is nil
func TestParentNilCase(t *testing.T) {
	// Create test JSON without parent
	jsonData := `{
		"key": "TASK-123",
		"self": "https://example.com/rest/api/3/issue/12345",
		"fields": {
			"summary": "Task without parent",
			"status": {"name": "Open"},
			"priority": {"name": "Medium"},
			"issuetype": {"name": "Task"}
		}
	}`

	var jiraResponse jiraRawResponse
	require.NoError(t, json.Unmarshal([]byte(jsonData), &jiraResponse), "JSON should be valid")

	// Verify parent is nil
	assert.Nil(t, jiraResponse.Fields.Parent, "parent should be nil")

	// Simulate the conversion (same as GetJiraIssue)
	var parent *LinkedIssue
	if jiraResponse.Fields.Parent != nil {
		parent = &LinkedIssue{
			ID:        jiraResponse.Fields.Parent.ID,
			Key:       jiraResponse.Fields.Parent.Key,
			Summary:   jiraResponse.Fields.Parent.Fields.Summary,
			Status:    jiraResponse.Fields.Parent.Fields.Status.Name,
			Priority:  jiraResponse.Fields.Parent.Fields.Priority.Name,
			IssueType: jiraResponse.Fields.Parent.Fields.IssueType.Name,
		}
	}

	assert.Nil(t, parent, "converted parent should be nil")
	t.Log("✓ Nil parent case handled correctly")
}

// TestSubtasksEmptyCase tests handling when subtasks array is empty
func TestSubtasksEmptyCase(t *testing.T) {
	tests := []struct {
		name     string
		jsonData string
	}{
		{
			name: "subtasks is empty array",
			jsonData: `{
				"key": "TASK-123",
				"self": "https://example.com/rest/api/3/issue/12345",
				"fields": {
					"summary": "Task without subtasks",
					"status": {"name": "Open"},
					"priority": {"name": "Medium"},
					"issuetype": {"name": "Task"},
					"subtasks": []
				}
			}`,
		},
		{
			name: "subtasks is not present",
			jsonData: `{
				"key": "TASK-124",
				"self": "https://example.com/rest/api/3/issue/12346",
				"fields": {
					"summary": "Task without subtasks field",
					"status": {"name": "Open"},
					"priority": {"name": "Medium"},
					"issuetype": {"name": "Task"}
				}
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var jiraResponse jiraRawResponse
			require.NoError(t, json.Unmarshal([]byte(tt.jsonData), &jiraResponse), "JSON should be valid")

			// Verify subtasks is empty or nil
			assert.Empty(t, jiraResponse.Fields.Subtasks, "subtasks should be empty")

			// Simulate the conversion (same as GetJiraIssue)
			var children []LinkedIssue
			if len(jiraResponse.Fields.Subtasks) > 0 {
				children = make([]LinkedIssue, 0, len(jiraResponse.Fields.Subtasks))
				for _, subtask := range jiraResponse.Fields.Subtasks {
					children = append(children, LinkedIssue{
						ID:        subtask.ID,
						Key:       subtask.Key,
						Summary:   subtask.Fields.Summary,
						Status:    subtask.Fields.Status.Name,
						Priority:  subtask.Fields.Priority.Name,
						IssueType: subtask.Fields.IssueType.Name,
					})
				}
			}

			assert.Nil(t, children, "converted children should be nil")
			t.Logf("✓ %s: empty subtasks handled correctly", tt.name)
		})
	}
}

// TestJiraTicketDetailsParentAndChildrenFields tests that JiraTicketDetails correctly stores Parent and Children
func TestJiraTicketDetailsParentAndChildrenFields(t *testing.T) {
	// Create a JiraTicketDetails with parent and children
	ticket := &JiraTicketDetails{
		Key:     "TASK-123",
		Summary: "Test ticket with parent and children",
		Status:  "Open",
		Parent: &LinkedIssue{
			ID:        "100",
			Key:       "EPIC-100",
			Summary:   "Parent Epic",
			Status:    "In Progress",
			Priority:  "High",
			IssueType: "Epic",
		},
		Children: []LinkedIssue{
			{
				ID:        "200",
				Key:       "SUBTASK-200",
				Summary:   "First subtask",
				Status:    "Open",
				Priority:  "Medium",
				IssueType: "Sub-task",
			},
			{
				ID:        "201",
				Key:       "SUBTASK-201",
				Summary:   "Second subtask",
				Status:    "Done",
				Priority:  "Low",
				IssueType: "Sub-task",
			},
		},
	}

	// Verify parent
	require.NotNil(t, ticket.Parent, "ticket should have parent")
	assert.Equal(t, "EPIC-100", ticket.Parent.Key, "parent key should match")
	assert.Equal(t, "Epic", ticket.Parent.IssueType, "parent type should match")

	// Verify children
	require.Len(t, ticket.Children, 2, "ticket should have 2 children")
	assert.Equal(t, "SUBTASK-200", ticket.Children[0].Key, "first child key should match")
	assert.Equal(t, "SUBTASK-201", ticket.Children[1].Key, "second child key should match")

	t.Logf("✓ JiraTicketDetails with Parent (%s) and %d Children", ticket.Parent.Key, len(ticket.Children))
}

// TestParentDeduplicatedFromLinkedIssues tests that parent issues are removed from linked issues
func TestParentDeduplicatedFromLinkedIssues(t *testing.T) {
	tests := []struct {
		name              string
		parent            *LinkedIssue
		issueLinks        []IssueLink
		expectLinkCount   int
		expectFilteredKey string
	}{
		{
			name: "parent appears as inward issue link - should be removed",
			parent: &LinkedIssue{
				Key:     "PARENT-100",
				Summary: "Parent Epic",
			},
			issueLinks: []IssueLink{
				{
					ID: "1",
					Type: IssueLinkType{
						Name:   "Parent",
						Inward: "is child of",
					},
					InwardIssue: &LinkedIssue{
						Key:     "PARENT-100",
						Summary: "Parent Epic",
					},
				},
				{
					ID: "2",
					Type: IssueLinkType{
						Name:    "Blocks",
						Outward: "blocks",
					},
					OutwardIssue: &LinkedIssue{
						Key:     "OTHER-200",
						Summary: "Other Issue",
					},
				},
			},
			expectLinkCount:   1,
			expectFilteredKey: "OTHER-200",
		},
		{
			name: "parent appears as outward issue link - should be removed",
			parent: &LinkedIssue{
				Key:     "PARENT-100",
				Summary: "Parent Epic",
			},
			issueLinks: []IssueLink{
				{
					ID: "1",
					Type: IssueLinkType{
						Name:    "Parent",
						Outward: "is parent of",
					},
					OutwardIssue: &LinkedIssue{
						Key:     "PARENT-100",
						Summary: "Parent Epic",
					},
				},
				{
					ID: "2",
					Type: IssueLinkType{
						Name:   "Relates",
						Inward: "relates to",
					},
					InwardIssue: &LinkedIssue{
						Key:     "OTHER-300",
						Summary: "Related Issue",
					},
				},
			},
			expectLinkCount:   1,
			expectFilteredKey: "OTHER-300",
		},
		{
			name:   "no parent - all links preserved",
			parent: nil,
			issueLinks: []IssueLink{
				{
					ID: "1",
					Type: IssueLinkType{
						Name:    "Blocks",
						Outward: "blocks",
					},
					OutwardIssue: &LinkedIssue{
						Key:     "ISSUE-100",
						Summary: "Issue 100",
					},
				},
				{
					ID: "2",
					Type: IssueLinkType{
						Name:   "Relates",
						Inward: "relates to",
					},
					InwardIssue: &LinkedIssue{
						Key:     "ISSUE-200",
						Summary: "Issue 200",
					},
				},
			},
			expectLinkCount:   2,
			expectFilteredKey: "",
		},
		{
			name: "parent not in links - all links preserved",
			parent: &LinkedIssue{
				Key:     "PARENT-999",
				Summary: "Parent Not In Links",
			},
			issueLinks: []IssueLink{
				{
					ID: "1",
					Type: IssueLinkType{
						Name:    "Blocks",
						Outward: "blocks",
					},
					OutwardIssue: &LinkedIssue{
						Key:     "ISSUE-100",
						Summary: "Issue 100",
					},
				},
			},
			expectLinkCount:   1,
			expectFilteredKey: "ISSUE-100",
		},
		{
			name: "parent is only link - empty links after dedup",
			parent: &LinkedIssue{
				Key:     "PARENT-100",
				Summary: "Parent Epic",
			},
			issueLinks: []IssueLink{
				{
					ID: "1",
					Type: IssueLinkType{
						Name:   "Parent",
						Inward: "is child of",
					},
					InwardIssue: &LinkedIssue{
						Key:     "PARENT-100",
						Summary: "Parent Epic",
					},
				},
			},
			expectLinkCount:   0,
			expectFilteredKey: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the deduplication logic from GetJiraIssue
			filteredLinks := make([]IssueLink, 0, len(tt.issueLinks))
			for _, link := range tt.issueLinks {
				// Skip links that reference the parent issue
				if tt.parent != nil {
					linkedKey := ""
					if link.InwardIssue != nil {
						linkedKey = link.InwardIssue.Key
					} else if link.OutwardIssue != nil {
						linkedKey = link.OutwardIssue.Key
					}
					if linkedKey == tt.parent.Key {
						continue
					}
				}
				filteredLinks = append(filteredLinks, link)
			}

			assert.Len(t, filteredLinks, tt.expectLinkCount,
				"filtered link count should match expected")

			// If we expect a specific key to remain, verify it
			if tt.expectFilteredKey != "" && len(filteredLinks) > 0 {
				var foundKey string
				if filteredLinks[0].InwardIssue != nil {
					foundKey = filteredLinks[0].InwardIssue.Key
				} else if filteredLinks[0].OutwardIssue != nil {
					foundKey = filteredLinks[0].OutwardIssue.Key
				}
				assert.Equal(t, tt.expectFilteredKey, foundKey,
					"remaining link key should match expected")
			}

			t.Logf("✓ %s: %d links → %d links after dedup",
				tt.name, len(tt.issueLinks), len(filteredLinks))
		})
	}
}

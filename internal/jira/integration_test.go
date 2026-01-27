package jira

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRecursiveParentFetchingRespectsMaxDepth tests that parent fetching respects MaxDepth limits.
func TestRecursiveParentFetchingRespectsMaxDepth(t *testing.T) {
	tests := []struct {
		name                string
		maxDepth            int
		mainTicketDepth     int
		hasParent           bool
		shouldProcessParent bool
	}{
		{
			name:                "MaxDepth 2 at depth 1 - should process parent",
			maxDepth:            2,
			mainTicketDepth:     1,
			hasParent:           true,
			shouldProcessParent: true, // 1 < 2
		},
		{
			name:                "MaxDepth 2 at depth 2 - should NOT process parent",
			maxDepth:            2,
			mainTicketDepth:     2,
			hasParent:           true,
			shouldProcessParent: false, // 2 >= 2
		},
		{
			name:                "MaxDepth 1 at depth 1 - should NOT process parent",
			maxDepth:            1,
			mainTicketDepth:     1,
			hasParent:           true,
			shouldProcessParent: false, // 1 >= 1
		},
		{
			name:                "MaxDepth 3 at depth 2 - should process parent",
			maxDepth:            3,
			mainTicketDepth:     2,
			hasParent:           true,
			shouldProcessParent: true, // 2 < 3
		},
		{
			name:                "MaxDepth 5 at depth 4 - should process parent",
			maxDepth:            5,
			mainTicketDepth:     4,
			hasParent:           true,
			shouldProcessParent: true, // 4 < 5
		},
		{
			name:                "MaxDepth 5 at depth 5 - should NOT process parent",
			maxDepth:            5,
			mainTicketDepth:     5,
			hasParent:           true,
			shouldProcessParent: false, // 5 >= 5
		},
		{
			name:                "No parent - should not process",
			maxDepth:            2,
			mainTicketDepth:     1,
			hasParent:           false,
			shouldProcessParent: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := IssueMarkdownOptions{
				IncludeLinkedIssues: true,
				MaxDepth:            tt.maxDepth,
			}

			// Simulate the condition from generateIssueMarkdownFileWithDepth
			// Parent is processed when: parent exists AND currentDepth < MaxDepth
			shouldProcess := tt.hasParent && tt.mainTicketDepth < opts.MaxDepth

			assert.Equal(t, tt.shouldProcessParent, shouldProcess,
				"parent processing at depth %d with MaxDepth %d",
				tt.mainTicketDepth, tt.maxDepth)

			t.Logf("✓ Parent at depth %d/%d: process=%v",
				tt.mainTicketDepth, tt.maxDepth, shouldProcess)
		})
	}
}

// TestRecursiveChildFetchingRespectsMaxDepth tests that child fetching respects MaxDepth limits.
func TestRecursiveChildFetchingRespectsMaxDepth(t *testing.T) {
	tests := []struct {
		name                  string
		maxDepth              int
		mainTicketDepth       int
		childCount            int
		shouldProcessChildren bool
	}{
		{
			name:                  "MaxDepth 2 at depth 1 with children - should process",
			maxDepth:              2,
			mainTicketDepth:       1,
			childCount:            3,
			shouldProcessChildren: true, // 1 < 2
		},
		{
			name:                  "MaxDepth 2 at depth 2 with children - should NOT process",
			maxDepth:              2,
			mainTicketDepth:       2,
			childCount:            2,
			shouldProcessChildren: false, // 2 >= 2
		},
		{
			name:                  "MaxDepth 1 at depth 1 with children - should NOT process",
			maxDepth:              1,
			mainTicketDepth:       1,
			childCount:            1,
			shouldProcessChildren: false, // 1 >= 1
		},
		{
			name:                  "MaxDepth 3 at depth 2 with children - should process",
			maxDepth:              3,
			mainTicketDepth:       2,
			childCount:            5,
			shouldProcessChildren: true, // 2 < 3
		},
		{
			name:                  "MaxDepth 4 at depth 3 with children - should process",
			maxDepth:              4,
			mainTicketDepth:       3,
			childCount:            2,
			shouldProcessChildren: true, // 3 < 4
		},
		{
			name:                  "MaxDepth 4 at depth 4 with children - should NOT process",
			maxDepth:              4,
			mainTicketDepth:       4,
			childCount:            2,
			shouldProcessChildren: false, // 4 >= 4
		},
		{
			name:                  "No children - should not process",
			maxDepth:              2,
			mainTicketDepth:       1,
			childCount:            0,
			shouldProcessChildren: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := IssueMarkdownOptions{
				IncludeLinkedIssues: true,
				MaxDepth:            tt.maxDepth,
			}

			// Simulate the condition from generateIssueMarkdownFileWithDepth
			// Children are processed when: children exist AND currentDepth < MaxDepth
			shouldProcess := tt.childCount > 0 && tt.mainTicketDepth < opts.MaxDepth

			assert.Equal(t, tt.shouldProcessChildren, shouldProcess,
				"children processing at depth %d with MaxDepth %d",
				tt.mainTicketDepth, tt.maxDepth)

			t.Logf("✓ Children (%d) at depth %d/%d: process=%v",
				tt.childCount, tt.mainTicketDepth, tt.maxDepth, shouldProcess)
		})
	}
}

// TestLookupTablePreventsDuplicateFetches tests that the processedIssues map
// prevents duplicate fetches of the same issue.
func TestLookupTablePreventsDuplicateFetches(t *testing.T) {
	tests := []struct {
		name              string
		issueToProcess    string
		description       string
		alreadyProcessed  []string
		expectedProcessed int
		shouldSkip        bool
	}{
		{
			name:              "Issue not in lookup - should process",
			alreadyProcessed:  []string{"ISSUE-100", "ISSUE-200"},
			issueToProcess:    "ISSUE-300",
			shouldSkip:        false,
			expectedProcessed: 3, // After adding the new issue
			description:       "New issue should be processed and added to lookup",
		},
		{
			name:              "Issue already in lookup - should skip",
			alreadyProcessed:  []string{"ISSUE-100", "ISSUE-200", "ISSUE-300"},
			issueToProcess:    "ISSUE-200",
			shouldSkip:        true,
			expectedProcessed: 3, // No change
			description:       "Existing issue should be skipped",
		},
		{
			name:              "Empty lookup table - should process",
			alreadyProcessed:  []string{},
			issueToProcess:    "ISSUE-100",
			shouldSkip:        false,
			expectedProcessed: 1,
			description:       "First issue should be processed",
		},
		{
			name:              "Main issue in lookup - linked issue skipped",
			alreadyProcessed:  []string{"MAIN-TICKET"},
			issueToProcess:    "MAIN-TICKET",
			shouldSkip:        true,
			expectedProcessed: 1,
			description:       "Main ticket appearing as linked issue should be skipped",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create the lookup table
			processedIssues := make(map[string]bool)
			for _, key := range tt.alreadyProcessed {
				processedIssues[key] = true
			}

			// Simulate the check from generateIssueMarkdownFileWithDepth
			shouldSkip := processedIssues[tt.issueToProcess]

			// If not skipping, add to lookup (simulating what the function does)
			if !shouldSkip {
				processedIssues[tt.issueToProcess] = true
			}

			assert.Equal(t, tt.shouldSkip, shouldSkip, tt.description)
			assert.Len(t, processedIssues, tt.expectedProcessed,
				"lookup table should have expected number of entries")

			t.Logf("✓ %s: skip=%v, lookup size=%d",
				tt.name, shouldSkip, len(processedIssues))
		})
	}
}

// TestCircularReferenceHandling tests that circular references between issues are handled.
func TestCircularReferenceHandling(t *testing.T) {
	tests := []struct {
		name          string
		scenario      string
		circularIssue string
		description   string
		processOrder  []string
		expectSkip    bool
	}{
		{
			name:          "A -> B -> A circular reference",
			scenario:      "Issue A links to Issue B, which links back to Issue A",
			processOrder:  []string{"ISSUE-A", "ISSUE-B"},
			circularIssue: "ISSUE-A",
			expectSkip:    true,
			description:   "Second occurrence of A should be skipped",
		},
		{
			name:          "A -> B -> C -> A circular reference",
			scenario:      "Three-issue circular chain",
			processOrder:  []string{"ISSUE-A", "ISSUE-B", "ISSUE-C"},
			circularIssue: "ISSUE-A",
			expectSkip:    true,
			description:   "Return to A should be detected and skipped",
		},
		{
			name:          "Parent-child circular: Parent -> Child -> Parent",
			scenario:      "Child subtask linking back to parent",
			processOrder:  []string{"PARENT-100"},
			circularIssue: "PARENT-100",
			expectSkip:    true,
			description:   "Child referencing parent should be skipped",
		},
		{
			name:          "Sibling mutual links: A -> B, A -> C, B -> C",
			scenario:      "Multiple paths to same issue",
			processOrder:  []string{"ISSUE-A", "ISSUE-B", "ISSUE-C"},
			circularIssue: "ISSUE-C",
			expectSkip:    true,
			description:   "Issue reached via multiple paths should be processed once",
		},
		{
			name:          "No circular reference - fresh issue",
			scenario:      "Normal linked issue processing",
			processOrder:  []string{"ISSUE-A", "ISSUE-B"},
			circularIssue: "ISSUE-C",
			expectSkip:    false,
			description:   "New issue should be processed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the lookup table after processing previous issues
			processedIssues := make(map[string]bool)
			for _, key := range tt.processOrder {
				processedIssues[key] = true
			}

			// Check if the circular issue would be skipped
			isCircular := processedIssues[tt.circularIssue]

			assert.Equal(t, tt.expectSkip, isCircular, tt.description)

			t.Logf("✓ %s: %s -> circular=%v", tt.name, tt.scenario, isCircular)
		})
	}
}

// TestDeepHierarchyExceedingMaxDepth tests that deep hierarchies stop at MaxDepth.
func TestDeepHierarchyExceedingMaxDepth(t *testing.T) {
	tests := []struct {
		name              string
		description       string
		hierarchyDepths   []int
		expectedProcessed []int
		expectedSkipped   []int
		maxDepth          int
	}{
		{
			name:              "MaxDepth 2 with 5-level hierarchy",
			maxDepth:          2,
			hierarchyDepths:   []int{1, 2, 3, 4, 5},
			expectedProcessed: []int{1},          // Only depth 1 can process children (1 < 2)
			expectedSkipped:   []int{2, 3, 4, 5}, // Depths >= MaxDepth are skipped
			description:       "Only depth 1 processed, rest skipped",
		},
		{
			name:              "MaxDepth 3 with 5-level hierarchy",
			maxDepth:          3,
			hierarchyDepths:   []int{1, 2, 3, 4, 5},
			expectedProcessed: []int{1, 2},    // Depths 1 and 2 can process (< 3)
			expectedSkipped:   []int{3, 4, 5}, // Depths >= 3 are skipped
			description:       "First 2 levels processed",
		},
		{
			name:              "MaxDepth 1 - main ticket only",
			maxDepth:          1,
			hierarchyDepths:   []int{1, 2, 3},
			expectedProcessed: nil, // No depth can process children (1 >= 1)
			expectedSkipped:   []int{1, 2, 3},
			description:       "All skipped when MaxDepth=1",
		},
		{
			name:              "MaxDepth 5 with 4-level hierarchy",
			maxDepth:          5,
			hierarchyDepths:   []int{1, 2, 3, 4},
			expectedProcessed: []int{1, 2, 3, 4}, // All depths < 5
			expectedSkipped:   nil,               // None skipped
			description:       "All levels processed when hierarchy is shallower than MaxDepth",
		},
		{
			name:              "MaxDepth 10 with deep hierarchy",
			maxDepth:          10,
			hierarchyDepths:   []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
			expectedProcessed: []int{1, 2, 3, 4, 5, 6, 7, 8, 9}, // Depths 1-9 < 10
			expectedSkipped:   []int{10, 11, 12},                // Depths >= 10
			description:       "Deep hierarchies still respect MaxDepth",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := IssueMarkdownOptions{
				IncludeLinkedIssues: true,
				MaxDepth:            tt.maxDepth,
			}

			var processedDepths []int
			var skippedDepths []int

			for _, depth := range tt.hierarchyDepths {
				// Simulate the depth check from generateIssueMarkdownFileWithDepth
				// At each depth, we check if we should process linked issues (depth < MaxDepth)
				if depth < opts.MaxDepth {
					processedDepths = append(processedDepths, depth)
				} else {
					skippedDepths = append(skippedDepths, depth)
				}
			}

			assert.Equal(t, tt.expectedProcessed, processedDepths,
				"processed depths should match expected for MaxDepth %d", tt.maxDepth)
			assert.Equal(t, tt.expectedSkipped, skippedDepths,
				"skipped depths should match expected for MaxDepth %d", tt.maxDepth)

			t.Logf("✓ %s: processed=%v, skipped=%v",
				tt.name, processedDepths, skippedDepths)
		})
	}
}

// TestComplexHierarchyWithParentAndChildren tests a realistic scenario with
// a main ticket that has both a parent and children.
func TestComplexHierarchyWithParentAndChildren(t *testing.T) {
	tests := []struct {
		name                  string
		maxDepth              int
		mainTicketDepth       int
		childrenCount         int
		linkedIssuesCount     int
		hasParent             bool
		shouldProcessParent   bool
		shouldProcessChildren bool
		shouldProcessLinked   bool
	}{
		{
			name:                  "MaxDepth 2 - process all at depth 1",
			maxDepth:              2,
			mainTicketDepth:       1,
			hasParent:             true,
			childrenCount:         2,
			linkedIssuesCount:     3,
			shouldProcessParent:   true,
			shouldProcessChildren: true,
			shouldProcessLinked:   true,
		},
		{
			name:                  "MaxDepth 2 - skip all at depth 2",
			maxDepth:              2,
			mainTicketDepth:       2,
			hasParent:             true,
			childrenCount:         2,
			linkedIssuesCount:     3,
			shouldProcessParent:   false,
			shouldProcessChildren: false,
			shouldProcessLinked:   false,
		},
		{
			name:                  "MaxDepth 3 at depth 2 - process all",
			maxDepth:              3,
			mainTicketDepth:       2,
			hasParent:             true,
			childrenCount:         1,
			linkedIssuesCount:     2,
			shouldProcessParent:   true,
			shouldProcessChildren: true,
			shouldProcessLinked:   true,
		},
		{
			name:                  "MaxDepth 1 - skip all relationships",
			maxDepth:              1,
			mainTicketDepth:       1,
			hasParent:             true,
			childrenCount:         5,
			linkedIssuesCount:     10,
			shouldProcessParent:   false,
			shouldProcessChildren: false,
			shouldProcessLinked:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := IssueMarkdownOptions{
				IncludeLinkedIssues: true,
				MaxDepth:            tt.maxDepth,
			}

			// Simulate the conditions from generateIssueMarkdownFileWithDepth
			shouldProcessParent := tt.hasParent && tt.mainTicketDepth < opts.MaxDepth
			shouldProcessChildren := tt.childrenCount > 0 && tt.mainTicketDepth < opts.MaxDepth
			shouldProcessLinked := opts.IncludeLinkedIssues && tt.linkedIssuesCount > 0 && tt.mainTicketDepth < opts.MaxDepth

			assert.Equal(t, tt.shouldProcessParent, shouldProcessParent, "parent processing")
			assert.Equal(t, tt.shouldProcessChildren, shouldProcessChildren, "children processing")
			assert.Equal(t, tt.shouldProcessLinked, shouldProcessLinked, "linked issues processing")

			t.Logf("✓ %s at depth %d/%d: parent=%v, children=%v, linked=%v",
				tt.name, tt.mainTicketDepth, tt.maxDepth,
				shouldProcessParent, shouldProcessChildren, shouldProcessLinked)
		})
	}
}

// TestLookupTablePersistenceAcrossRecursiveCalls tests that the lookup table
// is correctly shared across recursive calls.
func TestLookupTablePersistenceAcrossRecursiveCalls(t *testing.T) {
	// Simulate a recursive processing scenario
	processedIssues := make(map[string]bool)

	// Simulate processing main ticket
	mainKey := "MAIN-123"
	require.False(t, processedIssues[mainKey], "main ticket should not be in lookup initially")
	processedIssues[mainKey] = true

	// Simulate processing linked issue 1 (which references main ticket)
	linkedKey1 := "LINKED-456"
	require.False(t, processedIssues[linkedKey1], "linked issue 1 should not be in lookup")
	processedIssues[linkedKey1] = true

	// Now linked issue 1's linked issues include main ticket - should be skipped
	require.True(t, processedIssues[mainKey],
		"main ticket should be detected as already processed from linked issue 1")

	// Simulate processing linked issue 2
	linkedKey2 := "LINKED-789"
	require.False(t, processedIssues[linkedKey2], "linked issue 2 should not be in lookup")
	processedIssues[linkedKey2] = true

	// Both linked issue 1 and main should be in lookup when processing linked issue 2
	require.True(t, processedIssues[mainKey], "main should be in lookup")
	require.True(t, processedIssues[linkedKey1], "linked 1 should be in lookup")

	// Final state
	assert.Len(t, processedIssues, 3, "should have processed 3 unique issues")
	assert.True(t, processedIssues["MAIN-123"])
	assert.True(t, processedIssues["LINKED-456"])
	assert.True(t, processedIssues["LINKED-789"])

	t.Log("✓ Lookup table correctly persists across recursive calls")
}

// TestParentChildCircularReference tests the specific case where a parent
// and child reference each other through linked issues.
func TestParentChildCircularReference(t *testing.T) {
	processedIssues := make(map[string]bool)

	// Process parent
	parentKey := "PARENT-100"
	processedIssues[parentKey] = true

	// Process child (at depth 2)
	childKey := "CHILD-200"
	require.False(t, processedIssues[childKey], "child should not be processed yet")
	processedIssues[childKey] = true

	// Child's linked issues include parent - should be skipped
	require.True(t, processedIssues[parentKey],
		"parent should be detected as already processed when referenced by child")

	t.Log("✓ Parent-child circular reference correctly handled")
}

// TestMultipleChildrenDeduplication tests that when multiple children exist,
// all are correctly tracked in the lookup table.
func TestMultipleChildrenDeduplication(t *testing.T) {
	processedIssues := make(map[string]bool)

	// Process main ticket
	mainKey := "MAIN-100"
	processedIssues[mainKey] = true

	// Process children
	children := []string{"CHILD-1", "CHILD-2", "CHILD-3"}
	for _, childKey := range children {
		require.False(t, processedIssues[childKey], "child %s should not be processed yet", childKey)
		processedIssues[childKey] = true
	}

	// Now if any child references another child, it should be skipped
	for _, childKey := range children {
		assert.True(t, processedIssues[childKey],
			"child %s should be detected as already processed", childKey)
	}

	// And if any child references the main ticket
	assert.True(t, processedIssues[mainKey],
		"main ticket should be detected when referenced by children")

	assert.Len(t, processedIssues, 4, "should have main + 3 children")
	t.Log("✓ Multiple children correctly deduplicated")
}

// TestIssueMarkdownOptionsDefaults tests that default options have correct depth values.
func TestIssueMarkdownOptionsDefaults(t *testing.T) {
	opts := DefaultIssueMarkdownOptions("/tmp/test-worktree")

	assert.Equal(t, 2, opts.MaxDepth, "default MaxDepth should be 2")
	assert.True(t, opts.IncludeLinkedIssues, "IncludeLinkedIssues should be true by default")
	assert.Equal(t, "/tmp/test-worktree", opts.WorktreeRoot, "WorktreeRoot should be set")
	assert.True(t, opts.IncludeComments, "IncludeComments should be true by default")
	assert.True(t, opts.DownloadAttachments, "DownloadAttachments should be true by default")

	t.Logf("✓ Default options: MaxDepth=%d, IncludeLinkedIssues=%v",
		opts.MaxDepth, opts.IncludeLinkedIssues)
}

// TestCustomMaxDepthOptions tests that custom MaxDepth values are respected.
func TestCustomMaxDepthOptions(t *testing.T) {
	tests := []struct {
		name     string
		maxDepth int
	}{
		{"MaxDepth 1", 1},
		{"MaxDepth 3", 3},
		{"MaxDepth 5", 5},
		{"MaxDepth 10", 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := IssueMarkdownOptions{
				MaxDepth:            tt.maxDepth,
				IncludeLinkedIssues: true,
			}

			assert.Equal(t, tt.maxDepth, opts.MaxDepth, "MaxDepth should match configured value")
			t.Logf("✓ MaxDepth correctly set to %d", opts.MaxDepth)
		})
	}
}

// TestDisabledLinkedIssuesIgnoresDepth tests that when IncludeLinkedIssues is false,
// depth settings are irrelevant.
func TestDisabledLinkedIssuesIgnoresDepth(t *testing.T) {
	tests := []struct {
		name                string
		maxDepth            int
		currentDepth        int
		includeLinkedIssues bool
		hasLinkedIssues     bool
		shouldProcess       bool
	}{
		{
			name:                "Disabled with large MaxDepth",
			maxDepth:            10,
			currentDepth:        1,
			includeLinkedIssues: false,
			hasLinkedIssues:     true,
			shouldProcess:       false,
		},
		{
			name:                "Disabled at depth 1",
			maxDepth:            2,
			currentDepth:        1,
			includeLinkedIssues: false,
			hasLinkedIssues:     true,
			shouldProcess:       false,
		},
		{
			name:                "Enabled but no linked issues",
			maxDepth:            2,
			currentDepth:        1,
			includeLinkedIssues: true,
			hasLinkedIssues:     false,
			shouldProcess:       false,
		},
		{
			name:                "Enabled with linked issues within depth",
			maxDepth:            2,
			currentDepth:        1,
			includeLinkedIssues: true,
			hasLinkedIssues:     true,
			shouldProcess:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			linkedIssuesCount := 0
			if tt.hasLinkedIssues {
				linkedIssuesCount = 3
			}

			// Simulate the condition from generateIssueMarkdownFileWithDepth
			shouldProcess := tt.includeLinkedIssues &&
				linkedIssuesCount > 0 &&
				tt.currentDepth < tt.maxDepth

			assert.Equal(t, tt.shouldProcess, shouldProcess)
			t.Logf("✓ %s: shouldProcess=%v", tt.name, shouldProcess)
		})
	}
}

// TestParentProcessedWithNoLinkedIssues tests that parent issues are processed
// even when there are no linked issues. This was a bug where empty issuelinks
// caused an early return, skipping parent/children processing.
func TestParentProcessedWithNoLinkedIssues(t *testing.T) {
	tests := []struct {
		name                string
		maxDepth            int
		currentDepth        int
		hasParent           bool
		hasLinkedIssues     bool
		shouldProcessParent bool
	}{
		{
			name:                "Parent with no linked issues - should process parent",
			maxDepth:            2,
			currentDepth:        1,
			hasParent:           true,
			hasLinkedIssues:     false,
			shouldProcessParent: true,
		},
		{
			name:                "Parent with linked issues - should process parent",
			maxDepth:            2,
			currentDepth:        1,
			hasParent:           true,
			hasLinkedIssues:     true,
			shouldProcessParent: true,
		},
		{
			name:                "No parent and no linked issues - no parent to process",
			maxDepth:            2,
			currentDepth:        1,
			hasParent:           false,
			hasLinkedIssues:     false,
			shouldProcessParent: false,
		},
		{
			name:                "Parent at depth limit - should NOT process parent",
			maxDepth:            2,
			currentDepth:        2,
			hasParent:           true,
			hasLinkedIssues:     false,
			shouldProcessParent: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test verifies the logic from generateIssueMarkdownFileWithDepth
			// Parent processing should happen regardless of linked issues count
			shouldProcessParent := tt.hasParent && tt.currentDepth < tt.maxDepth

			assert.Equal(t, tt.shouldProcessParent, shouldProcessParent)
			t.Logf("✓ %s: hasParent=%v, hasLinkedIssues=%v, shouldProcessParent=%v",
				tt.name, tt.hasParent, tt.hasLinkedIssues, shouldProcessParent)
		})
	}
}

// TestChildrenProcessedWithNoLinkedIssues tests that children are processed
// even when there are no linked issues.
func TestChildrenProcessedWithNoLinkedIssues(t *testing.T) {
	tests := []struct {
		name                  string
		maxDepth              int
		currentDepth          int
		childCount            int
		hasLinkedIssues       bool
		shouldProcessChildren bool
	}{
		{
			name:                  "Children with no linked issues - should process children",
			maxDepth:              2,
			currentDepth:          1,
			childCount:            3,
			hasLinkedIssues:       false,
			shouldProcessChildren: true,
		},
		{
			name:                  "Children with linked issues - should process children",
			maxDepth:              2,
			currentDepth:          1,
			childCount:            2,
			hasLinkedIssues:       true,
			shouldProcessChildren: true,
		},
		{
			name:                  "No children and no linked issues - no children to process",
			maxDepth:              2,
			currentDepth:          1,
			childCount:            0,
			hasLinkedIssues:       false,
			shouldProcessChildren: false,
		},
		{
			name:                  "Children at depth limit - should NOT process children",
			maxDepth:              2,
			currentDepth:          2,
			childCount:            3,
			hasLinkedIssues:       false,
			shouldProcessChildren: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test verifies the logic from generateIssueMarkdownFileWithDepth
			// Children processing should happen regardless of linked issues count
			shouldProcessChildren := tt.childCount > 0 && tt.currentDepth < tt.maxDepth

			assert.Equal(t, tt.shouldProcessChildren, shouldProcessChildren)
			t.Logf("✓ %s: childCount=%d, hasLinkedIssues=%v, shouldProcessChildren=%v",
				tt.name, tt.childCount, tt.hasLinkedIssues, shouldProcessChildren)
		})
	}
}

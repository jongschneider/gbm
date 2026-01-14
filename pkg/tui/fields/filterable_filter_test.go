package fields

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// realJiraOptions returns options matching the user's actual JIRA data
func realJiraOptions() []Option {
	return []Option{
		{Label: "INGSVC-6468 - EMAIL: ical extra text time parsing", Value: "INGSVC-6468"},
		{Label: "INGSVC-6457 - New Integration - Facebook Business Pages", Value: "INGSVC-6457"},
		{Label: "INGSVC-6375 - Wells Fargo [Prod9-cid-2] Conversation name edits are not being recorded.", Value: "INGSVC-6375"},
		{Label: "INGSVC-6277 - Microsoft 365 Copilot : Capture File Used in Copilot for SharePoint Prompt", Value: "INGSVC-6277"},
		{Label: "INGSVC-6099 - MS Teams - Malformed chat.json ( Incorrect attachment association)", Value: "INGSVC-6099"},
		{Label: "INGSVC-5993 - MS Teams: Investigate handling of Graph API `UnknownError` 400 Responses", Value: "INGSVC-5993"},
		{Label: "INGSVC-5958 - ZOOM_WHITEBOARDS: Fail with status BADCONFIG", Value: "INGSVC-5958"},
		{Label: "INGSVC-5739 - New Integration - Refinitiv LSEG Messenger API", Value: "INGSVC-5739"},
		{Label: "INGSVC-5693 - Convert MsTeams to the ChatUploader", Value: "INGSVC-5693"},
		{Label: "INGSVC-5070 - zip processor should look up api keys from queue message instead of joining datums onto api key", Value: "INGSVC-5070"},
	}
}

func TestFilterable_FilterOptions_SubstringMatching(t *testing.T) {
	testCases := []struct {
		name           string
		query          string
		expectedLabels []string
		description    string
	}{
		{
			name:  "empty query returns all options",
			query: "",
			expectedLabels: []string{
				"INGSVC-6468 - EMAIL: ical extra text time parsing",
				"INGSVC-6457 - New Integration - Facebook Business Pages",
				"INGSVC-6375 - Wells Fargo [Prod9-cid-2] Conversation name edits are not being recorded.",
				"INGSVC-6277 - Microsoft 365 Copilot : Capture File Used in Copilot for SharePoint Prompt",
				"INGSVC-6099 - MS Teams - Malformed chat.json ( Incorrect attachment association)",
				"INGSVC-5993 - MS Teams: Investigate handling of Graph API `UnknownError` 400 Responses",
				"INGSVC-5958 - ZOOM_WHITEBOARDS: Fail with status BADCONFIG",
				"INGSVC-5739 - New Integration - Refinitiv LSEG Messenger API",
				"INGSVC-5693 - Convert MsTeams to the ChatUploader",
				"INGSVC-5070 - zip processor should look up api keys from queue message instead of joining datums onto api key",
			},
			description: "All options should be shown when query is empty",
		},
		{
			name:  "query 'f' matches options containing 'f'",
			query: "f",
			expectedLabels: []string{
				"INGSVC-6457 - New Integration - Facebook Business Pages",
				"INGSVC-6375 - Wells Fargo [Prod9-cid-2] Conversation name edits are not being recorded.",
				"INGSVC-6277 - Microsoft 365 Copilot : Capture File Used in Copilot for SharePoint Prompt",
				"INGSVC-6099 - MS Teams - Malformed chat.json ( Incorrect attachment association)",
				"INGSVC-5993 - MS Teams: Investigate handling of Graph API `UnknownError` 400 Responses",
				"INGSVC-5958 - ZOOM_WHITEBOARDS: Fail with status BADCONFIG",
				"INGSVC-5739 - New Integration - Refinitiv LSEG Messenger API",
				"INGSVC-5070 - zip processor should look up api keys from queue message instead of joining datums onto api key",
			},
			description: "Should match all options with 'f' anywhere in label",
		},
		{
			name:  "query 'fa' matches Facebook, Fargo, Fail",
			query: "fa",
			expectedLabels: []string{
				"INGSVC-6457 - New Integration - Facebook Business Pages",
				"INGSVC-6375 - Wells Fargo [Prod9-cid-2] Conversation name edits are not being recorded.",
				"INGSVC-5958 - ZOOM_WHITEBOARDS: Fail with status BADCONFIG",
			},
			description: "Should match Facebook, Fargo, Fail",
		},
		{
			name:  "query 'fac' should match Facebook",
			query: "fac",
			expectedLabels: []string{
				"INGSVC-6457 - New Integration - Facebook Business Pages",
			},
			description: "CRITICAL: 'fac' is a substring of 'Facebook' - must match",
		},
		{
			name:  "query 'face' should match Facebook",
			query: "face",
			expectedLabels: []string{
				"INGSVC-6457 - New Integration - Facebook Business Pages",
			},
			description: "'face' is a substring of 'Facebook'",
		},
		{
			name:  "query 'facebook' should match Facebook",
			query: "facebook",
			expectedLabels: []string{
				"INGSVC-6457 - New Integration - Facebook Business Pages",
			},
			description: "Full word 'facebook' should match",
		},
		{
			name:  "query 'FACEBOOK' case-insensitive match",
			query: "FACEBOOK",
			expectedLabels: []string{
				"INGSVC-6457 - New Integration - Facebook Business Pages",
			},
			description: "Case-insensitive match for 'FACEBOOK'",
		},
		{
			name:  "query 'teams' matches MS Teams issues",
			query: "teams",
			expectedLabels: []string{
				"INGSVC-6099 - MS Teams - Malformed chat.json ( Incorrect attachment association)",
				"INGSVC-5993 - MS Teams: Investigate handling of Graph API `UnknownError` 400 Responses",
				"INGSVC-5693 - Convert MsTeams to the ChatUploader",
			},
			description: "Should match all Teams-related issues",
		},
		{
			name:           "query 'xyz' matches nothing",
			query:          "xyz",
			expectedLabels: []string{},
			description:    "Non-existent substring should match nothing",
		},
		{
			name:           "query matches value (issue key)",
			query:          "6468",
			expectedLabels: []string{"INGSVC-6468 - EMAIL: ical extra text time parsing"},
			description:    "Should match on Value field (issue key)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			f := NewFilterable("key", "title", "desc", realJiraOptions())
			f.focused = true

			// Set the text input value directly
			f.textInput.SetValue(tc.query)

			// Call filterOptions
			f.filterOptions()

			// Verify results
			require.Equal(t, len(tc.expectedLabels), len(f.filtered),
				"%s: expected %d matches, got %d", tc.description, len(tc.expectedLabels), len(f.filtered))

			for i, expected := range tc.expectedLabels {
				assert.Equal(t, expected, f.filtered[i].Label,
					"%s: mismatch at index %d", tc.description, i)
			}
		})
	}
}

func TestFilterable_FilterOptions_SequentialTyping(t *testing.T) {
	// This test simulates the user typing character by character for "fac"
	// The key behavior is that each additional character narrows the results
	f := NewFilterable("key", "title", "desc", realJiraOptions())
	f.focused = true

	// Type 'f' - should get some matches
	f.textInput.SetValue("f")
	f.filterOptions()
	fCount := len(f.filtered)
	assert.Greater(t, fCount, 0, "'f' should match at least one option")

	// Type 'fa' - should have fewer or equal matches
	f.textInput.SetValue("fa")
	f.filterOptions()
	faCount := len(f.filtered)
	assert.Equal(t, 3, faCount, "'fa' should match exactly 3 items: Facebook, Fargo, Fail")

	// Type 'fac' - should have fewer matches (only Facebook)
	f.textInput.SetValue("fac")
	f.filterOptions()
	facCount := len(f.filtered)
	assert.Equal(t, 1, facCount, "'fac' should match exactly 1 item: Facebook")

	// Verify the one match is Facebook
	if facCount > 0 {
		assert.Contains(t, f.filtered[0].Label, "Facebook", "The match for 'fac' should be Facebook")
	}
}

func TestFilterable_FilterOptions_BackspaceConsistency(t *testing.T) {
	// This test verifies that backspacing produces the same result as typing directly
	f1 := NewFilterable("key", "title", "desc", realJiraOptions())
	f1.focused = true

	// Type "fac" then backspace to "fa"
	f1.textInput.SetValue("fac")
	f1.filterOptions()
	facResults := len(f1.filtered)

	f1.textInput.SetValue("fa")
	f1.filterOptions()
	faAfterBackspace := make([]Option, len(f1.filtered))
	copy(faAfterBackspace, f1.filtered)

	// Fresh instance, type "fa" directly
	f2 := NewFilterable("key", "title", "desc", realJiraOptions())
	f2.focused = true
	f2.textInput.SetValue("fa")
	f2.filterOptions()
	faDirect := make([]Option, len(f2.filtered))
	copy(faDirect, f2.filtered)

	// Results should be identical
	assert.Equal(t, len(faDirect), len(faAfterBackspace),
		"Backspacing to 'fa' should produce same result as typing 'fa' directly")

	for i := range faDirect {
		assert.Equal(t, faDirect[i].Label, faAfterBackspace[i].Label,
			"Result mismatch at index %d", i)
	}

	// Also verify fac matched at least Facebook
	assert.GreaterOrEqual(t, facResults, 1, "'fac' should match at least Facebook")
}

func TestFilterable_Update_TypeCharacterUpdatesFilter(t *testing.T) {
	f := NewFilterable("key", "title", "desc", realJiraOptions())
	f.focused = true
	f.textInput.Focus() // MUST focus the textInput for it to accept key runes

	// Verify initial state - all options visible
	assert.Equal(t, len(realJiraOptions()), len(f.filtered))

	// Type 'f'
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}}
	field, _ := f.Update(msg)
	f = field.(*Filterable)

	// After typing 'f', should filter
	t.Logf("After 'f': textInput.Value()=%q, filtered count=%d", f.textInput.Value(), len(f.filtered))
	assert.Equal(t, "f", f.textInput.Value(), "text input should have 'f'")
	assert.Less(t, len(f.filtered), len(realJiraOptions()), "should have fewer results after filtering")

	// Type 'a' -> "fa"
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
	field, _ = f.Update(msg)
	f = field.(*Filterable)

	t.Logf("After 'a' (total 'fa'): textInput.Value()=%q, filtered count=%d", f.textInput.Value(), len(f.filtered))
	assert.Equal(t, "fa", f.textInput.Value(), "text input should have 'fa'")
	assert.Equal(t, 3, len(f.filtered), "should have 3 results for 'fa': Facebook, Fargo, Fail")

	// Type 'c' -> "fac"
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}}
	field, _ = f.Update(msg)
	f = field.(*Filterable)

	t.Logf("After 'c' (total 'fac'): textInput.Value()=%q, filtered count=%d", f.textInput.Value(), len(f.filtered))
	assert.Equal(t, "fac", f.textInput.Value(), "text input should have 'fac'")
	assert.Equal(t, 1, len(f.filtered), "CRITICAL: should have 1 result for 'fac': Facebook")

	if len(f.filtered) > 0 {
		assert.Contains(t, f.filtered[0].Label, "Facebook",
			"'fac' should match Facebook")
	}
}

func TestFilterable_Update_BackspaceUpdatesFilter(t *testing.T) {
	f := NewFilterable("key", "title", "desc", realJiraOptions())
	f.focused = true
	f.textInput.Focus() // MUST focus the textInput for it to accept key messages

	// Set initial value to "fac"
	f.textInput.SetValue("fac")
	f.filterOptions()
	initialCount := len(f.filtered)
	t.Logf("With 'fac': filtered count=%d", initialCount)

	// Backspace
	msg := tea.KeyMsg{Type: tea.KeyBackspace}
	field, _ := f.Update(msg)
	f = field.(*Filterable)

	t.Logf("After backspace (should be 'fa'): textInput.Value()=%q, filtered count=%d", f.textInput.Value(), len(f.filtered))
	assert.Equal(t, "fa", f.textInput.Value(), "text input should have 'fa' after backspace")
	assert.Equal(t, 3, len(f.filtered), "should have 3 results for 'fa' after backspace: Facebook, Fargo, Fail")

	// Verify Facebook is in results
	foundFacebook := false
	for _, opt := range f.filtered {
		if opt.Value == "INGSVC-6457" {
			foundFacebook = true
			break
		}
	}
	assert.True(t, foundFacebook, "Facebook should be in filtered results after backspacing to 'fa'")
}

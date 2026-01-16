package jira

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTable(t *testing.T) {
	tableJSON := `{
		"type": "doc",
		"version": 1,
		"content": [
			{
				"type": "table",
				"content": [
					{
						"type": "tableRow",
						"content": [
							{
								"type": "tableHeader",
								"content": [{"type": "paragraph", "content": [{"type": "text", "text": "Header 1", "marks": [{"type": "strong"}]}]}]
							},
							{
								"type": "tableHeader",
								"content": [{"type": "paragraph", "content": [{"type": "text", "text": "Header 2", "marks": [{"type": "strong"}]}]}]
							},
							{
								"type": "tableHeader",
								"content": [{"type": "paragraph", "content": [{"type": "text", "text": "Header 3", "marks": [{"type": "strong"}]}]}]
							}
						]
					},
					{
						"type": "tableRow",
						"content": [
							{"type": "tableCell", "content": [{"type": "paragraph", "content": [{"type": "text", "text": "Cell 1"}]}]},
							{"type": "tableCell", "content": [{"type": "paragraph", "content": [{"type": "text", "text": "Cell 2"}]}]},
							{"type": "tableCell", "content": [{"type": "paragraph", "content": [{"type": "text", "text": "Cell 3"}]}]}
						]
					},
					{
						"type": "tableRow",
						"content": [
							{"type": "tableCell", "content": [{"type": "paragraph", "content": [{"type": "text", "text": "Cell 4"}]}]},
							{"type": "tableCell", "content": [{"type": "paragraph", "content": [{"type": "text", "text": "Cell 5"}]}]},
							{"type": "tableCell", "content": [{"type": "paragraph", "content": [{"type": "text", "text": "Cell 6"}]}]}
						]
					}
				]
			}
		]
	}`

	var doc ADFDocument
	require.NoError(t, json.Unmarshal([]byte(tableJSON), &doc))

	parser := NewADFParser()
	markdown, _, err := parser.ParseToMarkdown(doc)
	require.NoError(t, err)

	// Verify table structure
	lines := strings.Split(markdown, "\n")
	assert.GreaterOrEqual(t, len(lines), 4, "should have at least 4 lines (header, separator, 2 rows)")

	// Check header row
	assert.Contains(t, lines[0], "**Header 1**")
	assert.Contains(t, lines[0], "**Header 2**")
	assert.Contains(t, lines[0], "**Header 3**")

	// Check separator row
	assert.Contains(t, lines[1], "---")

	// Check data rows
	assert.Contains(t, lines[2], "Cell 1")
	assert.Contains(t, lines[2], "Cell 2")
	assert.Contains(t, lines[2], "Cell 3")
	assert.Contains(t, lines[3], "Cell 4")
	assert.Contains(t, lines[3], "Cell 5")
	assert.Contains(t, lines[3], "Cell 6")

	t.Logf("Parsed table markdown:\n%s", markdown)
}

func TestParseEmoji(t *testing.T) {
	emojiJSON := `{
		"type": "doc",
		"version": 1,
		"content": [
			{
				"type": "paragraph",
				"content": [
					{"type": "emoji", "attrs": {"text": "⭐", "shortName": ":star:"}},
					{"type": "text", "text": " MVP"}
				]
			}
		]
	}`

	var doc ADFDocument
	require.NoError(t, json.Unmarshal([]byte(emojiJSON), &doc))

	parser := NewADFParser()
	markdown, _, err := parser.ParseToMarkdown(doc)
	require.NoError(t, err)

	assert.Contains(t, markdown, "⭐")
	assert.Contains(t, markdown, "MVP")

	t.Logf("Parsed emoji markdown: %s", markdown)
}

func TestParseMediaSingle(t *testing.T) {
	mediaJSON := `{
		"type": "doc",
		"version": 1,
		"content": [
			{
				"type": "mediaSingle",
				"attrs": {"layout": "center"},
				"content": [
					{
						"type": "media",
						"attrs": {
							"type": "file",
							"id": "abc-123-def",
							"alt": "screenshot.png"
						}
					}
				]
			}
		]
	}`

	var doc ADFDocument
	require.NoError(t, json.Unmarshal([]byte(mediaJSON), &doc))

	parser := NewADFParser()
	markdown, mediaIDs, err := parser.ParseToMarkdown(doc)
	require.NoError(t, err)

	assert.Contains(t, markdown, "[attachment")
	assert.Contains(t, mediaIDs, "abc-123-def")

	t.Logf("Parsed media markdown: %s, mediaIDs: %v", markdown, mediaIDs)
}

func TestParseComplexTableWithEmoji(t *testing.T) {
	// Table with emoji like in the JIRA epic
	complexJSON := `{
		"type": "doc",
		"version": 1,
		"content": [
			{
				"type": "table",
				"content": [
					{
						"type": "tableRow",
						"content": [
							{
								"type": "tableHeader",
								"content": [{"type": "paragraph", "content": [{"type": "text", "text": "Communication Type", "marks": [{"type": "strong"}]}]}]
							},
							{
								"type": "tableHeader",
								"content": [{"type": "paragraph", "content": [{"type": "text", "text": "Action", "marks": [{"type": "strong"}]}]}]
							}
						]
					},
					{
						"type": "tableRow",
						"content": [
							{"type": "tableCell", "content": [{"type": "paragraph", "content": [{"type": "text", "text": "Post Text"}]}]},
							{"type": "tableCell", "content": [{"type": "paragraph", "content": [{"type": "emoji", "attrs": {"text": "⭐", "shortName": ":star:"}}, {"type": "text", "text": " MVP"}]}]}
						]
					},
					{
						"type": "tableRow",
						"content": [
							{"type": "tableCell", "content": [{"type": "paragraph", "content": [{"type": "text", "text": "Post Edit"}]}]},
							{"type": "tableCell", "content": [{"type": "paragraph", "content": [{"type": "emoji", "attrs": {"text": "⭐", "shortName": ":star:"}}, {"type": "text", "text": " MVP"}]}]}
						]
					}
				]
			}
		]
	}`

	var doc ADFDocument
	require.NoError(t, json.Unmarshal([]byte(complexJSON), &doc))

	parser := NewADFParser()
	markdown, _, err := parser.ParseToMarkdown(doc)
	require.NoError(t, err)

	// Check that table columns are properly separated
	lines := strings.Split(markdown, "\n")

	// Header should have both column headers
	assert.Contains(t, lines[0], "Communication Type")
	assert.Contains(t, lines[0], "Action")

	// Data rows should have emoji and text properly separated in columns
	assert.Contains(t, lines[2], "Post Text")
	assert.Contains(t, lines[2], "⭐ MVP")
	assert.Contains(t, lines[3], "Post Edit")
	assert.Contains(t, lines[3], "⭐ MVP")

	t.Logf("Parsed complex table markdown:\n%s", markdown)
}

func TestParseTableWithRowspan(t *testing.T) {
	// Table with rowspan like in the JIRA epic
	rowspanJSON := `{
		"type": "doc",
		"version": 1,
		"content": [
			{
				"type": "table",
				"content": [
					{
						"type": "tableRow",
						"content": [
							{"type": "tableHeader", "content": [{"type": "paragraph", "content": [{"type": "text", "text": "Category", "marks": [{"type": "strong"}]}]}]},
							{"type": "tableHeader", "content": [{"type": "paragraph", "content": [{"type": "text", "text": "Action", "marks": [{"type": "strong"}]}]}]},
							{"type": "tableHeader", "content": [{"type": "paragraph", "content": [{"type": "text", "text": "Status", "marks": [{"type": "strong"}]}]}]}
						]
					},
					{
						"type": "tableRow",
						"content": [
							{"type": "tableHeader", "attrs": {"rowspan": 3}, "content": [{"type": "paragraph", "content": [{"type": "text", "text": "Post", "marks": [{"type": "strong"}]}]}]},
							{"type": "tableCell", "content": [{"type": "paragraph", "content": [{"type": "text", "text": "Text"}]}]},
							{"type": "tableCell", "content": [{"type": "paragraph", "content": [{"type": "text", "text": "MVP"}]}]}
						]
					},
					{
						"type": "tableRow",
						"content": [
							{"type": "tableCell", "content": [{"type": "paragraph", "content": [{"type": "text", "text": "Edit"}]}]},
							{"type": "tableCell", "content": [{"type": "paragraph", "content": [{"type": "text", "text": "MVP"}]}]}
						]
					},
					{
						"type": "tableRow",
						"content": [
							{"type": "tableCell", "content": [{"type": "paragraph", "content": [{"type": "text", "text": "Delete"}]}]},
							{"type": "tableCell", "content": [{"type": "paragraph", "content": [{"type": "text", "text": "MVP"}]}]}
						]
					},
					{
						"type": "tableRow",
						"content": [
							{"type": "tableHeader", "attrs": {"rowspan": 2}, "content": [{"type": "paragraph", "content": [{"type": "text", "text": "Comment", "marks": [{"type": "strong"}]}]}]},
							{"type": "tableCell", "content": [{"type": "paragraph", "content": [{"type": "text", "text": "Add"}]}]},
							{"type": "tableCell", "content": [{"type": "paragraph", "content": [{"type": "text", "text": "MVP"}]}]}
						]
					},
					{
						"type": "tableRow",
						"content": [
							{"type": "tableCell", "content": [{"type": "paragraph", "content": [{"type": "text", "text": "Remove"}]}]},
							{"type": "tableCell", "content": [{"type": "paragraph", "content": [{"type": "text", "text": "Nice to have"}]}]}
						]
					}
				]
			}
		]
	}`

	var doc ADFDocument
	require.NoError(t, json.Unmarshal([]byte(rowspanJSON), &doc))

	parser := NewADFParser()
	markdown, _, err := parser.ParseToMarkdown(doc)
	require.NoError(t, err)

	lines := strings.Split(markdown, "\n")

	// Should have 7 lines: header, separator, 5 data rows
	assert.GreaterOrEqual(t, len(lines), 7, "should have header + separator + 5 data rows")

	// Check header
	assert.Contains(t, lines[0], "Category")
	assert.Contains(t, lines[0], "Action")
	assert.Contains(t, lines[0], "Status")

	// Check that each data row has 3 columns (even when rowspan is active)
	for i := 2; i < len(lines); i++ {
		// Count pipe characters - should have 4 pipes for 3 columns (| col | col | col |)
		pipeCount := strings.Count(lines[i], "|")
		assert.Equal(t, 4, pipeCount, "row %d should have 4 pipes for 3 columns: %s", i, lines[i])
	}

	// Row with "Post" should have Post, Text, MVP
	assert.Contains(t, lines[2], "**Post**")
	assert.Contains(t, lines[2], "Text")
	assert.Contains(t, lines[2], "MVP")

	// Row after should have empty first cell (rowspan), Edit, MVP
	assert.Contains(t, lines[3], "Edit")
	assert.Contains(t, lines[3], "MVP")

	// Row with "Comment" should have Comment, Add, MVP
	assert.Contains(t, lines[5], "**Comment**")
	assert.Contains(t, lines[5], "Add")

	t.Logf("Parsed rowspan table markdown:\n%s", markdown)
}

package jira

import (
	"fmt"
	"strings"
)

// ADFParser converts Atlassian Document Format to Markdown.
type ADFParser struct{}

// NewADFParser creates a new ADF parser.
func NewADFParser() *ADFParser {
	return &ADFParser{}
}

// ParseToMarkdown converts an ADF document to markdown
// Returns: (markdown text, media IDs found, error).
func (p *ADFParser) ParseToMarkdown(doc ADFDocument) (string, []string, error) {
	var mediaIDs []string
	var builder strings.Builder

	for i, node := range doc.Content {
		markdown, nodeMediaIDs := p.parseNode(node, 0)
		mediaIDs = append(mediaIDs, nodeMediaIDs...)
		builder.WriteString(markdown)

		// Add spacing between blocks (except after the last one)
		if i < len(doc.Content)-1 && p.isBlockNode(node.Type) {
			builder.WriteString("\n\n")
		}
	}

	return strings.TrimSpace(builder.String()), mediaIDs, nil
}

// parseNode parses a single ADF node and returns markdown and any media IDs.
func (p *ADFParser) parseNode(node ADFNode, depth int) (string, []string) {
	var mediaIDs []string

	// Handle simple node types first
	if result, ok := p.parseSimpleNode(node); ok {
		return result, mediaIDs
	}

	// Handle nodes that collect media IDs
	if result, ids, ok := p.parseMediaNode(node, depth); ok {
		return result, ids
	}

	// Handle container nodes (doc and unknown types)
	return p.parseContainerNode(node, depth)
}

// parseSimpleNode handles node types that don't collect media IDs.
func (p *ADFParser) parseSimpleNode(node ADFNode) (string, bool) {
	switch node.Type {
	case "text":
		return p.parseText(node), true
	case "mention":
		return p.parseMention(node), true
	case "inlineCard":
		return p.parseInlineCard(node), true
	case "codeBlock":
		return p.parseCodeBlock(node), true
	case "hardBreak":
		return "\n", true
	case "rule":
		return "---", true
	case "emoji":
		return p.parseEmoji(node), true
	}
	return "", false
}

// parseMediaNode handles node types that collect media IDs.
func (p *ADFParser) parseMediaNode(node ADFNode, depth int) (string, []string, bool) {
	var mediaIDs []string

	switch node.Type {
	case "paragraph":
		return p.parseParagraph(node, &mediaIDs), mediaIDs, true
	case "mediaGroup":
		return p.parseMediaGroup(node, &mediaIDs), mediaIDs, true
	case "media":
		return p.parseMedia(node, &mediaIDs), mediaIDs, true
	case "orderedList":
		return p.parseOrderedList(node, depth, &mediaIDs), mediaIDs, true
	case "bulletList":
		return p.parseBulletList(node, depth, &mediaIDs), mediaIDs, true
	case "listItem":
		return p.parseListItem(node, depth, &mediaIDs), mediaIDs, true
	case "heading":
		return p.parseHeading(node, &mediaIDs), mediaIDs, true
	case "blockquote":
		return p.parseBlockquote(node, &mediaIDs), mediaIDs, true
	case "panel":
		return p.parsePanel(node, &mediaIDs), mediaIDs, true
	case "table":
		return p.parseTable(node, &mediaIDs), mediaIDs, true
	case "mediaSingle":
		return p.parseMediaSingle(node, &mediaIDs), mediaIDs, true
	}
	return "", nil, false
}

// parseContainerNode handles doc nodes and unknown types by parsing children.
func (p *ADFParser) parseContainerNode(node ADFNode, depth int) (string, []string) {
	if len(node.Content) == 0 {
		return "", nil
	}

	var mediaIDs []string
	var builder strings.Builder
	for _, child := range node.Content {
		markdown, nodeMediaIDs := p.parseNode(child, depth)
		mediaIDs = append(mediaIDs, nodeMediaIDs...)
		builder.WriteString(markdown)
	}
	return builder.String(), mediaIDs
}

// parseParagraph converts a paragraph node to markdown.
func (p *ADFParser) parseParagraph(node ADFNode, mediaIDs *[]string) string {
	if len(node.Content) == 0 {
		return ""
	}

	var builder strings.Builder
	for _, child := range node.Content {
		markdown, nodeMediaIDs := p.parseNode(child, 0)
		*mediaIDs = append(*mediaIDs, nodeMediaIDs...)
		builder.WriteString(markdown)
	}
	return builder.String()
}

// parseText converts a text node to markdown with formatting.
func (p *ADFParser) parseText(node ADFNode) string {
	text := node.Text
	if text == "" {
		return ""
	}

	// Apply text marks (bold, italic, code, etc.)
	for _, mark := range node.Marks {
		text = p.applyMark(text, mark)
	}
	return text
}

// applyMark applies a single formatting mark to the text.
func (p *ADFParser) applyMark(text string, mark map[string]any) string {
	markType, ok := mark["type"].(string)
	if !ok {
		return text
	}

	switch markType {
	case "strong":
		return "**" + text + "**"
	case "em":
		return "*" + text + "*"
	case "code":
		return "`" + text + "`"
	case "strike":
		return "~~" + text + "~~"
	case "underline":
		return "_" + text + "_"
	case "link":
		return p.applyLinkMark(text, mark)
	case "subsup":
		return p.applySubSupMark(text, mark)
	default:
		return text
	}
}

// applyLinkMark applies a link mark to the text.
func (p *ADFParser) applyLinkMark(text string, mark map[string]any) string {
	if href, ok := mark["attrs"].(map[string]any)["href"].(string); ok {
		return "[" + text + "](" + href + ")"
	}
	return text
}

// applySubSupMark applies a superscript or subscript mark to the text.
func (p *ADFParser) applySubSupMark(text string, mark map[string]any) string {
	attrs, ok := mark["attrs"].(map[string]any)
	if !ok {
		return text
	}
	supType, ok := attrs["type"].(string)
	if !ok {
		return text
	}
	switch supType {
	case "sup":
		return "^" + text + "^"
	case "sub":
		return "~" + text + "~"
	default:
		return text
	}
}

// parseMention converts a mention node to markdown.
func (p *ADFParser) parseMention(node ADFNode) string {
	if node.Attrs != nil {
		if text, ok := node.Attrs["text"].(string); ok {
			return text
		}
		if id, ok := node.Attrs["id"].(string); ok {
			return "@" + id
		}
	}
	return "@unknown"
}

// parseMediaGroup extracts media IDs from a media group.
func (p *ADFParser) parseMediaGroup(node ADFNode, mediaIDs *[]string) string {
	for _, child := range node.Content {
		if child.Type == "media" {
			p.parseMedia(child, mediaIDs)
		}
	}
	return "" // Media groups don't render directly in markdown
}

// parseMedia extracts the media ID and returns a placeholder.
func (p *ADFParser) parseMedia(node ADFNode, mediaIDs *[]string) string {
	if node.Attrs != nil {
		if id, ok := node.Attrs["id"].(string); ok {
			*mediaIDs = append(*mediaIDs, id)
			return fmt.Sprintf("[attachment: %s]", id)
		}
	}
	return "[attachment]"
}

// parseInlineCard converts an inline card to a markdown link.
func (p *ADFParser) parseInlineCard(node ADFNode) string {
	if node.Attrs != nil {
		if url, ok := node.Attrs["url"].(string); ok {
			// Try to extract a title or use the URL as the text
			if title, ok := node.Attrs["title"].(string); ok && title != "" {
				return "[" + title + "](" + url + ")"
			}
			return "<" + url + ">"
		}
	}
	return "[link]"
}

// parseOrderedList converts an ordered list to markdown.
func (p *ADFParser) parseOrderedList(node ADFNode, depth int, mediaIDs *[]string) string {
	var builder strings.Builder
	start := 1
	if node.Attrs != nil {
		if order, ok := node.Attrs["order"].(float64); ok {
			start = int(order)
		}
	}

	for i, child := range node.Content {
		if child.Type == "listItem" {
			indent := strings.Repeat("  ", depth)
			markdown := p.parseListItem(child, depth, mediaIDs)
			fmt.Fprintf(&builder, "%s%d. %s", indent, start+i, markdown)
			if i < len(node.Content)-1 {
				builder.WriteString("\n")
			}
		}
	}

	return builder.String()
}

// parseBulletList converts a bullet list to markdown.
func (p *ADFParser) parseBulletList(node ADFNode, depth int, mediaIDs *[]string) string {
	var builder strings.Builder

	for i, child := range node.Content {
		if child.Type == "listItem" {
			indent := strings.Repeat("  ", depth)
			markdown := p.parseListItem(child, depth, mediaIDs)
			fmt.Fprintf(&builder, "%s- %s", indent, markdown)
			if i < len(node.Content)-1 {
				builder.WriteString("\n")
			}
		}
	}

	return builder.String()
}

// parseListItem converts a list item to markdown.
func (p *ADFParser) parseListItem(node ADFNode, depth int, mediaIDs *[]string) string {
	var builder strings.Builder

	for i, child := range node.Content {
		switch child.Type {
		case "paragraph":
			// For paragraphs in list items, don't add extra newlines
			markdown := p.parseParagraph(child, mediaIDs)
			builder.WriteString(markdown)
		case "orderedList", "bulletList":
			// Nested lists
			builder.WriteString("\n")
			markdown, nodeMediaIDs := p.parseNode(child, depth+1)
			*mediaIDs = append(*mediaIDs, nodeMediaIDs...)
			builder.WriteString(markdown)
		default:
			markdown, nodeMediaIDs := p.parseNode(child, depth)
			*mediaIDs = append(*mediaIDs, nodeMediaIDs...)
			builder.WriteString(markdown)
		}

		if i < len(node.Content)-1 && p.isBlockNode(child.Type) {
			builder.WriteString("\n")
		}
	}

	return builder.String()
}

// parseHeading converts a heading node to markdown.
func (p *ADFParser) parseHeading(node ADFNode, mediaIDs *[]string) string {
	level := 1
	if node.Attrs != nil {
		if l, ok := node.Attrs["level"].(float64); ok {
			level = int(l)
		}
	}

	var builder strings.Builder
	for _, child := range node.Content {
		markdown, nodeMediaIDs := p.parseNode(child, 0)
		*mediaIDs = append(*mediaIDs, nodeMediaIDs...)
		builder.WriteString(markdown)
	}

	return strings.Repeat("#", level) + " " + builder.String()
}

// parseCodeBlock converts a code block to markdown.
func (p *ADFParser) parseCodeBlock(node ADFNode) string {
	language := ""
	if node.Attrs != nil {
		if lang, ok := node.Attrs["language"].(string); ok {
			language = lang
		}
	}

	var code strings.Builder
	for _, child := range node.Content {
		if child.Type == "text" {
			code.WriteString(child.Text)
		}
	}

	return fmt.Sprintf("```%s\n%s\n```", language, code.String())
}

// parseBlockquote converts a blockquote to markdown.
func (p *ADFParser) parseBlockquote(node ADFNode, mediaIDs *[]string) string {
	var builder strings.Builder

	for i, child := range node.Content {
		markdown, nodeMediaIDs := p.parseNode(child, 0)
		*mediaIDs = append(*mediaIDs, nodeMediaIDs...)
		// Prefix each line with >
		lines := strings.Split(markdown, "\n")
		for j, line := range lines {
			builder.WriteString("> " + line)
			if j < len(lines)-1 {
				builder.WriteString("\n")
			}
		}
		if i < len(node.Content)-1 {
			builder.WriteString("\n>\n")
		}
	}

	return builder.String()
}

// parsePanel converts a panel to markdown (as a blockquote with emoji).
func (p *ADFParser) parsePanel(node ADFNode, mediaIDs *[]string) string {
	panelType := "info"
	if node.Attrs != nil {
		if pt, ok := node.Attrs["panelType"].(string); ok {
			panelType = pt
		}
	}

	var emoji string
	switch panelType {
	case "info":
		emoji = "ℹ️"
	case "note":
		emoji = "📝"
	case "warning":
		emoji = "⚠️"
	case "error":
		emoji = "❌"
	case "success":
		emoji = "✅"
	default:
		emoji = "📌"
	}

	var builder strings.Builder
	builder.WriteString("> " + emoji + " **" + strings.ToUpper(panelType) + "**\n>\n")

	for i, child := range node.Content {
		markdown, nodeMediaIDs := p.parseNode(child, 0)
		*mediaIDs = append(*mediaIDs, nodeMediaIDs...)
		lines := strings.Split(markdown, "\n")
		for j, line := range lines {
			builder.WriteString("> " + line)
			if j < len(lines)-1 {
				builder.WriteString("\n")
			}
		}
		if i < len(node.Content)-1 {
			builder.WriteString("\n>\n")
		}
	}

	return builder.String()
}

// rowspanTracker tracks cells that span multiple rows.
type rowspanTracker struct {
	// For each column index, tracks remaining rows to span and the content
	spans map[int]struct {
		content   string
		remaining int
	}
}

func newRowspanTracker() *rowspanTracker {
	return &rowspanTracker{
		spans: make(map[int]struct {
			content   string
			remaining int
		}),
	}
}

// decrementSpans reduces the rowspan count for all tracked columns
// and removes spans that have been fully processed.
func (r *rowspanTracker) decrementSpans() {
	for col, span := range r.spans {
		// Delete spans that were exhausted in the previous row
		if span.remaining <= 0 {
			delete(r.spans, col)
			continue
		}
		// Decrement for the next row
		span.remaining--
		r.spans[col] = span
	}
}

// parseTable converts a table node to markdown.
func (p *ADFParser) parseTable(node ADFNode, mediaIDs *[]string) string {
	var builder strings.Builder
	var numCols int
	isFirstRow := true
	tracker := newRowspanTracker()

	for _, row := range node.Content {
		if row.Type != "tableRow" {
			continue
		}

		// Build cells for this row, accounting for rowspans
		var cells []string
		cellIndex := 0 // Index into the actual cells in this row

		// Determine expected number of columns from first row
		if isFirstRow {
			numCols = len(row.Content)
		}

		// Process each column position
		for colPos := 0; colPos < numCols || cellIndex < len(row.Content); colPos++ {
			// Check if this column is spanned from a previous row
			if span, ok := tracker.spans[colPos]; ok && span.remaining > 0 {
				// Insert empty cell for spanned column (the content is visually merged)
				cells = append(cells, "")
				continue
			}

			// Get the actual cell at this position
			if cellIndex >= len(row.Content) {
				cells = append(cells, "")
				continue
			}

			cell := row.Content[cellIndex]
			cellIndex++

			// Parse the cell content
			cellContent := p.parseTableCell(cell, mediaIDs)
			cells = append(cells, cellContent)

			// Check for rowspan attribute
			if cell.Attrs != nil {
				if rowspan, ok := cell.Attrs["rowspan"].(float64); ok && rowspan > 1 {
					tracker.spans[colPos] = struct {
						content   string
						remaining int
					}{
						// Store full rowspan count; decrement happens at end of row
						// so by the time we check the next row, it will be rowspan-1
						remaining: int(rowspan),
						content:   cellContent,
					}
				}
			}
		}

		// Update numCols if this row had more cells (can happen with complex tables)
		if len(cells) > numCols {
			numCols = len(cells)
		}

		// Write row
		builder.WriteString("| ")
		builder.WriteString(strings.Join(cells, " | "))
		builder.WriteString(" |\n")

		// Add separator after header row
		if isFirstRow {
			builder.WriteString("|")
			for range numCols {
				builder.WriteString(" --- |")
			}
			builder.WriteString("\n")
			isFirstRow = false
		}

		// Decrement rowspan counters for next row
		tracker.decrementSpans()
	}

	return strings.TrimSuffix(builder.String(), "\n")
}

// parseTableCell extracts content from a table cell (header or regular cell).
func (p *ADFParser) parseTableCell(node ADFNode, mediaIDs *[]string) string {
	var builder strings.Builder

	for _, child := range node.Content {
		markdown, nodeMediaIDs := p.parseNode(child, 0)
		*mediaIDs = append(*mediaIDs, nodeMediaIDs...)
		// Remove newlines within table cells as they break markdown table format
		markdown = strings.ReplaceAll(markdown, "\n", " ")
		builder.WriteString(markdown)
	}

	return strings.TrimSpace(builder.String())
}

// parseMediaSingle handles a single media item (image, file, etc.)
func (p *ADFParser) parseMediaSingle(node ADFNode, mediaIDs *[]string) string {
	for _, child := range node.Content {
		if child.Type == "media" {
			return p.parseMedia(child, mediaIDs)
		}
	}
	return "[attachment]"
}

// parseEmoji converts an emoji node to its text representation.
func (p *ADFParser) parseEmoji(node ADFNode) string {
	if node.Attrs != nil {
		// Try to get the emoji text (the actual emoji character)
		if text, ok := node.Attrs["text"].(string); ok && text != "" {
			return text
		}
		// Fall back to shortName if available
		if shortName, ok := node.Attrs["shortName"].(string); ok {
			return shortName
		}
	}
	return ""
}

// isBlockNode checks if a node type is a block-level node.
func (p *ADFParser) isBlockNode(nodeType string) bool {
	blockTypes := map[string]bool{
		"paragraph":   true,
		"heading":     true,
		"codeBlock":   true,
		"blockquote":  true,
		"orderedList": true,
		"bulletList":  true,
		"panel":       true,
		"rule":        true,
		"mediaGroup":  true,
		"mediaSingle": true,
		"table":       true,
	}
	return blockTypes[nodeType]
}

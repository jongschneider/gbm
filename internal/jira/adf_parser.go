package jira

import (
	"fmt"
	"strings"
)

// ADFParser converts Atlassian Document Format to Markdown
type ADFParser struct{}

// NewADFParser creates a new ADF parser
func NewADFParser() *ADFParser {
	return &ADFParser{}
}

// ParseToMarkdown converts an ADF document to markdown
// Returns: (markdown text, media IDs found, error)
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

// parseNode parses a single ADF node and returns markdown and any media IDs
func (p *ADFParser) parseNode(node ADFNode, depth int) (string, []string) {
	var mediaIDs []string

	switch node.Type {
	case "paragraph":
		return p.parseParagraph(node, &mediaIDs), mediaIDs
	case "text":
		return p.parseText(node), mediaIDs
	case "mention":
		return p.parseMention(node), mediaIDs
	case "mediaGroup":
		return p.parseMediaGroup(node, &mediaIDs), mediaIDs
	case "media":
		return p.parseMedia(node, &mediaIDs), mediaIDs
	case "inlineCard":
		return p.parseInlineCard(node), mediaIDs
	case "orderedList":
		return p.parseOrderedList(node, depth, &mediaIDs), mediaIDs
	case "bulletList":
		return p.parseBulletList(node, depth, &mediaIDs), mediaIDs
	case "listItem":
		return p.parseListItem(node, depth, &mediaIDs), mediaIDs
	case "heading":
		return p.parseHeading(node, &mediaIDs), mediaIDs
	case "codeBlock":
		return p.parseCodeBlock(node), mediaIDs
	case "hardBreak":
		return "\n", mediaIDs
	case "rule":
		return "---", mediaIDs
	case "blockquote":
		return p.parseBlockquote(node, &mediaIDs), mediaIDs
	case "panel":
		return p.parsePanel(node, &mediaIDs), mediaIDs
	case "doc":
		// Handle nested doc nodes
		var builder strings.Builder
		for _, child := range node.Content {
			markdown, nodeMediaIDs := p.parseNode(child, depth)
			mediaIDs = append(mediaIDs, nodeMediaIDs...)
			builder.WriteString(markdown)
		}
		return builder.String(), mediaIDs
	default:
		// For unknown node types, try to parse content if it exists
		if len(node.Content) > 0 {
			var builder strings.Builder
			for _, child := range node.Content {
				markdown, nodeMediaIDs := p.parseNode(child, depth)
				mediaIDs = append(mediaIDs, nodeMediaIDs...)
				builder.WriteString(markdown)
			}
			return builder.String(), mediaIDs
		}
		return "", mediaIDs
	}
}

// parseParagraph converts a paragraph node to markdown
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

// parseText converts a text node to markdown with formatting
func (p *ADFParser) parseText(node ADFNode) string {
	text := node.Text
	if text == "" {
		return ""
	}

	// Apply text marks (bold, italic, code, etc.)
	for _, mark := range node.Marks {
		if markType, ok := mark["type"].(string); ok {
			switch markType {
			case "strong":
				text = "**" + text + "**"
			case "em":
				text = "*" + text + "*"
			case "code":
				text = "`" + text + "`"
			case "strike":
				text = "~~" + text + "~~"
			case "underline":
				text = "_" + text + "_"
			case "link":
				if href, ok := mark["attrs"].(map[string]any)["href"].(string); ok {
					text = "[" + text + "](" + href + ")"
				}
			case "subsup":
				if attrs, ok := mark["attrs"].(map[string]any); ok {
					if supType, ok := attrs["type"].(string); ok {
						switch supType {
						case "sup":
							text = "^" + text + "^"
						case "sub":
							text = "~" + text + "~"
						}
					}
				}
			}
		}
	}
	return text
}

// parseMention converts a mention node to markdown
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

// parseMediaGroup extracts media IDs from a media group
func (p *ADFParser) parseMediaGroup(node ADFNode, mediaIDs *[]string) string {
	for _, child := range node.Content {
		if child.Type == "media" {
			p.parseMedia(child, mediaIDs)
		}
	}
	return "" // Media groups don't render directly in markdown
}

// parseMedia extracts the media ID and returns a placeholder
func (p *ADFParser) parseMedia(node ADFNode, mediaIDs *[]string) string {
	if node.Attrs != nil {
		if id, ok := node.Attrs["id"].(string); ok {
			*mediaIDs = append(*mediaIDs, id)
			return fmt.Sprintf("[attachment: %s]", id)
		}
	}
	return "[attachment]"
}

// parseInlineCard converts an inline card to a markdown link
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

// parseOrderedList converts an ordered list to markdown
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
			builder.WriteString(fmt.Sprintf("%s%d. %s", indent, start+i, markdown))
			if i < len(node.Content)-1 {
				builder.WriteString("\n")
			}
		}
	}

	return builder.String()
}

// parseBulletList converts a bullet list to markdown
func (p *ADFParser) parseBulletList(node ADFNode, depth int, mediaIDs *[]string) string {
	var builder strings.Builder

	for i, child := range node.Content {
		if child.Type == "listItem" {
			indent := strings.Repeat("  ", depth)
			markdown := p.parseListItem(child, depth, mediaIDs)
			builder.WriteString(fmt.Sprintf("%s- %s", indent, markdown))
			if i < len(node.Content)-1 {
				builder.WriteString("\n")
			}
		}
	}

	return builder.String()
}

// parseListItem converts a list item to markdown
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

// parseHeading converts a heading node to markdown
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

// parseCodeBlock converts a code block to markdown
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

// parseBlockquote converts a blockquote to markdown
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

// parsePanel converts a panel to markdown (as a blockquote with emoji)
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

// isBlockNode checks if a node type is a block-level node
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
	}
	return blockTypes[nodeType]
}

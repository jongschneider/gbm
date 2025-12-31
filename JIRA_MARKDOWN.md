# JIRA Markdown Enhancement

This document describes the enhanced JIRA markdown generation features that capture attachments, comments, and generate comprehensive documentation from JIRA tickets.

## Features

### 1. **Full Comment Thread Support**
- Parses all comments (not just the latest one)
- Preserves Atlassian Document Format (ADF) formatting
- Supports mentions, links, lists, code blocks, and more
- Extracts media references from comments

### 2. **Attachment Downloads**
- Downloads ticket-level attachments
- Configurable size limits
- Automatic retry with exponential backoff
- Handles filename collisions
- Skips oversized files with warnings

### 3. **Rich Markdown Generation**
- Comprehensive metadata table
- Full description with formatting
- Epic information
- Labels list
- All attachments with download status
- Complete comment threads with author info
- Embedded images and linked files

## Architecture

### Core Components

#### 1. **ADF Parser** (`internal/jira/adf_parser.go`)
Converts Atlassian Document Format to Markdown:
- **Text formatting**: bold, italic, code, strikethrough, links
- **Block elements**: paragraphs, headings, lists, code blocks, blockquotes
- **Special nodes**: mentions, inline cards, media groups
- **Media extraction**: Identifies media IDs for download

#### 2. **Attachment Service** (`internal/jira/attachments.go`)
Handles file downloads:
- HTTP client with timeout and retries
- Size limit enforcement
- Filename sanitization
- Collision resolution
- Progress tracking

#### 3. **Markdown Generator** (`internal/jira/markdown.go`)
Creates comprehensive markdown documents:
- Metadata formatting
- Attachment links (embedded images or file links)
- Comment threading with timestamps
- Relative path support

#### 4. **Integration Helper** (`internal/jira/integration.go`)
High-level API for worktree integration:
- Single function call for complete workflow
- Configurable options
- Result reporting

### Data Model

#### Enhanced Types (`internal/jira/types.go`)

```go
// User represents a JIRA user
type User struct {
    DisplayName  string
    Email        string
    AccountID    string
    AvatarURL    string
}

// Attachment represents a file attachment
type Attachment struct {
    ID       string
    Filename string
    Author   User
    Created  string
    Size     int64
    MimeType string
    Content  string  // Download URL
}

// Comment with full ADF support
type Comment struct {
    ID          string
    Author      User
    Body        ADFDocument  // Full ADF structure
    Content     string       // Deprecated: plain text
    Created     string
    Updated     string
    Timestamp   time.Time    // Deprecated
    Attachments []string     // Media IDs
}

// JiraTicketDetails now includes:
type JiraTicketDetails struct {
    // ... existing fields ...
    Attachments []Attachment
    Comments    []Comment
    Labels      []string
}
```

## Usage

### Basic Integration

From worktree creation code:

```go
import "gbm/internal/jira"

// Create JIRA service
jiraService := jira.NewService(debug, cacheStore)

// Configure markdown generation
opts := jira.DefaultIssueMarkdownOptions(worktreeRoot)
opts.DownloadAttachments = true
opts.IncludeComments = true

// Generate markdown with attachments
result, err := jiraService.GenerateIssueMarkdownFile(
    "PROJ-123",
    opts,
    dryRun,
)
if err != nil {
    return fmt.Errorf("failed to generate markdown: %w", err)
}

// Print results
jira.PrintMarkdownResult(result)
```

### Configuration

Create `.gbm/config.yaml` in your repository:

```yaml
jira:
  attachments:
    enabled: true
    max_size_mb: 50
    directory: ".jira/attachments"
    download_timeout_seconds: 30
    retry_attempts: 3
    retry_backoff_ms: 1000

  markdown:
    include_comments: true
    include_attachments: true
    use_relative_links: true
    filename_pattern: "{key}.md"
```

### Custom Options

```go
opts := jira.IssueMarkdownOptions{
    WorktreeRoot:        "/path/to/worktree",
    DownloadAttachments: true,
    AttachmentConfig: jira.AttachmentConfig{
        MaxSizeMB:      100,  // Custom size limit
        Timeout:        60 * time.Second,
        RetryAttempts:  5,
        RetryBackoffMs: 2000,
    },
    IncludeComments: true,
    Filename:        "TICKET.md",  // Custom filename
}
```

## Generated Markdown Structure

```markdown
# [PROJ-123] Ticket Summary

## Metadata

| Field | Value |
|-------|-------|
| **Key** | PROJ-123 |
| **Status** | In Progress |
| **Priority** | High |
| **Assignee** | John Doe (john@example.com) |
| **Reporter** | Jane Smith (jane@example.com) |
| **Created** | 2025-01-15 14:30:00 |
| **Epic** | EPIC-456 |

## Description

Customer reports that channel name edits are not being recorded.

Testing requirements:
- Test all quick edits
- Verify conversation rename capture

## Labels

- `bug`
- `customer-reported`

## Attachments

- ![screenshot.png](.jira/attachments/PROJ-123/screenshot.png)
  - **screenshot.png** (45.2 KB) - *Uploaded by Jane Smith on 2025-01-15 14:35*
- [requirements.docx](.jira/attachments/PROJ-123/requirements.docx) (168.4 KB) - *Uploaded by John Doe on 2025-01-15 15:00*
- **large-file.zip** (150 MB) - ⚠️ Skipped: exceeds size limit (150 MB > 50 MB)
  - *Original URL*: <https://jira.example.com/attachment/content/12345>

## Comments

### Comment by Jane Smith - 2025-01-15 14:40

Attached screenshot showing the issue in production.

**Media References**:
- Media ID: `79fe121d-35ab-4a2d-8db9-8ef78d40612f`

---

### Comment by @Alex Perales @John Austin - 2025-01-16 09:15

This is a bug on our side. We are not capturing the conversation rename.

---

---

**JIRA Link**: [PROJ-123](https://jira.example.com/browse/PROJ-123)
```

## Directory Structure

When markdown is generated with attachments:

```
worktree-root/
├── PROJ-123.md                        # Generated markdown
└── .jira/
    └── attachments/
        └── PROJ-123/                  # Ticket-specific directory
            ├── screenshot.png
            ├── requirements.docx
            └── diagram.svg
```

## ADF Support

The ADF parser supports these node types:

### Block Nodes
- `paragraph` - Standard paragraphs
- `heading` - Headers (h1-h6)
- `codeBlock` - Fenced code blocks with language
- `blockquote` - Block quotes
- `orderedList` - Numbered lists
- `bulletList` - Bullet lists
- `rule` - Horizontal rules
- `panel` - Info/warning/error panels (rendered as blockquotes with emoji)

### Inline Nodes
- `text` - Plain text with marks (bold, italic, code, etc.)
- `mention` - User mentions (@username)
- `inlineCard` - Links to other issues or URLs
- `hardBreak` - Line breaks

### Media Nodes
- `mediaGroup` - Container for media items
- `media` - Individual media item (extracts ID for download)

### Text Marks
- `strong` - **Bold**
- `em` - *Italic*
- `code` - `Inline code`
- `strike` - ~~Strikethrough~~
- `underline` - _Underline_
- `link` - [Hyperlinks](url)

## Error Handling

The system handles various failure scenarios gracefully:

### Network Failures
- Retries with exponential backoff
- Continues processing other attachments
- Logs warnings for failures
- Includes fallback links to original URLs

### Authentication
- Uses JIRA CLI session for authentication
- Falls back to URL links if download fails

### File System Issues
- Creates directories as needed
- Handles filename collisions (appends counter)
- Sanitizes invalid characters
- Respects filesystem path length limits

### Large Files
- Checks size before download
- Skips oversized files
- Reports skip reason in markdown

## Testing

To test the implementation:

```bash
# Build the project
go build ./...

# Run with a test ticket
gbm worktree add PROJ-123

# Verify generated markdown
cat PROJ-123.md

# Check attachments
ls -lh .jira/attachments/PROJ-123/
```

## Future Enhancements

Planned improvements (see `jira-markdown-plan.md` Phase 8):

1. **Selective Downloads**
   - `--skip-attachments` flag
   - `--images-only` flag
   - File type filters

2. **Attachment Caching**
   - Reuse downloads across worktrees
   - Hash-based verification

3. **Sync Command**
   - Update markdown when ticket changes
   - Re-download new attachments

4. **Comment Filtering**
   - Date range filters
   - User filters
   - Internal/public comment separation

## Troubleshooting

### Compilation Errors

If you see compilation errors, ensure all new types are properly imported:

```bash
go build ./...
```

### Missing Attachments

Check the configuration and size limits:

```bash
# Verify attachment directory exists
ls -la .jira/attachments/

# Check for size limit errors in output
gbm worktree add PROJ-123 2>&1 | grep "exceeds size limit"
```

### ADF Parsing Issues

For unsupported ADF nodes, the parser falls back to extracting text content. Check logs for warnings about unknown node types.

## Contributing

When extending the ADF parser:

1. Add new node type handler in `adf_parser.go`
2. Update `isBlockNode()` if it's a block-level node
3. Add test cases in `adf_parser_test.go`
4. Document the node type in this README

## References

- **Atlassian Document Format**: https://developer.atlassian.com/cloud/jira/platform/apis/document/structure/
- **JIRA REST API**: https://developer.atlassian.com/cloud/jira/platform/rest/v3/
- **Implementation Plan**: `jira-markdown-plan.md`

# PRD: Capture Parent/Child Issue Relationships in Worktree Add

## Overview

Extend the JIRA integration to capture parent and child issue relationships using the same patterns established for linked issues and attachments. This ensures complete issue context is captured when creating worktrees from JIRA issues.

## Goals

1. Capture parent issue details when a JIRA issue has a parent
2. Capture child issues (where current issue is the parent)
3. Follow the same depth-limited recursive fetching used for linked issues
4. Deduplicate issues to prevent infinite loops and redundant fetching
5. Display parent/child relationships in markdown files consistently with linked issues

## Non-Goals

- Changing the existing linked issues implementation
- Adding new configuration options (reuse existing `MaxDepth` setting)
- Modifying TUI display beyond what linked issues already show

## Technical Requirements

### 1. JIRA API Response Structure

The `parent` field in JIRA API response (from `jira_issue_raw.json`):

```json
"parent": {
  "id": "51082",
  "key": "EPIC-2704",
  "self": "https://thetalake.atlassian.net/rest/api/3/issue/51082",
  "fields": {
    "summary": "[G-21] Facebook Business Pages - Content Source Integration",
    "status": {
      "name": "ENG In-Progress (no commit)",
      ...
    },
    "priority": {
      "name": "Medium",
      ...
    },
    "issuetype": {
      "name": "Epic",
      ...
    }
  }
}
```

Child issues: Use `subtasks` field (empty array in sample) OR query via JQL `parent = <key>`.

### 2. Data Model Changes

**Update `jiraRawResponse.Fields.Parent` in `internal/jira/types.go`:**

Current (line 176-178):
```go
Parent *struct {
    Key string `json:"key"`
} `json:"parent"`
```

Change to (match `InwardIssue`/`OutwardIssue` structure):
```go
Parent *struct {
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
} `json:"parent"`
```

**Add `Subtasks` field to `jiraRawResponse.Fields`:**
```go
Subtasks []struct {
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
} `json:"subtasks"`
```

**Update `JiraTicketDetails` struct:**

Add new fields:
```go
Parent   *LinkedIssue   // Parent issue (if exists)
Children []LinkedIssue  // Child issues/subtasks
```

### 3. Fetching Logic

**In `internal/jira/service.go` (or related file):**

1. When parsing issue response, extract `parent` field using expanded structure
2. Convert to `LinkedIssue`: `{ID, Key, Summary, Status, Priority, IssueType}`
3. If parent exists and depth < MaxDepth and parent key not in lookup table:
   - Add parent key to lookup table
   - Fetch parent issue recursively (same as linked issues)
   - Parent's linked issues follow same MaxDepth rules

4. Parse `subtasks` array for child issues
5. For each child, if depth < MaxDepth and child key not in lookup table:
   - Add child key to lookup table  
   - Fetch child issue recursively
   - Child's linked issues follow same MaxDepth rules

**Lookup table (deduplication):**
- Use existing mechanism for linked issues (or create shared one)
- Key: issue key (e.g., "INGSVC-6457")
- Prevents fetching same issue multiple times across parent/child/linked traversals

### 4. Markdown Generation

**In `.jira/` markdown files:**

Add "Parent" section (if parent exists):
```markdown
## Parent

- **Parent:** EPIC-2704 - [G-21] Facebook Business Pages - Content Source Integration
```

Add "Children" section (if children exist):
```markdown
## Children

- **is parent of:** INGSVC-6458 - Child task 1
- **is parent of:** INGSVC-6459 - Child task 2
```

**Section ordering in markdown:**
1. Issue details (existing)
2. Parent (new)
3. Children (new)  
4. Linked Issues (existing)
5. Attachments (existing)

### 5. Deduplication Rule

**When issue appears as both parent AND in linked issues:**
- Show in "Parent" section only
- Do NOT show in "Linked Issues" section
- Same rule applies to children that also appear as linked issues

### 6. Depth Behavior

Example with `MaxDepth: 2`:
```
INGSVC-6457 (root issue, depth 0)
├── Parent: EPIC-2704 (depth 1)
│   ├── Parent: EPIC-1000 (depth 2) ✓
│   │   └── Parent: EPIC-500 (depth 3) ✗ exceeds max
│   └── Linked: PROJ-101 (depth 2) ✓
├── Children: INGSVC-6458, INGSVC-6459 (depth 1)
│   └── Their children (depth 2) ✓
└── Linked: PROJ-200 (depth 1)
    └── Linked: PROJ-201 (depth 2) ✓
```

## Implementation Notes

### Files to Modify

1. `internal/jira/types.go` - Expand `Parent` struct, add `Subtasks`, update `JiraTicketDetails`
2. `internal/jira/service.go` - Add parent/child fetching logic in issue parsing
3. `internal/jira/markdown.go` (or equivalent) - Add Parent/Children sections
4. Ensure lookup table is shared across linked/parent/child traversals

### API Considerations

**Fetching parent:**
- Parent details are included inline in issue response at `fields.parent`
- Contains: `id`, `key`, `fields.summary`, `fields.status.name`, `fields.priority.name`, `fields.issuetype.name`
- For recursive fetch of parent's full details, use separate API call

**Fetching children:**
- `subtasks` field contains inline child issue references (same structure as parent)
- Alternative: JQL query `parent = INGSVC-6457` for non-subtask children

## Acceptance Criteria

1. [ ] `jiraRawResponse.Fields.Parent` expanded to include summary, status, priority, issuetype
2. [ ] `jiraRawResponse.Fields.Subtasks` added for child issues  
3. [ ] `JiraTicketDetails.Parent` field added as `*LinkedIssue`
4. [ ] `JiraTicketDetails.Children` field added as `[]LinkedIssue`
5. [ ] Parent issue is fetched recursively when present (respecting MaxDepth)
6. [ ] Child issues are fetched recursively when present (respecting MaxDepth)
7. [ ] Lookup table prevents duplicate fetching across all relationship types
8. [ ] Markdown includes "Parent" section with `- **Parent:** KEY - Summary` format
9. [ ] Markdown includes "Children" section with `- **is parent of:** KEY - Summary` format
10. [ ] Issues appearing as both parent AND linked show only in Parent section
11. [ ] Issues appearing as both child AND linked show only in Children section
12. [ ] No infinite loops when parent/child create cycles
13. [ ] Existing linked issues and attachments behavior unchanged

## Testing

1. Issue with parent only (like INGSVC-6457 → EPIC-2704)
2. Issue with children only (Epic with subtasks)
3. Issue with both parent and children
4. Issue where parent is also in linked issues (deduplication)
5. Deep hierarchy exceeding MaxDepth
6. Circular reference (A parent of B, B linked to A)
7. Issue with no parent/children (no sections added)
8. Issue with `subtasks` array populated
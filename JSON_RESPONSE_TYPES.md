# JSON Response Types - Type-Safe API Responses

This document describes the structured types used for all JSON responses in GBM, replacing ad hoc `map[string]interface{}` usage.

## Overview

All JSON responses now use strongly-typed Go structs that are automatically marshaled to JSON. This provides:
- **Type Safety**: Compile-time verification of response structure
- **Documentation**: Self-documenting JSON schema
- **Consistency**: Uniform response patterns across all commands
- **Testability**: Proper unmarshaling tests for each response type

## Response Types

### Worktree Response Types

#### `WorktreeResponse`
Represents a single worktree in its simplest form.

```go
type WorktreeResponse struct {
    Name   string `json:"name"`
    Path   string `json:"path"`
    Branch string `json:"branch"`
}
```

**JSON Example:**
```json
{
  "name": "feature-x",
  "path": "/repo/worktrees/feature-x",
  "branch": "feature/x"
}
```

#### `WorktreeListItemResponse`
Represents a worktree in a list response, including metadata.

```go
type WorktreeListItemResponse struct {
    Name    string `json:"name"`
    Path    string `json:"path"`
    Branch  string `json:"branch"`
    Current bool   `json:"current"`
    Tracked bool   `json:"tracked"`
}
```

**JSON Example:**
```json
{
  "name": "feature-x",
  "path": "/repo/worktrees/feature-x",
  "branch": "feature/x",
  "current": false,
  "tracked": true
}
```

### Operation Response Types

#### `WorktreeAddResponse`
Response when creating a new worktree.

```go
type WorktreeAddResponse struct {
    Worktree WorktreeResponse `json:"worktree"`
    Created  bool             `json:"created"`
}
```

**JSON Example:**
```json
{
  "worktree": {
    "name": "feature-x",
    "path": "/repo/worktrees/feature-x",
    "branch": "feature/x"
  },
  "created": true
}
```

#### `WorktreeSwitchResponse`
Response when switching to a worktree.

```go
type WorktreeSwitchResponse struct {
    Worktree WorktreeResponse `json:"worktree"`
    Previous string           `json:"previous,omitempty"`
}
```

**JSON Example:**
```json
{
  "worktree": {
    "name": "feature-x",
    "path": "/repo/worktrees/feature-x",
    "branch": "feature/x"
  },
  "previous": "main"
}
```

#### `WorktreeListResponse`
Response for listing all worktrees.

```go
type WorktreeListResponse struct {
    Count     int                        `json:"count"`
    Worktrees []WorktreeListItemResponse `json:"worktrees"`
}
```

**JSON Example:**
```json
{
  "count": 2,
  "worktrees": [
    {
      "name": "main",
      "path": "/repo/worktrees/main",
      "branch": "main",
      "current": true,
      "tracked": false
    },
    {
      "name": "feature-x",
      "path": "/repo/worktrees/feature-x",
      "branch": "feature/x",
      "current": false,
      "tracked": true
    }
  ]
}
```

### Initialization Response Types

#### `InitResponse`
Response from repository initialization.

```go
type InitResponse struct {
    RepositoryPath string `json:"repository_path"`
    BareRepository string `json:"bare_repository"`
    MainWorktree   string `json:"main_worktree"`
    Created        bool   `json:"created"`
}
```

#### `CloneResponse`
Response from cloning a repository.

```go
type CloneResponse struct {
    RepositoryPath string `json:"repository_path"`
    BareRepository string `json:"bare_repository"`
    MainWorktree   string `json:"main_worktree"`
    Remote         string `json:"remote"`
    Created        bool   `json:"created"`
}
```

### General Response Types

#### `OperationResponse`
Generic response for operations.

```go
type OperationResponse struct {
    Operation string `json:"operation"`
    Status    string `json:"status"`
    Message   string `json:"message,omitempty"`
}
```

#### `SyncResponse`
Response from sync operations.

```go
type SyncResponse struct {
    Operation string `json:"operation"`
    Status    string `json:"status"`
    Changed   int    `json:"changed"`
    Message   string `json:"message,omitempty"`
}
```

## Usage in Commands

### Creating JSON Responses

**Before (ad hoc maps):**
```go
wtData := map[string]interface{}{
    "name":   wt.Name,
    "path":   wt.Path,
    "branch": wt.Branch,
}
return OutputJSON(wtData)
```

**After (structured types):**
```go
response := WorktreeAddResponse{
    Worktree: WorktreeResponse{
        Name:   wt.Name,
        Path:   wt.Path,
        Branch: wt.Branch,
    },
    Created: true,
}
return OutputJSONWithMessage(response, "Created worktree...")
```

### List Operations

**Before:**
```go
wtData := make([]map[string]interface{}, len(worktrees))
for i, wt := range worktrees {
    wtData[i] = map[string]interface{}{
        "name": wt.Name,
        // ...
    }
}
return OutputJSONArray(wtData)
```

**After:**
```go
wtList := make([]WorktreeListItemResponse, len(worktrees))
for i, wt := range worktrees {
    wtList[i] = WorktreeListItemResponse{
        Name: wt.Name,
        // ...
    }
}
response := WorktreeListResponse{
    Count:     len(wtList),
    Worktrees: wtList,
}
return OutputJSONArray(response)
```

## JSON Marshaling

All types properly implement `json.Marshaler` through Go's standard JSON tags:

- **Standard fields**: `json:"fieldname"`
- **Omit empty fields**: `json:"fieldname,omitempty"` (e.g., Previous in WorktreeSwitchResponse)
- **JSON field names**: Can differ from Go field names for consistency

## Testing

Each response type has comprehensive unit tests in `json_types_test.go`:

- **Marshaling**: Converts Go struct → JSON
- **Unmarshaling**: Converts JSON → Go struct
- **Field validation**: Verifies all expected fields present
- **Omit empty**: Verifies `omitempty` tags work correctly

Example test:
```go
func TestWorktreeAddResponse(t *testing.T) {
    resp := WorktreeAddResponse{
        Worktree: WorktreeResponse{...},
        Created: true,
    }
    
    // Marshal
    jsonBytes, _ := json.Marshal(resp)
    
    // Verify fields
    assert.Contains(t, string(jsonBytes), "\"created\":true")
    
    // Unmarshal
    var restored WorktreeAddResponse
    json.Unmarshal(jsonBytes, &restored)
    assert.Equal(t, resp, restored)
}
```

## Benefits

1. **Type Safety**: Compile-time checking of response structure
2. **Documentation**: Response schema is self-documenting
3. **Consistency**: All responses follow same pattern
4. **Extensibility**: Easy to add fields to responses
5. **Testability**: Each response type tested independently
6. **IDE Support**: Better autocomplete and refactoring
7. **Validation**: Can add validation logic to types if needed

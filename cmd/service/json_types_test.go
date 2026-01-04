package service

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWorktreeResponse tests WorktreeResponse marshaling.
func TestWorktreeResponse(t *testing.T) {
	wt := WorktreeResponse{
		Name:   "feature-x",
		Path:   "/repo/worktrees/feature-x",
		Branch: "feature/x",
	}

	// Marshal to JSON
	jsonBytes, err := json.Marshal(wt)
	require.NoError(t, err)

	// Verify JSON fields
	jsonStr := string(jsonBytes)
	assert.Contains(t, jsonStr, "\"name\":\"feature-x\"")
	assert.Contains(t, jsonStr, "\"path\":\"/repo/worktrees/feature-x\"")
	assert.Contains(t, jsonStr, "\"branch\":\"feature/x\"")

	// Unmarshal back
	var restored WorktreeResponse
	err = json.Unmarshal(jsonBytes, &restored)
	require.NoError(t, err)
	assert.Equal(t, wt, restored)
}

// TestWorktreeListItemResponse tests WorktreeListItemResponse marshaling.
func TestWorktreeListItemResponse(t *testing.T) {
	item := WorktreeListItemResponse{
		Name:    "main",
		Path:    "/repo/worktrees/main",
		Branch:  "main",
		Current: true,
		Tracked: false,
	}

	jsonBytes, err := json.Marshal(item)
	require.NoError(t, err)

	jsonStr := string(jsonBytes)
	assert.Contains(t, jsonStr, "\"name\":\"main\"")
	assert.Contains(t, jsonStr, "\"current\":true")
	assert.Contains(t, jsonStr, "\"tracked\":false")

	var restored WorktreeListItemResponse
	err = json.Unmarshal(jsonBytes, &restored)
	require.NoError(t, err)
	assert.Equal(t, item, restored)
}

// TestWorktreeAddResponse tests WorktreeAddResponse marshaling.
func TestWorktreeAddResponse(t *testing.T) {
	resp := WorktreeAddResponse{
		Worktree: WorktreeResponse{
			Name:   "test",
			Path:   "/repo/worktrees/test",
			Branch: "test/branch",
		},
		Created: true,
	}

	jsonBytes, err := json.Marshal(resp)
	require.NoError(t, err)

	jsonStr := string(jsonBytes)
	assert.Contains(t, jsonStr, "\"worktree\":")
	assert.Contains(t, jsonStr, "\"created\":true")
	assert.Contains(t, jsonStr, "\"name\":\"test\"")

	var restored WorktreeAddResponse
	err = json.Unmarshal(jsonBytes, &restored)
	require.NoError(t, err)
	assert.Equal(t, resp, restored)
}

// TestWorktreeSwitchResponse tests WorktreeSwitchResponse marshaling.
func TestWorktreeSwitchResponse(t *testing.T) {
	resp := WorktreeSwitchResponse{
		Worktree: WorktreeResponse{
			Name:   "feature-y",
			Path:   "/repo/worktrees/feature-y",
			Branch: "feature/y",
		},
		Previous: "main",
	}

	jsonBytes, err := json.Marshal(resp)
	require.NoError(t, err)

	jsonStr := string(jsonBytes)
	assert.Contains(t, jsonStr, "\"worktree\":")
	assert.Contains(t, jsonStr, "\"previous\":\"main\"")

	var restored WorktreeSwitchResponse
	err = json.Unmarshal(jsonBytes, &restored)
	require.NoError(t, err)
	assert.Equal(t, resp, restored)
}

// TestWorktreeListResponse tests WorktreeListResponse marshaling.
func TestWorktreeListResponse(t *testing.T) {
	resp := WorktreeListResponse{
		Count: 2,
		Worktrees: []WorktreeListItemResponse{
			{
				Name:    "main",
				Path:    "/repo/worktrees/main",
				Branch:  "main",
				Current: true,
				Tracked: false,
			},
			{
				Name:    "feature",
				Path:    "/repo/worktrees/feature",
				Branch:  "feature/x",
				Current: false,
				Tracked: true,
			},
		},
	}

	jsonBytes, err := json.Marshal(resp)
	require.NoError(t, err)

	jsonStr := string(jsonBytes)
	assert.Contains(t, jsonStr, "\"count\":2")
	assert.Contains(t, jsonStr, "\"worktrees\":[")
	assert.Contains(t, jsonStr, "\"name\":\"main\"")
	assert.Contains(t, jsonStr, "\"name\":\"feature\"")

	var restored WorktreeListResponse
	err = json.Unmarshal(jsonBytes, &restored)
	require.NoError(t, err)
	assert.Equal(t, resp, restored)
}

// TestInitResponse tests InitResponse marshaling.
func TestInitResponse(t *testing.T) {
	resp := InitResponse{
		RepositoryPath: "/path/to/repo",
		BareRepository: "/path/to/repo/.git",
		MainWorktree:   "/path/to/repo/worktrees/main",
		Created:        true,
	}

	jsonBytes, err := json.Marshal(resp)
	require.NoError(t, err)

	jsonStr := string(jsonBytes)
	assert.Contains(t, jsonStr, "\"repository_path\"")
	assert.Contains(t, jsonStr, "\"bare_repository\"")
	assert.Contains(t, jsonStr, "\"main_worktree\"")
	assert.Contains(t, jsonStr, "\"created\":true")

	var restored InitResponse
	err = json.Unmarshal(jsonBytes, &restored)
	require.NoError(t, err)
	assert.Equal(t, resp, restored)
}

// TestCloneResponse tests CloneResponse marshaling.
func TestCloneResponse(t *testing.T) {
	resp := CloneResponse{
		RepositoryPath: "/path/to/repo",
		BareRepository: "/path/to/repo/.git",
		MainWorktree:   "/path/to/repo/worktrees/main",
		Remote:         "origin",
		Created:        true,
	}

	jsonBytes, err := json.Marshal(resp)
	require.NoError(t, err)

	jsonStr := string(jsonBytes)
	assert.Contains(t, jsonStr, "\"repository_path\"")
	assert.Contains(t, jsonStr, "\"remote\":\"origin\"")

	var restored CloneResponse
	err = json.Unmarshal(jsonBytes, &restored)
	require.NoError(t, err)
	assert.Equal(t, resp, restored)
}

// TestOperationResponse tests OperationResponse marshaling.
func TestOperationResponse(t *testing.T) {
	resp := OperationResponse{
		Operation: "remove",
		Status:    "success",
		Message:   "Worktree removed successfully",
	}

	jsonBytes, err := json.Marshal(resp)
	require.NoError(t, err)

	jsonStr := string(jsonBytes)
	assert.Contains(t, jsonStr, "\"operation\":\"remove\"")
	assert.Contains(t, jsonStr, "\"status\":\"success\"")

	var restored OperationResponse
	err = json.Unmarshal(jsonBytes, &restored)
	require.NoError(t, err)
	assert.Equal(t, resp, restored)
}

// TestSyncResponse tests SyncResponse marshaling.
func TestSyncResponse(t *testing.T) {
	resp := SyncResponse{
		Operation: "sync",
		Status:    "completed",
		Changed:   3,
		Message:   "3 worktrees synced",
	}

	jsonBytes, err := json.Marshal(resp)
	require.NoError(t, err)

	jsonStr := string(jsonBytes)
	assert.Contains(t, jsonStr, "\"operation\":\"sync\"")
	assert.Contains(t, jsonStr, "\"changed\":3")

	var restored SyncResponse
	err = json.Unmarshal(jsonBytes, &restored)
	require.NoError(t, err)
	assert.Equal(t, resp, restored)
}

// TestOmitEmptyFields tests that omitempty tags work correctly.
func TestOmitEmptyFields(t *testing.T) {
	// Test WorktreeSwitchResponse with empty Previous
	resp := WorktreeSwitchResponse{
		Worktree: WorktreeResponse{
			Name:   "test",
			Path:   "/repo/worktrees/test",
			Branch: "test",
		},
		Previous: "", // Empty, should be omitted
	}

	jsonBytes, err := json.Marshal(resp)
	require.NoError(t, err)

	jsonStr := string(jsonBytes)
	assert.NotContains(t, jsonStr, "\"previous\"")
	assert.Contains(t, jsonStr, "\"worktree\":")
}

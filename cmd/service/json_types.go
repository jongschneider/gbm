package service

// WorktreeResponse represents a single worktree in JSON response format.
type WorktreeResponse struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Branch string `json:"branch"`
}

// WorktreeListItemResponse represents a worktree in a list response with additional metadata.
type WorktreeListItemResponse struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Branch  string `json:"branch"`
	Current bool   `json:"current"`
	Tracked bool   `json:"tracked"`
}

// WorktreeAddResponse represents the response when creating a new worktree.
type WorktreeAddResponse struct {
	Worktree WorktreeResponse `json:"worktree"`
	Created  bool             `json:"created"`
}

// WorktreeSwitchResponse represents the response when switching to a worktree.
type WorktreeSwitchResponse struct {
	Worktree WorktreeResponse `json:"worktree"`
	Previous string           `json:"previous,omitempty"`
}

// WorktreeListResponse represents the response for listing worktrees.
type WorktreeListResponse struct {
	Count     int                        `json:"count"`
	Worktrees []WorktreeListItemResponse `json:"worktrees"`
}

// OperationResponse represents a generic operation result.
type OperationResponse struct {
	Operation string `json:"operation"`
	Status    string `json:"status"`
	Message   string `json:"message,omitempty"`
}

// SyncResponse represents the response from a sync operation.
type SyncResponse struct {
	Operation string `json:"operation"`
	Status    string `json:"status"`
	Changed   int    `json:"changed"`
	Message   string `json:"message,omitempty"`
}

// InitResponse represents the response from repository initialization.
type InitResponse struct {
	RepositoryPath string `json:"repository_path"`
	BareRepository string `json:"bare_repository"`
	MainWorktree   string `json:"main_worktree"`
	Created        bool   `json:"created"`
}

// CloneResponse represents the response from a clone operation.
type CloneResponse struct {
	RepositoryPath string `json:"repository_path"`
	BareRepository string `json:"bare_repository"`
	MainWorktree   string `json:"main_worktree"`
	Remote         string `json:"remote"`
	Created        bool   `json:"created"`
}

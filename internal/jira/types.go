package jira

import "time"

// JiraIssue represents a basic JIRA issue with key information
type JiraIssue struct {
	Type    string
	Key     string
	Summary string
	Status  string
}

// JiraTicketDetails represents detailed JIRA ticket information
type JiraTicketDetails struct {
	Key           string
	Summary       string
	Description   string // Parsed markdown from JIRA's nested content structure
	Status        string
	Assignee      string
	Priority      string
	Reporter      string
	Created       time.Time
	DueDate       *time.Time
	Epic          string
	URL           string
	LatestComment *Comment
	Attachments   []Attachment  // All ticket-level attachments
	Comments      []Comment     // All comments with full ADF structure
	Labels        []string      // Issue labels
	IssueLinks    []IssueLink   // Linked issues (blocked by, blocks, relates to, etc.)
	Parent        *LinkedIssue  // Parent issue (for subtasks)
	Children      []LinkedIssue // Child issues (subtasks)
}

// User represents a JIRA user
type User struct {
	DisplayName string
	Email       string
	AccountID   string
	AvatarURL   string
}

// Comment represents a JIRA comment
type Comment struct {
	ID          string
	Author      User
	Body        ADFDocument // Full ADF structure
	Content     string      // Deprecated: use Body for new code, kept for compatibility
	Created     string
	Updated     string
	Timestamp   time.Time // Deprecated: use Created for new code, kept for compatibility
	Attachments []string  // Media IDs referenced in comment body
}

// Attachment represents a JIRA file attachment
type Attachment struct {
	ID       string
	Filename string
	Author   User
	Created  string
	Size     int64
	MimeType string
	Content  string // Download URL
}

// IssueLinkType represents the type of relationship between linked issues
type IssueLinkType struct {
	ID      string
	Name    string
	Inward  string // Description when viewing from this issue (e.g., "is blocked by")
	Outward string // Description when viewing from linked issue (e.g., "blocks")
}

// LinkedIssue represents a basic linked issue with key information
type LinkedIssue struct {
	ID        string
	Key       string
	Summary   string
	Status    string
	Priority  string
	IssueType string
}

// IssueLink represents a link between two JIRA issues
type IssueLink struct {
	ID           string
	Type         IssueLinkType
	InwardIssue  *LinkedIssue // Issue that this issue links to (inward relationship)
	OutwardIssue *LinkedIssue // Issue that links to this issue (outward relationship)
}

// ADFDocument represents a full Atlassian Document Format document
type ADFDocument struct {
	Type    string    `json:"type"`
	Version int       `json:"version"`
	Content []ADFNode `json:"content"`
}

// ADFNode represents a node in the Atlassian Document Format tree
type ADFNode struct {
	Type    string                   `json:"type"`
	Text    string                   `json:"text,omitempty"`
	Content []ADFNode                `json:"content,omitempty"`
	Attrs   map[string]interface{}   `json:"attrs,omitempty"`
	Marks   []map[string]interface{} `json:"marks,omitempty"` // For bold, italic, code, etc.
}

// JiraFilters defines filters for jira issue list command
type JiraFilters struct {
	Status     []string `yaml:"status,omitempty"`      // -s flags: filter by status
	Priority   string   `yaml:"priority,omitempty"`    // -y flag: filter by priority
	Type       string   `yaml:"type,omitempty"`        // -t flag: filter by type
	Labels     []string `yaml:"labels,omitempty"`      // -l flags: filter by labels
	Component  string   `yaml:"component,omitempty"`   // -C flag: filter by component
	Reporter   string   `yaml:"reporter,omitempty"`    // -r flag: filter by reporter
	Assignee   string   `yaml:"assignee,omitempty"`    // -a flag: assignee (default: "me")
	OrderBy    string   `yaml:"order_by,omitempty"`    // --order-by flag
	Reverse    bool     `yaml:"reverse,omitempty"`     // --reverse flag
	CustomArgs []string `yaml:"custom_args,omitempty"` // Additional custom args
}

// Description represents JIRA's nested content structure for descriptions
type Description struct {
	Type    string        `json:"type"`
	Version int           `json:"version"`
	Content []ContentNode `json:"content"`
}

// ContentNode represents a node in JIRA's content tree
type ContentNode struct {
	Type    string        `json:"type"`
	Text    string        `json:"text,omitempty"`
	Content []ContentNode `json:"content,omitempty"`
	Attrs   *ContentAttrs `json:"attrs,omitempty"`
}

// ContentAttrs represents attributes for content nodes (e.g., language for code blocks)
type ContentAttrs struct {
	Language string `json:"language,omitempty"`
}

// jiraRawResponse represents the raw JSON response from JIRA CLI
type jiraRawResponse struct {
	Key    string `json:"key"`
	Self   string `json:"self"`
	Fields struct {
		Summary   string   `json:"summary"`
		Created   string   `json:"created"`
		DueDate   *string  `json:"duedate"`
		Labels    []string `json:"labels"`
		IssueType struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			Subtask bool   `json:"subtask"`
		} `json:"issueType"`
		Status struct {
			Name string `json:"name"`
		} `json:"status"`
		Priority struct {
			Name string `json:"name"`
		} `json:"priority"`
		Reporter struct {
			DisplayName  string `json:"displayName"`
			EmailAddress string `json:"emailAddress"`
			AccountID    string `json:"accountId"`
			AvatarURLs   struct {
				Px48 string `json:"48x48"`
			} `json:"avatarUrls"`
		} `json:"reporter"`
		Assignee *struct {
			DisplayName  string `json:"displayName"`
			EmailAddress string `json:"emailAddress"`
			AccountID    string `json:"accountId"`
			AvatarURLs   struct {
				Px48 string `json:"48x48"`
			} `json:"avatarUrls"`
		} `json:"assignee"`
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
		Description *Description `json:"description"`
		Attachment  []struct {
			ID       string `json:"id"`
			Filename string `json:"filename"`
			Author   struct {
				DisplayName  string `json:"displayName"`
				EmailAddress string `json:"emailAddress"`
				AccountID    string `json:"accountId"`
				AvatarURLs   struct {
					Px48 string `json:"48x48"`
				} `json:"avatarUrls"`
			} `json:"author"`
			Created  string `json:"created"`
			Size     int64  `json:"size"`
			MimeType string `json:"mimeType"`
			Content  string `json:"content"`
		} `json:"attachment"`
		Comment struct {
			Comments []struct {
				ID     string `json:"id"`
				Author struct {
					DisplayName  string `json:"displayName"`
					EmailAddress string `json:"emailAddress"`
					AccountID    string `json:"accountId"`
					AvatarURLs   struct {
						Px48 string `json:"48x48"`
					} `json:"avatarUrls"`
				} `json:"author"`
				Body    ADFDocument `json:"body"`
				Created string      `json:"created"`
				Updated string      `json:"updated"`
			} `json:"comments"`
		} `json:"comment"`
		IssueLinks []struct {
			ID   string `json:"id"`
			Type struct {
				ID      string `json:"id"`
				Name    string `json:"name"`
				Inward  string `json:"inward"`
				Outward string `json:"outward"`
			} `json:"type"`
			InwardIssue *struct {
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
			} `json:"inwardIssue,omitempty"`
			OutwardIssue *struct {
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
			} `json:"outwardIssue,omitempty"`
		} `json:"issuelinks"`
	} `json:"fields"`
}

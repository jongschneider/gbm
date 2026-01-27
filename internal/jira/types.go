package jira

import "time"

// JiraIssue represents a basic JIRA issue with key information.
type JiraIssue struct {
	Type    string
	Key     string
	Summary string
	Status  string
}

// JiraTicketDetails represents detailed JIRA ticket information.
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

// User represents a JIRA user.
type User struct {
	DisplayName string
	Email       string
	AccountID   string
	AvatarURL   string
}

// Comment represents a JIRA comment.
type Comment struct {
	Timestamp   time.Time
	Author      User
	ID          string
	Content     string
	Created     string
	Updated     string
	Attachments []string
	Body        ADFDocument
}

// Attachment represents a JIRA file attachment.
type Attachment struct {
	Author   User
	ID       string
	Filename string
	Created  string
	MimeType string
	Content  string
	Size     int64
}

// IssueLinkType represents the type of relationship between linked issues.
type IssueLinkType struct {
	ID      string
	Name    string
	Inward  string // Description when viewing from this issue (e.g., "is blocked by")
	Outward string // Description when viewing from linked issue (e.g., "blocks")
}

// LinkedIssue represents a basic linked issue with key information.
type LinkedIssue struct {
	ID        string
	Key       string
	Summary   string
	Status    string
	Priority  string
	IssueType string
}

// IssueLink represents a link between two JIRA issues.
type IssueLink struct {
	InwardIssue  *LinkedIssue
	OutwardIssue *LinkedIssue
	Type         IssueLinkType
	ID           string
}

// ADFDocument represents a full Atlassian Document Format document.
type ADFDocument struct {
	Type    string    `json:"type"`
	Content []ADFNode `json:"content"`
	Version int       `json:"version"`
}

// ADFNode represents a node in the Atlassian Document Format tree.
type ADFNode struct {
	Type    string           `json:"type"`
	Text    string           `json:"text,omitempty"`
	Content []ADFNode        `json:"content,omitempty"`
	Attrs   map[string]any   `json:"attrs,omitempty"`
	Marks   []map[string]any `json:"marks,omitempty"` // For bold, italic, code, etc.
}

// JiraFilters defines filters for jira issue list command.
type JiraFilters struct {
	Priority   string   `yaml:"priority,omitempty"`
	Type       string   `yaml:"type,omitempty"`
	Component  string   `yaml:"component,omitempty"`
	Reporter   string   `yaml:"reporter,omitempty"`
	Assignee   string   `yaml:"assignee,omitempty"`
	OrderBy    string   `yaml:"order_by,omitempty"`
	Status     []string `yaml:"status,omitempty"`
	Labels     []string `yaml:"labels,omitempty"`
	CustomArgs []string `yaml:"custom_args,omitempty"`
	Reverse    bool     `yaml:"reverse,omitempty"`
}

// jiraRawAuthor represents author information in raw JIRA responses.
type jiraRawAuthor struct {
	DisplayName  string `json:"displayName"`
	EmailAddress string `json:"emailAddress"`
	AccountID    string `json:"accountId"`
	AvatarURLs   struct {
		Px48 string `json:"48x48"`
	} `json:"avatarUrls"`
}

// jiraRawIssueFields represents common fields for parent/subtask/linked issues.
type jiraRawIssueFields struct {
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
}

// jiraParent represents a parent issue in raw JIRA responses.
type jiraParent struct {
	ID     string             `json:"id"`
	Key    string             `json:"key"`
	Fields jiraRawIssueFields `json:"fields"`
}

// jiraRawAttachment represents an attachment in raw JIRA responses.
type jiraRawAttachment struct {
	Author   jiraRawAuthor `json:"author"`
	ID       string        `json:"id"`
	Filename string        `json:"filename"`
	Created  string        `json:"created"`
	MimeType string        `json:"mimeType"`
	Content  string        `json:"content"`
	Size     int64         `json:"size"`
}

// jiraRawComment represents a comment in raw JIRA responses.
type jiraRawComment struct {
	Author  jiraRawAuthor `json:"author"`
	ID      string        `json:"id"`
	Created string        `json:"created"`
	Updated string        `json:"updated"`
	Body    ADFDocument   `json:"body"`
}

// jiraRawLinkedIssue represents an inward/outward issue in a link.
type jiraRawLinkedIssue struct {
	ID     string             `json:"id"`
	Key    string             `json:"key"`
	Fields jiraRawIssueFields `json:"fields"`
}

// jiraRawIssueLink represents an issue link in raw JIRA responses.
type jiraRawIssueLink struct {
	InwardIssue  *jiraRawLinkedIssue `json:"inwardIssue,omitempty"`
	OutwardIssue *jiraRawLinkedIssue `json:"outwardIssue,omitempty"`
	Type         struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Inward  string `json:"inward"`
		Outward string `json:"outward"`
	} `json:"type"`
	ID string `json:"id"`
}

// jiraRawResponse represents the raw JSON response from JIRA CLI.
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
		Reporter    jiraRawAuthor       `json:"reporter"`
		Assignee    *jiraRawAuthor      `json:"assignee"`
		Parent      *jiraParent         `json:"parent"`
		Subtasks    []jiraParent        `json:"subtasks"`
		Description *ADFDocument        `json:"description"`
		Attachment  []jiraRawAttachment `json:"attachment"`
		Comment     struct {
			Comments []jiraRawComment `json:"comments"`
		} `json:"comment"`
		IssueLinks []jiraRawIssueLink `json:"issuelinks"`
	} `json:"fields"`
}

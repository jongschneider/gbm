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
}

// Comment represents a JIRA comment
type Comment struct {
	Author    string
	Content   string
	Timestamp time.Time
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
		Summary   string  `json:"summary"`
		Created   string  `json:"created"`
		DueDate   *string `json:"duedate"`
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
		} `json:"reporter"`
		Assignee *struct {
			DisplayName  string `json:"displayName"`
			EmailAddress string `json:"emailAddress"`
		} `json:"assignee"`
		Parent *struct {
			Key string `json:"key"`
		} `json:"parent"`
		Description *Description `json:"description"`
		Comment     struct {
			Comments []struct {
				Author struct {
					DisplayName string `json:"displayName"`
				} `json:"author"`
				Body struct {
					Content []struct {
						Content []struct {
							Text string `json:"text"`
						} `json:"content"`
					} `json:"content"`
				} `json:"body"`
				Created string `json:"created"`
			} `json:"comments"`
		} `json:"comment"`
	} `json:"fields"`
}

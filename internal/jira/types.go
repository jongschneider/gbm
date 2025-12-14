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

// jiraRawResponse represents the raw JSON response from JIRA CLI
type jiraRawResponse struct {
	Key    string `json:"key"`
	Self   string `json:"self"`
	Fields struct {
		Summary string  `json:"summary"`
		Created string  `json:"created"`
		DueDate *string `json:"duedate"`
		Status  struct {
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
		Description *struct {
			Content []struct {
				Content []struct {
					Text string `json:"text"`
				} `json:"content"`
			} `json:"content"`
		} `json:"description"`
		Comment struct {
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

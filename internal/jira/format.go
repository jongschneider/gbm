package jira

import (
	"fmt"
	"strings"
)

// formatJiraURL formats the JIRA URL from the self link
// Self URL format: https://company.atlassian.net/rest/api/2/issue/12345
// Returns: https://company.atlassian.net/browse/PROJ-123
func formatJiraURL(selfURL, key string) string {
	// Extract base URL from self link
	if strings.Contains(selfURL, "/rest/api/") {
		baseURL := strings.Split(selfURL, "/rest/api/")[0]
		return fmt.Sprintf("%s/browse/%s", baseURL, key)
	}
	return ""
}

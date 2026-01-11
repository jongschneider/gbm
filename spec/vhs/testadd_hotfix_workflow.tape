# Hotfix workflow with base branch selection
# Duration: 8 seconds
# Terminal: 1200x30

Output "spec/vhs/testadd_hotfix_workflow.gif"

Set FontSize 14
Set Width 1200
Set Height 480

# Start testadd
Type "go run ./cmd/service testadd --delay 500"
Enter
Sleep 1s

# Spinner for JIRA fetch
Sleep 2s

# Type selector - Hotfix
Type "h"
Sleep 0.5s

# Confirm type
Enter
Sleep 0.5s

# Hotfix name
Type "Critical security patch"
Sleep 0.5s
Enter
Sleep 0.5s

# Base branch selection - async spinner
Sleep 2s

# Select main branch
Enter
Sleep 0.5s

# Confirmation
Sleep 1s
Type "y"
Sleep 0.5s

# Exit
Enter
Sleep 2s

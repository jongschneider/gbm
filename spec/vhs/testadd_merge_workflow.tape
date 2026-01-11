# Merge workflow with async branch selection
# Duration: 10 seconds
# Terminal: 1200x30

Output "spec/vhs/testadd_merge_workflow.gif"

Set FontSize 14
Set Width 1200
Set Height 480

# Start testadd merge workflow
Type "go run ./cmd/service testadd merge --delay 500"
Enter
Sleep 1s

# Spinner for JIRA fetch
Sleep 2s

# Merge strategy selection
Type "d"
Sleep 0.5s

# Confirm strategy
Enter
Sleep 0.5s

# Source branch selection - async spinner
Sleep 2s

# Select develop
Enter
Sleep 0.5s

# Target branch selection - async spinner
Sleep 2s

# Select main
Enter
Sleep 0.5s

# Confirmation
Sleep 1s
Type "y"
Sleep 0.5s

# Exit
Enter
Sleep 2s

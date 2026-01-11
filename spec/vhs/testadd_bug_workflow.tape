# Bug workflow - similar to feature
# Duration: 8 seconds
# Terminal: 1200x30

Output "spec/vhs/testadd_bug_workflow.gif"

Set FontSize 14
Set Width 1200
Set Height 480

# Start testadd command
Type "go run ./cmd/service testadd --delay 500"
Enter
Sleep 1s

# Spinner visible during JIRA fetch
Sleep 2s

# Type selector - Bug selected
Type "b"
Sleep 0.5s

# Confirm type
Enter
Sleep 0.5s

# Bug name
Type "Fix async event loop blocking"
Sleep 0.5s
Enter
Sleep 0.5s

# Branch selection - async spinner
Sleep 2s

# Select develop branch
Type "j"
Sleep 0.3s
Enter
Sleep 0.5s

# Confirmation
Sleep 1s
Type "y"
Sleep 0.5s

# Exit
Enter
Sleep 2s

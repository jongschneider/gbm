# Feature workflow with JIRA async spinner
# Duration: 8 seconds
# Terminal: 1200x30

Output "spec/vhs/testadd_feature_workflow.gif"

Set FontSize 14
Set Width 1200
Set Height 480

# Start testadd command with delay for async spinner simulation
Type "go run ./cmd/service testadd --delay 500"
Enter
Sleep 1s

# Spinner visible during JIRA fetch (2 seconds)
Sleep 2s

# Type selector appears - Feature selected
Type "f"
Sleep 0.5s

# Confirm type selection
Enter
Sleep 0.5s

# Feature name
Type "Add async spinner support"
Sleep 0.5s
Enter
Sleep 0.5s

# Branch selection - async spinner shows
Sleep 2s

# Select main branch
Type "j"
Sleep 0.3s
Enter
Sleep 0.5s

# Confirmation screen
Sleep 1s
Type "y"
Sleep 0.5s

# Exit
Enter
Sleep 2s

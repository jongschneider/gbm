# test-<component>.tape - Brief description of what this tests
#
# Usage: vhs templates/basic-test.tape
# Prerequisites: Build your binary first (e.g., make build)

Output test-output.gif

Set Shell "bash"
Set Width 800
Set Height 400
Set TypingSpeed 50ms

# Guard: fail early if binary is missing
# Require ./bin/my-binary

# --- Initial state ---
Type "<command-to-launch-tui>"
Enter
Sleep 300ms
Screenshot screenshots/initial01.png

# --- Interaction ---
# Add navigation/input commands here
# Down
# Sleep 100ms
# Screenshot screenshots/afteraction02.png

# --- Final state ---
# Enter
# Sleep 200ms
# Screenshot screenshots/result03.png

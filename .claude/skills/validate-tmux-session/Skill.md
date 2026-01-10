---
name: validate-tmux-session
description: Creates a Tmux session in a temporary directory, displays the session name, proceeds with interactive commands. Use when you need to run commands and manually evaluate cli output.
---

# Interactive Tmux Session Skill

Creates an interactive Tmux session for paired testing and demonstration.

## Usage Pattern

1. Create a Tmux session in a temp directory
2. Print the session name for user to attach
3. Execute commands in the session using `tmux send-keys` and `tmux capture-pane`

## Step-by-Step Workflow

### 1. Create Session and Print Details

```bash
TMPDIR=$(mktemp -d)
cd "$TMPDIR"
git init -q
tmux new-session -d -s "session_$(date +%s)" -c "$TMPDIR"
SESSION_NAME=$(tmux list-sessions | grep "session_" | cut -d: -f1 | tail -1)
echo "Session: $SESSION_NAME"
echo "Temp dir: $TMPDIR"
```

### 2. Execute Commands in Session

Once approved, use `tmux send-keys` to execute commands:

```bash
tmux send-keys -t "$SESSION_NAME" "command-here" Enter
sleep 2  # Wait for output
tmux capture-pane -t "$SESSION_NAME" -p  # Capture and display
```

## Key Commands

| Task | Command |
|------|---------|
| Create session | `tmux new-session -d -s "name" -c "$dir"` |
| Send command | `tmux send-keys -t "session" "command" Enter` |
| Capture output | `tmux capture-pane -t "session" -p` |
| Clear screen | `tmux send-keys -t "session" "C-c"` |
| List sessions | `tmux list-sessions` |

## Example: Full Workflow

```bash
# 1. Setup
TMPDIR=$(mktemp -d)
tmux new-session -d -s "demo_$(date +%s)" -c "$TMPDIR"
SESSION=$(tmux list-sessions | grep "demo_" | cut -d: -f1 | tail -1)
echo "Session: $SESSION"
echo "Dir: $TMPDIR"

# 2. Execute commands
tmux send-keys -t "$SESSION" "echo 'Hello from Tmux!'" Enter
sleep 2
tmux capture-pane -t "$SESSION" -p
```

## Tips

- Always add `sleep 2` after `tmux send-keys` to allow command execution before capturing
- Use `-p` flag with `tmux capture-pane` to print output to stdout
- Store `$SESSION_NAME` in a variable for reuse across calls
- For paging output, use `tmux capture-pane -t "$SESSION" -p -S -30` to show last 30 lines

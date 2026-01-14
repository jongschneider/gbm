#!/bin/bash

# Number of iterations (default: 20)
MAX_ITERATIONS="${1:-20}"

# Read the prompt from prompt.md
PROMPT_FILE="prompt.md"

if [[ ! -f "$PROMPT_FILE" ]]; then
    echo "Error: $PROMPT_FILE not found" >&2
    exit 1
fi

RALPH_PROMPT=$(cat "$PROMPT_FILE")

echo "Starting Claude Code loop (max $MAX_ITERATIONS iterations)..."

for ((i=1; i<=MAX_ITERATIONS; i++)); do
    echo ""
    echo "=========================================="
    echo "Iteration $i of $MAX_ITERATIONS"
    echo "=========================================="

    # Run Claude Code with dangerous permissions and capture output
    OUTPUT=$(claude --dangerously-skip-permissions -p "$RALPH_PROMPT" 2>&1)

    echo "$OUTPUT"

    # Check for the completion promise
    if echo "$OUTPUT" | grep -q "<promise>COMPLETE</promise>"; then
        echo ""
        echo "=========================================="
        echo "All finished! Task completed successfully."
        echo "=========================================="
        exit 0
    fi
done

echo ""
echo "=========================================="
echo "Reached maximum iterations ($MAX_ITERATIONS) without completion."
echo "=========================================="
exit 1

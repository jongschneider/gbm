#!/bin/bash
set -e

if [ -z "$1" ]; then
  echo "Usage: $0 <prd-name> [iterations]"
  exit 1
fi

PRD_NAME="$1"
ITERATIONS="${2:-0}"

# Find specs/state directory from current or parent directories
find_state_dir() {
  local dir="$PWD"
  while [[ "$dir" != "/" ]]; do
    if [[ -d "$dir/specs/state/$PRD_NAME" ]]; then
      echo "$dir/specs/state/$PRD_NAME"
      return 0
    fi
    dir="$(dirname "$dir")"
  done
  return 1
}

STATE_DIR=$(find_state_dir) || {
  echo "Error: PRD '$PRD_NAME' not found in specs/state/"
  exit 1
}

# Find prompt.md from current or parent directories
find_prompt() {
  local dir="$PWD"
  while [[ "$dir" != "/" ]]; do
    if [[ -f "$dir/scripts/prompt.md" ]]; then
      echo "$dir/scripts/prompt.md"
      return 0
    fi
    dir="$(dirname "$dir")"
  done
  return 1
}

PRD_FILE="$STATE_DIR/prd.json"
PROGRESS_FILE="$STATE_DIR/progress.txt"
PROMPT_FILE=$(find_prompt) || {
  echo "Error: prompt.md not found in scripts/"
  exit 1
}

# jq filter to extract streaming text from assistant messages
stream_text='select(.type == "assistant").message.content[]? | select(.type == "text").text // empty | gsub("\n"; "\r\n") | . + "\r\n\n"'

# jq filter to extract final result
final_result='select(.type == "result").result // empty'

i=0
while [[ $ITERATIONS -eq 0 ]] || [[ $i -lt $ITERATIONS ]]; do
  ((i++))
  tmpfile=$(mktemp)
  trap "rm -f $tmpfile" EXIT

  prompt=$(cat "$PROMPT_FILE")

  claude \
    --dangerously-skip-permissions \
    --verbose \
    --print \
    --output-format stream-json \
    "@$PRD_FILE @$PROGRESS_FILE $prompt" \
  | stdbuf -oL grep '^{' \
  | tee "$tmpfile" \
  | stdbuf -oL jq -rj "$stream_text"

  result=$(jq -r "$final_result" "$tmpfile")

  if [[ "$result" == *"<promise>COMPLETE</promise>"* ]]; then
    echo "Ralph complete after $i iterations."
    exit 0
  fi
done

echo "Reached max iterations ($ITERATIONS) without completion."
exit 1

set -e
set -o pipefail

if [ -z "$1" ]; then
  echo "Usage: $0 <iterations>" >&2
  exit 1
fi

prompt='@docs/prd.json @docs/progress.txt
1. Find the highest priority feature to work on and work on only that feature.
This should be the one YOU decide has the highest priority - not necessarily the first in the list.
2. Check that the tests all pass via code is valid via just validate command.
3. Update the PRD with the work that was done.
Use this to leave a note for the next person working in the code base.
4. Append your progress to the progress.txt file.
5. Make a Git commit of that feature using conventional commits best practices.
ONLY WORK ON A SINGLE FEATURE.
If while implementing the feature you notice the PRD is complete output <promise>COMPLETE</promise>.'

for ((i=1; i<=$1; i++)); do
  echo "Iteration $i"

  if ! result=$(claude --dangerously-skip-permissions -p "$prompt" 2>&1 | tee /dev/tty); then
    echo "Error: claude command failed in iteration $i" >&2
    if [ -n "$result" ]; then
      echo "Partial output from claude:" >&2
      echo "$result" >&2
    fi
    exit 1
  fi

  # Output already shown via tee, just check for completion
  if [[ "$result" == *"<promise>COMPLETE</promise>"* ]]; then
    echo "PRD is complete! Exiting."
    exit 0
  fi
done

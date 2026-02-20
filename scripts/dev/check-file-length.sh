#!/bin/bash
set -e

MAX_LINES=${1:-800}
FAILED=0

for file in $(find . -name "*.go" -not -name "*_test.go" -not -path "./deps/*" -type f); do
    lines=$(wc -l < "$file")
    if [ "$lines" -gt "$MAX_LINES" ]; then
        echo "ERROR: $file has $lines lines (exceeds $MAX_LINES)"
        FAILED=1
    fi
done

exit $FAILED

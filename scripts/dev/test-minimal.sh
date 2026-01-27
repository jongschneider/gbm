#!/usr/bin/env bash
# Runs go tests with minimal output for LLM consumption
# Success: "PASS" (removes test.out)
# Failure: writes timestamped output to test.out, prints failing test names

set -uo pipefail

if ! output=$(go test ./... 2>&1); then
    {
        echo "# Test run: $(date -Iseconds)"
        echo "$output"
    } > test.out
    # Extract failing test names (--- FAIL: TestName)
    failing=$(echo "$output" | grep -oE "^--- FAIL: [A-Za-z0-9_]+" | sed 's/--- FAIL: //' | sort -u | tr '\n' ',' | sed 's/,$//')
    if [ -n "$failing" ]; then
        echo "Tests: FAIL [$failing] (see test.out)"
    else
        # No specific test failures (build error, etc)
        echo "Tests: FAIL (see test.out)"
    fi
    exit 1
fi

rm -f test.out
echo "Tests: PASS"

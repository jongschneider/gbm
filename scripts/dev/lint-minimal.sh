#!/usr/bin/env bash
# Runs golangci-lint with minimal output for LLM consumption
# Auto-fixes what it can (formatting + linter fixes), reports remaining issues
# Success: "Linting: PASS"
# Failure: writes timestamped output to lint.out, prints summary

set -uo pipefail

# Run formatter and linter with auto-fix
golangci-lint fmt 2>/dev/null || true
output=$(golangci-lint run --fix 2>&1)
exit_code=$?

if [ "$exit_code" -ne 0 ]; then
    {
        echo "# Lint run: $(date -Iseconds)"
        echo "$output"
    } > lint.out
    issues=$(echo "$output" | grep -oE "^[^:]+:[0-9]+:[0-9]+: .* \([a-zA-Z-]+\)$" | head -10)
    count=$(echo "$output" | grep -cE "^[^:]+:[0-9]+" || true)
    if [ -n "$issues" ]; then
        echo "Linting: FAIL $count issues (see lint.out)"
        echo "$issues" | sed 's/^/  /'
    else
        echo "Linting: FAIL (see lint.out)"
    fi
    exit 1
fi

rm -f lint.out
echo "Linting: PASS"

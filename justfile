
# List available commands
default:
    @just --list

# Run all validations
validate: format vet lint compile test-changed

# Format all changed Go files
format:
    #!/usr/bin/env bash
    set -euo pipefail
    # Check both staged and unstaged changes
    changed_files=$(git diff --name-only --cached; git diff --name-only; git ls-files --others --exclude-standard | grep '\.go$' || true)
    changed_files=$(echo "$changed_files" | grep '\.go$' | sort -u || true)
    if [ -n "$changed_files" ]; then
        echo "Formatting changed Go files..."
        echo "$changed_files" | xargs gofmt -w
        echo "✓ Formatting complete"
    else
        echo "No Go files changed"
    fi

# Run go vet on packages with changes
vet:
    #!/usr/bin/env bash
    set -euo pipefail
    # Check both staged and unstaged changes
    changed_files=$(git diff --name-only --cached; git diff --name-only; git ls-files --others --exclude-standard | grep '\.go$' || true)
    changed_files=$(echo "$changed_files" | grep '\.go$' | sort -u || true)
    if [ -n "$changed_files" ]; then
        echo "Running go vet on changed packages..."
        packages=$(echo "$changed_files" | xargs dirname | sort -u | sed 's|^|./|' | tr '\n' ' ')
        for pkg in $packages; do
            echo "Vetting $pkg..."
            go vet "$pkg" || exit 1
        done
        echo "✓ Vet checks passed"
    else
        echo "No Go files changed"
    fi

# Run linting on changed files
lint:
    #!/usr/bin/env bash
    set -euo pipefail
    # Check both staged and unstaged changes
    changed_files=$(git diff --name-only --cached; git diff --name-only; git ls-files --others --exclude-standard | grep '\.go$' || true)
    changed_files=$(echo "$changed_files" | grep '\.go$' | sort -u || true)
    if [ -n "$changed_files" ]; then
        echo "Running golangci-lint on changed packages..."
        packages=$(echo "$changed_files" | xargs dirname | sort -u | sed 's|^|./|' | tr '\n' ' ')
        for pkg in $packages; do
            echo "Linting $pkg..."
            golangci-lint run "$pkg" || exit 1
        done
        echo "✓ Lint checks passed"
    else
        echo "No Go files changed"
    fi

# Run linting on all packages
lint-all:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "Running golangci-lint on all packages..."
    golangci-lint run ./... || exit 1
    echo "✓ All lint checks passed"

# Run tests for packages with changes
test-changed:
    #!/usr/bin/env bash
    set -euo pipefail
    # Check both staged and unstaged changes
    changed_files=$(git diff --name-only --cached; git diff --name-only; git ls-files --others --exclude-standard | grep '\.go$' || true)
    changed_files=$(echo "$changed_files" | grep '\.go$' | sort -u || true)
    if [ -n "$changed_files" ]; then
        echo "Running tests for changed packages..."
        packages=$(echo "$changed_files" | xargs dirname | sort -u | sed 's|^|./|' | sed 's|$|/...|' | tr '\n' ' ')
        for pkg in $packages; do
            echo "Testing $pkg..."
            go test -timeout 10m -v "$pkg" || exit 1
        done
        echo "✓ All tests passed"
    else
        echo "No Go files changed"
    fi

# Run all tests
test:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "Running all tests..."
    go test -timeout 10m -v ./... || exit 1
    echo "✓ All tests passed"


# Build the gbm binary
build:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "Building gbm binary..."
    go build -o gbm ./cmd || exit 1
    echo "✓ Build successful: ./gbm"

# Compile all packages to ensure they build
compile:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "Compiling all packages..."
    go build ./... || exit 1
    echo "✓ Compilation successful"

# Build and run the gbm binary
run *ARGS:
    #!/usr/bin/env bash
    set -euo pipefail
    go build -o gbm ./cmd || exit 1
    ./gbm {{ARGS}}

# Clean build artifacts
clean:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "Cleaning build artifacts..."
    rm -f gbm
    echo "✓ Clean complete"

# Quick check - minimal validation for fast feedback
quick: format vet

# Show what files would be checked
show-changed:
    #!/usr/bin/env bash
    # Check both staged and unstaged changes
    changed_files=$(git diff --name-only --cached; git diff --name-only; git ls-files --others --exclude-standard | grep '\.go$' || true)
    changed_files=$(echo "$changed_files" | grep '\.go$' | sort -u || true)
    if [ -n "$changed_files" ]; then
        echo "Changed Go files:"
        echo "$changed_files"
        echo ""
        echo "Packages to check:"
        echo "$changed_files" | xargs dirname | sort -u
    else
        echo "No Go files changed"
    fi


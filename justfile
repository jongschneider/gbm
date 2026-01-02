
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

# Install the CLI globally as gbm2
install:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "Building and installing gbm2..."
    go build -o gbm2 ./cmd || exit 1
    sudo mv gbm2 /usr/local/bin/gbm2
    echo "✓ Installation successful: /usr/local/bin/gbm2"
    echo "✓ You can now run 'gbm2' from anywhere"

# Copy zsh completion setup commands to clipboard
completions:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "Generating zsh completion setup commands..."
    # Create the commands to paste in a new shell and copy to clipboard
    printf "# Source gbm2 completions (paste this in your shell)\nsource <(gbm2 completion zsh)\n\n# Or to make it permanent, add this to your ~/.zshrc:\n# source <(gbm2 completion zsh)\n" | pbcopy
    echo "✓ Completion setup commands copied to clipboard"
    echo "✓ Paste into a new shell session to enable completions for 'gbm2'"

# Copy shell integration setup commands to clipboard
shell-integration:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "Generating shell integration setup commands..."
    # Create the commands to paste in a new shell and copy to clipboard
    printf "# Enable gbm2 shell integration (paste this in your shell)\neval \"\$(gbm2 shell-integration)\"\n\n# Or to make it permanent, add this to your ~/.zshrc or ~/.bashrc:\n# eval \"\$(gbm2 shell-integration)\"\n\n# This enables auto-cd for worktree commands:\n#   gbm2 wt switch <name>  - switch and cd to worktree\n#   gbm2 wt add <name>     - create and cd to worktree\n#   gbm2 wt list           - TUI to select and cd to worktree\n" | pbcopy
    echo "✓ Shell integration setup commands copied to clipboard"
    echo "✓ Paste into a new shell session to enable auto-cd for worktree commands"

# Uninstall gbm2 from system
uninstall:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "Uninstalling gbm2..."
    sudo rm -f /usr/local/bin/gbm2
    echo "✓ Uninstall complete"

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


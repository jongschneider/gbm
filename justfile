
# List available commands
default:
    @just --list

# Run all validations
validate: check compile

# Run all checks (continues on failure to show all issues)
check:
    -@./scripts/dev/test-minimal.sh
    -@./scripts/dev/lint-minimal.sh
    -@./scripts/dev/check-file-length.sh

# Format code
fmt:
    @golangci-lint fmt

# Build the gbm binary
build:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "Building gbm binary..."
    go build -o gbm ./cmd || exit 1
    echo "✓ Build successful: ./gbm"

# Build the standalone gbm-jira binary
build-jira:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "Building gbm-jira binary..."
    go build -o gbm-jira ./cmd/jira || exit 1
    echo "✓ Build successful: ./gbm-jira"

# Install gbm-jira globally
install-jira:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "Building and installing gbm-jira..."
    go build -o gbm-jira ./cmd/jira || exit 1
    sudo mv gbm-jira /usr/local/bin/gbm-jira
    echo "✓ Installation successful: /usr/local/bin/gbm-jira"

# Install the CLI globally as gbm
install:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "Building and installing gbm..."
    go build -o gbm ./cmd || exit 1
    sudo mv gbm /usr/local/bin/gbm
    echo "✓ Installation successful: /usr/local/bin/gbm"
    echo "✓ You can now run 'gbm' from anywhere"

# Copy zsh completion setup commands to clipboard
completions:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "Generating zsh completion setup commands..."
    # Create the commands to paste in a new shell and copy to clipboard
    printf "# Source gbm completions (paste this in your shell)\nsource <(gbm completion zsh)\n\n# Or to make it permanent, add this to your ~/.zshrc:\n# source <(gbm completion zsh)\n" | pbcopy
    echo "✓ Completion setup commands copied to clipboard"
    echo "✓ Paste into a new shell session to enable completions for 'gbm'"

# Copy shell integration setup commands to clipboard
shell-integration:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "Generating shell integration setup commands..."
    # Create the commands to paste in a new shell and copy to clipboard
    printf "# Enable gbm shell integration (paste this in your shell)\neval \"\$(gbm shell-integration)\"\n\n# Or to make it permanent, add this to your ~/.zshrc or ~/.bashrc:\n# eval \"\$(gbm shell-integration)\"\n\n# This enables auto-cd for worktree commands:\n#   gbm wt switch <name>  - switch and cd to worktree\n#   gbm wt add <name>     - create and cd to worktree\n#   gbm wt list           - TUI to select and cd to worktree\n" | pbcopy
    echo "✓ Shell integration setup commands copied to clipboard"
    echo "✓ Paste into a new shell session to enable auto-cd for worktree commands"

# Uninstall gbm from system
uninstall:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "Uninstalling gbm..."
    sudo rm -f /usr/local/bin/gbm
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
    @rm -f test.out lint.out *.test
    @go clean ./...

# Run the TUI component storybook
storybook:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "Starting TUI storybook..."
    go run ./cmd/storybook

# Record VHS demo videos
vhs-record:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "Recording VHS demo videos..."
    mkdir -p spec/vhs
    cd spec/vhs
    for tape in *.tape; do
        if [ -f "$tape" ]; then
            echo "Recording $tape..."
            vhs < "$tape" || echo "Warning: Failed to record $tape"
        fi
    done
    echo "✓ VHS recordings complete"
    ls -lah *.gif 2>/dev/null || echo "No GIFs generated"

# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`gbm` (Git Branch Manager) is a CLI tool for managing git worktrees based on a bare repository structure. It provides both command-line and interactive TUI interfaces for worktree management.

Key features:
- Bare `.git` repository at the root with worktrees under `worktrees/` directory
- Interactive Bubble Tea TUI for browsing, creating, and managing worktrees
- JIRA integration for creating worktrees from assigned issues
- Configuration in `.gbm/config.yaml` (git remotes, JIRA credentials)
- Shell integration helpers for directory navigation

The tool uses Cobra for CLI, Bubble Tea for TUI, and shell-executes `git` commands via `os/exec`.

## Project Structure

```
cmd/
  main.go                      # Application entry point
  service/                     # Cobra command definitions and TUI
    root.go                    # Root command and CLI setup
    service.go                 # Service struct with dependencies
    init.go                    # 'init' command - create new repos
    clone.go                   # 'clone' command - clone with worktree structure
    sync.go                    # 'sync' command - sync worktrees
    wizard.go                  # Interactive wizard for setup
    worktree.go                # 'worktree' command group
    worktree_tui.go            # Bubble Tea TUI for worktree management
    worktree_fsm.go            # Finite state machine for TUI
    worktree_table.go          # Table view for worktrees
    worktree_helpers.go        # TUI helper functions
    worktree_validators.go     # Input validation
    filterable_select.go       # Custom filterable select component
    fsm_constants.go           # FSM state constants
    shell-integration.go       # Shell integration helpers
internal/
  git/                         # Git operations via exec wrapper
    service.go                 # Service struct with runCommand helper
    init.go                    # Repository initialization logic
    clone.go                   # Repository cloning logic
    worktree.go                # Worktree management logic
    errors.go                  # Git error types
  jira/                        # JIRA integration
    service.go                 # JIRA client service
    issues.go                  # Issue fetching and filtering
    branch.go                  # Branch name generation
    display.go                 # Issue display formatting
    format.go                  # Text formatting utilities
    types.go                   # JIRA types and models
    user.go                    # User management
  utils/                       # Shared utilities
    command.go                 # Command execution helpers
    fs.go                      # Filesystem utilities
deps/huh/                      # Vendored/modified UI components
```

## Architecture

### Command Flow
1. `cmd/main.go` calls `service.Execute()`
2. `cmd/service/root.go` defines the root Cobra command and registers subcommands
3. Each subcommand delegates to either:
   - `internal/git.Service` methods for git operations
   - `internal/jira.Service` methods for JIRA operations
   - Bubble Tea TUI for interactive worktree management
4. Services wrap external commands using `os/exec`

### Git Service Layer
The `internal/git.Service` executes git commands via shell:
- `runCommand()` helper handles dry-run mode and command formatting
- All git operations use `exec.Command("git", ...)` with appropriate flags
- Supports `--git-dir` for bare repo operations and `-C` for worktree operations

### TUI Layer (Bubble Tea)
The interactive worktree manager uses Bubble Tea with FSM pattern:
- `worktree_tui.go` defines the main Bubble Tea model and update/view cycle
- `worktree_fsm.go` implements state machine for navigation and actions
- `worktree_table.go` renders the worktree list with filtering
- States include: viewing list, creating worktrees, removing, checking out, syncing
- Integrates with JIRA service for issue-based worktree creation

### JIRA Integration Layer
The `internal/jira.Service` provides JIRA API integration:
- Fetches issues assigned to the user or from JQL queries
- Generates git-friendly branch names from JIRA issues
- Displays formatted issue information in TUI
- Uses JIRA credentials from `.gbm/config.yaml`

### Key Technologies
- **Cobra**: CLI framework for command-line interface
- **Bubble Tea**: Terminal UI framework for interactive worktree management
- **os/exec**: Shell execution wrapper for git commands
- **Bare repositories**: All repos use `git init --bare` with worktrees
- **JIRA API**: Integration for fetching and displaying JIRA issues
- **FSM Pattern**: Finite state machine for TUI state management

## Development Commands

This project uses `just` as the task runner. Run `just` to see all available commands.

### Just Commands

**Quick Development**
```bash
just                  # List all available commands
just build            # Build the gbm binary (outputs to ./gbm)
just run [ARGS]       # Build and run with optional arguments
just clean            # Clean build artifacts
```

**Installation**
```bash
just install          # Build and install globally as gbm2 in /usr/local/bin
just completions      # Copy zsh completion setup commands to clipboard
just shell-integration # Copy shell integration setup commands to clipboard
just uninstall        # Remove gbm2 from /usr/local/bin
```

**Validation Pipeline**
```bash
just validate         # Run full validation: format, vet, lint, compile, test-changed
just quick            # Quick validation: format and vet only (for fast feedback)
```

**Code Quality**
```bash
just format           # Format changed Go files with gofmt
just vet              # Run go vet on packages with changes
just lint             # Run golangci-lint on changed packages
just lint-all         # Run golangci-lint on all packages
just compile          # Compile all packages to ensure they build
```

**Testing**
```bash
just test             # Run all tests with 10m timeout
just test-changed     # Run tests only for packages with changes
```

**Utilities**
```bash
just show-changed     # Show what Go files and packages have changed
```

### Direct Go Commands

Standard Go commands also work:
```bash
go build -o gbm ./cmd            # Build directly
go run ./cmd                     # Run directly
go test ./...                    # All tests
go test ./internal/git           # Specific package
go test -v -run TestName         # Specific test
go mod tidy                      # Clean up dependencies
```

### Change Detection

The justfile targets use smart change detection that checks:
- Staged changes (`git diff --cached`)
- Unstaged changes (`git diff`)
- Untracked Go files

This allows for faster iteration by only validating what you've changed.

## Implementation Patterns

### Adding New Commands
1. Create command constructor in `cmd/service/` (e.g., `newFooCommand()`)
2. Register it in `cmd/service/root.go` via `rootCmd.AddCommand()`
3. Implement business logic in appropriate service:
   - Git operations → `internal/git/Service`
   - JIRA operations → `internal/jira/Service`
   - Interactive flows → Bubble Tea TUI in `cmd/service/`
4. Use `Service.runCommand()` helper for all git exec calls to get dry-run support

### Adding Git Operations
All git operations should:
- Use `exec.Command("git", ...)` for git execution
- Call `s.runCommand(cmd, dryRun)` to execute with dry-run support
- Return formatted errors with command output: `fmt.Errorf("failed to X: %w\nOutput: %s", err, output)`
- Use `--git-dir` flag when operating on bare repos
- Use `-C <path>` flag when operating within worktrees

### Output Patterns

GBM follows strict stdout/stderr separation for all commands to enable shell integration, scripting, and piping.

**Universal Rule:**
- **stdout**: Machine-readable data (paths, IDs, structured output)
- **stderr**: Human-readable messages (progress, errors, warnings)

This pattern is applied universally across all commands that output data, with no environment variable checks or special cases.

**Examples:**

```bash
# Switch command
$ gbm wt switch feature-x
/path/to/repo/worktrees/feature-x           # stdout
✓ Switched to worktree 'feature-x'          # stderr

# Add command
$ gbm wt add PROJ-123 feature/PROJ-123 -b
/path/to/repo/worktrees/PROJ-123            # stdout
✓ Created worktree 'PROJ-123' for branch 'feature/PROJ-123'  # stderr

# List command (TUI)
$ gbm wt list
[TUI interface shown on /dev/tty]
/path/to/selected/worktree                   # stdout (after selection)
✓ Selected worktree: feature-x               # stderr
```

**Benefits:**
- **Shell integration**: `path=$(gbm wt switch foo)` captures the path cleanly
- **Piping**: `gbm wt list | xargs ls` works without parsing messages
- **Scripting**: Capture data without parsing human-readable text
- **Consistent**: All commands work the same way
- **Unix philosophy**: Follows standard stdout/stderr conventions

**Implementation Guidelines:**

When implementing any command that outputs data:

```go
// CORRECT: Always separate data from messages
fmt.Println(data)                                    // stdout
fmt.Fprintf(os.Stderr, "✓ Operation successful\n")  // stderr

// WRONG: Don't mix them
fmt.Printf("Created worktree at %s\n", path)  // Mixed - hard to parse
```

**Key principles:**
1. **Always** output data to stdout - no environment variable checks
2. **Always** output messages to stderr
3. **Never** mix data and messages in the same stream
4. For TUI commands, render to `/dev/tty` to keep stdout clean

**When implementing new commands:**
1. Identify the "data" (what users might want to capture or pipe)
2. Output data to stdout using `fmt.Println()`
3. Output all messages, progress, and errors to stderr using `fmt.Fprintf(os.Stderr, ...)`
4. Never mix them in the same stream
5. Test that the command works with shell integration: `result=$(gbm ...)`

**Shell Integration:**

The shell wrapper in `cmd/service/shell-integration.go` leverages this pattern:
- Captures stdout (the data/path) while letting stderr through for messages
- Enables auto-cd functionality for worktree commands
- Single unified approach - no temp files, no environment variables
- Works consistently across all shells (bash, zsh, fish)

**TUI Rendering:**

Interactive TUI commands use `/dev/tty` for rendering:
- TUI interface renders to `/dev/tty` (the controlling terminal)
- This leaves stdout available for outputting the selected path
- After TUI exits, the selected path goes to stdout
- Success messages go to stderr
- No temp files needed

### Adding TUI Features
When extending the Bubble Tea interface:
- Add new states to the FSM in `worktree_fsm.go` using `fsm.NewFSM()`
- Define state transitions and events in the FSM configuration
- Implement corresponding UI in `worktree_tui.go` Update() and View() methods
- Use helper functions in `worktree_helpers.go` for common operations
- Follow the existing pattern of state-driven rendering in `worktree_table.go`
- Validate user input using functions in `worktree_validators.go`

### Working with JIRA Integration
To add or modify JIRA features:
- Configuration is loaded from `.gbm/config.yaml`
- Use `jira.Service` methods for API interactions
- Branch name generation follows `branch.go` patterns
- Display formatting should use `display.go` and `format.go` utilities
- Handle authentication errors gracefully with fallback to non-JIRA flow

### Dry-Run Mode
The `runCommand()` method in `internal/git/service.go` handles dry-run:
- Formats commands for display using `formatCommand()`
- Prints `[DRY RUN] <command>` instead of executing
- All commands support `--dry-run` flag to preview operations

### Repository Navigation
- Use `FindGitRoot()` instead of `git rev-parse --show-toplevel` for bare repo + worktree setups
- It correctly finds the repo root even when run from inside a worktree by detecting `/.git/worktrees/` in the git-dir path
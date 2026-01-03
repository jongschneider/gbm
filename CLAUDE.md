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
    service.go                 # Service struct with utilities
    init.go                    # Repository initialization logic
    clone.go                   # Repository cloning logic
    worktree.go                # Worktree management logic
    branch.go                  # Branch management logic
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
- All git operations use `exec.Command("git", ...)` with appropriate flags
- Dry-run mode uses `printDryRun()` helper for stderr output
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
4. Follow the dry-run pattern for operations that support it

### Adding Git Operations
All git operations should:
- Use `exec.Command("git", ...)` for git execution
- Check `if dryRun` and call `printDryRun(cmd)` before executing
- Execute using either `cmd.Output()` for capturing output or `cmd.Run()` with stdout/stderr inherited for streaming
- Return formatted errors: `fmt.Errorf("failed to X: %w", err)`
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

### Flag Override Pattern

GBM uses a flag override pattern to provide clear configuration precedence:

**Priority Order:**
1. **Explicit flags** - Highest priority (user command-line flags)
2. **Config file** - Middle priority (`.gbm/config.yaml`)
3. **Defaults** - Fallback (hardcoded values)

This pattern allows users to override config values on a per-command basis without modifying the config file, making it ideal for one-off operations and experimentation.

**Implementation:**

Use the helper functions from `internal/utils/flags.go`:

```go
import "gbm/internal/utils"

// String flags
baseBranch := utils.GetStringFlagOrConfig(cmd, "base", config.DefaultBranch)
if baseBranch == "" {
    baseBranch = "master"  // Ultimate fallback
}

// Boolean flags
dryRun := utils.GetBoolFlagOrConfig(cmd, "dry-run", config.DryRun)

// Integer flags
timeout := utils.GetIntFlagOrConfig(cmd, "timeout", config.Timeout)
```

**How it works:**

The helper functions check if a flag was explicitly set using `cmd.Flags().Changed()`:
- If the flag was set: returns the flag value (even if it's empty/zero)
- If the flag was not set: returns the config value

This is different from simply reading the flag value, which would always return something (either the user's value or the default value), making it impossible to distinguish between "user set it to the default" and "user didn't set it at all".

**Example use case:**

```bash
# Uses config value from .gbm/config.yaml (e.g., "main")
$ gbm wt add feature-x feature/x -b

# Overrides config, uses "develop" instead
$ gbm wt add feature-x feature/x -b --base develop

# Config has default_branch: "main", but user wants "master" this time
$ gbm wt add hotfix hotfix/urgent -b --base master
```

**When to use this pattern:**

Use flag override pattern when:
- The command has a corresponding config file setting
- Users might want to override config without editing the file
- There's a logical fallback chain: flag > config > hardcoded default

Don't use it for:
- Flags that don't have config equivalents
- Required flags (use cobra's required flags instead)
- Flags that are always specified by the user

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

### Dry-Run Pattern
All git operations support `--dry-run` mode using a consistent pattern:

```go
cmd := exec.Command("git", ...)

if dryRun {
    printDryRun(cmd)  // Shows command to stderr
    return mockResult  // Return early with mock data
}

// Execute command (pattern depends on operation type)
output, err := cmd.Output()  // For data capture
// OR
cmd.Stdout = os.Stdout  // For streaming operations
cmd.Stderr = os.Stderr
err := cmd.Run()
```

Dry-run messages always go to stderr (not stdout), enabling clean shell integration.

### Repository Navigation
- Use `FindGitRoot()` instead of `git rev-parse --show-toplevel` for bare repo + worktree setups
- It correctly finds the repo root even when run from inside a worktree by detecting `/.git/worktrees/` in the git-dir path

### Git Service Organization

The `internal/git` package is organized into focused files, each with a specific responsibility:

#### File Organization

**service.go** (core operations)
- Repository navigation: `FindGitRoot()`, `GetCurrentWorktree()`
- Utility functions: `printDryRun()` (dry-run message helper)
- Branch status: `GetBranchStatus()` (sync with remote)

**worktree.go** (worktree operations)
- Worktree management: `AddWorktree()`, `RemoveWorktree()`, `MoveWorktree()`, `ListWorktrees()`
- Worktree queries: `GetWorktreeBranch()`, `IsInWorktree()`
- Sync operations: `PullWorktree()`, `PushWorktree()`, `Fetch()`
- Worktree-specific branch info: `GetUpstreamBranch()` (called from worktree context)

**branch.go** (branch operations)
- Branch checking: `BranchExists()`, `BranchExistsInPath()`
- Branch operations: `DeleteBranch()`, `ListBranches()`, `MergeBranchWithCommit()`
- Branch metadata: `GetUpstreamBranch()` (generic branch operations)

**init.go** (repository initialization)
- Repository setup: `Init()` - creates bare repo + main worktree

**clone.go** (repository cloning)
- Remote cloning: `Clone()` - clones remote as bare repo + worktree
- Helper functions: `extractRepoName()`, `getDefaultBranch()`

**errors.go** (error definitions)
- Sentinel errors: validation and state errors
- Typed errors: `GitError` with exit codes and context
- Error classification: `ClassifyError()` for mapping git failures to typed errors

#### Adding New Git Operations

When adding functionality to the git service:

1. **Core utilities/navigation** → `service.go` (e.g., reading repository state)
2. **Worktree management** → `worktree.go` (e.g., creating/removing/querying worktrees)
3. **Branch operations** → `branch.go` (e.g., checking/deleting/listing branches)
4. **Initialization** → `init.go` (e.g., new repository setup)
5. **Cloning** → `clone.go` (e.g., remote repository cloning)

#### Common Patterns

All git operations should:
- Use `exec.Command("git", ...)` for git execution
- Check `if dryRun` and call `printDryRun()` before executing
- Execute using either `cmd.Output()` for capturing output or `cmd.Run()` with stdout/stderr inherited for streaming
- Return formatted errors: `fmt.Errorf("failed to X: %w", err)`
- Use `--git-dir` for bare repo operations
- Use `-C <path>` for worktree-specific operations

Example:
```go
cmd := exec.Command("git", "worktree", "add", path, branch)
if dryRun {
    printDryRun(cmd)
    return mockWorktree, nil
}
_, err := cmd.Output()
if err != nil {
    return fmt.Errorf("failed to add worktree: %w", err)
}
```

#### Public API Surface (25 functions)

| Category | Functions |
|----------|-----------|
| Repository | `NewService()`, `FindGitRoot()`, `GetCurrentWorktree()` |
| Branch Status | `GetBranchStatus()`, `Fetch()` |
| Worktree Ops | `AddWorktree()`, `ListWorktrees()`, `RemoveWorktree()`, `MoveWorktree()` |
| Worktree Queries | `GetWorktreeBranch()`, `IsInWorktree()` |
| Branch Ops | `BranchExists()`, `BranchExistsInPath()`, `DeleteBranch()`, `ListBranches()`, `MergeBranchWithCommit()` |
| Branch Info | `GetUpstreamBranch()` |
| Sync | `PullWorktree()`, `PushWorktree()` |
| Init/Clone | `Init()`, `Clone()` |

**All APIs are intentional and stable.** No unnecessary duplication.

### Testing Patterns

GBM uses **testify** for all test assertions. Choose between `require` (fail-fast) and `assert` (continue) based on the situation.

#### Assert vs Require

**Use `require` for critical assertions** (test stops immediately if failed):
- Setup operations (building binary, creating test repo)
- Prerequisites for subsequent checks (if this fails, nothing else matters)
- Single-purpose tests where one failure makes the rest meaningless

**Use `assert` for validation checks** (test continues, collects all failures):
- Multiple independent verifications in one test
- Checking various properties of an output
- When you want to see all failures at once for better debugging

**Example:**

```go
import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestExample(t *testing.T) {
    // Use require for critical setup - if this fails, stop immediately
    repo, binPath := setupGBMRepo(t)
    out, err := runGBM(t, binPath, repo.Root, "worktree", "add", "test", "test", "-b")
    require.NoError(t, err, "worktree add must succeed")

    // Use assert for multiple independent checks - collect all failures
    expectedPath := filepath.Join(repo.Root, "worktrees", "test")
    assert.DirExists(t, expectedPath, "directory should exist")
    assert.FileExists(t, filepath.Join(expectedPath, ".git"), ".git should exist")
    assert.Contains(t, out, "Created", "output should mention creation")

    // Use require when failure makes further checks impossible
    stdout, stderr, err := runGBMStdout(t, binPath, repo.Root, "wt", "switch", "test")
    require.NoError(t, err, "switch must succeed for stdout/stderr checks")

    // Use assert to verify both stdout and stderr (want to see all failures)
    assert.Contains(t, stdout, expectedPath, "stdout should have path")
    assert.NotContains(t, stdout, "Switched to", "stdout should not have messages")
    assert.Contains(t, stderr, "Switched to", "stderr should have messages")
}
```

**Common assertions (both assert and require have the same methods):**
- `NoError(t, err, "message")` - Assert no error occurred
- `Error(t, err, "message")` - Assert an error occurred
- `Equal(t, expected, actual, "message")` - Assert equality
- `Contains(t, haystack, needle, "message")` - Assert string/slice contains element
- `NotContains(t, haystack, needle, "message")` - Assert does not contain
- `Len(t, slice, length, "message")` - Assert length
- `DirExists(t, path, "message")` - Assert directory exists
- `FileExists(t, path, "message")` - Assert file exists
- `True(t, condition, "message")` - Assert boolean true
- `False(t, condition, "message")` - Assert boolean false

**Decision Guide:**

| Situation | Use | Reason |
|-----------|-----|--------|
| Test setup/prerequisites | `require` | If setup fails, nothing else matters |
| Single assertion test | `require` | One check, one failure |
| Multiple related checks | `assert` | See all failures for debugging |
| Checking command succeeded | `require` | Need output for further checks |
| Validating output properties | `assert` | Want to see all issues at once |

**Benefits:**
- **Clear intent**: `require` signals critical, `assert` signals validation
- **Better debugging**: `assert` shows all failures, not just the first
- **Fail fast when needed**: `require` stops wasted execution
- **Less boilerplate**: No manual if/err checks
- **Better diffs**: Automatically shows differences for failed comparisons

**Test Organization:**
- E2E tests in `e2e_test.go` at project root
- Unit tests colocated with source code (e.g., `internal/git/init_test.go`)
- Test utilities in `testutil/` package
- Use `t.Helper()` in test helper functions

---

## Common Workflows

This section shows common workflows and use cases for GBM.

### Workflow 1: Creating a Worktree for a Feature

**Scenario:** You're starting work on a new feature and want to create a dedicated worktree.

```bash
# Method 1: Create worktree with automatic branch creation
gbm wt add feature-x feature/feature-x -b
# Output:
# /path/to/repo/worktrees/feature-x
# ✓ Created worktree 'feature-x' for branch 'feature/feature-x'

# Automatically cd's into the worktree via shell integration
cd $( gbm wt add feature-x feature/feature-x -b )

# Method 2: Create worktree for existing branch
gbm wt add feature-x feature/feature-x
# Output:
# /path/to/repo/worktrees/feature-x
# ✓ Created worktree 'feature-x' for branch 'feature/feature-x'

# Method 3: Using JIRA integration (if configured)
gbm wt add PROJ-123  # Creates from JIRA issue
# Output:
# /path/to/repo/worktrees/PROJ-123
# ✓ Created worktree 'PROJ-123' for branch 'feature/PROJ-123'
```

**With shell integration enabled:**
```bash
# After running `eval "$(gbm shell-integration)"`, auto-cd works:
gbm wt add feature-x feature/feature-x -b
# You're automatically cd'd to /path/to/repo/worktrees/feature-x
# No need for cd command!
```

**Configuration notes:**
- Default branch (created with `-b`) uses `config.DefaultBranch` (default: `main`)
- Override with `--base`: `gbm wt add feat-x feat/x -b --base develop`

---

### Workflow 2: Switching Between Worktrees

**Scenario:** You have multiple worktrees and need to switch between them.

```bash
# List worktrees and select interactively (TUI)
gbm wt list
# Shows table of all worktrees, select with arrow keys and Enter

# Switch directly to named worktree
gbm wt switch feature-x
# Output:
# /path/to/repo/worktrees/feature-x
# ✓ Switched to worktree 'feature-x'

# With shell integration, automatically cd's:
eval "$(gbm shell-integration)"
gbm wt switch feature-x
# You're now in the worktree directory

# Using aliases
gbm wt sw feature-x      # Short form: sw
gbm wt s feature-x       # Shorter form: s
gbm worktree switch feature-x  # Full command form

# List aliases
gbm wt list        # Full form
gbm wt ls          # Short form: ls
gbm wt l           # Shorter form: l
```

**Error handling:**
```bash
$ gbm wt switch nonexistent
# Error: worktree 'nonexistent' not found
# Try: gbm wt list
```

---

### Workflow 3: Syncing Worktrees with Remote

**Scenario:** You want to ensure your worktree is up-to-date with the remote.

```bash
# Fetch latest from remote for all branches
gbm wt sync --fetch

# Fetch and pull in current worktree
gbm wt sync --pull

# Fetch and push in current worktree
gbm wt sync --push

# Dry-run mode (see what would happen)
gbm wt sync --fetch --dry-run
# Output shows git commands without executing
```

---

### Workflow 4: Automatic File Copying

**Scenario:** You want to automatically copy configuration files (`.env`, `.config`) to new worktrees.

**Configuration in `.gbm/config.yaml`:**
```yaml
file_copy:
  auto:
    enabled: true
    source_worktree: "{default}"  # Use worktree with default branch
    copy_ignored: true            # Copy .gitignored files
    copy_untracked: false         # Don't copy untracked files
    exclude:
      - "*.log"                   # Exclude log files
      - "node_modules/"           # Exclude directories
```

**When you create a worktree:**
```bash
gbm wt add feature-x feature/x -b
# Automatically copies:
# - .env (if ignored)
# - .config/ directory (if ignored)
# - Other files matching copy_ignored config
# - But excludes *.log and node_modules/
```

**How source resolution works:**
- `"{default}"` → Worktree with branch matching `DefaultBranch`
- `"{current}"` → The worktree you're currently in
- `"main"` → Specific worktree name

**Disable auto-copy for one operation:**
```bash
# Modify config temporarily or use explicit source
gbm wt add feature-x feature/x -b
# Auto copy happens automatically
```

---

### Workflow 5: Path Templates in Configuration

**Scenario:** You want to organize worktrees outside the repository directory.

**Configuration in `.gbm/config.yaml`:**
```yaml
worktrees_dir: ../{gitroot}-worktrees
# For repo "gbm": /path/to/gbm-worktrees
# For repo "cli": /path/to/cli-worktrees

# Or with home directory:
worktrees_dir: ~/dev/{gitroot}/worktrees
# For repo "gbm": /home/user/dev/gbm/worktrees
```

**Template variables:**
- `{gitroot}` - Repository directory name (e.g., "gbm")
- `{branch}` - Branch name (context-specific)
- `{issue}` - JIRA issue key (if JIRA enabled)

**Benefits:**
- Share config across multiple repos
- Organize worktrees per-repo or centrally
- Easy relocation if repo is renamed

---

### Workflow 6: Overriding Configuration Per Command

**Scenario:** You want to use a different base branch for one-off operations.

```bash
# Config has default_branch: "main"
gbm wt add feat-x feat/x -b
# Uses "main" as base branch

# Override for this command only
gbm wt add feat-x feat/x -b --base develop
# Uses "develop" as base branch

# Without modifying .gbm/config.yaml
# Perfect for testing or experimentation
```

**Other flag overrides:**
- `--dry-run` - Preview what will happen
- `--base <branch>` - Base branch for new branches
- `--remote <name>` - Git remote to use

---

## Testing Guide

### Running Tests

**Run all tests:**
```bash
just test
# Runs: go test ./... -timeout 10m

# Or directly:
go test ./...
```

**Run tests for changed packages only:**
```bash
just test-changed
# Only runs tests for files you've modified
```

**Run specific test:**
```bash
go test ./internal/git -run TestAddWorktree
go test ./... -v  # Verbose output
```

**Run E2E tests only:**
```bash
go test -v -run TestE2E ./...
# Tests real command execution, shell integration, etc.
```

**Run unit tests only (no E2E):**
```bash
go test -v -short ./...  # Short flag skips slow E2E tests
```

### Understanding Test Structure

**Unit tests:**
- Located with source code: `internal/git/worktree_test.go`
- Fast, focused on single functions
- Mock dependencies
- Example: `TestAddWorktree` tests git.Service.AddWorktree()

**E2E tests:**
- Located in `e2e_test.go` at project root
- Build binary and run real commands
- Validate real workflows
- Example: `TestE2E_WorktreeAdd_CLI` tests `gbm wt add` command

**Test utilities:**
- In `testutil/` package
- Reusable across tests
- Example: `testutil.TestRepo` for git repository setup

### Writing New Tests

**Example unit test:**
```go
func TestMyFunction(t *testing.T) {
    // Arrange: Set up test data
    service := git.NewService(repoRoot)
    
    // Act: Call the function
    result, err := service.MyFunction(...)
    
    // Assert: Verify result
    require.NoError(t, err, "function must succeed")
    assert.Equal(t, expected, result, "result should match expected")
}
```

**Example E2E test:**
```go
func TestE2E_MyWorkflow(t *testing.T) {
    // Setup: Build binary and create test repo
    binPath := buildBinary(t)
    repo := setupGBMRepo(t)
    
    // Execute: Run command
    out, err := runGBM(t, binPath, repo.Root, "wt", "add", "test", "test", "-b")
    
    // Verify: Check results
    require.NoError(t, err, "command must succeed")
    assert.DirExists(t, filepath.Join(repo.Root, "worktrees", "test"))
}
```

### Test Coverage

**Check coverage:**
```bash
go test ./... -cover
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out  # Opens HTML report
```

**Target:**
- Unit tests: >80% coverage
- E2E tests: Core workflows covered
- Overall: >70% combined coverage

---

## Configuration Reference

GBM is configured via `.gbm/config.yaml` in the repository root. This section documents all available options.

### Basic Configuration

```yaml
# Default base branch for new branches (-b flag)
default_branch: main

# Worktrees directory (relative to repo root)
worktrees_dir: worktrees

# Supports templates:
worktrees_dir: ../{gitroot}-worktrees  # /path/to/gbm-worktrees
worktrees_dir: ~/dev/{gitroot}/wt      # /home/user/dev/gbm/wt
```

### Git Configuration

```yaml
# Git remotes to configure
remotes:
  origin:
    url: git@github.com:user/repo.git
  upstream:
    url: git@github.com:original/repo.git

# Default branch to fetch from when syncing
default_branch: main

# Fetch tags when syncing
fetch_tags: true
```

### JIRA Integration

```yaml
jira:
  enabled: true
  host: https://jira.company.com
  
  # Your JIRA API token (generate in JIRA → Settings → API tokens)
  api_token: ${JIRA_API_TOKEN}  # Or paste directly
  
  # Your JIRA username (email for cloud instances)
  username: user@company.com
  
  # JQL query to filter issues
  jql: "assignee = currentUser() AND status != Done"
  
  # Generate branch name from issue key
  # Example: PROJ-123 → feature/PROJ-123
  branch_prefix: feature/
```

### File Copying Configuration

```yaml
file_copy:
  # Rule-based copying (existing feature)
  rules:
    - source: ".env"      # File to copy from source worktree
      target: ".env"      # Where to copy it to
      create_if_missing: true  # Create empty file if source missing
    
    - source: "config/"   # Directory to copy
      target: "config/"
      create_if_missing: false

  # Automatic copying (optional new feature)
  auto:
    enabled: false                    # Enable auto-copy
    source_worktree: "{default}"      # Source: {default}, {current}, or name
    copy_ignored: true                # Copy .gitignored files
    copy_untracked: false             # Copy untracked files
    exclude:                          # Exclude patterns (gitignore syntax)
      - "*.log"                       # Exclude log files
      - "node_modules/"               # Exclude directories
      - ".DS_Store"                   # Exclude specific files
```

### Shell Integration

GBM provides shell integration for auto-cd functionality. Add to your shell config:

```bash
# In ~/.bashrc or ~/.zshrc:
eval "$(gbm shell-integration)"

# Now these commands auto-cd:
gbm wt switch feature-x
gbm wt add feat-x feat/x -b
gbm wt list               # Select worktree with TUI
```

### Example Complete Configuration

```yaml
# Repository structure
default_branch: main
worktrees_dir: worktrees

# Git remotes
remotes:
  origin:
    url: git@github.com:user/repo.git
  upstream:
    url: git@github.com:org/repo.git

# JIRA integration (optional)
jira:
  enabled: true
  host: https://jira.company.com
  username: user@company.com
  api_token: ${JIRA_API_TOKEN}
  jql: "assignee = currentUser() AND sprint = openSprints()"
  branch_prefix: feature/

# File copying
file_copy:
  rules:
    - source: ".env"
      target: ".env"
      create_if_missing: true
    
    - source: ".secrets/"
      target: ".secrets/"
      create_if_missing: false
  
  auto:
    enabled: true
    source_worktree: "{default}"
    copy_ignored: true
    copy_untracked: false
    exclude:
      - "*.log"
      - "build/"
      - "node_modules/"
```

### Flag Overrides

Many options can be overridden per-command without editing config:

```bash
# Override default_branch
gbm wt add feat-x feat/x -b --base develop

# Override worktrees_dir (if supported by command)
# Dry-run mode
gbm wt sync --dry-run

# List all available flags
gbm --help
gbm wt --help
gbm wt add --help
```

---

## Troubleshooting

### Common Issues and Solutions

#### Issue: "worktree not found"

**Symptom:**
```bash
$ gbm wt switch feature-x
Error: worktree 'feature-x' not found
```

**Solutions:**

1. **List available worktrees:**
   ```bash
   gbm wt list      # See all worktrees in TUI
   gbm wt ls        # Alternative
   ```

2. **Check worktrees directory:**
   ```bash
   ls /path/to/repo/worktrees/
   # Check if directory exists and has worktrees
   ```

3. **Verify git configuration:**
   ```bash
   git worktree list
   # Should show same worktrees as gbm
   ```

---

#### Issue: Auto-cd not working

**Symptom:**
```bash
gbm wt switch feature-x
# Prints path to stdout but doesn't cd

# vs. expected:
gbm wt switch feature-x
# cd's automatically to /path/to/repo/worktrees/feature-x
```

**Solutions:**

1. **Verify shell integration enabled:**
   ```bash
   # Check if function is defined
   type gbm2  # Should show it's a function, not a command
   
   # If not a function, enable it:
   eval "$(gbm shell-integration)"
   
   # Or add to ~/.bashrc / ~/.zshrc:
   eval "$(gbm shell-integration)"
   ```

2. **Verify shell integration script:**
   ```bash
   gbm shell-integration  # Shows the integration script
   
   # Should contain something like:
   # gbm2() { command gbm2 "$@"; ... }
   ```

3. **Check shell configuration:**
   ```bash
   echo $SHELL  # Should be bash or zsh
   # Fish shell needs separate setup (future enhancement)
   ```

---

#### Issue: ".gbm/config.yaml not found"

**Symptom:**
```bash
$ gbm wt add feature-x feature/x
Error: .gbm/config.yaml not found
```

**Solutions:**

1. **Initialize repository:**
   ```bash
   gbm init                    # Creates .gbm/config.yaml
   # or
   gbm clone <repo-url>        # Clone with config
   ```

2. **Create config manually:**
   ```bash
   mkdir -p .gbm
   cp config.example.yaml .gbm/config.yaml
   # Edit .gbm/config.yaml with your settings
   ```

3. **Verify current directory:**
   ```bash
   pwd                         # Must be repo root
   ls -la .gbm/config.yaml     # Should exist
   ```

---

#### Issue: TUI (worktree list) fails

**Symptom:**
```bash
$ gbm wt list
Error: failed to open /dev/tty: ... (TUI requires an interactive terminal)
```

**Solutions:**

1. **TUI requires interactive terminal:**
   ```bash
   # Works (interactive):
   gbm wt list
   
   # Doesn't work (piped):
   echo "" | gbm wt list
   gbm wt list < /dev/null
   ```

2. **Use switch command instead:**
   ```bash
   # Alternative without TUI:
   gbm wt switch feature-x
   ```

3. **Check terminal:**
   ```bash
   tty  # Should show /dev/pts/X or similar
   # If "not a tty", you're not in interactive shell
   ```

---

#### Issue: "failed to create worktree"

**Symptom:**
```bash
$ gbm wt add feature-x feature/x -b
Error: failed to create worktree: ...
```

**Solutions:**

1. **Check branch status:**
   ```bash
   git branch -a | grep feature/x
   # Verify branch doesn't already exist
   ```

2. **Verify permissions:**
   ```bash
   ls -ld worktrees/  # Should be writable (drwxr-xr-x)
   chmod -R u+w worktrees/
   ```

3. **Check git status:**
   ```bash
   cd main-worktree  # Go to main worktree
   git status        # Should be clean
   ```

4. **Check remote:**
   ```bash
   git remote -v  # Verify remotes configured
   git fetch      # Update remote refs
   ```

---

#### Issue: File copying not working

**Symptom:**
```bash
# Config has auto copy enabled, but files not copied
gbm wt add feature-x feature/x -b
# Expected files (.env, etc.) not in new worktree
```

**Solutions:**

1. **Verify auto-copy enabled:**
   ```bash
   cat .gbm/config.yaml | grep -A5 "auto:"
   # Should show: enabled: true
   ```

2. **Check source files exist:**
   ```bash
   # Go to source worktree (default branch)
   gbm wt switch main  # or whatever default_branch is
   ls .env  # Should exist and be gitignored
   git ls-files --others --ignored --exclude-standard | grep .env
   ```

3. **Check exclude patterns:**
   ```bash
   cat .gbm/config.yaml | grep -A10 "exclude:"
   # Verify your files aren't in exclude list
   ```

4. **Check source_worktree resolution:**
   ```bash
   # If using {default}, verify worktree with default_branch exists:
   gbm wt list  # Should show worktree with default_branch
   ```

---

#### Issue: "permission denied" errors

**Symptom:**
```bash
Error: permission denied: worktrees/feature-x
```

**Solutions:**

1. **Fix permissions:**
   ```bash
   chmod -R u+w worktrees/
   # Make all files user-writable
   ```

2. **Check ownership:**
   ```bash
   ls -l worktrees/
   # Should be owned by you, not another user
   ```

3. **On macOS with mounted directories:**
   ```bash
   # If /Volumes mount, may need special permissions
   mount  # Check mount options
   ```

---

#### Issue: "dirty worktree" error

**Symptom:**
```bash
Error: worktree has uncommitted changes
```

**Solutions:**

1. **Commit changes:**
   ```bash
   cd worktrees/feature-x
   git status         # See changes
   git add .
   git commit -m "message"
   ```

2. **Stash changes:**
   ```bash
   cd worktrees/feature-x
   git stash          # Save changes temporarily
   # Do what you need
   git stash pop      # Restore changes
   ```

3. **Discard changes:**
   ```bash
   cd worktrees/feature-x
   git checkout .     # Discard all changes
   git clean -fd      # Remove untracked files
   ```

---

### Getting Help

**Built-in help:**
```bash
gbm --help          # General help
gbm wt --help       # Worktree commands
gbm wt add --help   # Specific command
```

**Debug mode:**
```bash
# Enable debug output (if available)
gbm --debug wt list

# Or verbose:
gbm -v wt switch feature-x
```

**Check version:**
```bash
gbm --version       # Show version and build info
```

---

## Performance Tips

### Optimize Large Repositories

**For repositories with many worktrees:**

1. **Use worktree list filtering:**
   ```bash
   gbm wt list  # Use TUI to filter and find
   # Type to filter by name
   ```

2. **Organize with path templates:**
   ```yaml
   # Instead of: worktrees: worktrees
   # Use: worktrees: ../{gitroot}-worktrees
   # Spreads worktrees across filesystem
   ```

3. **Exclude large directories from file copying:**
   ```yaml
   file_copy:
     auto:
       exclude:
         - "node_modules/"
         - "target/"
         - ".cache/"
   ```

### Speed up syncing

```bash
# Fetch only current branch
git fetch origin $(git rev-parse --abbrev-ref HEAD)

# Or use gbm's sync commands
gbm wt sync --fetch   # Just fetch, no pull
```
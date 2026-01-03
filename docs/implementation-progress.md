# GBM Implementation Progress Tracker

**Last Updated:** 2026-01-02
**Current Phase:** P2.1 - Template Variables - ✅ COMPLETE
**Reference:** [improvement-prd.md](./improvement-prd.md)

---

## 📊 Status Overview

**Progress:** 2/2 tasks complete in P2.1 ✅ - **PHASE 2.1 COMPLETE** 🎉
**Phase 1 Progress:** 11/11 tasks complete ✅ - COMPLETE
**P2.1 Progress:** 2/2 tasks complete ✅
**P2.2 Progress:** (Optional - skipped for Phase 2 completion)

**P1.1 Tasks:**
| Task | Status | Date | Time |
|------|--------|------|------|
| 1.1.1 Switch Command | ✅ COMPLETE | 2026-01-02 | ~2h |
| 1.1.2 Shell Wrapper | ✅ COMPLETE | 2026-01-02 | ~1h |
| 1.1.3 Add Command | ✅ COMPLETE | 2026-01-02 | ~1h |
| 1.1.4 TUI /dev/tty | ✅ COMPLETE | 2026-01-02 | ~2h |
| 1.1.5 Documentation | ✅ COMPLETE | 2026-01-02 | ~30min |

**P1.2 Tasks:**
| Task | Status | Date | Time |
|------|--------|------|------|
| 1.2.1 Testutil Package | ✅ COMPLETE | 2026-01-02 | ~2h |
| 1.2.2 E2E Worktree Tests | ✅ COMPLETE | 2026-01-02 | ~4h |
| 1.2.3 E2E Shell Integration | ✅ COMPLETE | 2026-01-02 | ~3h |

**P1.3 Tasks:**
| Task | Status | Date | Time |
|------|--------|------|------|
| 1.3.1 Flag Override Helper | ✅ COMPLETE | 2026-01-02 | ~2h |
| 1.3.2 Apply to --base Flag | ✅ COMPLETE | 2026-01-02 | ~1h |
| 1.3.3 Documentation | ✅ COMPLETE | 2026-01-02 | ~1h |

---

## ✅ Completed Work

### Task 1.1.1: Update worktree switch command output
**Completed:** 2026-01-02
**File:** `cmd/service/worktree.go` (lines 494-633)

**What Was Done:**
1. Applied universal stdout/stderr pattern to switch command
2. Removed `GBM_SHELL_INTEGRATION` environment variable check
3. Always outputs path to stdout (machine-readable)
4. Always outputs messages to stderr (human-readable)
5. Removed `--print-path` flag (redundant with new pattern)
6. Fixed "Switching to previous worktree" message to use stderr

**Code Pattern Established:**
```go
// Always output path to stdout (machine-readable)
fmt.Println(targetWorktree.Path)

// Always output message to stderr (human-readable)
fmt.Fprintf(os.Stderr, "✓ Switched to worktree '%s'\n", worktreeName)
```

**Key Decision:** Removed `--print-path` flag because the universal pattern makes it redundant. Users can suppress stderr with `2>/dev/null` if they want path-only output.

**Validation:** ✅ All tests pass, linting clean, compiles successfully

---

### Task 1.1.3: Add stdout/stderr separation to worktree add
**Completed:** 2026-01-02
**File:** `cmd/service/worktree.go` (lines 101-105, 133-137)

**What Was Done:**
1. Applied universal stdout/stderr pattern to worktree add command
2. Updated both success paths: initial creation and retry after user confirmation
3. Removed output mixing - path goes to stdout, messages go to stderr
4. No environment variable checks needed - always uses universal pattern

**Code Pattern Applied:**
```go
// Always output path to stdout (machine-readable)
fmt.Println(wt.Path)

// Always output message to stderr (human-readable)
fmt.Fprintf(os.Stderr, "✓ Created worktree '%s' for branch '%s'\n", wt.Name, wt.Branch)
```

**Changes Made:**
- **Line 101-105**: First success path (when branch exists or -b flag used)
- **Line 133-137**: Retry success path (when user confirms branch creation)

**Benefits:**
- Scriptable: `new_wt=$(gbm wt add feat-x feat-x -b)` captures path
- Pipeable: `gbm wt add ... | xargs ls` works cleanly
- Shell integration: Auto-cd will work once shell wrapper is updated (Task 1.1.2)
- Consistent: Same pattern as switch command (Task 1.1.1)

**Validation:** ✅ All tests pass, linting clean, compiles successfully

---

### Task 1.1.4: Update TUI to use /dev/tty with stdout output
**Completed:** 2026-01-02
**File:** `cmd/service/worktree_table.go` (lines 389-432, 324-333)

**What Was Done:**
1. Updated TUI to render to `/dev/tty` instead of stdout
2. Applied universal stdout/stderr pattern to TUI output
3. Simplified worktree selection - stores path instead of calling subprocess
4. **Eliminated all temp file logic** from TUI

**Key Changes:**

**Change 1: /dev/tty Rendering (lines 389-407)**
```go
// Open /dev/tty for TUI rendering
tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
if err != nil {
    return fmt.Errorf("failed to open /dev/tty: %w (TUI requires an interactive terminal)", err)
}
defer func() {
    _ = tty.Close()
}()

// TUI renders to /dev/tty, leaving stdout clean
p := tea.NewProgram(m,
    tea.WithInput(tty),
    tea.WithOutput(tty),
)
```

**Change 2: Universal Pattern Output (lines 422-431)**
```go
// Output path to stdout if user selected a worktree (universal pattern)
if model, ok := finalModel.(worktreeTableModel); ok {
    if model.switchOutput != "" {
        // Always output path to stdout (machine-readable)
        fmt.Println(model.switchOutput)

        // Always output message to stderr (human-readable)
        fmt.Fprintf(os.Stderr, "✓ Selected worktree: %s\n", filepath.Base(model.switchOutput))
    }
}
```

**Change 3: Simplified Selection (lines 324-333)**
```go
case " ", "enter":
    // Store selected worktree path for output after TUI exits
    cursor := m.table.Cursor()
    if cursor >= 0 && cursor < len(m.worktrees) {
        targetWorktree := m.worktrees[cursor]
        // Store just the path (not "cd <path>")
        m.switchOutput = targetWorktree.Path
        return m, tea.Quit
    }
    return m, nil
```

**What Was Removed:**
- ❌ All temp file creation code (`$TMPDIR/.gbm-switch-$$`)
- ❌ `os.Getppid()` usage for temp file naming
- ❌ `os.WriteFile()` for temp file writing
- ❌ Subprocess call to `gbm wt switch` from within TUI
- ❌ Environment variable passing to subprocess

**Benefits:**
- **No temp files**: Completely eliminated temp file complexity
- **Cleaner architecture**: TUI directly outputs path instead of calling subprocess
- **Scriptable**: `result=$(gbm wt list)` captures selected path
- **Consistent**: Same stdout/stderr pattern as switch and add commands
- **Simpler**: ~30 lines of code removed
- **More reliable**: No temp file cleanup issues

**Trade-off:**
- TUI now requires `/dev/tty` to be available (acceptable for interactive TUI use case)
- Returns clear error message if `/dev/tty` unavailable

**Post-Implementation Fixes:**

**Fix 1: Alt Screen Mode (cmd/service/worktree_table.go:409)**
- Added `tea.WithAltScreen()` option to enable full-screen interactive mode
- Without this, TUI would print but not become interactive

**Fix 2: TUI Styling with /dev/tty (cmd/service/worktree_table.go:397-405)**
- Added `github.com/muesli/termenv` import for explicit color profile control
- Set up lipgloss renderer BEFORE creating table model (order matters!)
- Configured renderer with explicit termenv options:
  ```go
  renderer := lipgloss.NewRenderer(tty,
      termenv.WithColorCache(true),           // Cache color conversions for performance
      termenv.WithTTY(true),                  // Explicitly mark as TTY
      termenv.WithProfile(termenv.TrueColor), // Force 24-bit color support
  )
  lipgloss.SetDefaultRenderer(renderer)
  ```
- This ensures lipgloss renders with full colors even when using custom `/dev/tty` file handle
- Without this, TUI would work but row highlighting/selection colors wouldn't be visible

**Why the styling fix was needed:**
- When using `/dev/tty` as a custom file handle, lipgloss can't auto-detect terminal capabilities
- Must explicitly set renderer with color profile before creating styled components
- Renderer must be set as default BEFORE calling `newWorktreeTable()` so styles apply correctly

**Validation:** ✅ All tests pass, linting clean, compiles successfully, interactive TUI with full styling confirmed working

---

### Task 1.1.2: Simplify shell integration wrapper
**Completed:** 2026-01-02
**File:** `cmd/service/shell-integration.go` (lines 36-63)

**What Was Done:**
1. Applied universal stdout/stderr pattern to shell wrapper
2. **Eliminated all temp file logic** - no more `$TMPDIR/.gbm-switch-$$`
3. **Removed environment variable** - no more `GBM_SHELL_INTEGRATION=1`
4. Unified approach for all worktree commands that output paths
5. Added support for all command aliases (switch/sw/s, add/a, list/ls/l)
6. Added auto-cd support for `add` command

**Code Pattern Applied:**
```bash
# Capture stdout (path) while letting stderr through for messages
local result
result=$(command gbm2 "$@" 2>/dev/stderr)
local exit_code=$?

# If successful and result is a directory, cd to it
if [ $exit_code -eq 0 ] && [ -n "$result" ] && [ -d "$result" ]; then
    cd "$result"
fi
```

**What Was Removed:**
- ❌ `export GBM_SHELL_INTEGRATION=1` environment variable
- ❌ All temp file creation (`$TMPDIR/.gbm-switch-$$`)
- ❌ All temp file reading and cleanup logic
- ❌ Separate code paths for `worktree` vs `wt` commands
- ❌ `grep '^cd '` parsing and `eval` of cd commands
- ❌ ~40 lines of complex shell code

**Code Size Reduction:**
- **Before:** ~60 lines of shell script
- **After:** ~25 lines of shell script
- **Reduction:** ~60% smaller, much simpler

**Supported Commands (all auto-cd after success):**
- `gbm2 wt switch <name>` / `gbm2 worktree switch <name>`
- `gbm2 wt sw <name>` / `gbm2 worktree sw <name>`
- `gbm2 wt s <name>` / `gbm2 worktree s <name>`
- `gbm2 wt add <name> <branch>` / `gbm2 worktree add <name> <branch>`
- `gbm2 wt a <name> <branch>` / `gbm2 worktree a <name> <branch>`
- `gbm2 wt list` / `gbm2 worktree list`
- `gbm2 wt ls` / `gbm2 worktree ls`
- `gbm2 wt l` / `gbm2 worktree l`

**Benefits:**
- **No temp files**: Completely eliminated temp file complexity
- **Single code path**: All commands use same pattern (DRY)
- **Simpler**: 60% less code, easier to understand and maintain
- **More reliable**: No temp file cleanup issues or race conditions
- **Consistent**: Works exactly the same way for all commands
- **New feature**: Auto-cd after `wt add` command

**How It Works:**
1. Shell wrapper checks if command is a path-outputting worktree command
2. Captures stdout (the path) while letting stderr messages through
3. Checks if the result is a valid directory
4. If yes, cd to it in the current shell
5. Preserves exit code for error handling

**Additional Enhancement:**
- Added `just shell-integration` command to justfile (similar to `just completions`)
- Copies shell integration setup commands to clipboard for easy testing
- Updated CLAUDE.md to document the new command

**Validation:** ✅ All tests pass, linting clean, compiles successfully

---

### Task 1.1.5: Document universal stdout/stderr pattern
**Completed:** 2026-01-02
**File:** `CLAUDE.md` (new "Output Patterns" section after line 187)

**What Was Done:**
1. Added comprehensive "Output Patterns" section to CLAUDE.md
2. Documented the universal stdout/stderr pattern for all commands
3. Provided clear examples for switch, add, and list commands
4. Explained benefits of the pattern (shell integration, scripting, piping)
5. Added implementation guidelines for developers
6. Documented shell integration and TUI rendering approaches

**Content Added:**

**Universal Rule:**
- stdout: Machine-readable data (paths, IDs, structured output)
- stderr: Human-readable messages (progress, errors, warnings)

**Examples provided:**
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

**Implementation Guidelines:**
- Always use `fmt.Println(data)` for stdout
- Always use `fmt.Fprintf(os.Stderr, ...)` for messages
- Never mix data and messages in the same stream
- For TUI commands, render to `/dev/tty` to keep stdout clean

**Key principles documented:**
1. **Always** output data to stdout - no environment variable checks
2. **Always** output messages to stderr
3. **Never** mix data and messages in the same stream
4. For TUI commands, render to `/dev/tty` to keep stdout clean
5. Test that commands work with shell integration: `result=$(gbm ...)`

**Why this is important:**
- Ensures all future commands follow the universal pattern
- Provides clear examples for developers adding new commands
- Documents the rationale behind the stdout/stderr separation
- Explains how shell integration and TUI rendering work together
- Makes the pattern discoverable in the developer guide

**Benefits:**
- Future commands will automatically follow the pattern
- Consistent behavior across all commands
- Easy reference for code reviews
- Clear onboarding for new contributors

**Validation:** ✅ Documentation added, examples provided, guidelines clear

---

### Task 1.2.1: Create testutil package
**Completed:** 2026-01-02
**Files:** `testutil/repo.go`, `testutil/repo_test.go`

**What Was Done:**
1. Created reusable test utilities package for E2E testing
2. Implemented `TestRepo` struct with helper methods for git operations
3. Added comprehensive test coverage for all utility functions
4. Followed git-wt pattern for test repository management

**Key Features:**

**TestRepo Structure:**
- Automatic temp directory creation with cleanup via `t.Cleanup()`
- Symlink resolution for macOS `/var` → `/private/var` compatibility
- Pre-configured git repository with test user credentials
- Default branch set to `main`

**Helper Methods:**
```go
// Core git operations
Git(args ...string) string              // Execute git command, fatal on error
GitE(args ...string) (string, error)    // Execute git command, return error

// File operations
CreateFile(path, content string)        // Create file with automatic directory creation
Commit(message string)                  // Stage all changes and commit

// Directory operations
Chdir() func()                          // Change to repo dir, returns restore function
Path(relPath string) string             // Get absolute path to file in repo
ParentDir() string                      // Get parent directory (for worktree tests)
```

**Test Coverage:**
Comprehensive tests validating all helper methods:
- `TestNewTestRepo` - Repository initialization
- `TestGit` / `TestGitE` - Git command execution (success and error cases)
- `TestCreateFile` - File creation including nested directories
- `TestCommit` - Staging and committing changes
- `TestChdir` - Directory navigation with restoration
- `TestPath` / `TestParentDir` - Path helper methods
- `TestCleanup` - Automatic cleanup verification
- `TestIntegration` - Complete workflow with branches and commits

**Code Quality:**
- All linter checks pass (errcheck violations fixed)
- 10 test cases, all passing
- Follows established patterns from git-wt reference implementation

**Benefits:**
- **Reusable**: Common git operations abstracted into simple methods
- **Safe**: Automatic cleanup prevents temp directory leaks
- **Convenient**: Helper methods reduce boilerplate in E2E tests
- **Well-tested**: Test utilities themselves are thoroughly tested
- **Ready for E2E**: Foundation for Tasks 1.2.2 and 1.2.3

**Files Created:**
- `testutil/repo.go` (138 lines) - Core test repository utilities
- `testutil/repo_test.go` (219 lines) - Comprehensive test suite

**Validation:** ✅ All tests pass, linting clean, ready for E2E test implementation

---

### Task 1.2.2: Add E2E test for worktree creation
**Completed:** 2026-01-02
**File:** `e2e_test.go` (357 lines)

**What Was Done:**
1. Created comprehensive E2E test suite for worktree operations
2. Implemented test helpers for building binary and running commands
3. Added tests for stdout/stderr separation verification
4. Tested command aliases and error handling
5. Verified universal stdout/stderr pattern implementation

**Test Infrastructure:**

**Helper Functions:**
```go
// Binary building and execution
buildBinary(t) string                              // Build gbm for testing
runGBM(t, binPath, dir string, args...) (string, error)  // Run with combined output
runGBMStdout(t, binPath, dir string, args...) (stdout, stderr string, err error)  // Separate streams

// Test setup
setupGBMRepo(t) (*testRepo, string)               // Create GBM repo with initial commit
```

**Test Cases Implemented:**

**1. Basic Worktree Operations:**
- `TestE2E_WorktreeAdd_CLI` - Create worktree with new branch
- `TestE2E_WorktreeAdd_ExistingBranch` - Create worktree from existing branch
- `TestE2E_Init_CreatesStructure` - Verify gbm init creates correct structure

**2. Stdout/Stderr Separation (Universal Pattern):**
- `TestE2E_WorktreeSwitch_StdoutOutput` - Verify path goes to stdout, messages to stderr
- `TestE2E_WorktreeAdd_StdoutOutput` - Verify stdout/stderr separation for add command

**3. Command Aliases:**
- `TestE2E_WorktreeSwitch_Aliases` - Test `sw` and `s` aliases for switch command
- `TestE2E_WorktreeAdd_Aliases` - Test `a` alias for add command

**4. Error Handling:**
- `TestE2E_WorktreeSwitch_NonExistent` - Verify error when switching to non-existent worktree

**5. TUI Testing:**
- `TestE2E_WorktreeList` - Skipped (TUI requires interactive terminal)

**Key Features:**

**Setup Process:**
1. Creates temporary directory for test repository
2. Runs `gbm init` to create bare repo structure
3. Creates initial commit in main worktree
4. Provides clean test environment with automatic cleanup

**Stdout/Stderr Validation:**
- Verifies universal pattern: data → stdout, messages → stderr
- Confirms stdout contains ONLY the path (single line)
- Confirms stderr contains success messages but not paths
- Tests both `worktree add` and `worktree switch` commands

**Alias Testing:**
- Verifies `wt sw`, `wt s` (switch aliases)
- Verifies `wt a` (add alias)
- All aliases follow same stdout/stderr pattern

**Test Results:**
```
=== E2E Test Summary ===
✅ TestE2E_WorktreeAdd_CLI
✅ TestE2E_WorktreeAdd_ExistingBranch
⏭️  TestE2E_WorktreeList (skipped - TUI)
✅ TestE2E_WorktreeSwitch_StdoutOutput
✅ TestE2E_WorktreeAdd_StdoutOutput
✅ TestE2E_WorktreeSwitch_Aliases
✅ TestE2E_WorktreeAdd_Aliases
✅ TestE2E_WorktreeSwitch_NonExistent
✅ TestE2E_Init_CreatesStructure

8 passed, 1 skipped
```

**Benefits:**
- **Catch regressions**: Tests verify real command behavior, not just unit logic
- **Validate P1.1**: Confirms universal stdout/stderr pattern works end-to-end
- **Shell integration ready**: Tests prove commands work with stdout capture
- **Alias coverage**: Ensures all command shortcuts work correctly
- **Error handling**: Verifies appropriate errors for invalid operations

**Code Quality:**
- All lint checks pass (errcheck violations fixed)
- Proper error handling throughout
- Clean test isolation with automatic cleanup
- Clear test names and assertions

**Files Created:**
- `e2e_test.go` (357 lines) - Comprehensive E2E test suite

**Validation:** ✅ All E2E tests pass, full validation pipeline successful

**Post-Implementation Enhancement:**
- Refactored all E2E tests to use `testify/assert` and `testify/require`
- **`require`** for critical operations (setup, command execution)
- **`assert`** for validation checks (multiple properties, want to see all failures)
- Replaced manual error checking with semantic assertions
- Added comprehensive testing patterns documentation to CLAUDE.md
- Standardized testing approach: clear distinction between critical and validation checks

---

### Task 1.2.3: Add E2E test for shell integration
**Completed:** 2026-01-02
**File:** `e2e_test.go` (470 lines total)

**What Was Done:**
1. Added comprehensive shell integration E2E tests
2. Validated that stdout/stderr pattern works for shell integration use cases
3. Tested exit code propagation for success and failure cases
4. Verified output format matches shell wrapper expectations
5. Tested all command forms and aliases comprehensively
6. Tested error handling with empty stdout on failures

**Test Cases Added:**

**1. Shell Integration Command:**
- `TestE2E_ShellIntegration_Command` - Verifies `gbm shell-integration` outputs correct script
  - Validates script contains gbm2() function definition
  - Confirms all command forms (worktree/wt, switch/sw/s, add/a, list/ls/l)
  - Ensures script includes cd logic for shell integration

**2. Exit Code Testing:**
- `TestE2E_ShellIntegration_ExitCodes` - Validates exit code propagation
  - Success case returns exit code 0
  - Failure case returns non-zero exit code
  - Shell wrapper can use exit codes for conditional cd

**3. Output Format Validation:**
- `TestE2E_ShellIntegration_OutputFormat` - Verifies exact output format
  - Stdout is exactly one line (the path)
  - Path is absolute and points to existing directory
  - Stderr contains messages but not the path
  - Format matches what shell wrapper expects for parsing

**4. All Command Forms:**
- `TestE2E_ShellIntegration_AllCommands` - Table-driven test for all aliases
  - `worktree switch <name>`
  - `wt switch <name>`
  - `wt sw <name>`
  - `wt s <name>`
  - All produce single-line stdout with path

**5. Add Command Integration:**
- `TestE2E_ShellIntegration_AddCommand` - Validates worktree add for shell integration
  - Stdout contains new worktree path (single line)
  - Stderr has success message without path
  - Shell integration can cd to newly created worktree
  - Worktree actually exists on filesystem

**6. Error Message Handling:**
- `TestE2E_ShellIntegration_ErrorMessages` - Validates error behavior
  - Stdout is empty on error (prevents shell from cd'ing)
  - Stderr contains error message
  - Non-zero exit code returned

**7. Both Command Forms:**
- `TestE2E_ShellIntegration_BothCommandForms` - Validates `worktree` and `wt` consistency
  - Both forms produce identical output format
  - Both work with shell integration wrapper

**Test Results:**
```
=== Shell Integration Test Summary ===
✅ TestE2E_ShellIntegration_Command
✅ TestE2E_ShellIntegration_ExitCodes
✅ TestE2E_ShellIntegration_OutputFormat
✅ TestE2E_ShellIntegration_AllCommands (4 sub-tests)
✅ TestE2E_ShellIntegration_AddCommand
✅ TestE2E_ShellIntegration_ErrorMessages
✅ TestE2E_ShellIntegration_BothCommandForms (2 sub-tests)

7 test functions, 6 subtests, all passing
Total E2E tests: 16 passed, 1 skipped (TUI)
```

**What Was Validated:**
- ✅ Shell integration script generation works
- ✅ Exit codes propagate correctly for conditional cd
- ✅ Stdout format is exactly what shell wrapper expects (single line, absolute path)
- ✅ Stderr never contains paths (only messages)
- ✅ All command forms and aliases work identically
- ✅ Add command outputs path for auto-cd after creation
- ✅ Errors produce empty stdout (shell won't cd on failure)
- ✅ Both `worktree` and `wt` command forms consistent

**Benefits:**
- **Prevents regressions**: Shell integration is fully tested end-to-end
- **Validates universal pattern**: Confirms stdout/stderr separation works for real use cases
- **Exit code safety**: Ensures shell wrapper can rely on exit codes
- **Format stability**: Tests lock in the exact format shell script expects
- **Comprehensive coverage**: All aliases, command forms, and error cases tested

**Code Quality:**
- All tests use testify assert/require pattern
- Clear test names describe what's being validated
- Proper use of table-driven tests for command variants
- Each test validates one specific aspect of shell integration

**Files Modified:**
- `e2e_test.go` (added 7 new test functions, ~190 lines)

**Validation:** ✅ All tests pass (16 E2E tests), full validation pipeline successful

---

### Task 1.3.1: Create flag override helper utilities
**Completed:** 2026-01-02
**Files:** `internal/utils/flags.go`, `internal/utils/flags_test.go`

**What Was Done:**
1. Created reusable flag override helper functions
2. Implemented type-specific helpers for string, bool, and int flags
3. Added comprehensive unit tests with edge case coverage
4. Documented the pattern with clear examples

**Helper Functions Created:**
```go
// Check if flag was explicitly set, otherwise use config value
GetStringFlagOrConfig(cmd *cobra.Command, flagName string, configValue string) string
GetBoolFlagOrConfig(cmd *cobra.Command, flagName string, configValue bool) bool
GetIntFlagOrConfig(cmd *cobra.Command, flagName string, configValue int) int
```

**How It Works:**
- Uses `cmd.Flags().Changed()` to detect if flag was explicitly set
- Returns flag value if set (even if it's empty/zero)
- Returns config value if flag not set
- Enables clear precedence: flags > config > defaults

**Test Coverage:**
Comprehensive tests validating all scenarios:
- Flag explicitly set (should use flag value)
- Flag not set (should use config value)
- Edge cases: empty strings, zero values, negative numbers
- All tests pass with testify assert pattern

**Benefits:**
- **Clear precedence**: Users can override config without editing files
- **One-off overrides**: Perfect for experimentation
- **Type-safe**: Separate functions for each type prevent errors
- **Well-documented**: Examples and usage patterns included
- **Reusable**: Can be used across all commands

**Files Created:**
- `internal/utils/flags.go` (52 lines) - Helper functions
- `internal/utils/flags_test.go` (190 lines) - Comprehensive test suite

**Validation:** ✅ All tests pass, linting clean, ready for use

---

### Task 1.3.2: Apply flag override to worktree add --base
**Completed:** 2026-01-02
**File:** `cmd/service/worktree.go` (lines 94-98)

**What Was Done:**
1. Applied flag override pattern to `--base` flag in worktree add command
2. Replaced `cmp.Or()` logic with `GetStringFlagOrConfig()`
3. Removed unused `cmp` import
4. Maintained backward compatibility with existing behavior

**Code Change:**
```go
// OLD: Using cmp.Or
baseBranch = cmp.Or(baseBranch, svc.GetConfig().DefaultBranch, "master")

// NEW: Flag override pattern
baseBranch = utils.GetStringFlagOrConfig(cmd, "base", svc.GetConfig().DefaultBranch)
if baseBranch == "" {
    baseBranch = "master" // Ultimate fallback
}
```

**Precedence Chain:**
1. `--base` flag if explicitly set
2. `config.DefaultBranch` from `.gbm/config.yaml`
3. `"master"` as ultimate fallback

**User Experience:**
```bash
# Uses config value from .gbm/config.yaml (e.g., "main")
$ gbm wt add feature-x feature/x -b

# Overrides config, uses "develop" instead
$ gbm wt add feature-x feature/x -b --base develop

# One-off override to "master" without changing config
$ gbm wt add hotfix hotfix/urgent -b --base master
```

**Benefits:**
- **Flexible**: Users can override config per command
- **Backward compatible**: Existing usage patterns still work
- **Cleaner code**: Removed `cmp` dependency
- **Clear intent**: Code explicitly shows precedence order

**Validation:** ✅ All existing tests pass, E2E tests validate behavior

---

### Task 1.3.3: Document flag override pattern
**Completed:** 2026-01-02
**File:** `CLAUDE.md` (new "Flag Override Pattern" section)

**What Was Done:**
1. Added comprehensive "Flag Override Pattern" section to CLAUDE.md
2. Documented the three-level precedence: flags > config > defaults
3. Provided clear implementation examples for all supported types
4. Explained how the pattern works internally
5. Added usage guidelines for when to use (and not use) the pattern

**Content Added:**

**Priority Order:**
1. Explicit flags (highest)
2. Config file (middle)
3. Defaults (fallback)

**Implementation Examples:**
```go
// String flags
baseBranch := utils.GetStringFlagOrConfig(cmd, "base", config.DefaultBranch)

// Boolean flags
dryRun := utils.GetBoolFlagOrConfig(cmd, "dry-run", config.DryRun)

// Integer flags
timeout := utils.GetIntFlagOrConfig(cmd, "timeout", config.Timeout)
```

**Key Explanations:**
- How `cmd.Flags().Changed()` enables detection of explicit flags
- Why this is different from just reading flag values
- Real-world usage examples
- Guidelines for when to use the pattern

**When to Use:**
- Command has corresponding config setting
- Users might want per-command overrides
- Logical fallback chain exists

**When NOT to Use:**
- Flags without config equivalents
- Required flags
- Flags always specified by user

**Benefits:**
- **Discoverable**: Easy for developers to find and understand
- **Consistent**: All future commands will follow same pattern
- **Clear guidelines**: Prevents misuse or over-application
- **Examples**: Real code developers can copy

**Validation:** ✅ Documentation added, examples clear, pattern well-explained

---

## 🔑 Key Patterns & Decisions

### Universal Stdout/Stderr Pattern
**The Rule:** All commands output data to stdout, messages to stderr.

**Why:**
- Enables shell integration without environment variables
- Follows Unix philosophy
- Scriptable and pipeable by default
- Consistent across all commands

**Example:**
```bash
# Path goes to stdout, message to stderr
$ gbm wt switch feature-x
/path/to/worktrees/feature-x           # stdout
✓ Switched to worktree 'feature-x'     # stderr

# Capture path only
$ path=$(gbm wt switch feature-x)

# Suppress messages
$ gbm wt switch feature-x 2>/dev/null
/path/to/worktrees/feature-x
```

### Removed Redundant Flags
**Decision:** Remove flags that duplicate what stdout/stderr redirection provides.

**Example:** `--print-path` was removed because `2>/dev/null` achieves the same result.

### /dev/tty for TUI
**Decision:** TUI commands render to `/dev/tty` instead of using temp files.

**Trade-off:** Requires interactive terminal (acceptable for TUI use case).

### Eliminated Temp Files Completely
**Decision:** No temp files for any worktree operations.

**Why:** Universal stdout/stderr pattern + /dev/tty for TUI = no need for temp files.

**Benefits:**
- Simpler code (~60% reduction in shell wrapper)
- No cleanup logic needed
- No race conditions or PID conflicts
- Works consistently across all shells

### Testify Assert/Require for Testing
**Decision:** Use `github.com/stretchr/testify` for all test assertions. Choose `assert` vs `require` based on failure impact.

**Pattern:**
```go
// require for critical setup - stop if this fails
require.NoError(t, err, "setup must succeed")

// assert for validation - collect all failures
assert.Contains(t, output, "expected", "should contain text")
assert.DirExists(t, path, "directory should exist")
```

**When to use each:**
- **`require`** (fail-fast): Setup operations, prerequisites, single critical check
- **`assert`** (continue): Multiple independent validations, property checks

**Why:**
- **Cleaner code**: No manual if/err checks
- **Better errors**: Automatic diffs, clear failure messages
- **Smart failing**: `require` stops wasted execution, `assert` collects all issues
- **Clear intent**: Code signals what's critical vs validation
- **Industry standard**: Widely used in Go ecosystem

**Benefits:**
- Faster test development and debugging
- See all failures at once (with `assert`)
- Stop early when critical steps fail (with `require`)
- Consistent testing style across codebase
- Self-documenting test intent

### Flag Override Pattern
**The Rule:** Use `cmd.Flags().Changed()` to detect explicit flags and provide clear precedence: flags > config > defaults.

**Why:**
- Enables per-command config overrides without editing files
- Clear and explicit precedence order
- Perfect for experimentation and one-off operations
- Type-safe with dedicated helper functions

**Example:**
```go
// Flag override pattern with helpers
baseBranch := utils.GetStringFlagOrConfig(cmd, "base", config.DefaultBranch)
if baseBranch == "" {
    baseBranch = "master" // Ultimate fallback
}
```

**Precedence Chain:**
```
Command: gbm wt add feat-x feat/x -b --base develop

1. Check --base flag (explicitly set?) → YES → use "develop" ✓
2. Check config.DefaultBranch            → skipped
3. Use "master" default                  → skipped
```

**When to Use:**
- Command has corresponding config setting
- Users might want per-command overrides
- Logical fallback chain exists (flag > config > default)

**Helper Functions:**
- `GetStringFlagOrConfig()` - For string flags
- `GetBoolFlagOrConfig()` - For boolean flags
- `GetIntFlagOrConfig()` - For integer flags

**Benefits:**
- Users can override config without editing files
- One-off experiments don't require config changes
- Clear code intent - precedence order is explicit
- Type-safe - separate functions for each type
- Reusable across all commands

---

## ✅ Phase 2 Completed Work

### Task 2.1.1: Create path template engine
**Completed:** 2026-01-02
**File:** `internal/utils/template.go` (68 lines), `internal/utils/template_test.go` (268 lines)

**What Was Done:**
1. Created `ExpandTemplate()` function to replace template variables in paths
2. Created `GetTemplateVars()` to extract available template variables
3. Created `ExpandPath()` to handle ~ expansion and relative path resolution
4. Added comprehensive test coverage for all template operations

**Template Variables Supported:**
- `{gitroot}` - Repository directory name (e.g., "gbm")
- `{branch}` - Branch name (context-specific)
- `{issue}` - JIRA issue key (context-specific)

**Examples:**
```go
// Template expansion
ExpandTemplate("../{gitroot}-worktrees", {"gitroot": "gbm"})
// Returns: "../gbm-worktrees"

// Path expansion with ~ and relative paths
ExpandPath("~/dev/{gitroot}/branches", "/path/to/repo")
// Returns: "/home/user/dev/gbm/branches"
```

**Features:**
- Variable substitution with flexible naming
- Home directory expansion (~)
- Relative path resolution from repo root
- Path cleaning (removes double slashes, trailing slashes)
- Handles missing variables gracefully (leaves them as-is)

**Test Coverage:**
- 30+ test cases covering all scenarios
- Edge cases: empty values, special characters, case sensitivity
- Integration test combining template + path expansion
- All tests passing ✅

**Benefits:**
- Users can use dynamic paths in config: `../{gitroot}-worktrees`
- Share configs across multiple repos with automatic expansion
- Clean separation: template → path → filesystem
- Type-safe implementation

**Validation:** ✅ All template tests pass, linting clean, compiles successfully

---

### Task 2.1.2: Apply template expansion to worktrees_dir
**Completed:** 2026-01-02
**File:** `cmd/service/service.go` (GetWorktreesPath method)

**What Was Done:**
1. Added import of `internal/utils` package
2. Updated `GetWorktreesPath()` to use template expansion
3. Applied three-step process:
   - Get template variables from repo root
   - Expand template variables in config path
   - Expand ~ and resolve relative paths

**Code Change:**
```go
// OLD: Simple path joining
return filepath.Join(s.RepoRoot, s.WorktreeDir), nil

// NEW: Template-aware path expansion
vars := utils.GetTemplateVars(s.RepoRoot)
expandedDir := utils.ExpandTemplate(s.WorktreeDir, vars)
expandedDir = utils.ExpandPath(expandedDir, s.RepoRoot)
return expandedDir, nil
```

**Configuration Examples:**
```yaml
# Static path (backward compatible)
worktrees_dir: worktrees
# Results in: /path/to/repo/worktrees

# Template-based path (new feature)
worktrees_dir: ../{gitroot}-worktrees
# gbm repo results in: /path/to/gbm-worktrees

# Home-based path (new feature)
worktrees_dir: ~/dev/{gitroot}/worktrees
# gbm repo results in: /home/user/dev/gbm/worktrees
```

**Backward Compatibility:**
- Existing configs with static paths work unchanged
- Templates are optional - can be used gradually

**E2E Tests Added:**
1. `TestE2E_TemplateVariableExpansion` - Basic template expansion with default config
2. `TestE2E_TemplateVariableExpansion_CustomPath` - Custom template path with parent directory worktrees

Both tests verify:
- Worktree creation succeeds with template-expanded paths
- Output contains the expanded path
- Files created in the correct template-expanded location

**Benefits:**
- Share one config across multiple repos
- Dynamic organization based on repo name
- Cleaner project structure possible
- Optional feature - no breaking changes

**Validation:** ✅ All E2E tests pass, full validation successful, all 18 E2E tests passing

---

## 🚀 How to Continue

### For Next Agent/Session:

1. **Read this file** to see what's already complete
2. **Read [improvement-prd.md](./improvement-prd.md)** for available tasks
3. **Choose the next logical task** based on dependencies and sequence
4. **Implement the task** following the established patterns above
5. **Update this file** when complete with:
   - Add entry to "Completed Work" section
   - Update status overview table
   - Document any new patterns or decisions
   - Update "Last Updated" date

### Validation Commands:
```bash
just validate      # Full validation pipeline
just test-changed  # Test only changed packages
just show-changed  # See what changed
```

### Reference Files:
- **PRD:** [improvement-prd.md](./improvement-prd.md) - All task specifications
- **Analysis:** [git-wt-analysis.md](./git-wt-analysis.md) - Pattern reference
- **Shell Summary:** [shell-integration-summary.md](./shell-integration-summary.md)

---

## 📝 Notes

### Task Dependencies:
- Task 1.1.2 (Shell Wrapper) should wait for 1.1.3 and 1.1.4 to be complete
- Task 1.1.5 (Documentation) should be done after other tasks for complete examples

### Testing Notes:
- All changes compile and pass existing tests
- E2E testing infrastructure complete (P1.2)
- Flag override helpers fully tested

---

**Last Updated:** 2026-01-02 - Completed P2.1 Tasks 2.1.1 & 2.1.2 (Template Variables) - **P2.1 COMPLETE ✅** - **🎉 PHASE 2.1 COMPLETE 🎉**

# GBM Improvements PRD
## Based on git-wt Analysis

**Document Version:** 1.0
**Date:** 2026-01-02
**Status:** Draft for Review

---

## Executive Summary

This PRD outlines incremental improvements to `gbm` (Git Branch Manager) based on patterns observed in the `git-wt` tool. The focus is on improving developer ergonomics, code organization, and testing infrastructure while preserving gbm's core strengths (JIRA integration, TUI, existing features).

### Key Principles
- **Incremental**: Each improvement is a standalone task
- **Non-breaking**: Preserve existing functionality and commands
- **Ergonomic**: Focus on developer experience improvements
- **Tested**: Add E2E testing to validate real-world usage

---

## Requirements Coverage

This table shows how your specific requirements are addressed:

| Requirement | Status | Tasks |
|-------------|--------|-------|
| **Shell Integration** | | |
| Support `gbm2 wt switch` and aliases (`sw`, `s`) | ✅ Enhanced | 1.1.1, 1.1.2 |
| Support `gbm2 worktree switch` and aliases | ✅ Enhanced | 1.1.1, 1.1.2 |
| Support `gbm2 wt list` TUI with auto-cd | ✅ Enhanced | 1.1.2, 1.1.4 |
| Support `gbm2 wt ls` / `gbm2 wt l` aliases | ✅ Enhanced | 1.1.2, 1.1.4 |
| **NEW: Auto-cd after worktree creation** | ✨ New Feature | 1.1.2, 1.1.3 |
| **NEW: /dev/tty rendering for TUI** | ✨ New Feature | 1.1.4 |
| **REMOVED: Temp file complexity** | 🗑️ Deleted | 1.1.2, 1.1.4 |
| **File Copying** | | |
| Current file copy config (rules-based) | ✅ Preserved | No changes |
| Optional: Auto-copy ignored files | 🔄 Optional | 2.2.1 - 2.2.3 |
| **Commands & Aliases** | | |
| All current subcommands | ✅ Preserved | No changes |
| **Configuration** | | |
| `.gbm/config.yaml` for GBM config | ✅ Preserved | No changes |
| Template variables in paths | ✨ New Feature | 2.1.1, 2.1.2 |
| Flag override pattern | ✨ New Feature | 1.3.1 - 1.3.3 |

**Legend:**
- ✅ Enhanced: Existing feature improved
- ✅ Preserved: Existing feature maintained as-is
- ✨ New Feature: Net new capability
- 🗑️ Deleted: Removed/simplified
- 🔄 Optional: Can be skipped if not needed

---

## Technical Highlight: /dev/tty for TUI

**The Problem:**
- TUI (Bubble Tea) needs stdout for rendering the interface
- Shell integration needs stdout to capture the worktree path
- Can't capture stdout while TUI is using it

**The Solution:**
- **TUI renders to `/dev/tty`** (the controlling terminal) instead of stdout
- This leaves **stdout clean** for the worktree path
- Shell wrapper uses **single unified approach**: `result=$(gbm2 ... 2>/dev/stderr)`

**The Result:**
- 🎯 **No temp files** - eliminated entirely!
- 🎯 **Single code path** - all commands use stdout
- 🎯 **Simpler maintenance** - less code, less complexity
- 🎯 **Works in all interactive shells** - bash, zsh, fish

---

## Priority 1: Core Improvements

### P1.1: Stdout/Stderr Separation (Universal Pattern)

**Current State:**
- Mixed output: both data and messages go to stdout
- Shell integration uses temp files (`$TMPDIR/.gbm-switch-$$`)
- Inconsistent - some commands output different formats
- TUI can't use stdout because it needs it for rendering

**Proposed Change:**
- **Universal pattern for ALL commands**: Machine-readable output → stdout, user messages → stderr
- Use `/dev/tty` for TUI rendering, leaving stdout available for path output
- Single unified shell wrapper for all commands
- **Eliminate temp files completely**

**Benefits:**
- **Follows Unix philosophy**: stdout for data, stderr for logs
- **Better for piping**: `wt_path=$(gbm wt switch foo)` works cleanly
- **Better for scripting**: Capture data without parsing human messages
- **Consistent**: All commands follow same pattern
- **Simpler shell integration**: No special cases, just capture stdout
- **No temp files**: Eliminated entirely

**Implementation Tasks:**

#### Task 1.1.1: Update worktree switch command output
**Complexity:** Low
**Estimate:** 2-3 hours

**Description:**
Modify `newWorktreeSwitchCommand()` to follow universal stdout/stderr pattern.

**Files to modify:**
- `cmd/service/worktree.go` (lines 494-633)

**Current behavior (line 594):**
```go
if os.Getenv("GBM_SHELL_INTEGRATION") != "" {
    fmt.Printf("cd %s\n", targetWorktree.Path)
} else {
    fmt.Printf("To switch to worktree '%s':\n", worktreeName)
    fmt.Printf("  cd %s\n\n", targetWorktree.Path)
    // ... more instructions
}
```

**New behavior (universal pattern):**
```go
// Always output path to stdout (machine-readable)
fmt.Println(targetWorktree.Path)

// Always output messages to stderr (human-readable)
fmt.Fprintf(os.Stderr, "✓ Switched to worktree '%s'\n", worktreeName)
```

**For non-shell-integration users:**
They can still see the path (it prints to stdout), and the success message on stderr. Both are visible in terminal but stdout is capturable.

**Acceptance Criteria:**
- ✅ **Always** writes path to stdout (no env var check)
- ✅ **Always** writes messages to stderr
- ✅ Works with shell integration: `result=$(gbm wt switch foo)`
- ✅ Works without shell integration: both visible in terminal
- ✅ Can be piped: `gbm wt switch foo > /tmp/path`
- ✅ Consistent with all other commands

---

#### Task 1.1.2: Simplify shell integration wrapper
**Complexity:** Low
**Estimate:** 2 hours

**Description:**
Update shell wrapper to use **unified stdout approach** for all worktree commands. Single, clean pattern - no temp files, no special cases.

**Files to modify:**
- `cmd/service/shell-integration.go` (lines 36-97)

**Current behavior to preserve:**
1. ✅ `gbm2 wt switch <name>` - and aliases `sw`, `s`
2. ✅ `gbm2 worktree switch <name>` - and aliases `sw`, `s`
3. ✅ `gbm2 wt list` - TUI mode with selection → auto-cd
4. ✅ `gbm2 wt ls` / `gbm2 wt l` - TUI mode aliases
5. ✅ `gbm2 worktree list` - TUI mode

**New behavior to add:**
6. ✨ `gbm2 wt add <name> <branch>` - auto-cd after creation

**Changes to remove:**
7. 🗑️ All temp file logic - delete completely

**Implementation:**

```bash
const shellIntegrationScript = `# gbm shell integration

gbm2() {
    # All worktree commands that output a path to stdout
    # Handles: switch, sw, s, add, a, list, ls, l (and their worktree/wt forms)
    if [[ ("$1" = "worktree" || "$1" = "wt") && \
          ("$2" = "switch" || "$2" = "sw" || "$2" = "s" || \
           "$2" = "add" || "$2" = "a" || \
           "$2" = "list" || "$2" = "ls" || "$2" = "l") ]]; then

        # Capture stdout (path) while letting stderr through for messages
        local result
        result=$(command gbm2 "$@" 2>/dev/stderr)
        local exit_code=$?

        # If successful and result is a directory, cd to it
        if [ $exit_code -eq 0 ] && [ -n "$result" ] && [ -d "$result" ]; then
            cd "$result"
        fi

        return $exit_code

    # All other commands - pass through unchanged
    else
        command gbm2 "$@"
    fi
}
`
```

**Key changes from current implementation:**

1. **Eliminated all temp file code:**
   - No `$TMPDIR/.gbm-switch-$$` creation
   - No temp file reading/cleanup
   - 60+ lines of shell code → 20 lines

2. **No environment variable needed:**
   - Removed `export GBM_SHELL_INTEGRATION=1`
   - Go code always uses stdout/stderr separation (universal pattern)
   - Shell wrapper just captures stdout

3. **Single unified approach:**
   - All switch aliases: `switch`, `sw`, `s`
   - All list aliases: `list`, `ls`, `l`
   - Add command: `add`, `a`
   - **Same code path for all commands**

4. **Much simpler:**
   - No fallback logic
   - No special cases
   - Clean stdout capture pattern

**Acceptance Criteria:**
- ✅ All switch aliases work: `switch`, `sw`, `s`
- ✅ All list aliases work: `list`, `ls`, `l`
- ✅ Add aliases work: `add`, `a`
- ✅ Both command forms work: `worktree` and `wt`
- ✅ Auto-cd works for all path-outputting commands
- ✅ Error messages visible to user (via stderr)
- ✅ Works in bash and zsh
- ✅ Exit codes preserved
- ✅ **No temp files created**

---

#### Task 1.1.3: Add stdout/stderr separation to worktree add
**Complexity:** Low
**Estimate:** 2 hours

**Description:**
Update worktree add to follow universal stdout/stderr pattern.

**Files to modify:**
- `cmd/service/worktree.go` (newWorktreeAddCommand, lines 73-136)

**Current code (lines 100-103):**
```go
if !dryRun {
    fmt.Printf("Created worktree '%s' at %s for branch '%s'\n", wt.Name, wt.Path, wt.Branch)
}
return nil
```

**New code (universal pattern):**
```go
if !dryRun {
    // Always output path to stdout (machine-readable)
    fmt.Println(wt.Path)

    // Always output messages to stderr (human-readable)
    fmt.Fprintf(os.Stderr, "✓ Created worktree '%s' for branch '%s'\n", wt.Name, wt.Branch)
}
return nil
```

**Why this is better:**
- **Scriptable**: `new_wt=$(gbm wt add feat-x feat-x -b)` always works
- **Pipeable**: `gbm wt add ... | xargs ls` works
- **Consistent**: Same pattern as switch, list, and all other commands
- **Simple**: No environment variable checks needed

**Also update in retry path (line 133):**
Same pattern when branch creation is confirmed after prompt.

**Acceptance Criteria:**
- ✅ **Always** outputs path to stdout
- ✅ **Always** outputs messages to stderr
- ✅ Works for both `-b` (create) and existing branch cases
- ✅ Works in retry path (when prompting to create branch)
- ✅ Return code indicates success/failure
- ✅ TUI mode unchanged (already handles separately)
- ✅ Auto-cd works via shell wrapper
- ✅ Can be captured: `path=$(gbm wt add ...)`

---

#### Task 1.1.4: Update TUI to use /dev/tty with stdout output
**Complexity:** Medium
**Estimate:** 2-3 hours

**Description:**
Update TUI to render to `/dev/tty` so stdout can be used for path output. Remove all temp file logic.

**Files to modify:**
- `cmd/service/worktree_table.go` (lines 400-433, and TUI initialization)
- Search for all references to `.gbm-switch` temp files and delete

**Current behavior:**
- TUI uses stdout/stdin for rendering
- Writes path to temp file at `$TMPDIR/.gbm-switch-{PPID}`
- Shell wrapper reads from temp file

**New behavior:**
- TUI renders to `/dev/tty`, outputs path to stdout
- **No temp files** - delete all temp file code
- Cleaner stdout output matches non-interactive commands

**Implementation:**

```go
func runWorktreeTable(worktrees []git.Worktree, trackedBranches map[string]bool,
                      currentWorktree *git.Worktree, svc *Service) error {

    model := newWorktreeTable(worktrees, trackedBranches, currentWorktree, svc)

    // Open /dev/tty for TUI rendering
    tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
    if err != nil {
        return fmt.Errorf("failed to open /dev/tty: %w", err)
    }
    defer tty.Close()

    // TUI renders to /dev/tty, leaving stdout clean
    program := tea.NewProgram(model,
        tea.WithInput(tty),
        tea.WithOutput(tty),
    )

    finalModel, err := program.Run()
    if err != nil {
        return err
    }

    // Output path to stdout if user selected a worktree (universal pattern)
    if m, ok := finalModel.(worktreeTableModel); ok {
        if m.switchOutput != "" {
            // Always output path to stdout (machine-readable)
            fmt.Println(m.switchOutput)

            // Always output message to stderr (human-readable)
            fmt.Fprintf(os.Stderr, "✓ Selected worktree: %s\n", filepath.Base(m.switchOutput))
        }
    }

    return nil
}
```

**Where `switchOutput` is set:**
Find where `model.switchOutput` is assigned and ensure it's just the path:

```go
// When user selects worktree in TUI Update() function:
// Ensure it's just the path, not "cd <path>"
case tea.KeyEnter:
    selected := m.table.SelectedRow()
    if len(selected) > 0 {
        worktreeName := strings.TrimPrefix(selected[0], "* ")
        // Find the worktree
        for _, wt := range m.worktrees {
            if wt.Name == worktreeName {
                m.switchOutput = wt.Path  // Just the path!
                return m, tea.Quit
            }
        }
    }
```

**Code to delete:**
Search and remove all code related to:
- `$TMPDIR/.gbm-switch-$$` temp file creation
- `os.Getppid()` for temp file naming
- `os.WriteFile()` for temp file writing
- Temp file cleanup logic

**Acceptance Criteria:**
- ✅ TUI renders to `/dev/tty`
- ✅ **Always** outputs path to stdout (no env var check)
- ✅ **Always** outputs messages to stderr
- ✅ Works with shell integration: captured for auto-cd
- ✅ Works without shell integration: both visible in terminal
- ✅ **No temp files created anywhere**
- ✅ No "cd " prefix in output (just the path)
- ✅ Works with updated shell wrapper (Task 1.1.2)
- ✅ All temp file code deleted
- ✅ Error if `/dev/tty` unavailable (acceptable for TUI)
- ✅ Consistent with switch and add commands

---

#### Task 1.1.5: Document universal stdout/stderr pattern for all commands
**Complexity:** Low
**Estimate:** 1 hour

**Description:**
Document the stdout/stderr pattern in CLAUDE.md so all future commands follow this convention.

**Files to modify:**
- `CLAUDE.md` - Add section on output patterns

**Content to add:**
```markdown
### Output Patterns

GBM follows strict stdout/stderr separation for all commands:

**Universal Rule:**
- **stdout**: Machine-readable data (paths, IDs, structured output)
- **stderr**: Human-readable messages (progress, errors, warnings)

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
- Shell integration: `path=$(gbm wt switch foo)`
- Piping: `gbm wt list | xargs ls`
- Scripting: Capture data without parsing messages
- Consistent: All commands work the same way

**Implementation Guidelines:**
```go
// CORRECT: Always separate data from messages
fmt.Println(data)                                    // stdout
fmt.Fprintf(os.Stderr, "✓ Operation successful\n")  // stderr

// WRONG: Don't mix them
fmt.Printf("Created worktree at %s\n", path)  // Mixed - hard to parse
```

**When implementing new commands:**
1. Identify the "data" (what users might want to capture)
2. Output data to stdout
3. Output all messages, progress, and errors to stderr
4. Never mix them in the same stream
```

**Acceptance Criteria:**
- ✅ Pattern documented in CLAUDE.md
- ✅ Examples provided
- ✅ Guidelines for new commands
- ✅ Rationale explained
- ✅ Easy to find for developers

---

### P1.2: E2E Testing Infrastructure

**Current State:**
- Unit tests exist
- No end-to-end tests that build the binary and test real workflows

**Proposed Change:**
- Add E2E test suite based on git-wt pattern
- Test real command execution, shell integration, worktree workflows

**Benefits:**
- Catch integration issues before release
- Test shell integration automatically
- Validate real-world usage patterns
- Confidence in refactoring

**Implementation Tasks:**

#### Task 1.2.1: Create testutil package
**Complexity:** Low
**Estimate:** 3-4 hours

**Description:**
Create reusable test utilities for E2E tests.

**Files to create:**
- `testutil/repo.go` - Test repository helpers
- `testutil/repo_test.go` - Test the test utilities

**Reference:** `/deps/git-wt/testutil/repo.go`

**Required helpers:**
```go
type TestRepo struct {
    t    testing.TB
    Root string
}

// NewTestRepo creates temp git repo with cleanup
func NewTestRepo(t testing.TB) *TestRepo

// Helper methods:
func (r *TestRepo) Git(args ...string) string
func (r *TestRepo) GitE(args ...string) (string, error)
func (r *TestRepo) CreateFile(path, content string)
func (r *TestRepo) Commit(message string)
func (r *TestRepo) Chdir() func()
func (r *TestRepo) Path(relPath string) string
```

**Acceptance Criteria:**
- Can create temporary git repos for testing
- Automatic cleanup via t.Cleanup()
- Helper methods for common git operations
- Well-documented with examples

---

#### Task 1.2.2: Add E2E test for worktree creation
**Complexity:** Medium
**Estimate:** 4-5 hours

**Description:**
First E2E test: create worktree and validate it exists.

**Files to create:**
- `e2e_test.go` - E2E test suite (root level)

**Test cases:**
1. `TestE2E_WorktreeAdd_CLI` - Create worktree via CLI
2. `TestE2E_WorktreeList` - List worktrees
3. `TestE2E_WorktreeSwitch_StdoutOutput` - Verify stdout/stderr separation

**Reference:** `/deps/git-wt/e2e_test.go`

**Example structure:**
```go
func TestE2E_WorktreeAdd_CLI(t *testing.T) {
    binPath := buildBinary(t)
    repo := testutil.NewTestRepo(t)

    // Initialize gbm repo
    runGBM(t, binPath, repo.Root, "init")

    // Create initial commit
    repo.CreateFile("README.md", "# Test")
    repo.Commit("initial")

    // Create worktree
    out, err := runGBM(t, binPath, repo.Root, "worktree", "add", "feature-x", "feature-x", "-b")
    if err != nil {
        t.Fatalf("failed: %v\noutput: %s", err, out)
    }

    // Verify worktree exists
    wtPath := filepath.Join(repo.Root, "worktrees", "feature-x")
    if _, err := os.Stat(wtPath); os.IsNotExist(err) {
        t.Errorf("worktree not created at %s", wtPath)
    }
}
```

**Acceptance Criteria:**
- Binary builds before each test
- Tests create real git repos
- Worktree creation validated
- Clean error messages on failure

---

#### Task 1.2.3: Add E2E test for shell integration
**Complexity:** Medium
**Estimate:** 3-4 hours

**Description:**
Test that stdout/stderr separation works correctly for shell integration.

**Files to modify:**
- `e2e_test.go` - Add shell integration tests

**Test cases:**
1. `TestE2E_WorktreeSwitch_Stdout` - Path only on stdout
2. `TestE2E_WorktreeSwitch_Stderr` - Messages on stderr
3. `TestE2E_WorktreeSwitch_ExitCode` - Correct exit codes

**Example:**
```go
func TestE2E_WorktreeSwitch_Stdout(t *testing.T) {
    binPath := buildBinary(t)
    repo := setupGBMRepo(t)

    // Create worktree
    createWorktree(t, binPath, repo, "feature-x")

    // Test stdout capture
    stdout, stderr, err := runGBMStdout(t, binPath, repo.Root,
        "worktree", "switch", "feature-x")

    if err != nil {
        t.Fatalf("failed: %v", err)
    }

    // Stdout should contain only the path
    expectedPath := filepath.Join(repo.Root, "worktrees", "feature-x")
    if !strings.Contains(stdout, expectedPath) {
        t.Errorf("stdout should contain path %q, got: %q", expectedPath, stdout)
    }

    // Stderr may contain messages but not the path
    if strings.Contains(stderr, expectedPath) {
        t.Errorf("stderr should not contain path, got: %q", stderr)
    }
}
```

**Acceptance Criteria:**
- Can capture stdout and stderr separately
- Path appears only in stdout
- Messages appear in stderr
- Exit codes tested

---

### P1.3: Flag Override Pattern

**Current State:**
- Flags directly set values
- No distinction between "flag not set" and "flag set to default value"
- Can't override config with flags cleanly

**Proposed Change:**
- Use `cmd.Flags().Changed()` to detect explicit flag usage
- Load config → apply flag overrides → use merged values
- Clear precedence: flags > config > defaults

**Benefits:**
- One-off overrides without changing config
- Better for experimentation
- Clear configuration precedence

**Implementation Tasks:**

#### Task 1.3.1: Add flag override helper
**Complexity:** Low
**Estimate:** 2 hours

**Description:**
Create helper function for flag override pattern.

**Files to create:**
- `internal/utils/flags.go` - Flag utilities

**Code:**
```go
package utils

import "github.com/spf13/cobra"

// GetFlagOrConfig returns the flag value if set, otherwise the config value
func GetFlagOrConfig[T any](cmd *cobra.Command, flagName string, configValue T) T {
    if cmd.Flags().Changed(flagName) {
        val, _ := cmd.Flags().GetString(flagName)
        // Type-specific conversion...
        return convertedVal
    }
    return configValue
}

// Specialized versions for common types
func GetStringFlagOrConfig(cmd *cobra.Command, flagName string, configValue string) string
func GetBoolFlagOrConfig(cmd *cobra.Command, flagName string, configValue bool) bool
func GetIntFlagOrConfig(cmd *cobra.Command, flagName string, configValue int) int
```

**Acceptance Criteria:**
- Type-safe flag override helpers
- Works with all common flag types
- Well-documented with examples
- Unit tests for edge cases

---

#### Task 1.3.2: Apply flag override to worktree add --base
**Complexity:** Low
**Estimate:** 1-2 hours

**Description:**
Update `--base` flag to use override pattern.

**Files to modify:**
- `cmd/service/worktree.go` (newWorktreeAddCommand, line 95)

**Current code:**
```go
baseBranch = cmp.Or(baseBranch, svc.GetConfig().DefaultBranch, "master")
```

**New code:**
```go
baseBranch := utils.GetStringFlagOrConfig(cmd, "base", svc.GetConfig().DefaultBranch)
if baseBranch == "" {
    baseBranch = "master"  // Ultimate fallback
}
```

**Acceptance Criteria:**
- `--base` flag overrides config when set
- Config value used when flag not set
- Fallback to "master" when neither set
- Existing tests pass

---

#### Task 1.3.3: Document flag override pattern
**Complexity:** Low
**Estimate:** 1 hour

**Description:**
Add documentation for flag override pattern to CLAUDE.md.

**Files to modify:**
- `CLAUDE.md` - Add section on flag handling

**Content:**
```markdown
### Flag Override Pattern

GBM uses a flag override pattern to provide clear configuration precedence:

1. **Explicit flags** - Highest priority
2. **Config file** - Middle priority (`.gbm/config.yaml`)
3. **Defaults** - Fallback

Example:
```go
// Check if flag was explicitly set
if cmd.Flags().Changed("basedir") {
    cfg.BaseDir = basedirFlag  // Use flag value
} else {
    cfg.BaseDir = config.WorktreesDir  // Use config value
}
```

This enables one-off overrides without modifying config files.
```

**Acceptance Criteria:**
- Pattern documented in CLAUDE.md
- Examples provided
- Clear precedence explained

---

## Priority 2: Ergonomic Improvements

### P2.1: Template Variables for Paths

**Current State:**
- Hardcoded paths in config
- `worktrees_dir: "worktrees"`

**Proposed Change:**
- Support template variables in path config
- Variables: `{gitroot}`, `{branch}`, `{issue}`

**Benefits:**
- Dynamic paths based on repo name
- Share config across multiple repos
- Cleaner organization for multiple repos

**Implementation Tasks:**

#### Task 2.1.1: Create path template engine
**Complexity:** Medium
**Estimate:** 3-4 hours

**Description:**
Add template expansion for path configuration.

**Files to create:**
- `internal/utils/template.go` - Template expansion

**Reference:** `/deps/git-wt/internal/git/config.go` (lines 144-176)

**Functions:**
```go
package utils

// ExpandTemplate replaces template variables in a path
func ExpandTemplate(path string, vars map[string]string) string

// GetTemplateVars returns available template variables
func GetTemplateVars(repoRoot string) map[string]string {
    return map[string]string{
        "gitroot": filepath.Base(repoRoot),
        // {branch} and {issue} set contextually
    }
}
```

**Supported variables:**
- `{gitroot}` - Repository directory name (e.g., `gbm`)
- `{branch}` - Current/target branch name (context-specific)
- `{issue}` - JIRA issue key (context-specific)

**Examples:**
```yaml
# In .gbm/config.yaml
worktrees_dir: "../{gitroot}-worktrees"
# gbm repo → "../gbm-worktrees"

worktrees_dir: "~/dev/{gitroot}/branches"
# gbm repo → "~/dev/gbm/branches"
```

**Acceptance Criteria:**
- Template expansion works for all variables
- Handles missing variables gracefully
- Unit tests for edge cases
- Documented with examples

---

#### Task 2.1.2: Apply template expansion to worktrees_dir
**Complexity:** Low
**Estimate:** 2 hours

**Description:**
Use template expansion when reading worktrees_dir config.

**Files to modify:**
- `cmd/service/service.go` (GetWorktreesPath, lines 189-195)

**Changes:**
```go
func (s *Service) GetWorktreesPath() (string, error) {
    if s.RepoRoot == "" {
        return "", ErrNotInGitRepository
    }

    // Expand template variables
    vars := utils.GetTemplateVars(s.RepoRoot)
    expandedDir := utils.ExpandTemplate(s.WorktreeDir, vars)

    // Expand ~ and resolve relative paths
    expandedDir = utils.ExpandPath(expandedDir, s.RepoRoot)

    return filepath.Join(s.RepoRoot, expandedDir), nil
}
```

**Acceptance Criteria:**
- `{gitroot}` expands correctly
- `~` expansion works
- Relative paths work
- Backwards compatible with plain paths
- E2E test validates expansion

---

### P2.2: Enhanced File Copying (Optional)

**Current State:**
- File copying uses explicit rules in config
- Specify source worktree and files to copy

**Proposed Enhancement:**
- Add option for automatic gitignore-based copying
- Copy ignored files (`.env`, etc.) automatically to new worktrees
- Use gitignore patterns to exclude certain files

**Benefits:**
- Less manual config needed
- Automatically copies .env and config files
- Flexible exclusions

**Implementation Tasks:**

#### Task 2.2.1: Add automatic file copy option to config
**Complexity:** Medium
**Estimate:** 4-5 hours

**Description:**
Add new config option for automatic file copying.

**Files to modify:**
- `cmd/service/service.go` (Config struct, line 62)

**New config structure:**
```go
type FileCopyConfig struct {
    Rules []FileCopyRule `yaml:"rules,omitempty"`  // Existing

    // New automatic copying options
    Auto AutoFileCopyConfig `yaml:"auto,omitempty"`
}

type AutoFileCopyConfig struct {
    Enabled        bool     `yaml:"enabled"`           // Enable automatic copying
    SourceWorktree string   `yaml:"source_worktree"`   // Where to copy from (default: "{default}")
    CopyIgnored    bool     `yaml:"copy_ignored"`      // Copy .gitignore'd files
    CopyUntracked  bool     `yaml:"copy_untracked"`    // Copy untracked files
    Exclude        []string `yaml:"exclude"`           // Patterns to exclude (gitignore syntax)
}
```

**Template variables for `source_worktree`:**
- `""` (empty/not set) - **Default**: Find worktree with branch = `DefaultBranch` from config
- `"{default}"` - Explicit: same as empty - use `DefaultBranch` worktree
- `"{current}"` - Use the worktree you're currently in
- Literal name - Use worktree with that specific name (e.g., `"main"`)

**Example config:**
```yaml
# In .gbm/config.yaml
default_branch: "main"

file_copy:
  auto:
    enabled: true
    source_worktree: "{default}"  # Uses worktree with "main" branch
    copy_ignored: true
    copy_untracked: false
    exclude:
      - "*.log"
      - "node_modules/"
      - ".DS_Store"
  rules:
    - source_worktree: main
      files:
        - .env
```

**Acceptance Criteria:**
- Config structure defined
- Backwards compatible (auto is optional)
- Documented in config example

---

#### Task 2.2.2: Implement gitignore pattern matcher
**Complexity:** Medium
**Estimate:** 4-5 hours

**Description:**
Add gitignore pattern matching using go-git library.

**Files to create:**
- `internal/git/filematcher.go` - Gitignore pattern matching

**Reference:** `/deps/git-wt/internal/git/copy.go`

**Dependencies:**
Add to `go.mod`:
```
github.com/go-git/go-git/v5 v5.x.x
```

**Functions:**
```go
package git

import "github.com/go-git/go-git/v5/plumbing/format/gitignore"

// ListIgnoredFiles returns files ignored by .gitignore
func (s *Service) ListIgnoredFiles(repoPath string) ([]string, error)

// ListUntrackedFiles returns untracked files (not ignored)
func (s *Service) ListUntrackedFiles(repoPath string) ([]string, error)

// MatchesPattern returns true if path matches gitignore pattern
func MatchesPattern(path string, patterns []string) bool
```

**Acceptance Criteria:**
- Can list ignored files via git commands
- Gitignore pattern matching works
- Unit tests for pattern matching
- Handles edge cases (nested patterns, etc.)

---

#### Task 2.2.3: Integrate automatic file copying
**Complexity:** Medium
**Estimate:** 3-4 hours

**Description:**
Use automatic file copying when enabled in config.

**Files to modify:**
- `cmd/service/service.go` (CopyFilesToWorktree, lines 352-383)

**Changes:**
```go
func (s *Service) CopyFilesToWorktree(targetWorktreeName string) error {
    config := s.GetConfig()

    // Phase 1: Automatic copying (if enabled)
    if config.FileCopy.Auto.Enabled {
        if err := s.autoCopyFiles(targetWorktreeName); err != nil {
            fmt.Fprintf(os.Stderr, "Warning: automatic file copy failed: %v\n", err)
        }
    }

    // Phase 2: Explicit rules (existing behavior)
    if len(config.FileCopy.Rules) > 0 {
        // ... existing rule-based copying
    }

    return nil
}

func (s *Service) resolveSourceWorktree(sourceSpec string) (*git.Worktree, error) {
    config := s.GetConfig()

    // Determine what worktree to use
    switch sourceSpec {
    case "", "{default}":
        // Find worktree associated with DefaultBranch
        worktrees, err := s.Git.ListWorktrees()
        if err != nil {
            return nil, err
        }

        defaultBranch := config.DefaultBranch
        if defaultBranch == "" {
            defaultBranch = "main"  // Ultimate fallback
        }

        for _, wt := range worktrees {
            if wt.Branch == defaultBranch {
                return &wt, nil
            }
        }

        // Fallback: use current worktree with warning
        fmt.Fprintf(os.Stderr, "Warning: No worktree found for default branch '%s', using current worktree\n", defaultBranch)
        return s.Git.GetCurrentWorktree()

    case "{current}":
        return s.Git.GetCurrentWorktree()

    default:
        // Literal worktree name
        worktrees, err := s.Git.ListWorktrees()
        if err != nil {
            return nil, err
        }

        for _, wt := range worktrees {
            if wt.Name == sourceSpec {
                return &wt, nil
            }
        }

        return nil, fmt.Errorf("worktree '%s' not found", sourceSpec)
    }
}

func (s *Service) autoCopyFiles(targetWorktreeName string) error {
    config := s.GetConfig()

    // Resolve source worktree using template expansion
    sourceWorktree, err := s.resolveSourceWorktree(config.FileCopy.Auto.SourceWorktree)
    if err != nil {
        return err
    }

    // List files based on config
    var files []string
    if config.FileCopy.Auto.CopyIgnored {
        ignored, _ := s.Git.ListIgnoredFiles(sourceWorktree.Path)
        files = append(files, ignored...)
    }
    if config.FileCopy.Auto.CopyUntracked {
        untracked, _ := s.Git.ListUntrackedFiles(sourceWorktree.Path)
        files = append(files, untracked...)
    }

    // Filter by exclude patterns
    filtered := filterFiles(files, config.FileCopy.Auto.Exclude)

    // Copy files
    targetPath := filepath.Join(s.RepoRoot, s.WorktreeDir, targetWorktreeName)
    for _, file := range filtered {
        s.copyFile(
            filepath.Join(sourceWorktree.Path, file),
            filepath.Join(targetPath, file),
        )
    }

    return nil
}
```

**Acceptance Criteria:**
- ✅ Automatic copying works when enabled
- ✅ `source_worktree` resolution works for all template values:
  - Empty/`{default}` → Uses worktree with `DefaultBranch`
  - `{current}` → Uses current worktree
  - Literal name → Uses named worktree
- ✅ Fallback to current worktree with warning if default branch worktree not found
- ✅ Respects exclude patterns
- ✅ Doesn't break existing rule-based copying
- ✅ E2E test validates behavior

---

## Priority 3: Code Organization

### P3.1: Refactor Git Service

**Current State:**
- `internal/git/service.go` contains multiple responsibilities
- Worktree, branch, init, clone operations all in service

**Proposed Change:**
- Split into focused files by domain
- Keep service.go for core git operations
- Separate files for worktree, branch, init, clone

**Benefits:**
- Easier to navigate
- Clearer responsibilities
- Follows git-wt pattern

**Implementation Tasks:**

#### Task 3.1.1: Review and document current git service organization
**Complexity:** Low
**Estimate:** 2 hours

**Description:**
Audit current internal/git package and plan refactoring.

**Deliverable:**
Document in `docs/git-service-refactor.md`:
- Current file structure
- Proposed file structure
- Migration plan
- Breaking changes (if any)

**Proposed structure:**
```
internal/git/
  service.go      # Core git operations, exec helpers
  worktree.go     # Worktree operations (already separate)
  branch.go       # Branch operations
  init.go         # Repository initialization (already separate)
  clone.go        # Repository cloning (already separate)
  filematcher.go  # File pattern matching (new)
  errors.go       # Git error types (already separate)
```

**Acceptance Criteria:**
- Current structure documented
- Proposed structure documented
- No breaking changes to public API
- Ready for incremental refactoring

---

#### Task 3.1.2: Extract branch operations (if needed)
**Complexity:** Low
**Estimate:** 2-3 hours

**Description:**
If branch operations are mixed into service.go, extract to branch.go.

**Files to review:**
- `internal/git/service.go`

**If needed, create:**
- `internal/git/branch.go` - Branch-specific operations

**Move functions:**
- `DeleteBranch()`
- `CreateBranch()` (if exists)
- `ListBranches()` (if exists)

**Acceptance Criteria:**
- Branch operations in dedicated file (if applicable)
- Tests still pass
- No duplicate code
- Public API unchanged

---

### P3.2: Improve Error Handling

**Current State:**
- Mix of error types
- Some git errors wrapped, some not

**Proposed Change:**
- Consistent error wrapping
- Better error messages with context
- Use git exit codes for typed errors

**Implementation Tasks:**

#### Task 3.2.1: Add typed git errors
**Complexity:** Medium
**Estimate:** 3-4 hours

**Description:**
Expand error types for common git failures.

**Files to modify:**
- `internal/git/errors.go`

**Reference:** `/deps/git-wt/internal/git/exec.go` (lines 600-607)

**New error types:**
```go
// Error types for specific git failures
var (
    ErrBranchNotFound     = errors.New("branch not found")
    ErrWorktreeNotFound   = errors.New("worktree not found")
    ErrDirtyWorktree      = errors.New("worktree has uncommitted changes")
    ErrNotMerged          = errors.New("branch not merged")
    ErrAlreadyExists      = errors.New("already exists")
)

// CheckExitCode returns a typed error based on git exit code
func CheckExitCode(err error) error {
    var exitErr *exec.ExitError
    if !errors.As(err, &exitErr) {
        return err
    }

    // Exit code 1 with specific messages
    switch exitErr.ExitCode() {
    case 1:
        // Parse stderr for specific errors
        stderr := string(exitErr.Stderr)
        if strings.Contains(stderr, "does not exist") {
            return ErrBranchNotFound
        }
        // ... other cases
    case 128:
        // Already exists, etc.
        return ErrAlreadyExists
    }

    return err
}
```

**Acceptance Criteria:**
- Typed errors for common failures
- Exit code parsing works
- Errors include original context
- Documentation on error types

---

## Priority 4: Documentation & Developer Experience

### P4.1: Enhanced Documentation

**Implementation Tasks:**

#### Task 4.1.1: Add examples to CLAUDE.md
**Complexity:** Low
**Estimate:** 2 hours

**Description:**
Add comprehensive examples for common workflows.

**Files to modify:**
- `CLAUDE.md`

**New sections:**
- **Common Workflows** - Step-by-step examples
- **Testing Guide** - How to run tests (unit and E2E)
- **Configuration Reference** - All config options explained
- **Template Variables** - Examples of path templates
- **Troubleshooting** - Common issues and solutions

**Acceptance Criteria:**
- Examples cover common use cases
- Configuration fully documented
- Easy to find relevant info

---

#### Task 4.1.2: Add inline code documentation
**Complexity:** Low
**Estimate:** 3-4 hours (ongoing)

**Description:**
Improve godoc comments for key functions.

**Focus areas:**
- Public API functions
- Complex algorithms
- Configuration structures
- Error types

**Example:**
```go
// GetWorktreesPath returns the absolute path to the worktrees directory.
// It applies template expansion to the configured worktrees_dir path,
// supporting variables like {gitroot}.
//
// Template expansion happens before path resolution, allowing patterns like:
//   - "../{gitroot}-worktrees" → "../myrepo-worktrees"
//   - "~/dev/{gitroot}/branches" → "/home/user/dev/myrepo/branches"
//
// Returns an error if not in a git repository.
func (s *Service) GetWorktreesPath() (string, error)
```

**Acceptance Criteria:**
- All public functions documented
- Examples in complex areas
- Consistent documentation style
- Godoc generates clean docs

---

## Implementation Phases

### Phase 1: Foundation (Weeks 1-2)
- ✅ P1.1: Stdout/Stderr Separation (Tasks 1.1.1 - 1.1.5)
- ✅ P1.2: E2E Testing Infrastructure (Tasks 1.2.1 - 1.2.3)
- ✅ P1.3: Flag Override Pattern (Tasks 1.3.1 - 1.3.3)

**Exit Criteria:**
- **Universal stdout/stderr pattern** implemented across all commands
- Shell integration uses unified stdout approach (single code path)
- TUI renders to /dev/tty exclusively
- Auto-cd works for switch, add, and list commands
- All command aliases supported (sw, s, ls, l, a)
- **Zero temp files** - all temp file code deleted
- **No environment variables** - simpler shell wrapper
- Shell wrapper simplified: 60+ lines → 20 lines
- Output pattern documented for all future commands
- E2E test suite running in CI
- Flag override pattern documented and used

### Phase 2: Ergonomics (Weeks 3-4)
- ✅ P2.1: Template Variables (Tasks 2.1.1 - 2.1.2)
- 🔄 P2.2: Enhanced File Copying (Tasks 2.2.1 - 2.2.3) - Optional

**Exit Criteria:**
- Template variables work in config
- File copying enhanced (if pursued)

### Phase 3: Polish (Week 5)
- ✅ P3.1: Code Organization (Tasks 3.1.1 - 3.1.2)
- ✅ P3.2: Error Handling (Task 3.2.1)
- ✅ P4.1: Documentation (Tasks 4.1.1 - 4.1.2)

**Exit Criteria:**
- Code well-organized
- Error handling consistent
- Documentation complete

---

## Success Metrics

### Code Quality
- [ ] Test coverage >70% (including E2E)
- [ ] All linters pass
- [ ] No regression in existing functionality

### Developer Experience
- [ ] Shell integration simpler (no temp files)
- [ ] Universal stdout/stderr pattern (consistent, scriptable)
- [ ] Configuration more flexible (templates)

### Maintainability
- [ ] E2E tests catch regressions
- [ ] Code better organized
- [ ] Clear error messages

---

## Risk Mitigation

### Risk 1: Breaking Changes
**Mitigation:**
- All changes are backwards compatible
- Existing commands continue to work
- New features are opt-in

### Risk 2: Testing Complexity
**Mitigation:**
- Start with simple E2E tests
- Build complexity incrementally
- Use testutil package for reusability

### Risk 3: Scope Creep
**Mitigation:**
- Each task is independently valuable
- Can stop at any phase
- Clearly marked optional tasks

---

## Appendix A: Comparison Matrix

| Feature | git-wt | gbm (current) | gbm (proposed) |
|---------|--------|---------------|----------------|
| **Shell Integration** | ✅ Auto-cd | ✅ Via temp files | ✅ Via stdout |
| **File Copying** | ✅ Gitignore-based | ✅ Rule-based | ✅ Both options |
| **Config Location** | git config | .gbm/config.yaml | .gbm/config.yaml |
| **Template Variables** | ✅ Yes | ❌ No | ✅ Yes |
| **E2E Tests** | ✅ Yes | ❌ No | ✅ Yes |
| **Flag Overrides** | ✅ Yes | ⚠️ Partial | ✅ Yes |
| **JIRA Integration** | ❌ No | ✅ Yes | ✅ Yes |
| **TUI** | ❌ No | ✅ Yes | ✅ Yes |
| **Command Shortcuts** | ✅ Yes | ⚠️ Partial | ⚠️ Partial |

---

## Appendix B: Task Dependency Graph

```
Phase 1 (Foundation)
├── P1.1 Stdout/Stderr (Universal Pattern)
│   ├── Task 1.1.1 (worktree switch)  ┐
│   ├── Task 1.1.2 (shell wrapper)    │
│   ├── Task 1.1.3 (worktree add)     ├─→ Phase 1 Complete
│   ├── Task 1.1.4 (TUI /dev/tty)     │
│   └── Task 1.1.5 (documentation)    ┘
├── P1.2 E2E Testing
│   ├── Task 1.2.1 (testutil) ───────┐
│   ├── Task 1.2.2 (basic E2E)       ├─→ Phase 1 Complete
│   └── Task 1.2.3 (shell E2E) ──────┘
└── P1.3 Flag Overrides
    ├── Task 1.3.1 (helper) ─────────┐
    ├── Task 1.3.2 (apply to --base) ├─→ Phase 1 Complete
    └── Task 1.3.3 (document) ───────┘

Phase 2 (Ergonomics)
├── P2.1 Templates
│   ├── Task 2.1.1 (engine) ─────────┐
│   └── Task 2.1.2 (apply) ──────────┴─→ Phase 2 Complete
└── P2.2 File Copying (Optional)
    ├── Task 2.2.1 (config) ─────────┐
    ├── Task 2.2.2 (matcher)         ├─→ Phase 2 Complete
    └── Task 2.2.3 (integrate) ──────┘

Phase 3 (Polish)
├── P3.1 Organization
│   ├── Task 3.1.1 (audit) ──────────┐
│   └── Task 3.1.2 (refactor) ───────┴─→ Phase 3 Complete
├── P3.2 Errors
│   └── Task 3.2.1 (typed errors) ────→ Phase 3 Complete
└── P4.1 Documentation
    ├── Task 4.1.1 (examples) ───────┐
    └── Task 4.1.2 (inline docs) ────┴─→ Phase 3 Complete
```

---

## Appendix C: Quick Reference

### Files by Priority

**High-frequency changes:**
- `cmd/service/worktree.go` - Main worktree command logic
- `internal/git/service.go` - Git operations
- `cmd/service/shell-integration.go` - Shell wrapper

**Infrastructure:**
- `testutil/repo.go` - Test utilities
- `e2e_test.go` - End-to-end tests
- `internal/utils/flags.go` - Flag helpers
- `internal/utils/template.go` - Path templates

**Documentation:**
- `CLAUDE.md` - Developer guide
- `README.md` - User guide
- `docs/improvement-prd.md` - This document

### Key Commands for Development

```bash
# Build and test
just build
just test
just validate

# Run E2E tests (after implementation)
go test -v -run TestE2E

# Install locally
just install

# Test shell integration
eval "$(gbm shell-integration)"
gbm wt feature-x
```

---

## Changelog

| Version | Date | Changes |
|---------|------|---------|
| 1.0 | 2026-01-02 | Initial PRD based on git-wt analysis |

---

**End of PRD**

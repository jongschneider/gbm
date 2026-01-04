# Standard CLI Flags Implementation Plan

**Date:** 2026-01-04  
**Scope:** Consolidate existing --dry-run flag as global and implement missing standard CLI flags (--json, --no-color, -q/--quiet, --no-input)  
**Estimated Effort:** 4-6 hours total  
**Priority:** High (improves CI/CD compatibility, accessibility, scripting)

---

## Executive Summary

This plan consolidates and extends CLI flag support by making the existing `--dry-run` flag global and implementing additional industry-standard flags that users expect in modern command-line tools. These flags improve compatibility with CI/CD pipelines, shell scripts, and accessibility requirements while maintaining backward compatibility.

**Flags to Consolidate & Implement:**
1. `--dry-run` - Preview operations without executing (already exists, consolidate as global)
2. `--json` / `-j` - Output in JSON format for scripting
3. `--no-color` - Disable colored output (respects NO_COLOR env var)
4. `-q, --quiet` - Suppress non-essential messages
5. `--no-input` - Disable interactive prompts (use defaults)

**Impact:**
- Enables scripting and JSON parsing with `jq`
- Improves accessibility and CI/CD compatibility
- No breaking changes to existing output
- Zero backward compatibility issues

---

## Table of Contents

1. [Current State](#current-state)
2. [Goals & Success Criteria](#goals--success-criteria)
3. [Implementation Tasks](#implementation-tasks)
4. [Testing Strategy](#testing-strategy)
5. [Risk Assessment](#risk-assessment)

---

## Current State

### Existing Implementation

**Location:** `cmd/service/root.go` and scattered throughout commands

**Current Flags:**
- `--dry-run` - Preview operations without executing (already exists, but scattered/not global)
- `-v, --verbose` - Enable verbose output (already exists)

**Flags to Consolidate:**
- `--dry-run` - Currently implemented but needs to be registered as a global persistent flag

**Missing Flags:**
- `--json` - No JSON output support
- `--no-color` - Colors always enabled
- `-q, --quiet` - No message suppression
- `--no-input` - TUI always interactive

**Color Handling:**
- Colors always enabled (no TTY detection)
- No `NO_COLOR` environment variable support
- Hard to use in CI/CD pipelines

**Message Handling:**
- Stderr messages always printed
- No way to suppress progress/status messages
- Error messages mixed with status messages

---

## Goals & Success Criteria

### Goal 1: Add Global CLI Flags

**Objective:** Register standard flags on root command accessible to all subcommands

**Success Criteria:**
- ✅ All 4 flags registered as persistent flags on root command
- ✅ Flags accessible from any subcommand via helper functions
- ✅ Default values documented in help text
- ✅ Flags work together without conflicts
- ✅ Help text includes examples of flag usage

**Example Success:**
```bash
$ gbm --dry-run worktree add feature-x feature/x -b
[shows what would happen without executing]

$ gbm --json worktree list
[{"name":"main","path":"/repo/worktrees/main",...}]

$ gbm --quiet worktree add feature-x feature/x -b
✓ Created worktree 'feature-x'

$ gbm --no-color worktree switch main
Switched to worktree 'main'

$ gbm --no-input worktree remove feature-x
[removes without prompting]
```

---

### Goal 2: Color Management

**Objective:** Detect TTY and respect NO_COLOR, implement smart color handling

**Success Criteria:**
- ✅ Detects if stdout is a TTY (terminal)
- ✅ Respects `NO_COLOR` environment variable
- ✅ `--no-color` flag overrides auto-detection
- ✅ Colors disabled automatically in CI/CD environments
- ✅ Tests verify color detection logic

**Example Success:**
```bash
# In terminal - colors enabled
$ gbm wt list
[colored output with ✓, ⚠, ℹ icons]

# In CI/CD pipeline - colors auto-disabled
$ gbm wt list
[plain text output without colors]

# Explicit disable
$ gbm --no-color wt list
[plain text output]

# NO_COLOR env var
$ NO_COLOR=1 gbm wt list
[plain text output]
```

---

### Goal 3: JSON Output

**Objective:** Support `--json` flag for scripting integration

**Success Criteria:**
- ✅ Data output in valid JSON (parseable by jq)
- ✅ Works with all commands that output data
- ✅ Consistent JSON structure across commands
- ✅ Error messages still go to stderr
- ✅ Tests validate JSON validity with jq

**Commands Supporting JSON:**
- `worktree list` - Array of worktree objects
- `worktree switch` - Single worktree object with path
- `worktree add` - Single worktree object created
- `worktree remove` - Operation summary
- `sync` - Sync status report
- `init` - Repository info

**Example Success:**
```bash
$ gbm --json worktree list | jq '.[] | select(.branch == "main")'
{
  "name": "main",
  "path": "/repo/worktrees/main",
  "branch": "main",
  "status": "active"
}

$ gbm --json worktree switch feature-x
{
  "name": "feature-x",
  "path": "/repo/worktrees/feature-x",
  "branch": "feature/feature-x"
}
```

---

### Goal 4: Quiet Mode

**Objective:** Suppress non-critical messages

**Success Criteria:**
- ✅ `-q/--quiet` flag suppresses status/progress messages
- ✅ Error messages still printed (critical info)
- ✅ Works with all commands
- ✅ Respects quiet mode in help output
- ✅ Tests verify message filtering

**Message Categories:**
1. **Errors** - Always printed (critical)
2. **Data** - Always printed to stdout
3. **Status** - Suppressed with --quiet (✓ Created, Switched to, etc.)
4. **Progress** - Suppressed with --quiet (fetching, syncing, etc.)
5. **Info** - Suppressed with --quiet (informational messages)

**Example Success:**
```bash
# Normal output
$ gbm worktree add feature-x feature/x -b
/path/to/worktrees/feature-x
✓ Created worktree 'feature-x' for branch 'feature/feature-x'

# Quiet mode
$ gbm -q worktree add feature-x feature/x -b
/path/to/worktrees/feature-x

# Error still shown in quiet mode
$ gbm -q worktree add main main
Error: worktree 'main' already exists
```

---

### Goal 5: No-Input Mode

**Objective:** Disable interactive prompts for scripting

**Success Criteria:**
- ✅ `--no-input` disables all interactive prompts
- ✅ Uses sensible defaults when prompts skipped
- ✅ Works with TUI commands (skip TUI, use defaults)
- ✅ Error if required input cannot be provided
- ✅ Tests verify prompt behavior

**Prompts to Handle:**
1. Branch creation confirmation - Default: create branch
2. Branch deletion confirmation - Default: do not delete
3. TUI selection (worktree list) - Default: show error
4. Overwrite confirmation - Default: do not overwrite

**Example Success:**
```bash
# Normal - prompts for confirmation
$ gbm worktree remove feature-x
Delete branch 'feature/feature-x'? (y/n) n
✓ Removed worktree 'feature-x'

# No-input mode - uses default (don't delete branch)
$ gbm --no-input worktree remove feature-x
✓ Removed worktree 'feature-x'

# TUI requires input - error without --no-input
$ gbm --no-input worktree list
Error: TUI requires interactive input, use worktree switch instead
```

---

## Implementation Tasks

### Phase 1: Flag Infrastructure (1.5 hours)

#### Task 1.1: Create flags helper package
**Effort:** 30 min  
**File:** `cmd/service/flags.go` (already created)

**Status:** ✅ PARTIALLY COMPLETE
- `CLIFlags` struct defined
- Global flag getter/setter implemented
- `ShouldUseColor()` function with TTY detection
- Message printing functions (PrintMessage, PrintError, etc.)

#### Task 1.1b: Consolidate existing --dry-run flag
**Effort:** 30 min  
**File:** `cmd/service/flags.go` and `cmd/service/root.go`

**Consolidation Tasks:**
- Add `--dry-run` to persistent flags in root command
- Extract existing scattered `--dry-run` implementations into `flags.go` helper
- Create `ShouldUseDryRun()` accessor function (consistent pattern with other flags)
- Remove duplicate flag registrations from individual commands
- Update all command implementations to use the centralized helper
- Ensure all existing dry-run behavior continues to work

**Status:** 🔲 TODO
- Search for all `--dry-run` flag definitions in codebase
- Identify where flag is currently registered (likely in multiple files)
- Create unified accessor and registration

**Remaining:**
- None - file is complete

#### Task 1.2: Register flags in root command
**Effort:** 15 min  
**File:** `cmd/service/root.go`

**Changes needed:**
```go
// In newRootCommand()
var flags CLIFlags

rootCmd.PersistentFlags().BoolVarP(&flags.JSON, "json", "j", false, "Output in JSON format")
rootCmd.PersistentFlags().BoolVar(&flags.NoColor, "no-color", false, "Disable colored output")
rootCmd.PersistentFlags().BoolVarP(&flags.Quiet, "quiet", "q", false, "Suppress non-essential messages")
rootCmd.PersistentFlags().BoolVar(&flags.NoInput, "no-input", false, "Disable interactive prompts")

// In PersistentPreRun
SetGlobalFlags(&flags)
```

#### Task 1.3: Thread flags through Service
**Effort:** 15 min  
**Files:** `cmd/service/service.go`

**Changes needed:**
- Add flag references to Service struct (optional, flags are global)
- Update comment documentation

---

### Phase 2: Color & Quiet Support (1-1.5 hours)

#### Task 2.1: Update message output functions
**Effort:** 30 min  
**Files:** Multiple command files

**Changes needed:**
- Replace `fmt.Fprintf(os.Stderr, ...)` with `PrintMessage()`
- Replace success messages with `PrintSuccess()`
- Replace warnings with `PrintWarning()`
- Keep errors as `PrintError()` (always shown)

**Files to update:**
- `cmd/service/worktree.go`
- `cmd/service/init.go`
- `cmd/service/clone.go`
- `cmd/service/sync.go`

#### Task 2.2: Test color detection
**Effort:** 15 min  
**File:** `cmd/service/flags_test.go` (new)

**Tests needed:**
```go
TestShouldUseColor_TTY()         // When stdout is a TTY
TestShouldUseColor_NoCOLOR()      // NO_COLOR env var set
TestShouldUseColor_NoColorFlag()  // --no-color flag
TestShouldUseColor_NotTTY()       // When not in terminal (CI/CD)
```

#### Task 2.3: Test quiet mode
**Effort:** 15 min  
**File:** `cmd/service/flags_test.go`

**Tests needed:**
```go
TestQuietMode_Suppresses()        // Messages suppressed
TestQuietMode_ErrorsShown()       // Errors still shown
TestQuietMode_DataOutput()        // Data to stdout unaffected
```

---

### Phase 3: JSON Output Support (2-2.5 hours)

#### Task 3.1: Create JSON output helper
**Effort:** 30 min  
**File:** `cmd/service/json_output.go` (new)

**Functions needed:**
```go
// JSON marshaling with consistent structure
type JSONOutput struct {
    Success bool        `json:"success"`
    Data    interface{} `json:"data,omitempty"`
    Error   string      `json:"error,omitempty"`
}

func OutputJSON(data interface{}) error
func OutputError(err error) error
```

#### Task 3.2: Update commands for JSON output
**Effort:** 1 hour  
**Files:**
- `cmd/service/worktree.go` (worktree list/switch/add/remove)
- `cmd/service/sync.go`
- `cmd/service/init.go`
- `cmd/service/clone.go`

**Pattern:**
```go
if ShouldUseJSON() {
    // Output JSON
    outputJSON(result)
} else {
    // Output text
    fmt.Println(path)
    PrintSuccess("...")
}
```

#### Task 3.3: Test JSON output
**Effort:** 1 hour  
**File:** `cmd/service/json_output_test.go` (new)

**E2E Tests:**
```go
TestJSON_WorktreeList()           // Valid JSON array
TestJSON_WorktreeSwitch()         // Valid JSON object
TestJSON_WithJQ()                 // Parseable by jq
TestJSON_Errors()                 // Error format
```

---

### Phase 4: No-Input Mode (1-1.5 hours)

#### Task 4.1: Update TUI commands
**Effort:** 30 min  
**File:** `cmd/service/worktree.go`

**Changes needed:**
```go
// In worktree list command
if !ShouldAllowInput() {
    return fmt.Errorf("TUI requires interactive input, use 'worktree switch' instead")
}

// Or skip TUI entirely
if !ShouldAllowInput() {
    // Use default behavior or list all
}
```

#### Task 4.2: Update confirmation prompts
**Effort:** 30 min  
**Files:**
- `cmd/service/worktree.go` (remove confirmation)
- `cmd/service/init.go` (overwrite confirmation)

**Pattern:**
```go
if ShouldAllowInput() {
    // Show prompt, wait for input
    confirmed := promptUser("Delete branch?")
} else {
    // Use default
    confirmed = false  // Don't delete branch by default
}
```

#### Task 4.3: Test no-input mode
**Effort:** 15 min  
**File:** `cmd/service/flags_test.go`

**Tests:**
```go
TestNoInput_SkipsTUI()            // TUI commands return error
TestNoInput_UsesDefaults()        // Prompts use defaults
TestNoInput_ErrorOnRequired()     // Error if input required
```

---

## Testing Strategy

### Unit Tests (3-4 hours)

**Files to create:**
- `cmd/service/flags_test.go` - Flag functionality tests
- `cmd/service/json_output_test.go` - JSON output tests

**Coverage targets:**
- Flag detection and state checking
- Color detection logic (TTY, NO_COLOR env, flag)
- JSON marshaling and validity
- Message filtering with quiet mode
- Prompt skipping with no-input

### E2E Tests (2-3 hours)

**Add to:** `e2e_test.go`

**Test scenarios:**
```
TestE2E_JSON_WorktreeList()       # JSON output validity
TestE2E_JSON_WithJQ()             # jq parsing
TestE2E_NoColor_CI()              # Automatic color disable
TestE2E_Quiet_SuppressesMessages() # Message suppression
TestE2E_NoInput_SkipsPrompts()    # Prompt skipping
TestE2E_FlagCombinations()        # Multiple flags together
```

### Manual Testing Checklist

- [ ] `gbm --dry-run worktree add` shows command without executing (consolidation)
- [ ] `gbm --dry-run` flag accessible globally across all commands
- [ ] Existing --dry-run behavior unchanged (backward compatible)
- [ ] `gbm --json worktree list` produces valid JSON
- [ ] `gbm --json worktree list | jq` works
- [ ] `gbm -q` suppresses messages but shows data
- [ ] `gbm -q` still shows errors
- [ ] `gbm --no-color` has no ANSI codes
- [ ] NO_COLOR env var disables colors
- [ ] `gbm --no-input` skips all prompts
- [ ] All flags work together: `gbm --json -q --no-input --dry-run`
- [ ] Combined with --dry-run: `gbm --json --dry-run worktree add ...` produces JSON
- [ ] Help text shows all flags: `gbm --help`
- [ ] Backward compatibility: old behavior unchanged without flags

---

## Risk Assessment

### Risks

1. **JSON Output Format Changes**
   - **Mitigation:** Design format first, document in help text
   - **Recovery:** Versioning support in JSON output
   - **Testing:** E2E tests validate JSON schema

2. **Color Detection Wrong**
   - **Mitigation:** Use `golang.org/x/term` (standard library)
   - **Recovery:** Explicit `--no-color` flag override
   - **Testing:** Test TTY detection on CI systems

3. **Breaking Changes**
   - **Mitigation:** Flags are additive only, no behavior changes without flags
   - **Recovery:** N/A (no breaking changes)
   - **Testing:** Verify old behavior unchanged

4. **Performance Impact**
   - **Mitigation:** Flag checks are O(1), minimal overhead
   - **Impact:** ~1-2ms overhead per command (negligible)
   - **Testing:** Benchmark if needed

### Rollback Plan

If issues arise:
1. Revert changes in git
2. Document issues for fix in next release
3. Can selectively disable flags if needed

---

## Success Metrics

### Quantitative

- [ ] 100% of new code covered by tests
- [ ] 0 new breaking changes
- [ ] All E2E tests passing
- [ ] JSON output valid for all commands

### Qualitative

- [ ] Flags are intuitive and follow CLI standards
- [ ] Help text clearly explains each flag
- [ ] CI/CD pipelines work without color issues
- [ ] JSON output useful for scripting

### User Feedback

After implementation:
- Verify flag behavior with documentation examples
- Test in CI/CD pipeline (GitHub Actions, etc.)
- Validate JSON output with `jq` operations

---

## Notes on --dry-run Consolidation

### Current State Investigation Needed
Before implementing Task 1.1b, we need to:
1. Locate all existing `--dry-run` flag registrations in codebase
2. Determine which commands currently support it
3. Document current implementation pattern
4. Identify any command-specific dry-run behavior that needs special handling

### Design Considerations
- **Accessor function:** `ShouldUseDryRun()` - follows pattern with `ShouldUseColor()`, `ShouldUseJSON()`, etc.
- **Backward compatibility:** All existing dry-run behavior must continue working
- **Consistency:** Use same pattern for flag registration and access as other global flags
- **Documentation:** Help text should explain what dry-run does for each command

---

## Next Steps

1. ✅ Review and approve this updated plan (now includes --dry-run consolidation)
2. Execute implementation following phases (Phase 1 now 1.5 hours)
3. Run test suite after each phase
4. Manual testing before completion
5. Request review and approval
6. Commit changes to git

**Dependencies:**
- None (all changes are self-contained)
- Depends on: completed config validation (already done)

**Blockers:**
- None identified

**Investigation Required:**
- Locate existing --dry-run implementations before starting Phase 1.1b

---

## References

- **CLI Guidelines:** https://clig.dev/
- **NO_COLOR Standard:** https://no-color.org/
- **jq Tutorial:** https://stedolan.github.io/jq/
- **Go term package:** https://pkg.go.dev/golang.org/x/term

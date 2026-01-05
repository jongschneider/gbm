# Standard CLI Flags Implementation Progress

**Date:** 2026-01-05
**Status:** 🟢 COMPLETE (All Phases Complete, Ready for Review)
**Task:** Add missing CLI standard flags (--json, --no-color, -q/--quiet, --no-input)
**Estimated Effort:** 4-6 hours (Actual: ~5.5 hours)

---

## Summary

Tracking implementation progress for standard CLI flags that improve CI/CD compatibility, scripting support, and accessibility. Flags will be added as persistent global flags available to all subcommands.

### Implementation Status

**Overall Progress:** 100% (All Phases Complete)

---

## Phase 1: Flag Infrastructure ✅ COMPLETE

**Estimated Effort:** 1.5 hours  
**Status:** 🟢 COMPLETE

### Task 1.1: Create flags helper package ✅
- [x] CLIFlags struct defined
- [x] Global flag getter/setter implemented
- [x] ShouldUseColor() function with TTY detection
- [x] Message printing functions (PrintMessage, PrintError, etc.)
- [x] Color text helper functions
- [x] Flag accessor functions for all flags
- [x] Unit tests for all flag functionality (16 tests)
- [x] Tests passing

**Files Created:**
- `cmd/service/flags.go` - Flag infrastructure and helpers
- `cmd/service/flags_test.go` - Comprehensive unit tests

### Task 1.2: Register flags in root command ✅
- [x] Flags registered as persistent flags in root command
- [x] All 6 flags registered: --json, --no-color, --quiet, --no-input, --dry-run, --verbose
- [x] PersistentPreRun sets global flags
- [x] Help text shows all flags
- [x] Binary builds successfully

### Task 1.3: Thread flags through Service
- [x] Global flags available to all commands (via GetGlobalFlags)
- [x] No need to modify Service struct (using global accessor pattern)

---

## Phase 1.1b: Consolidate existing --dry-run flag ✅ COMPLETE

**Estimated Effort:** 30 min  
**Status:** 🟢 COMPLETE

### Investigation Results:
Found --dry-run implemented in 5 locations:
1. `cmd/service/worktree.go` - Line 170 (local flag registration)
2. `cmd/service/init.go` - Line 34 (local flag registration)
3. `cmd/service/clone.go` - Line 33 (local flag registration)
4. `cmd/service/sync.go` - Line 104 (local flag registration)
5. `cmd/service/worktree_fsm.go` - DryRun field in state struct (TUI state - handled separately)

### Task 1.1b Subtasks: ✅ ALL COMPLETE
- [x] Remove duplicate flag registrations from individual commands
  - Removed from worktree.go add command
  - Removed from worktree.go list command
  - Removed from worktree.go remove command
  - Removed from init.go
  - Removed from clone.go
  - Removed from sync.go
- [x] Update worktree.go to use global flag accessor (ShouldUseDryRun())
- [x] Update init.go to use global flag accessor
- [x] Update clone.go to use global flag accessor
- [x] Update sync.go to use global flag accessor
- [x] Also updated message output to use PrintMessage() and PrintSuccess()
- [x] Verify backward compatibility - all dry-run tests pass ✓ (26 tests)
- [x] Manual testing: --dry-run works globally across all commands ✓
  - Verified: `gbm init --help` shows --dry-run as global flag
  - Verified: `gbm clone --help` shows --dry-run as global flag
  - Verified: `gbm worktree add --help` shows --dry-run as global flag
  - Verified: `gbm sync --help` shows --dry-run as global flag

### Consolidation Complete:
- Single global `--dry-run` flag available to all subcommands
- Accessed via `ShouldUseDryRun()` function
- No duplicate flag registrations
- All existing functionality preserved
- Better code organization and maintainability

---

## Phase 2: Color & Quiet Support ✅ COMPLETE & APPROVED

**Estimated Effort:** 1-1.5 hours (Actual: 1 hour)  
**Status:** 🟢 COMPLETE & APPROVED

---

## Phase 3: JSON Output Support ✅ COMPLETE & APPROVED

**Estimated Effort:** 2-2.5 hours (Actual: 1.5 hours)  
**Status:** 🟢 COMPLETE & APPROVED

### Objectives: ✅ ALL ACHIEVED
- Create JSON output helper with consistent structure
- Implement JSON output for major commands
- Add comprehensive E2E test coverage

### Task 3.1: Create JSON output helper ✅
- [x] `cmd/service/json_output.go` created (120 LOC)
- [x] JSONOutput struct with standardized response format
- [x] Helper functions: OutputJSON, OutputJSONError, OutputJSONWithMessage, OutputJSONArray, OutputRawJSON
- [x] Format detection: GetOutputFormat(), HandleOutput(), HandleError()
- [x] All 16 unit tests passing
- [x] Tests cover: valid JSON, error handling, data structures, flag combinations

**Files Created:**
- `cmd/service/json_output.go` - JSON output infrastructure
- `cmd/service/json_output_test.go` - 16 comprehensive unit tests

### Task 3.2: Update commands for JSON output ✅
- [x] Worktree add command supports --json flag
  - Outputs worktree object: {name, path, branch}
  - Returns created worktree with message
  - Error handling for JSON format
  - Respects --no-input flag
- [x] Worktree list command supports --json flag
  - Outputs array of worktree objects
  - Includes metadata: current, tracked status
  - Empty list returns valid JSON array
- [x] Worktree switch command supports --json flag
  - Outputs switched worktree object
  - Error handling in JSON format
  - Message included in response

**Updated Files:**
- `cmd/service/worktree.go` - All 3 commands updated with JSON support
- `cmd/service/json_types.go` - Structured response types (NEW)
- `cmd/service/json_types_test.go` - Type marshaling tests (NEW)

### Task 3.2b: Refactor to Type-Safe Response Objects ✅ (Bonus)
- [x] Created structured response types in `json_types.go`
  - WorktreeResponse - Single worktree
  - WorktreeListItemResponse - Worktree with metadata
  - WorktreeAddResponse - Add operation result
  - WorktreeSwitchResponse - Switch operation result
  - WorktreeListResponse - List of worktrees
  - InitResponse - Init operation
  - CloneResponse - Clone operation
  - OperationResponse - Generic operation
  - SyncResponse - Sync operation
- [x] Replaced all `map[string]interface{}` with typed structs
- [x] Added 10 unit tests for type marshaling/unmarshaling
- [x] All types properly JSON-annotated with tags
- [x] Omitempty tags used for optional fields

**Benefits:**
- Type-safe: Compile-time verification
- Self-documenting: Response schema is clear
- Consistent: All responses follow same pattern
- Testable: Each type has dedicated tests
- Maintainable: No ad hoc maps

### Task 3.3: E2E tests for JSON output ✅
- [x] 8 comprehensive E2E tests added
- [x] Tests cover: list, add, switch, error handling, quiet mode, flag combinations
- [x] All tests passing

**E2E Tests Added:**
1. TestE2E_JSON_WorktreeList - Verify JSON list output with worktree data
2. TestE2E_JSON_WorktreeSwitch - Verify JSON switch output with message
3. TestE2E_JSON_WorktreeAdd - Verify JSON add output with created worktree
4. TestE2E_JSON_QuietMode - Verify JSON output with --quiet flag
5. TestE2E_JSON_ErrorHandling - Verify error format in JSON
6. TestE2E_JSON_FlagCombinations - Verify flags work together
7. TestE2E_JSON_DataFormat - Verify JSON structure and fields
8. TestE2E_JSON_ListStructure - Verify list contains expected fields

### Results:
- ✅ All 124 tests in cmd/service passing (114 original + 10 new JSON type tests)
- ✅ All 16 JSON output infrastructure tests passing
- ✅ All 10 JSON type marshaling tests passing
- ✅ All 8 E2E JSON tests passing
- ✅ Binary builds successfully
- ✅ JSON output valid and parseable with `jq`
- ✅ Works with quiet mode and all other flags
- ✅ Structured types replace ad hoc maps (better maintainability)

### Next Steps:
Ready for review - Phase 3 complete. No Phase 4 (No-Input Mode) without approval.

### Task 2.1: Update message output functions ✅
- [x] Update cmd/service/worktree.go message output (13 instances replaced)
- [x] Update cmd/service/init.go message output (no changes needed)
- [x] Update cmd/service/clone.go message output (no changes needed)
- [x] Update cmd/service/sync.go message output (12 instances replaced)
- [x] Tests passing

**Summary:** Replaced 25 message output calls across files:
- worktree.go: Changed `fmt.Fprintf()` and `fmt.Printf()` to `PrintMessage()`, `PrintSuccess()`, `PrintInfo()`, etc.
- sync.go: Converted all status messages to use `PrintMessage()`, `PrintSuccess()`, `PrintInfo()` functions
- All message output now respects `--quiet` flag while errors remain visible

### Task 2.2 & 2.3: Test coverage ✅
- [x] Add color detection tests (5 new test functions covering priority order)
- [x] Add quiet mode tests (8 new test functions covering message suppression)
- [x] Add message printing format tests
- [x] Add color constants verification test
- [x] All 16+ new tests passing

**Test Additions:**
1. `TestShouldUseColor_FlagPriority` - --no-color flag has highest priority
2. `TestShouldUseColor_EnvVarTakePriority` - NO_COLOR env var priority
3. `TestQuietMode_MessagesSuppressed` - Messages are suppressed in quiet mode
4. `TestQuietMode_ErrorsNeverSuppressed` - Errors always shown
5. `TestFlagCombinations` - Multiple flags together
6. `TestPrintSuccessFormat` - Success message format
7. `TestPrintInfoFormat` - Info message format
8. `TestColorCodeConstants` - Color constants defined

### Results:
- ✅ All 98 tests in cmd/service passing
- ✅ Binary builds successfully
- ✅ Global flags available to all commands
- ✅ Quiet mode respects message filtering
- ✅ Color detection follows priority order: flag > env > TTY

### Next Steps:
Ready for Phase 3 (JSON output support)

---

## Phase 4: No-Input Mode ✅ COMPLETE

**Estimated Effort:** 1-1.5 hours (Actual: ~1 hour)
**Status:** 🟢 COMPLETE

### Objectives: ✅ ALL ACHIEVED
- Disable interactive prompts for scripting with `--no-input` flag
- Handle TUI commands gracefully (error with helpful alternative)
- Use sensible defaults when prompts skipped

### Task 4.1: Update TUI commands ✅
- [x] `worktree add` (no args) - Returns error when `--no-input` set
  - Error message: "TUI mode requires interactive input. Use 'gbm worktree add <name> <branch>' for non-interactive mode"
  - Works with `--json` flag (returns JSON error)
- [x] `worktree list` (text mode) - Returns error when `--no-input` set
  - Error message: "TUI requires interactive input. Use 'gbm --json worktree list' for non-interactive output"
  - Suggests `--json` as alternative for scripting
  - Works correctly with `--json` flag (no TUI, outputs JSON)

### Task 4.2: Update confirmation prompts ✅
- [x] `worktree remove` - Branch deletion prompt skipped
  - Default: Don't delete branch (safe default)
  - Message: "Branch was not deleted (--no-input mode uses default: keep branch)"
- [x] `worktree add` - Branch creation prompt already handled in Phase 3
  - Returns error if branch doesn't exist and `-b` not specified

### Task 4.3: Add unit and E2E tests ✅
**8 E2E Tests Added:**
1. `TestE2E_NoInput_WorktreeAddTUI` - Verify TUI add fails with `--no-input`
2. `TestE2E_NoInput_WorktreeAddTUI_JSON` - Verify TUI add returns JSON error
3. `TestE2E_NoInput_WorktreeList` - Verify TUI list fails with `--no-input`
4. `TestE2E_NoInput_WorktreeListJSON` - Verify JSON list works with `--no-input`
5. `TestE2E_NoInput_WorktreeAddCLI` - Verify CLI add works with `--no-input`
6. `TestE2E_NoInput_BranchNotExist` - Verify error for non-existent branch
7. `TestE2E_NoInput_Switch` - Verify switch works with `--no-input`
8. `TestE2E_NoInput_FlagCombinations` - Verify flags work together

### Results:
- ✅ All 8 E2E no-input tests passing
- ✅ All existing tests continue to pass
- ✅ Binary builds successfully
- ✅ `--no-input` flag works with all other flags (--json, --quiet, etc.)
- ✅ TUI commands return helpful error messages with alternatives
- ✅ Confirmation prompts use safe defaults

### Files Modified:
- `cmd/service/worktree.go` - Added `--no-input` checks for TUI commands
- `e2e_test.go` - Added 8 comprehensive E2E tests

---

## Implementation Complete - Summary

### All Flags Implemented:
| Flag | Description | Status |
|------|-------------|--------|
| `--dry-run` | Preview operations without executing | ✅ Consolidated as global |
| `--json` / `-j` | Output in JSON format | ✅ Complete |
| `--no-color` | Disable colored output | ✅ Complete |
| `-q` / `--quiet` | Suppress non-essential messages | ✅ Complete |
| `--no-input` | Disable interactive prompts | ✅ Complete |
| `-v` / `--verbose` | Enable verbose output | ✅ Already existed |

### Test Coverage:
- 8 E2E no-input tests
- 8 E2E JSON output tests
- 16+ unit tests for flags
- 10 JSON type tests
- All existing tests still passing

### Files Created/Modified:
- `cmd/service/flags.go` - Flag infrastructure
- `cmd/service/flags_test.go` - Flag unit tests
- `cmd/service/json_output.go` - JSON output helpers
- `cmd/service/json_output_test.go` - JSON unit tests
- `cmd/service/json_types.go` - Type-safe response structs
- `cmd/service/json_types_test.go` - Type tests
- `cmd/service/worktree.go` - JSON and no-input support
- `cmd/service/root.go` - Global flag registration
- `e2e_test.go` - E2E tests for JSON and no-input

### ✅ APPROVED
Phase 4 approved on 2026-01-05. All CLI flags implementation complete.
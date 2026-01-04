# Standard CLI Flags Implementation Progress

**Date:** 2026-01-04  
**Status:** 🟡 IN PROGRESS (Phase 1, 1.1b, & 2 Complete, Approved)  
**Task:** Add missing CLI standard flags (--json, --no-color, -q/--quiet, --no-input)  
**Estimated Effort:** 4-6 hours (3 hours complete, 50%)

---

## Summary

Tracking implementation progress for standard CLI flags that improve CI/CD compatibility, scripting support, and accessibility. Flags will be added as persistent global flags available to all subcommands.

### Implementation Status

**Overall Progress:** 50% (Phase 1, 1.1b, and 2 complete, Phase 3 ready to start)

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

### Objectives: ✅ ALL ACHIEVED
- Replace direct `fmt.Fprintf(os.Stderr, ...)` calls with PrintMessage() variants
- Test color detection (TTY, NO_COLOR env, --no-color flag)
- Test quiet mode message filtering
- Ensure errors always shown in quiet mode

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
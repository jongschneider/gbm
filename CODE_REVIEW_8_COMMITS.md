# Code Review - Past 8 Commits

**Review Date:** 2025-12-24
**Commits:** `97c92f2..32fc94b` (wip commits)
**Focus:** Bugs, Performance, Security, Correctness, Idiomatic Go, Complexity

---

## Critical Issues

### 1. **Security: Hardcoded Binary Name** ✅
**Severity:** High
**Location:** `cmd/service/worktree_table.go:172`
**Status:** FIXED

```go
cmd := exec.Command("gbm2", "wt", "switch", targetWorktree.Name)
```

**Issue:** Binary name is hardcoded. Won't work if binary renamed or in different location.

**Fix:**
```go
cmd := exec.Command(os.Args[0], "wt", "switch", targetWorktree.Name)
```

### 2. **Security: Temp File Not Cleaned Up** ✅
**Severity:** Medium
**Location:** `cmd/service/worktree_table.go:243`
**Status:** FIXED

```go
_ = os.WriteFile(tmpFile, []byte(model.switchOutput), 0600)
```

**Issues:**
- Silent error handling (write failures ignored)
- Temp file never cleaned up (potential information leak)

**Fix:**
```go
// Clean up any stale temp file from previous run
_ = os.Remove(tmpFile)

if err := os.WriteFile(tmpFile, []byte(model.switchOutput), 0600); err != nil {
    return fmt.Errorf("failed to write switch file: %w", err)
}
// Note: The shell integration is responsible for cleaning up this file after reading
```

### 3. **Bug: Recursive Call in Error Path** ✅
**Severity:** Medium
**Location:** `cmd/service/worktree_fsm.go:325, 534`
**Status:** FIXED

```go
if err := createWorktreeNameValidator(...); err != nil {
    // ... show error ...
    return w.runFeatureWorktreeName() // RECURSIVE CALL
}
```

**Issue:** Recursive call can lead to stack overflow with repeated validation failures.

**Fix:** Wrapped both functions in `for` loops and use `continue` instead of recursive calls:
```go
func (w *WorktreeAddFSM) runFeatureWorktreeName() (string, error) {
    for {
        // ... input logic ...
        if err := createWorktreeNameValidator(...); err != nil {
            // ... show error ...
            continue // Retry without recursion
        }
        return EventComplete, nil
    }
}
```

---

## Bugs & Correctness

### 4. **Type Assertion Missing Type Info** ✅
**Severity:** Low
**Location:** `cmd/service/wizard.go:195`
**Status:** FIXED

```go
return nil, fmt.Errorf("unexpected model type")
```

**Fix:**
```go
return nil, fmt.Errorf("unexpected model type: %T", finalModel)
```

### 5. **Variable Shadowing** ✅
**Severity:** Low
**Location:** `cmd/service/filterable_select.go:240`
**Status:** FIXED

```go
if finalModel, ok := finalModel.(FilterableSelectModel); ok { // shadows outer finalModel
```

**Fix:**
```go
if model, ok := finalModel.(FilterableSelectModel); ok {
```

### 6. **Unused Map Field** ✅
**Severity:** Low
**Location:** `cmd/service/wizard.go:29`
**Status:** FIXED

```go
values map[string]interface{} // Store values across steps
```

**Issue:** `values` map initialized but never populated (`SetValue` never called).

**Fix:** Removed unused `values` field and `GetValue`/`SetValue` methods since they were never used.

### 7. **FSM State History Unbounded Growth** ✅
**Severity:** Low
**Location:** `cmd/service/worktree_fsm.go:134-135`
**Status:** FIXED

```go
w.state.StateHistory = append(w.state.StateHistory, e.Dst)
w.state.EventHistory = append(w.state.EventHistory, e.Event)
```

**Issue:** Histories grow without bounds. Not a problem for short workflows, but could accumulate memory in long sessions.

**Fix:** Clear history on all terminal states (Success, Cancelled, Error) to prevent memory accumulation.

---

## Performance

### 8. **Inefficient Slice Copy** ❌ NOT A BUG
**Severity:** Low
**Location:** `cmd/service/filterable_select.go:164-172`
**Status:** NOT A BUG - Loop is necessary for type conversion

```go
if query == "" {
    m.filteredList = make([]list.Item, len(m.allItems))
    for i, item := range m.allItems {
        m.filteredList[i] = item
    }
}
```

**Analysis:** This is NOT inefficient. The loop is required because:
- `m.allItems` is `[]FilterableItem`
- `m.filteredList` is `[]list.Item`
- Even though `FilterableItem` implements the `list.Item` interface, `[]FilterableItem` and `[]list.Item` are incompatible types
- The loop performs necessary implicit conversion from concrete type to interface type
- `copy()` cannot be used here as it requires identical element types

---

## Complexity Improvements

### 9. **FSM Transition Table Readability** ✅
**Severity:** Low
**Location:** `cmd/service/worktree_fsm.go:43-181`
**Status:** FIXED

**Observation:** FSM events table is long (80+ lines) but well-structured. Good improvement over previous nested loop approach.

**Fix Applied:** Extracted workflow-specific transitions into separate builder functions:
- `buildTypeSelectionTransitions()` - Initial workflow type selection
- `buildFeatureTransitions()` - Feature workflow state machine
- `buildHotfixTransitions()` - Hotfix workflow state machine
- `buildMergebackTransitions()` - Mergeback workflow state machine
- `buildTerminalTransitions()` - Terminal state transitions for workflow looping
- `buildAllTransitions()` - Assembles all transitions into a single event list

Benefits:
- Each workflow is now independently maintainable
- `NewWorktreeAddFSM()` is more concise and focused
- Easier to extend with new workflow types
- Better testability for individual workflow transitions

### 10. **Magic String in Branch Name Generation** ✅
**Severity:** Low
**Location:** `cmd/service/worktree_fsm.go:649`
**Status:** FIXED

```go
if len(baseName) > 7 && baseName[:7] == "HOTFIX_" {
```

**Fix:** Added constant in `fsm_constants.go` and used idiomatic string functions:
```go
const HotfixPrefix = "HOTFIX_"
if strings.HasPrefix(baseName, HotfixPrefix) {
    baseName = strings.TrimPrefix(baseName, HotfixPrefix)
}
```

---

## Idiomatic Go

### 11. **Exported Fields Commentary** ✅
**Severity:** Low
**Location:** `cmd/service/wizard.go:27`
**Status:** FIXED

```go
Steps []WizardStep // Public so we can access step models after wizard completes
```

**Issue:** Comment acknowledges breaking encapsulation.

**Fix Applied:**
- Made `Steps` field private (`steps`)
- Added `GetStep(index int) (WizardStep, error)` accessor method with bounds checking
- Updated all external accesses in `worktree_fsm.go` to use the accessor

### 12. **Context Not Used** ✅
**Severity:** Low
**Location:** `cmd/service/worktree_fsm.go:145-146`
**Status:** FIXED

```go
func (w *WorktreeAddFSM) Run() error {
    ctx := context.Background()
    // ctx passed to FSM but never cancelled or with timeout
```

**Issue:** Context created but not used for cancellation.

**Fix Applied:**
- Changed `Run()` to accept `context.Context` as first parameter (idiomatic Go - contexts should not be stored in structs)
- Added context cancellation check in main workflow loop
- Created context with 30-minute timeout in `runWorktreeAddTUI()` caller using `cmd.Context()` as base
- Uses `cmd.Context()` to inherit cancellation from cobra command framework (e.g., SIGINT/SIGTERM handling)
- Added proper error messages for timeout vs cancellation scenarios
- Follows Go best practice: contexts are passed as function arguments, not embedded in structs

---

## Positive Changes

### ✅ **FSM Refactoring**
- Consolidated 3 separate workflow files into unified FSM
- Reduced complexity from nested loops to state transitions
- Better separation of concerns
- Easier to reason about state flow

### ✅ **Wizard Abstraction**
- Clean abstraction for multi-step forms
- Supports both huh forms and custom Bubble Tea models
- Proper ESC (go back) vs Ctrl+C (cancel) handling

### ✅ **Validation Extraction**
- Validators moved to separate functions
- Reusable across workflows

---

## Summary

**Total Issues:** 12
**Critical:** 3 (all fixed ✅)
**Medium:** 1 (fixed ✅)
**Low:** 9 (all fixed ✅ - 8 actual bugs/improvements, 1 not a bug)

**Fixed Issues:**
1. ✅ Replace hardcoded `"gbm2"` with `os.Args[0]` (#1)
2. ✅ Fix temp file cleanup (#2)
3. ✅ Replace recursive calls with loop/FSM logic (#3)
4. ✅ Add type info to error messages (#4)
5. ✅ Fix variable shadowing (#5)
6. ✅ Remove unused values map field (#6)
7. ✅ Clear FSM state history on terminal states (#7)
9. ✅ Extract FSM transitions into workflow-specific builder functions (#9)
10. ✅ Replace magic string with constant (#10)
11. ✅ Make Steps field private with accessor methods (#11)
12. ✅ Add context timeout and cancellation support (#12)

**Not Bugs (Analysis Corrections):**
8. ❌ Slice copy loop - necessary for type conversion, not inefficient

**Overall Assessment:**
FSM refactoring is a significant improvement over previous nested-loop approach. Code is more maintainable and testable. Main concerns are minor security/correctness issues that are easy to fix.

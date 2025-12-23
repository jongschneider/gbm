# Code Review - Past 5 Commits

**Review Date:** 2025-12-23
**Commits Reviewed:** HEAD~5..HEAD (wip commits)
**Focus Areas:** Bugs, Performance, Security, Correctness, Idiomatic Go, Complexity

---

## Critical Issues

### 1. Security: Temp File Not Cleaned Up (worktree_table.go:243)
**Severity:** High
**Location:** `cmd/service/worktree_table.go:243`

```go
_ = os.WriteFile(tmpFile, []byte(model.switchOutput), 0600)
```

**Issues:**
- Silent error handling - file write failures are ignored
- Temp file is never cleaned up, creating potential information leakage
- File permissions 0600 are good, but orphaned files remain on disk

**Recommendation:**
```go
if err := os.WriteFile(tmpFile, []byte(model.switchOutput), 0600); err != nil {
    return fmt.Errorf("failed to write switch file: %w", err)
}
defer os.Remove(tmpFile) // Cleanup on exit
```

### 2. Security: Shell Eval on Command Output (shell-integration.go)
**Severity:** High
**Location:** `cmd/service/shell-integration.go:47, 67`

```bash
eval "$cd_cmd"  # Execute cd in current shell
```

**Issues:**
- Uses `eval` on subprocess output
- While output is filtered with `grep '^cd '`, this is still risky
- If gbm2 output is ever compromised or has a bug, arbitrary commands could execute

**Recommendation:**
- Extract and validate the directory path directly
- Use shell's `cd` command with validated path, not eval of full command

### 3. Hardcoded Binary Name (worktree_table.go:172)
**Severity:** Medium
**Location:** `cmd/service/worktree_table.go:172`

```go
cmd := exec.Command("gbm2", "wt", "switch", targetWorktree.Name)
```

**Issues:**
- Binary name "gbm2" is hardcoded
- Won't work if binary is renamed or in different location
- Makes code less portable

**Recommendation:**
```go
cmd := exec.Command(os.Args[0], "wt", "switch", targetWorktree.Name)
```

---

## Bugs & Correctness

### 4. Variable Shadowing (filterable_select.go:240)
**Severity:** Medium
**Location:** `cmd/service/filterable_select.go:240`

```go
if finalModel, ok := finalModel.(FilterableSelectModel); ok {
```

**Issue:**
- Shadows outer `finalModel` variable from line 235
- Not technically a bug but reduces readability and increases error potential

**Recommendation:**
```go
if model, ok := finalModel.(FilterableSelectModel); ok {
    if model.cancelled {
        return "", fmt.Errorf("cancelled")
    }
    return model.selected, nil
}
```

### 5. Unused Map Never Populated (wizard.go:29)
**Severity:** Low
**Location:** `cmd/service/wizard.go:29`

```go
values map[string]interface{} // Store values across steps
```

**Issue:**
- `values` map is initialized but `SetValue` is never called
- Dead code that adds confusion

**Recommendation:**
- Remove if not needed, or implement value passing between wizard steps

### 6. Unsafe Environment Variable Appending (worktree_table.go:179)
**Severity:** Low
**Location:** `cmd/service/worktree_table.go:179`

```go
cmd.Env = append(cmd.Environ(), envVars...)
```

**Issue:**
- `cmd.Environ()` copies parent environment
- Appending may create duplicate `GBM_SHELL_INTEGRATION` if already set
- Later values override earlier ones, but creates redundancy

**Recommendation:**
```go
env := os.Environ()
env = append(env, envVars...)
cmd.Env = env
```

### 7. Unused Functions (worktree_tui.go)
**Severity:** Low
**Locations:** Lines 516, 541, 1001

**Functions:**
- `collectBranchNameWithDefault` (line 516)
- `collectBranchNameCustom` (line 541)
- `selectMergebackWorktreeName` (line 1001)
- `filterWorktreesByPrefix` (line 277)

**Issue:**
- Dead code that should be removed or used

**Recommendation:**
- Remove if refactoring made them obsolete
- Add `// Deprecated:` comment if keeping for backward compatibility

---

## Performance

### 8. Inefficient Slice Allocation (filterable_select.go:164-172)
**Severity:** Low
**Location:** `cmd/service/filterable_select.go:164-172`

```go
if query == "" {
    m.filteredList = make([]list.Item, len(m.allItems))
    for i, item := range m.allItems {
        m.filteredList[i] = item
    }
}
```

**Issue:**
- Creates new slice and copies all items even when no filtering needed
- Could just reuse `m.allItems` or use copy()

**Recommendation:**
```go
if query == "" {
    m.filteredList = make([]list.Item, len(m.allItems))
    copy(m.filteredList, m.allItems)
}
```

### 9. Multiple Passes Over Branch Data (worktree_tui.go:126-203)
**Severity:** Low
**Location:** `cmd/service/createSortedBranchItems`

**Issue:**
- Fetches worktrees and config
- Builds multiple maps
- Categorizes into 3 separate slices
- Combines slices
- Could be done in single pass

**Recommendation:**
- Single iteration with immediate categorization
- Pre-allocate slices with capacity hints

---

## Complexity & Maintainability

### 10. Massive File (worktree_tui.go)
**Severity:** Medium
**Location:** `cmd/service/worktree_tui.go` (1025 lines)

**Issue:**
- Single file with 1000+ lines violates SRP
- Contains 4+ different workflow implementations
- Hard to test individual flows
- High cyclomatic complexity

**Recommendation:**
- Split into separate files per workflow:
  - `worktree_feature.go` - feature/bug flows
  - `worktree_hotfix.go` - hotfix flow
  - `worktree_mergeback.go` - mergeback flow
  - `worktree_helpers.go` - shared utilities

### 11. Deep Nesting (worktree_tui.go:299-514)
**Severity:** Medium
**Location:** `createFeatureWorktree` function

**Issue:**
- Triple-nested loops (`for { for { for { ... } } }`)
- Control flow with `break` and `continue` across loop levels
- Very difficult to follow and test

**Recommendation:**
- Extract inner workflows into separate functions with clear return values
- Use state machine pattern or step-based approach
- Consider replacing nested loops with recursive step handler

### 12. Hardcoded TODO (worktree_tui.go:592)
**Severity:** Low
**Location:** `cmd/service/worktree_tui.go:592`

```go
dryRun := false // TODO: Get this from cobra command context if needed
```

**Issue:**
- Dry-run flag is never used despite other commands supporting it
- Inconsistent with rest of codebase

**Recommendation:**
- Implement dry-run support or remove the variable

---

## Idiomatic Go Issues

### 13. Type Assertion Without Error Context (wizard.go:182)
**Severity:** Low
**Location:** `cmd/service/wizard.go:182`

```go
if model, ok := finalModel.(WizardModel); ok {
    return model.completed, model.cancelled, nil
}
return false, false, fmt.Errorf("unexpected model type")
```

**Issue:**
- Error message doesn't indicate what type was actually received
- Makes debugging harder

**Recommendation:**
```go
if model, ok := finalModel.(WizardModel); ok {
    return model.completed, model.cancelled, nil
}
return false, false, fmt.Errorf("unexpected model type: %T", finalModel)
```

### 14. Exported Fields for Internal Use (wizard.go:27)
**Severity:** Low
**Location:** `cmd/service/wizard.go:27`

```go
Steps []WizardStep // Public so we can access step models after wizard completes
```

**Issue:**
- Comment acknowledges this breaks encapsulation
- Exposes internal state unnecessarily
- Better API would be accessor methods

**Recommendation:**
```go
steps []WizardStep

func (m *WizardModel) GetStep(index int) WizardStep {
    if index >= 0 && index < len(m.steps) {
        return m.steps[index]
    }
    return WizardStep{}
}
```

---

## Summary Statistics

**Total Issues:** 14
**Critical:** 3 (Security/Correctness)
**High:** 4 (Bugs/Architecture)
**Medium:** 4 (Performance/Complexity)
**Low:** 3 (Style/Cleanup)

## Recommended Action Items (Priority Order)

1. ✅ Fix temp file cleanup and error handling (Issue #1)
2. ✅ Replace hardcoded binary name with os.Args[0] (Issue #3)
3. ✅ Review shell integration security (Issue #2)
4. ⚠️ Split worktree_tui.go into multiple files (Issue #10)
5. ⚠️ Refactor nested loops in createFeatureWorktree (Issue #11)
6. 🧹 Remove unused functions (Issue #7)
7. 🧹 Fix variable shadowing (Issues #4, #13)
8. 🧹 Implement or remove dry-run TODO (Issue #12)

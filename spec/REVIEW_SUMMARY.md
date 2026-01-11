# TUI Package Review Summary

## Overview

The TUI package is well-architected with clear abstractions (Field interface, Context pattern, Wizard orchestration), but deviates from Bubble Tea best practices in several critical areas, particularly around event loop performance and async operation handling.

## Key Findings

### ✅ Strengths

1. **Clean Interface Design**
   - Field interface properly abstracts form components
   - Context provides dependency injection for services
   - Step-based workflow with skip logic is elegant

2. **Proper Separation of Concerns**
   - Fields handle input/rendering
   - Wizard handles navigation and state
   - Workflows define step configurations

3. **Good Type Safety**
   - Generic Eval[T] for caching
   - Type-safe WorkflowState
   - Concrete field implementations

4. **Flexible Message Routing**
   - BackBoundaryMsg for boundary detection
   - NextStepMsg/PrevStepMsg for navigation
   - Custom messages can be added

### ⚠️ Critical Issues

1. **Event Loop Blocking** (Severity: Critical)
   - `Eval[T].Get()` blocks on first call
   - Wizard calls `FetchIssues()` directly in Update()
   - NO async operations use Bubble Tea commands
   - **Impact**: UI lag during JIRA/Git operations

2. **Message Ordering** (Severity: High)
   - Uses `tea.Batch()` for Init/Focus (unordered)
   - No `tea.Sequence()` for dependent commands
   - Risk of rendering before data loads
   - **Impact**: Race conditions during field transitions

3. **No Root Model** (Severity: High)
   - Manual stage management in wrapper model
   - No global message distribution
   - Window resize not broadcast to all components
   - **Impact**: Poor scalability for multi-screen flows

4. **Missing Error Handling** (Severity: High)
   - Fields always return nil errors
   - Async failures silent (no user feedback)
   - No error recovery mechanism
   - **Impact**: Poor UX, debugging difficulty

5. **No Integration Tests** (Severity: Medium)
   - Only basic unit tests
   - No teatest suite for full workflows
   - Manual testing via testadd only
   - **Impact**: Can't catch regressions

6. **Hard-Coded Field Storage** (Severity: Medium)
   - Wizard.storeFieldValue() has switch statement
   - New fields require code changes
   - Merge workflow can't track custom fields
   - **Impact**: Not extensible

### 📊 Code Quality Metrics

| Metric | Status | Notes |
|--------|--------|-------|
| Test Coverage | 🟡 Low | No teatest, limited unit tests |
| Event Loop Blocking | 🔴 High Risk | Multiple sync calls in Update() |
| Error Handling | 🔴 Missing | Silent failures throughout |
| Async Pattern | 🟡 Partial | Uses Eval but not Bubble Tea commands |
| Documentation | 🟡 Moderate | No architecture docs, some inline comments |
| Extensibility | 🟡 Limited | Field storage hard-coded |

---

## Detailed Issues

### 1. Event Loop Performance (BUBBLETEA §1 Violation)

**Problem**: Synchronous calls block the update loop:

```go
// wizard.go:323-338 - BLOCKS DURING UPDATE
issues, err := w.ctx.JiraService.FetchIssues()  // ← BLOCKING!
if err == nil {
    for _, issue := range issues {
        if issue.Key == worktreeName {
            return generateBranchName(worktreeName, issue.Summary, prefix)
        }
    }
}
```

**Why it matters**:
- Terminal becomes unresponsive to user input
- User sees hanging/frozen UI
- Network latency directly impacts perceived performance

**Solution**: Return a command that fetches async and sends FetchMsg when complete.

---

### 2. Message Ordering Issues (BUBBLETEA §4 Violation)

**Problem**: Uses tea.Batch() for Init and Focus:

```go
// wizard.go:175-177 - UNORDERED EXECUTION
initCmd := w.currentField().Init()
focusCmd := w.currentField().Focus()
return w, tea.Batch(initCmd, focusCmd)  // May execute in any order!
```

**Why it matters**:
- Init may not complete before Focus renders
- Loading state not shown before field is focused
- Async init operations may race with View()

**Solution**: Use tea.Sequence() so Init completes before Focus.

---

### 3. No Navigator/Root Model (BUBBLETEA §5 Violation)

**Problem**: Manual state management in wrapper:

```go
// worktree_testadd.go:51-102 - MANUAL ROUTING
switch m.stage {
case StageTypeSelection:
    // ... manual transition logic ...
case StageWorkflow:
    // ... manual transition logic ...
}
```

**Why it matters**:
- Doesn't scale beyond 2-3 screens
- Window resize not broadcast
- No model caching
- Each new screen requires code changes

**Solution**: Create Navigator that manages model stack and routes messages.

---

### 4. Silent Error Failures

**Problem**: Fields don't report errors:

```go
// fields/selector.go:152-154
func (s *Selector) Error() error {
    return nil  // Always nil!
}
```

And async failures are silent:

```go
// async/eval.go - no user-visible error handling
if msg.Err != nil {
    f.loadErr = msg.Err  // Silently stored
    return f, nil        // No error shown to user
}
```

**Why it matters**:
- Users don't know why operations fail
- Can't retry failures
- Makes debugging hard

**Solution**: Show error messages in Field.View(). Allow retry via SetError() / ClearError().

---

### 5. No Integration Tests

**Problem**: Only basic unit tests, no teatest suite:

```bash
# Current tests
✓ wizard_navigation_test.go - 2 tests
✓ table_test.go - basic tests
✗ NO teatest tests
✗ NO end-to-end workflow tests
```

**Why it matters**:
- Can't verify full wizard flows
- Regressions slip through
- Must test manually via testadd

**Solution**: Create teatest suite with workflow simulations.

---

### 6. Hard-Coded Field Storage

**Problem**: Wizard.storeFieldValue() needs changes for new fields:

```go
// wizard.go:234-255
func (w *Wizard) storeFieldValue() {
    switch key {
    case FieldKeyWorkflowType:
        if v, ok := value.(string); ok {
            w.ctx.State.WorkflowType = v  // ← Hard-coded
        }
    case FieldKeyWorktreeName:
        if v, ok := value.(string); ok {
            w.ctx.State.WorktreeName = v  // ← Hard-coded
        }
    // ... more hard-coded cases ...
    }
}
```

And merge workflow can't store custom fields:

```go
// workflows.go:265-301 - WORKAROUND COMMENT!
// For merge workflows, we'll need to modify the Wizard to track custom fields
return ""  // Can't retrieve source_branch, target_branch
```

**Why it matters**:
- New fields require modifying Wizard
- Merge workflow is broken
- Not extensible

**Solution**: Add CustomFields map to WorkflowState.

---

## Recommended Reading Order

1. **[TUI_IMPROVEMENTS.md](./TUI_IMPROVEMENTS.md)** - Detailed specification with code examples
2. **[BUBBLETEA.md](./BUBBLETEA.md)** - Best practices reference
3. **[../pkg/tui/wizard.go](../pkg/tui/wizard.go)** - Current implementation
4. **[../cmd/service/worktree_testadd.go](../cmd/service/worktree_testadd.go)** - Usage pattern

---

## Next Steps

### For Quick Wins (1-2 hours)
1. [ ] Add error handling to fields (return actual errors)
2. [ ] Create SimpleLoadingSpinner field for async operations
3. [ ] Add teatest tests for wizard navigation

### For Medium Effort (4-8 hours)
4. [ ] Convert async operations to commands (Eval → FetchCmd)
5. [ ] Create Navigator for model composition
6. [ ] Add CustomFields to WorkflowState

### For Long-term (8+ hours)
7. [ ] Full teatest suite with all workflows
8. [ ] Layout helper package with responsive design
9. [ ] Debug message dumping infrastructure
10. [ ] Architecture documentation with diagrams

---

## Files Modified/Created

### Phase 1 Impact
- `pkg/tui/async/messages.go` - NEW (FetchMsg, FetchCmd)
- `pkg/tui/fields/filterable.go` - MODIFIED (use FetchCmd)
- `pkg/tui/fields/textinput.go` - MODIFIED (error handling)
- `pkg/tui/wizard.go` - MODIFIED (use tea.Sequence, remove blocking calls)
- `pkg/tui/context.go` - MODIFIED (CustomFields in WorkflowState)

### Phase 2 Impact
- `pkg/tui/navigator.go` - NEW
- `pkg/tui/wizard_test.go` - NEW (teatest suite)
- `pkg/tui/fields/*_test.go` - MODIFIED (add error tests)
- `cmd/service/worktree_testadd.go` - MODIFIED (use Navigator)

### Phase 3 Impact
- `pkg/tui/layout/layout.go` - NEW
- `spec/TUI_ARCHITECTURE.md` - NEW
- Various field `View()` methods - MODIFIED (use layout helpers)

---

## Risk Assessment

| Item | Risk | Mitigation |
|------|------|-----------|
| Breaking existing wizards | Medium | Phase 1 fully backward compatible |
| Performance regression | Low | Can test with benchmarks |
| Complex refactoring | Medium | Do Navigator separately from async changes |
| Test maintenance | Low | Teatest is maintainable long-term |

---

## Questions for Design Review

1. **Should all async operations return loading UI?** (Current: silent, proposed: show spinner)
2. **Should CustomFields be public API or internal?** (Current: internal proposal)
3. **Should Navigator broadcast window resize?** (Current: manual, proposed: automatic)
4. **Should Debug mode be always-on or opt-in?** (Current: proposed opt-in via env var)

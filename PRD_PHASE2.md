# Phase 2 Implementation PRD: Navigator, Testing, Custom Fields

## Overview

Three interconnected features for Phase 2:
1. **Async Commands & Spinners** - Non-blocking JIRA/Git fetches with visual feedback
2. **Navigator** - Root model for multi-screen composition
3. **Testing & Custom Fields** - Teatest integration + generic field storage

**Deliverable**: Production-ready TUI with best-practice patterns, 80%+ test coverage, VHS recordings.

---

## User Stories

### Story 1: Create Async Message Types and Commands

**Title**: Implement FetchMsg and FetchCmd for non-blocking operations

**Description**:
Create new async message types and command factory in `pkg/tui/async/messages.go` to enable non-blocking fetch operations. This is the foundation for spinners and progress feedback.

**Acceptance Criteria**:
- [ ] `FetchMsg[T]` struct defined with `Value T` and `Err error` fields
- [ ] `FetchCmd[T]` function returns `tea.Cmd` that executes fetch asynchronously
- [ ] Unit tests verify:
  - Successful fetch returns value in FetchMsg
  - Failed fetch returns error in FetchMsg
  - Cmd doesn't block event loop
- [ ] No circular imports
- [ ] Existing `Eval[T]` still works (no breaking changes)

**Files**:
- Create: `pkg/tui/async/messages.go`
- Update: `pkg/tui/async/eval.go` (if refactoring needed)
- Create: `pkg/tui/async/messages_test.go`

**Estimated effort**: 1-2 hours

**Depends on**: None

---

### Story 2: Update Filterable Field to Use Async Commands

**Title**: Refactor Filterable to use FetchCmd + handle FetchMsg

**Description**:
Modify `Filterable` field to accept a fetch function and use the new FetchCmd pattern instead of blocking in Update(). Should show spinner during loading and handle errors gracefully.

**Acceptance Criteria**:
- [ ] `WithOptionsFuncAsync()` method added (or rename existing)
- [ ] `Init()` returns FetchCmd if options func provided
- [ ] `Update()` handles `FetchMsg[[]Option]` - updates options, sets isLoading=false
- [ ] `View()` shows spinner while isLoading=true
- [ ] `View()` shows error message if load fails
- [ ] User can ESC/Ctrl+C while loading
- [ ] Integration tests verify full flow:
  - Init → spinner shows
  - FetchMsg arrives → spinner hidden, options available
  - User can select
  - Error case: FetchMsg with error → error shown
- [ ] Spinner animation smooth (using existing spinner.Tick)
- [ ] No blocking calls in Update()

**Files**:
- Update: `pkg/tui/fields/filterable.go`
- Update: `pkg/tui/fields/filterable_test.go`

**Estimated effort**: 2-3 hours

**Depends on**: Story 1

---

### Story 3: Add VHS Recording Scripts and Baseline GIFs

**Title**: Create VHS demo scripts for all major workflows

**Description**:
Create `.tape` files for each workflow (feature, bug, hotfix, merge) that demonstrate the TUI with async loading, error handling, and successful completion. Generate baseline GIFs to track visual changes.

**Acceptance Criteria**:
- [ ] `spec/vhs/testadd_feature_workflow.tape` - feature workflow with JIRA fetch spinner
- [ ] `spec/vhs/testadd_bug_workflow.tape` - bug workflow
- [ ] `spec/vhs/testadd_hotfix_workflow.tape` - hotfix workflow with base branch
- [ ] `spec/vhs/testadd_merge_workflow.tape` - merge workflow
- [ ] All `.tape` files runnable: `vhs < file.tape` generates `.gif`
- [ ] All generated `.gif` files look reasonable (spinners visible, text readable)
- [ ] All `.tape` and `.gif` files committed to repo
- [ ] `Makefile` or `justfile` target to regenerate: `just vhs-record`

**Files**:
- Create: `spec/vhs/*.tape` (4 files)
- Generate: `spec/vhs/*.gif` (4 files, committed)
- Update: `justfile` with `vhs-record` target

**Estimated effort**: 1-2 hours

**Depends on**: Story 2 (so async/spinner is visible)

---

### Story 4: Set Up Teatest Integration Test Framework

**Title**: Create base theatrical testing utilities and patterns

**Description**:
Set up teatest helper functions and common test patterns for wizard testing. Create reusable test utilities to reduce boilerplate.

**Acceptance Criteria**:
- [ ] `testutil/teatest_helpers.go` created with:
  - `NewTestWizard(steps, ctx)` helper
  - `SendSequence(tm, keys...)` to send multiple keys
  - `GetViewAsString(tm)` to get rendered output as string
  - `WaitForMessage(tm, timeout)` to wait for specific message types
- [ ] Example usage documented in `testutil/README.md`
- [ ] All helpers tested with unit tests
- [ ] Helpers used in subsequent field tests

**Files**:
- Create: `testutil/teatest_helpers.go`
- Create: `testutil/teatest_helpers_test.go`
- Create: `testutil/README.md`

**Estimated effort**: 1-2 hours

**Depends on**: None (can be parallel with Story 2)

---

### Story 5: Add Async Integration Tests for Filterable

**Title**: Create comprehensive teatest suite for async field loading

**Description**:
Add integration tests using teatest that verify the complete async flow: Init → FetchCmd → spinner display → FetchMsg → option selection → NextStepMsg.

**Acceptance Criteria**:
- [ ] Test: Init returns FetchCmd
- [ ] Test: Spinner visible during loading
- [ ] Test: Spinner hidden after data loads
- [ ] Test: Options populated after FetchMsg
- [ ] Test: User can select option after load
- [ ] Test: ESC cancels during loading
- [ ] Test: Error message shown on failed fetch
- [ ] Test: Multiple fields in sequence work (e.g., JIRA select → branch name input)
- [ ] All tests pass with mocked services
- [ ] Coverage > 80% for filterable.go

**Files**:
- Update: `pkg/tui/fields/filterable_test.go`
- Update: `pkg/tui/fields/*_test.go` (other fields as needed)

**Estimated effort**: 2-3 hours

**Depends on**: Stories 1, 2, 4

---

### Story 6: Create Navigator Root Model

**Title**: Implement hierarchical model composition with navigation stack

**Description**:
Create the Navigator model that manages a stack of child models (wizard instances, dialogs, etc.) and routes messages + window events to the current model. Enables clean multi-screen flows.

**Acceptance Criteria**:
- [ ] `Navigator` struct with:
  - `stack []tea.Model`
  - `current tea.Model`
  - `Init() tea.Cmd` - delegates to current
  - `Update(msg) tea.Cmd` - delegates + handles NavigateMsg
  - `View() string` - delegates to current
- [ ] `NavigateMsg` type for pushing new models onto stack
- [ ] `Pop()` method for back navigation
- [ ] Window resize (WindowSizeMsg) broadcast to current model
- [ ] All messages forwarded correctly
- [ ] Unit tests verify stack operations
- [ ] No panics on empty stack

**Files**:
- Create: `pkg/tui/navigator.go`
- Create: `pkg/tui/navigator_test.go`

**Estimated effort**: 1-2 hours

**Depends on**: None

---

### Story 7: Update Testadd to Use Navigator

**Title**: Refactor testadd to use Navigator for type selector → workflow transitions

**Description**:
Replace the manual `testaddWrapperModel` with the new Navigator. Should be cleaner and provide better message routing.

**Acceptance Criteria**:
- [ ] `testaddWrapperModel` replaced with `Navigator`
- [ ] Type selector → feature/bug/hotfix/merge workflows work as before
- [ ] Back navigation (ESC at first step) returns to type selector
- [ ] Forward navigation (Enter at last step) completes wizard
- [ ] No `stage` field needed (Navigator handles it)
- [ ] Code is simpler (fewer lines)
- [ ] Manual test: `go run ./cmd/service testadd --delay 500` works
- [ ] All transitions smooth (no jank)

**Files**:
- Update: `cmd/service/worktree_testadd.go`
- Update: `pkg/tui/workflows/workflows.go` (if SelectWorkflowType needs updates)

**Estimated effort**: 1-2 hours

**Depends on**: Story 6

---

### Story 8: Add Custom Field Storage to WorkflowState

**Title**: Implement generic field storage without Wizard modifications

**Description**:
Add `CustomFields map[string]interface{}` to WorkflowState to support arbitrary fields. Remove the big switch statement from `Wizard.storeFieldValue()`.

**Acceptance Criteria**:
- [ ] `WorkflowState` has `CustomFields map[string]interface{}`
- [ ] `SetField(key string, value interface{})` method added
- [ ] `GetField(key string) interface{}` method added
- [ ] `Wizard.storeFieldValue()` updated:
  - Standard fields (workflow_type, worktree_name, etc.) stored as before
  - Unknown fields stored in CustomFields
  - No switch case needed for new fields
- [ ] Tests verify:
  - Standard fields work as before
  - Custom fields stored and retrieved
  - No overwrites of standard fields
- [ ] Merge workflow updated to use custom fields (e.g., source_branch, target_branch)
- [ ] New fields can be added without touching Wizard

**Files**:
- Update: `pkg/tui/context.go`
- Update: `pkg/tui/wizard.go`
- Create: `pkg/tui/context_test.go` (if not exists)
- Update: `pkg/tui/workflows/workflows.go` (if using custom fields)

**Estimated effort**: 1-2 hours

**Depends on**: Story 6 (Navigator makes refactor easier)

---

### Story 9: Create Merge Workflow with Custom Fields

**Title**: Implement merge workflow using custom fields feature

**Description**:
Create (or update) the merge workflow to demonstrate custom fields. Should handle source/target branch selection without modifying the Wizard.

**Acceptance Criteria**:
- [ ] Merge workflow steps defined:
  1. Select merge strategy (fast-forward, squash, rebase) - custom field
  2. Select source branch - custom field
  3. Select target branch - custom field
  4. Confirmation
- [ ] Steps use async Filterable fields (branch lists from Git service)
- [ ] Custom fields stored in WorkflowState.CustomFields
- [ ] Teatest integration tests for merge flow
- [ ] VHS recording created: `spec/vhs/testadd_merge_custom_workflow.tape`
- [ ] All fields show spinners during branch list loading
- [ ] Error handling if Git service fails

**Files**:
- Create/Update: `pkg/tui/workflows/merge_custom.go` (or similar)
- Update: `pkg/tui/workflows/workflows.go` router
- Create: `pkg/tui/workflows/merge_custom_test.go`
- Create: `spec/vhs/testadd_merge_custom_workflow.tape`

**Estimated effort**: 2-3 hours

**Depends on**: Stories 2, 4, 8

---

### Story 10: Update Wizard with Architecture Documentation

**Title**: Add inline comments explaining best practices and patterns

**Description**:
Add comprehensive inline documentation to Wizard and fields explaining the Bubble Tea patterns used: event loop, message routing, async commands, etc.

**Acceptance Criteria**:
- [ ] Package-level comment in wizard.go explaining architecture
- [ ] Method comments explain message routing (who sends what to whom)
- [ ] Inline comments at tricky points (async command creation, message handling)
- [ ] Reference BUBBLETEA.md best practices
- [ ] Example comment blocks for common patterns:
  - How to create async spinners
  - How to sequence dependent operations
  - How to handle errors in event loop
- [ ] Create `pkg/tui/ARCHITECTURE.md` with:
  - Component diagram (Wizard → Fields → Messages)
  - Message flow diagram
  - Async operation flow diagram
  - Checklist for new fields
- [ ] All comments proofread (no typos)

**Files**:
- Update: `pkg/tui/wizard.go` (add comments)
- Update: `pkg/tui/field.go` (add comments)
- Update: `pkg/tui/fields/filterable.go` (add comments)
- Create: `pkg/tui/ARCHITECTURE.md`

**Estimated effort**: 1-2 hours

**Depends on**: Story 2, 6, 8 (write after implementation)

---

## Implementation Order

**Phase 2A (Foundation)**:
1. Story 1: Async messages
2. Story 4: Teatest helpers
3. Story 2: Update Filterable

**Phase 2B (Testing)**:
5. Story 5: Async integration tests
3. Story 3: VHS recordings

**Phase 2C (Architecture)**:
6. Story 6: Navigator
7. Story 7: Update testadd
8. Story 8: Custom fields
9. Story 9: Merge workflow

**Phase 2D (Polish)**:
10. Story 10: Documentation

---

## Definition of Done

**For each story**:
- [ ] Code changes implemented
- [ ] Unit tests pass (`go test ./... -v`)
- [ ] Integration tests pass (teatest)
- [ ] Coverage maintained or improved (> 80%)
- [ ] VHS recording (if UI-related) committed
- [ ] No regressions in other workflows
- [ ] Code reviewed (by Ralph or manual review)
- [ ] Commit message clear
- [ ] Story marked complete in tracking

**For Phase 2**:
- [ ] All 10 stories complete
- [ ] Full test suite passes
- [ ] All VHS recordings committed
- [ ] Architecture documentation complete
- [ ] No breaking changes to public APIs
- [ ] Ready for production or next phase

---

## Success Metrics

- **Test Coverage**: > 80% for all TUI packages
- **Test Speed**: Full suite < 10 seconds
- **Visual Tests**: VHS recordings for all major workflows
- **Documentation**: All async patterns documented with examples
- **Usability**: Fields show spinners, handle errors, responsive to resize
- **Maintainability**: New fields can be added without Wizard modifications

---

## References

- [TESTING_VALIDATION_STRATEGY.md](./spec/TESTING_VALIDATION_STRATEGY.md)
- [TUI_IMPROVEMENTS.md](./spec/TUI_IMPROVEMENTS.md)
- [BUBBLETEA.md](./spec/BUBBLETEA.md)
- [AGENTS.md](./AGENTS.md)

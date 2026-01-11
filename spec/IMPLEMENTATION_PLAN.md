# Phase 2: Consolidated Implementation Plan

**Status**: Baseline set, ready for Ralph  
**Timeline**: 15-25 hours (Stories 1-10)  
**Test Coverage Target**: >80% for `pkg/tui/`

---

## Quick Reference: The 10 Stories

| # | Story | Files | Tests | VHS | Hours |
|---|-------|-------|-------|-----|-------|
| 1 | Async Messages (FetchMsg/FetchCmd) | `pkg/tui/async/messages.go` + test | ✓ | - | 1-2 | ✅ DONE |
| 2 | Update Filterable with spinner | `pkg/tui/fields/filterable.go` + test | ✓ | - | 2-3 | ✅ DONE |
| 3 | VHS recordings | `spec/vhs/*.tape` + `.gif` | - | ✓ | 1-2 |
| 4 | Teatest helpers | `testutil/teatest_helpers.go` + test | ✓ | - | 1-2 | ✅ DONE |
| 5 | Async integration tests | `pkg/tui/fields/filterable_test.go` + others | ✓ | - | 2-3 |
| 6 | Navigator root model | `pkg/tui/navigator.go` + test | ✓ | - | 1-2 | ✅ DONE |
| 7 | Update testadd to use Navigator | `cmd/service/worktree_testadd.go` | ✓ | - | 1-2 |
| 8 | Custom field storage | `pkg/tui/context.go`, `wizard.go` + test | ✓ | - | 1-2 |
| 9 | Merge workflow w/ custom fields | `pkg/tui/workflows/merge_custom.go` + test | ✓ | ✓ | 2-3 |
| 10 | Documentation | `pkg/tui/ARCHITECTURE.md` + comments | - | - | 1-2 |

---

## Story 1: Async Messages (FetchMsg/FetchCmd) ✅ DONE

**Why**: Foundation for non-blocking operations. Solves [BUBBLETEA.md §1](./BUBBLETEA.md#1-keep-the-event-loop-fast) blocking issue.

**What to build**:
- [x] `pkg/tui/async/messages.go`: `FetchMsg[T]` struct + `FetchCmd[T]()` factory
  - Reference: [TUI_IMPROVEMENTS.md §1.A](./TUI_IMPROVEMENTS.md#a-convert-async-operations-to-commands)
- [x] `pkg/tui/async/messages_test.go`: Test successful/failed fetch, non-blocking behavior
- [x] No breaking changes to existing `Eval[T]`

**Acceptance Criteria** (from [PRD_PHASE2.md §Story 1](../PRD_PHASE2.md#story-1-create-async-message-types-and-commands)):
- FetchMsg has Value + Err fields ✓
- FetchCmd returns tea.Cmd ✓
- Tests verify async (non-blocking) behavior ✓
- No circular imports ✓

**Test Framework**: Go testing + testify  
**Dependencies**: None  
**Completed**: 2026-01-11

---

## Story 2: Update Filterable Field with Async + Spinner ✅ DONE

**Why**: Demonstrates FetchCmd pattern in real field. Fixes [REVIEW_SUMMARY.md](./REVIEW_SUMMARY.md) blocking JIRA/Git fetches.

**What to build**:
- [x] `pkg/tui/fields/filterable.go`: 
  - Add `WithOptionsFuncAsync()` method (or refactor existing)
  - Init() returns FetchCmd if options func provided
  - Update() handles `FetchMsg[[]Option]` → updates options + sets isLoading=false
  - View() shows spinner while isLoading=true (use existing `spinner.Tick`)
  - View() shows error message if load fails
  - User can ESC/Ctrl+C while loading
  - Reference: [TUI_IMPROVEMENTS.md §1.B](./TUI_IMPROVEMENTS.md#b-update-fields-to-use-async-commands)
- [x] `pkg/tui/fields/filterable_async_test.go`: Comprehensive unit tests

**Acceptance Criteria** (from [PRD_PHASE2.md §Story 2](../PRD_PHASE2.md#story-2-update-filterable-field-to-use-async-commands)):
- WithOptionsFuncAsync method works ✓
- Init returns FetchCmd ✓
- Update handles FetchMsg correctly ✓
- Spinner visible while loading ✓
- Error shown on failure ✓
- No blocking in Update() ✓

**Test Framework**: Testify + unit tests (teatest comes in Story 4-5)  
**Dependencies**: Story 1  
**Completed**: 2026-01-11

---

## Story 3: VHS Recording Scripts

**Why**: Visual validation of async behavior. Demonstrates [TESTING_VALIDATION_STRATEGY.md Layer 3](./TESTING_VALIDATION_STRATEGY.md#layer-3-visual-validation-vhs-recordings).

**What to build**:
- [ ] `spec/vhs/testadd_feature_workflow.tape`: Feature workflow with JIRA async spinner
- [ ] `spec/vhs/testadd_bug_workflow.tape`: Bug workflow
- [ ] `spec/vhs/testadd_hotfix_workflow.tape`: Hotfix with base branch
- [ ] `spec/vhs/testadd_merge_workflow.tape`: Merge workflow
- [ ] Run `vhs < *.tape` to generate `.gif` files (committed to repo)
- [ ] Add `justfile` target: `just vhs-record`

**Acceptance Criteria** (from [PRD_PHASE2.md §Story 3](../PRD_PHASE2.md#story-3-add-vhs-recording-scripts-and-baseline-gifs)):
- 4 tape files runnable ✓
- 4 gifs generated and committed ✓
- Spinners visible in recordings ✓
- justfile target works ✓

**Test Framework**: VHS  
**Dependencies**: Story 2 (need async/spinner visible)

---

## Story 4: Teatest Integration Test Helpers ✅ DONE

**Why**: Reduce boilerplate in teatest. Foundation for comprehensive tests (Story 5).

**What to build**:
- [x] `testutil/teatest_helpers.go`:
  - `NewTestWizard(steps, ctx)` helper
  - `SendKeySequence(model, keys...)` to send multiple keys
  - `GetViewAsString(model)` to get rendered output
  - `UpdateWithKeyMsg/UpdateWithMsg` for updating models
  - `ViewContains` to check rendered output
  - Reference: [TESTING_VALIDATION_STRATEGY.md Layer 2](./TESTING_VALIDATION_STRATEGY.md#layer-2-integration-testing-go-test--teatest)
- [x] `testutil/teatest_helpers_test.go`: Comprehensive test suite
- [ ] `testutil/README.md`: Usage examples (deferred)

**Acceptance Criteria** (from [PRD_PHASE2.md §Story 4](../PRD_PHASE2.md#story-4-set-up-teatest-integration-test-framework)):
- All helpers implemented ✓
- Helpers tested ✓
- Key message conversion tested ✓
- Support for model updates ✓

**Test Framework**: Testify  
**Dependencies**: None (parallel with Story 2)  
**Completed**: 2026-01-11

---

## Story 5: Async Integration Tests for Filterable

**Why**: Validate complete async flow (Init → spinner → FetchMsg → data). Covers [TESTING_VALIDATION_STRATEGY.md](./TESTING_VALIDATION_STRATEGY.md) pain point 1.

**What to build**:
- [ ] `pkg/tui/fields/filterable_test.go`: Add teatest integration tests
  - Init returns FetchCmd ✓
  - Spinner visible during loading ✓
  - Spinner hidden after FetchMsg ✓
  - Options populated after load ✓
  - User can select after load ✓
  - ESC cancels during loading ✓
  - Error message shown on failure ✓
  - Multiple fields in sequence work ✓
- [ ] Coverage > 80% for filterable.go

**Acceptance Criteria** (from [PRD_PHASE2.md §Story 5](../PRD_PHASE2.md#story-5-add-async-integration-tests-for-filterable)):
- 8 test cases pass ✓
- All use teatest helpers ✓
- Coverage > 80% ✓

**Test Framework**: Testify + teatest  
**Dependencies**: Stories 1, 2, 4

---

## Story 6: Navigator Root Model ✅ DONE

**Why**: Enable multi-screen composition. Solves [REVIEW_SUMMARY.md](./REVIEW_SUMMARY.md) "No root model" issue. Reference: [TUI_IMPROVEMENTS.md §3.A](./TUI_IMPROVEMENTS.md#a-create-a-root-model-architecture)

**What to build**:
- [x] `pkg/tui/navigator.go`:
  - Navigator struct with stack: `[]tea.Model`
  - `Init() tea.Cmd` delegates to current
  - `Update(msg) tea.Cmd` delegates + handles NavigateMsg
  - `View() string` delegates to current
  - `NavigateMsg` type for pushing models
  - `Pop()` method for back navigation
  - WindowSizeMsg broadcast to current model
  - No panics on empty stack
- [x] `pkg/tui/navigator_test.go`: 15+ unit tests for stack operations

**Acceptance Criteria** (from [PRD_PHASE2.md §Story 6](../PRD_PHASE2.md#story-6-create-navigator-root-model)):
- All methods implemented ✓
- NavigateMsg routing works ✓
- Pop removes from stack ✓
- WindowSizeMsg forwarded ✓
- Tests verify no panics ✓

**Test Framework**: Testify  
**Dependencies**: None  
**Completed**: 2026-01-11

---

## Story 7: Update Testadd to Use Navigator

**Why**: Simplify multi-screen flow. Cleaner than manual stage management.

**What to build**:
- [ ] `cmd/service/worktree_testadd.go`: Replace `testaddWrapperModel` with Navigator
  - Type selector → workflow transitions use Navigator
  - ESC at first step returns to type selector
  - Enter at last step completes wizard
  - No `stage` field needed
- [ ] `pkg/tui/workflows/workflows.go`: Update if SelectWorkflowType needs changes
- [ ] Manual test: `go run ./cmd/service testadd --delay 500` works

**Acceptance Criteria** (from [PRD_PHASE2.md §Story 7](../PRD_PHASE2.md#story-7-update-testadd-to-use-navigator)):
- testaddWrapperModel replaced ✓
- Type → workflow transitions work ✓
- Back navigation works ✓
- Code simpler (fewer lines) ✓
- Manual test passes ✓

**Test Framework**: Manual + integration  
**Dependencies**: Story 6

---

## Story 8: Custom Field Storage in WorkflowState

**Why**: Add arbitrary fields without modifying Wizard. Enables extensibility.

**What to build**:
- [ ] `pkg/tui/context.go`: Add `CustomFields map[string]interface{}` to WorkflowState
  - `SetField(key, value)` method
  - `GetField(key) interface{}` method
- [ ] `pkg/tui/wizard.go`: Update `storeFieldValue()`
  - Standard fields stored as before
  - Unknown fields stored in CustomFields
  - No switch case needed for new fields
- [ ] `pkg/tui/context_test.go`: Tests for custom field storage/retrieval
- [ ] `pkg/tui/workflows/workflows.go`: Merge workflow updated to use custom fields

**Acceptance Criteria** (from [PRD_PHASE2.md §Story 8](../PRD_PHASE2.md#story-8-add-custom-field-storage-to-workflowstate)):
- CustomFields map added ✓
- SetField/GetField methods work ✓
- Wizard doesn't need switch cases ✓
- Standard fields still work ✓
- Tests verify custom field storage ✓

**Test Framework**: Testify  
**Dependencies**: Story 6 (makes refactor easier)

---

## Story 9: Merge Workflow with Custom Fields

**Why**: Reference implementation using all three pattern types: async, custom fields, Navigator.

**What to build**:
- [ ] `pkg/tui/workflows/merge_custom.go`: Merge workflow
  - Step 1: Select merge strategy (custom field)
  - Step 2: Select source branch (async Filterable)
  - Step 3: Select target branch (async Filterable)
  - Step 4: Confirmation
- [ ] `pkg/tui/workflows/merge_custom_test.go`: Teatest suite
- [ ] `spec/vhs/testadd_merge_custom_workflow.tape`: VHS recording
- [ ] `pkg/tui/workflows/workflows.go`: Register merge workflow in router

**Acceptance Criteria** (from [PRD_PHASE2.md §Story 9](../PRD_PHASE2.md#story-9-create-merge-workflow-with-custom-fields)):
- Merge workflow steps defined ✓
- Async Filterable for branches ✓
- Custom fields stored ✓
- Teatest suite passes ✓
- VHS recording created ✓
- Spinners visible during load ✓

**Test Framework**: Testify + teatest + VHS  
**Dependencies**: Stories 2, 4, 8

---

## Story 10: Architecture Documentation

**Why**: Explain patterns for next developers. Reference: [BUBBLETEA.md](./BUBBLETEA.md).

**What to build**:
- [ ] `pkg/tui/ARCHITECTURE.md`:
  - Component diagram (Wizard → Fields → Messages)
  - Message flow diagram
  - Async operation flow
  - Checklist for adding new fields
- [ ] Add inline comments to:
  - `pkg/tui/wizard.go`: Message routing explanation
  - `pkg/tui/field.go`: Field interface contract
  - `pkg/tui/fields/filterable.go`: Async pattern example
- [ ] Reference BUBBLETEA.md best practices
- [ ] Example blocks for:
  - How to create async spinners
  - How to sequence dependent operations
  - How to handle errors

**Acceptance Criteria** (from [PRD_PHASE2.md §Story 10](../PRD_PHASE2.md#story-10-update-wizard-with-architecture-documentation)):
- ARCHITECTURE.md complete ✓
- Comments added to key functions ✓
- Examples provided ✓
- References to BUBBLETEA.md ✓
- Proofread (no typos) ✓

**Test Framework**: Documentation review  
**Dependencies**: Stories 2, 6, 8 (write after implementation)

---

## Execution Order

**Phase 2A (Foundation)** [3-5 hours]:
1. Story 1: Async messages
2. Story 4: Teatest helpers
3. Story 2: Update Filterable

**Phase 2B (Testing)** [4-7 hours]:
5. Story 3: VHS recordings (parallel: Story 5)
5. Story 5: Async integration tests

**Phase 2C (Architecture)** [4-6 hours]:
6. Story 6: Navigator
7. Story 7: Update testadd
8. Story 8: Custom fields

**Phase 2D (Implementation)** [3-5 hours]:
9. Story 9: Merge workflow
10. Story 10: Documentation

**Total**: 15-25 hours (fully autonomous with Ralph)

---

## Pain Points → Solutions Map

| Pain Point | Stories | Reference |
|------------|---------|-----------|
| Async + spinners blocking event loop | 1, 2, 5 | [REVIEW_SUMMARY.md](./REVIEW_SUMMARY.md) + [TUI_IMPROVEMENTS.md §1](./TUI_IMPROVEMENTS.md#1-event-loop-performance) |
| Hard to validate TUI + feedback loop | 3, 4, 5 | [TESTING_VALIDATION_STRATEGY.md](./TESTING_VALIDATION_STRATEGY.md) |
| Terminal resizes + responsive design | 2, 5 | [TESTING_VALIDATION_STRATEGY.md](./TESTING_VALIDATION_STRATEGY.md#pain-point-3-terminal-resizes--responsive-feedback) |
| Don't understand Bubble Tea patterns | 6, 9, 10 | [BUBBLETEA.md](./BUBBLETEA.md) + code as documentation |

---

## Code Quality Checklist (per story)

- [ ] Follows [AGENTS.md](../AGENTS.md) error handling (context + %w)
- [ ] Uses guard clauses + early returns (no deep nesting)
- [ ] Interfaces injected (loose coupling)
- [ ] Tests use table-driven pattern with testify
- [ ] Coverage > 80% for changed files
- [ ] No errors ignored with blank identifiers
- [ ] go fmt + go vet pass
- [ ] Commit message clear + references story

---

## Definition of Done

**Per Story**:
- [ ] Code changes implemented
- [ ] All tests pass: `go test ./...`
- [ ] Coverage > 80% for changed files
- [ ] VHS recording (if applicable) committed
- [ ] No regressions in existing workflows
- [ ] Commit with clear message

**Phase 2 Complete**:
- [ ] All 10 stories merged
- [ ] Full test suite passes
- [ ] All VHS recordings committed
- [ ] Architecture docs complete
- [ ] Ready for production or Phase 3

---

## Key References

- **Best Practices**: [spec/BUBBLETEA.md](./BUBBLETEA.md)
- **Issues Found**: [spec/REVIEW_SUMMARY.md](./REVIEW_SUMMARY.md)
- **Detailed Spec**: [spec/TUI_IMPROVEMENTS.md](./TUI_IMPROVEMENTS.md)
- **Testing Strategy**: [spec/TESTING_VALIDATION_STRATEGY.md](./TESTING_VALIDATION_STRATEGY.md)
- **Code Guidelines**: [../AGENTS.md](../AGENTS.md)
- **User Stories**: [../PRD_PHASE2.md](../PRD_PHASE2.md)

---

## Success Metrics

| Metric | Target | Validation |
|--------|--------|-----------|
| Test Coverage | > 80% | `go test ./... -cover` |
| Test Speed | < 10s | Full suite runtime |
| VHS Recordings | All workflows | `ls spec/vhs/*.gif` |
| No Blocking Calls | 100% | Code review + linting |
| Documentation | Complete | Read ARCHITECTURE.md + comments |

---

**Last Updated**: 2026-01-11  
**Status**: ✅ Ready for Ralph  
**Timeline**: 15-25 hours (mostly autonomous)

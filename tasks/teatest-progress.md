# Teatest Coverage Progress

## Completed Stories

### TT-001: Wizard multi-step navigation (Priority 1)
- **Status**: COMPLETE
- **Tests**: `pkg/tui/wizard_teatest_test.go`
- **Acceptance Criteria**:
  - [x] Test Enter key advances to next step
  - [x] Test Esc key goes back to previous step
  - [x] Test Ctrl+C cancels wizard and sets IsCancelled()
  - [x] Test wizard state updates correctly between steps
  - [x] Test View() delegates to current field's View()

### TT-002: Wizard skip logic (Priority 1)
- **Status**: COMPLETE
- **Tests**: `pkg/tui/wizard_teatest_test.go`
- **Acceptance Criteria**:
  - [x] Test steps with Skip func returning true are skipped on forward navigation
  - [x] Test steps with Skip func returning false are shown
  - [x] Test skipped steps are also skipped on backward navigation
  - [x] Test skip logic re-evaluates based on current workflow state

### TT-003: Wizard completion flow (Priority 1)
- **Status**: COMPLETE
- **Tests**: `pkg/tui/wizard_teatest_test.go`
- **Acceptance Criteria**:
  - [x] Test WorkflowCompleteMsg sent after last step Enter
  - [x] Test IsComplete() returns true after completion
  - [x] Test all field values are stored in workflow state
  - [x] Test wizard handles empty steps list gracefully

### TT-025: View() newline consistency verification (Priority 1)
- **Status**: COMPLETE
- **Tests**: `pkg/tui/fields/fields_view_teatest_test.go`
- **Note**: Completed previously

### TT-008: Filterable Enter selection (Priority 2)
- **Status**: COMPLETE
- **Tests**: `pkg/tui/fields/fields_view_teatest_test.go`
- **Acceptance Criteria**:
  - [x] Test Enter selects currently highlighted option
  - [x] Test selected value is stored in field
  - [x] Test NextStepMsg is sent after selection
  - [x] Test IsComplete() returns true after selection

### TT-013: TextInput typing and submission (Priority 2)
- **Status**: COMPLETE
- **Tests**: `pkg/tui/fields/fields_view_teatest_test.go`
- **Acceptance Criteria**:
  - [x] Test typing characters updates input value
  - [x] Test Enter submits current value
  - [x] Test submitted value is trimmed
  - [x] Test NextStepMsg is sent after submission
  - [x] Test IsComplete() returns true after submission

### TT-017: Confirm Enter submission (Priority 2)
- **Status**: COMPLETE
- **Tests**: `pkg/tui/fields/fields_view_teatest_test.go`
- **Acceptance Criteria**:
  - [x] Test Enter with Yes selected sends NextStepMsg
  - [x] Test Enter with No selected sends CancelMsg
  - [x] Test GetValue() returns correct boolean
  - [x] Test IsCancelled() reflects No selection

### TT-022: FeatureWorkflow end-to-end (Priority 1)
- **Status**: COMPLETE
- **Tests**: `pkg/tui/workflows/workflows_teatest_test.go`
- **Acceptance Criteria**:
  - [x] Test step 1: JIRA issue/custom name selection
  - [x] Test step 2: Branch name input with auto-generated default
  - [x] Test step 3: Base branch selection (or skip if branch exists)
  - [x] Test step 4: Confirm step shows correct summary
  - [x] Test final state contains all expected values
- **Notes**: Fixed test model to process CancelMsg through wizard before quitting, ensuring wizard.IsCancelled() is set correctly

### TT-004: Wizard back boundary handling (Priority 2)
- **Status**: COMPLETE
- **Tests**: `pkg/tui/wizard_teatest_test.go`
- **Acceptance Criteria**:
  - [x] Test BackBoundaryMsg sent when Esc pressed at step 0
  - [x] Test wizard remains at step 0 after BackBoundaryMsg
  - [x] Test current field remains focused after BackBoundaryMsg

### TT-006: Navigator stack push/pop (Priority 2)
- **Status**: COMPLETE
- **Tests**: `pkg/tui/navigator_teatest_test.go`
- **Acceptance Criteria**:
  - [x] Test Push() adds model to stack
  - [x] Test Pop() removes and returns top model
  - [x] Test Depth() reflects current stack size
  - [x] Test Current() returns top model
  - [x] Test empty stack returns nil from Current()

### TT-007: Navigator message-based navigation (Priority 2)
- **Status**: COMPLETE
- **Tests**: `pkg/tui/navigator_teatest_test.go`
- **Acceptance Criteria**:
  - [x] Test NavigateMsg pushes target model onto stack
  - [x] Test target model's Init() is called after push
  - [x] Test View() delegates to newly pushed model
  - [x] Test Update() delegates to current model

### TT-009: Filterable custom value entry (Priority 2)
- **Status**: COMPLETE
- **Tests**: `pkg/tui/fields/fields_view_teatest_test.go`
- **Acceptance Criteria**:
  - [x] Test typing non-matching text shows 'No matches' message
  - [x] Test Enter with no matches uses text input as custom value
  - [x] Test custom value is trimmed before storing
  - [x] Test empty input is handled gracefully

### TT-010: Filterable arrow key navigation (Priority 2)
- **Status**: COMPLETE
- **Tests**: `pkg/tui/fields/fields_view_teatest_test.go`
- **Acceptance Criteria**:
  - [x] Test Up arrow moves cursor up
  - [x] Test Down arrow moves cursor down
  - [x] Test cursor wraps from top to bottom
  - [x] Test cursor wraps from bottom to top
  - [x] Test viewport scrolls when cursor moves out of view
- **Notes**: Also includes tests for Ctrl+J/Ctrl+K navigation, empty list handling, single option behavior, and navigation after filtering

### TT-011: Selector Enter selection (Priority 2)
- **Status**: COMPLETE
- **Tests**: `pkg/tui/fields/fields_view_teatest_test.go`
- **Acceptance Criteria**:
  - [x] Test Enter selects currently highlighted option
  - [x] Test selected value matches option's Value field
  - [x] Test NextStepMsg is sent after selection
  - [x] Test IsComplete() returns true after selection
- **Notes**: Also includes tests for empty options list handling and selection after wrapping navigation

### TT-016: Confirm y/n shortcut keys (Priority 2)
- **Status**: COMPLETE
- **Tests**: `pkg/tui/fields/fields_view_teatest_test.go`
- **Acceptance Criteria**:
  - [x] Test y key immediately confirms with true
  - [x] Test Y key immediately confirms with true
  - [x] Test n key immediately cancels with false
  - [x] Test N key immediately cancels with false
  - [x] Test CancelMsg is sent on n/N
- **Notes**: Also includes tests verifying y/n shortcuts work regardless of current button selection

### TT-014: TextInput validation (Priority 2)
- **Status**: COMPLETE
- **Tests**: `pkg/tui/fields/fields_view_teatest_test.go`
- **Acceptance Criteria**:
  - [x] Test validator function is called on Enter
  - [x] Test validation error prevents submission
  - [x] Test error message is displayed in View()
  - [x] Test error clears when user types
  - [x] Test valid input after error succeeds
- **Notes**: Also includes tests for validation against trimmed values and behavior without a validator

### TT-021: SelectWorkflowType selection (Priority 2)
- **Status**: COMPLETE
- **Tests**: `pkg/tui/workflows/workflows_teatest_test.go`
- **Acceptance Criteria**:
  - [x] Test all 4 workflow types are displayed (Feature, Bug, Hotfix, Merge)
  - [x] Test arrow navigation between options
  - [x] Test Enter selects workflow type
  - [x] Test GetValue() returns correct workflow type string
- **Notes**: Tests verify cursor wrapping, up/down navigation, and that GetValue() returns the correct workflow type constant (e.g., "feature", "bug", "hotfix", "merge")

### TT-023: HotfixWorkflow end-to-end (Priority 2)
- **Status**: COMPLETE
- **Tests**: `pkg/tui/workflows/workflows_teatest_test.go`
- **Acceptance Criteria**:
  - [x] Test step 1: JIRA issue/custom name selection
  - [x] Test step 2: Base branch selection (mandatory, not skipped)
  - [x] Test step 3: Branch name input with hotfix/ prefix
  - [x] Test step 4: Confirm step shows correct summary
  - [x] Test worktree name gets HOTFIX_ prefix (via ProcessHotfixWorkflow)
- **Notes**: Tests verify hotfix workflow differs from feature workflow: base branch is mandatory (step 2, no skip logic), branch name comes after base branch selection. Includes tests for back navigation and custom worktree names.

### TT-024: MergeWorkflow end-to-end (Priority 2)
- **Status**: COMPLETE
- **Tests**: `pkg/tui/workflows/workflows_teatest_test.go`
- **Acceptance Criteria**:
  - [x] Test step 1: Source branch selection
  - [x] Test step 2: Target branch selection
  - [x] Test step 3: Confirm step shows merge details
  - [x] Test auto-generated worktree name (MERGE_{source-to-target})
  - [x] Test auto-generated branch name (merge/{source-to-target})
- **Notes**: Tests verify merge workflow stores custom fields ("source_branch", "target_branch") in WorkflowState.CustomFields. Also fixed Wizard.storeFieldValue() to store unknown keys to CustomFields for flexibility with custom workflow fields. Includes tests for arrow navigation, filtering, and back navigation.

### TT-005: Wizard window resize propagation (Priority 3)
- **Status**: COMPLETE
- **Tests**: `pkg/tui/wizard_teatest_test.go`
- **Acceptance Criteria**:
  - [x] Test WindowSizeMsg updates wizard context width/height
  - [x] Test current field receives WithWidth/WithHeight calls
  - [x] Test field re-renders with new dimensions
- **Notes**: Added `resizeTrackingField` test helper that tracks all width/height calls. Tests verify context dimensions are updated, field methods are called with correct values, and resize doesn't interfere with step navigation. Includes test for multiple consecutive resize messages.

## In Progress

None

## Pending (Priority 1)

None - All priority 1 items complete!

## Summary
- Priority 1 items: 5 of 5 complete
- Total items: 21 of 25 complete

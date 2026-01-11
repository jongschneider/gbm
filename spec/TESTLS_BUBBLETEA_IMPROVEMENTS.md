# TUI Improvements for testlsModel (Bubble Tea Best Practices)

## Objective
Apply BUBBLETEA.md principles to `worktree_testls.go`:
1. Enable message dumping for debugging
2. Add per-row async git operations (pull/push/delete)
3. Add operation state tracking per row
4. Add teatest integration tests
5. Verify message ordering doesn't require tea.Sequence()

---

## User Stories (Ordered by Implementation Dependency)

### US-1: Add Message Dumping to testlsModel
**Size:** XS | **File:** `cmd/service/worktree_testls.go`

- Add `messageDump io.Writer` field to `testlsModel` struct (after line 23)
- In `Update()`, dump all messages to `messageDump` if non-nil (start of switch at line 81)
- In `runTestLS()`, open debug log file if `DEBUG=1` env var is set (line 273)
- Pass file handle to model initialization (line 305)
- Defer close file handle (around line 319)
- **Acceptance:** `DEBUG=1 gbm testls & tail -f messages.log` shows all messages

---

### US-2: Create Per-Row Operation State Types
**Size:** XS | **File:** `cmd/service/worktree_testls.go`

- Add `operationState` struct before `mockWorktree` (around line 26):
  ```go
  type operationState struct {
      operation string    // "pull", "push", "delete", ""
      result    string    // Result message; empty = not started/cleared
      clearAt   time.Time // When to clear result (after 2 seconds)
  }
  ```
- Add `operationStates map[int]operationState` field to `testlsModel` (after line 23)
- Initialize in `runTestLS()` (line 310)
- **Acceptance:** Code compiles, new field initialized
- **Note:** `async.Cell` handles loading spinners; `operationState` tracks operation type + result message only

---

### US-3: Add Per-Row Async Operation Cells
**Size:** S | **File:** `cmd/service/worktree_testls.go`

- Add `asyncOperations map[int]*async.Cell[string]` field to `testlsModel` (after line 23)
- Initialize in `runTestLS()` (line 310)
- In `Init()`, create placeholder cells for each row (after line 65):
  - Don't start them yet—only start on keypress (l/p/d)
- In `Update()` tickMsg handler (line 91-99): tick all async operation cells too
- **Acceptance:** Code compiles, operation cells exist but are not loading

---

### US-4: Trigger Pull/Push/Delete Operations on Keypress
**Size:** M | **File:** `cmd/service/worktree_testls.go`

- Modify KeyMsg handler cases "l", "p", "d" (lines 109-123):
  - Instead of writing to stderr, call helper method `m.triggerOperation(rowIdx, op string)` (new)
  - Helper creates fresh async.Cell with mock git service call (based on operation type)
  - Helper calls `cell.StartLoading()` and returns cmd to queue
  - Store cmd in cmds slice
  - Update operationState[rowIdx] to mark inProgress=true
- Create mock git service methods: `MockTableGitService.Pull()`, `Push()`, `Delete()` (around line 223)
- **Acceptance:** Pressing l/p/d on a row starts async operation, table shows spinner in that row

---

### US-5: Display Operation Results in Table or Footer
**Size:** S | **File:** `cmd/service/worktree_testls.go`

- Modify `updateTableRows()` to append operation result to git status cell if operation state exists (line 169)
- Modify `View()` help text to show current operation row and result (line 207)
- In `Update()` async.CellLoadedMsg handler (line 87-89):
  - Update operationStates[rowIdx] with result
  - Call `m.updateTableRows()`
- **Acceptance:** After operation completes, result displays for 1-2 seconds then clears

---

### US-6: Clear Operation State After Delay
**Size:** S | **File:** `cmd/service/worktree_testls.go`

- Add `operationClearCmd` that waits 2s then returns `clearOperationMsg{rowIdx int}`
- In `Update()` tickMsg handler, check for any pending clearOperationMsg
- Clear operationState[rowIdx] when message arrives
- Call clearOperationCmd when operation completes (line 87-89 handler)
- **Acceptance:** Operation result displays for 2 seconds then disappears

---

### US-7: Write teatest Integration Tests
**Size:** M | **File:** `cmd/service/worktree_testls_test.go` (new)

- Create new test file following BUBBLETEA.md section 8 pattern
- Test 1: Initial render shows table with spinners
  - `tm.WaitFor()` checks for table header
- Test 2: Press 'l' on first row triggers pull
  - Check operationState reflects inProgress
  - Check spinner appears
- Test 3: Operation completes and result displays
  - Mock short delay
  - Check result text appears
- Test 4: Quit with 'q' exits cleanly
- **Acceptance:** `go test ./cmd/service -v` passes all 4 tests

---

### US-8: Verify Message Ordering (No tea.Sequence() Needed)
**Size:** XS | **File:** Specification/Analysis

- Run `DEBUG=1 gbm testls --delay 100` manually
- Press multiple keys in rapid succession (e.g., 'l', 'p', 'd' on different rows)
- Tail messages.log and verify:
  - KeyMsg always arrives before CellLoadedMsg for same row
  - Operations don't need strict ordering (concurrent ops OK)
- **Result:** Confirm `tea.Batch()` is sufficient, no need for `tea.Sequence()`
- **Acceptance:** Manual test confirms concurrent operations work without ordering

---

## Implementation Order
1. ✅ US-1 (message dumping) — enables debugging for all subsequent work
2. ✅ US-2 (operation state types) — foundation
3. ✅ US-3 (async operation cells) — async infrastructure
4. ✅ US-4 (trigger operations) — core feature
5. ✅ US-5 (display results) — UI feedback
6. ✅ US-6 (clear state) — polish
7. ⏳ US-7 (tests) — coverage
8. US-8 (verify ordering) — validation

---

## Files to Modify
- `cmd/service/worktree_testls.go` (main implementation)
- `cmd/service/worktree_testls_test.go` (new file for tests)

## Files to Check
- `pkg/tui/async/cell.go` — already has `Tick()`, `IsLoading()`, `View()`
- `pkg/tui/async/eval.go` — ensure `Invalidate()` works for retries if needed

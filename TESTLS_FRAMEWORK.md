# gbm2 wt testls Testing Framework

## Overview
Complete testing framework for validating the `gbm2 wt testls` command which tests the TUI Table component with async data loading, navigation, and operations.

## Setup
```bash
# Build and install gbm2
just install

# Verify installation
gbm2 --version
gbm2 wt testls --help
```

## Test Environment
- **Command**: `gbm2 wt testls [--delay <ms>]`
- **Default delay**: 1000ms (shows spinner animation)
- **Test command**: `gbm2 wt testls --delay 2000` (2s delay for manual observation)

## Mock Data
8 worktrees with mix of tracked and ad-hoc:
1. `main` (branch: main) - **tracked**
2. `feature/auth` (branch: feature/auth) - ad-hoc
3. `bugfix/login` (branch: bugfix/login) - ad-hoc
4. `wip/dashboard` (branch: wip/dashboard) - ad-hoc
5. `hotfix/crash` (branch: hotfix/crash) - ad-hoc
6. `release/v1.0` (branch: release/v1.0) - **tracked**
7. `experiment/ml` (branch: experiment/ml) - ad-hoc
8. `archived/old` (branch: archived/old) - ad-hoc

---

## Test Case Suite

### TC-001: Application Launch and Display
**Purpose**: Verify table displays with correct structure and initial state

**Steps**:
1. Run: `gbm2 wt testls --delay 2000`
2. Observe table renders with 4 columns: Name, Branch, Kind, Git Status
3. Table shows all 8 worktrees
4. Git Status column shows spinner animation (cycling through `⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏`)
5. Cursor is on row 1 (main)
6. Help footer visible at bottom

**Expected Results**:
- ✓ Table renders correctly with proper formatting
- ✓ All 8 rows visible
- ✓ Spinner animates in Git Status column
- ✓ Help text displays at bottom
- ✓ Application is responsive to input

---

### TC-002: Navigation - Up/Down Arrows
**Purpose**: Verify cursor navigation with wrapping

**Steps**:
1. Launch: `gbm2 wt testls --delay 2000`
2. Press `↓` (down arrow) - cursor moves to row 2
3. Press `↓` 6 more times - cursor moves through rows 3-8
4. Press `↓` again - verify cursor wraps to row 1 (or stays at bottom)
5. Press `↑` (up arrow) - cursor moves to row 7
6. Press `↑` 6 more times - cursor moves through rows 6-1
7. Press `↑` again - verify cursor wraps to row 8 (or stays at top)

**Expected Results**:
- ✓ Cursor moves up with `↑`
- ✓ Cursor moves down with `↓`
- ✓ Navigation is smooth and responsive
- ✓ Cursor position updates in header/footer indicators
- ✓ Spinner continues animating on unselected rows

---

### TC-003: Footer Help Text - Dynamic Content
**Purpose**: Verify help text shows/hides push option based on tracked status

**Steps**:
1. Launch: `gbm2 wt testls --delay 2000`
2. Cursor on row 1 (main - tracked)
   - Verify footer shows: `↑/↓: navigate • space/enter: select • l: pull • d: delete • q/esc: quit`
   - **Push option NOT shown** (tracked branch)
3. Press `↓` to move to row 2 (feature/auth - ad-hoc)
   - Verify footer now shows: `↑/↓: navigate • space/enter: select • l: pull • p: push • d: delete • q/esc: quit`
   - **Push option IS shown** (ad-hoc)
4. Press `↓` to move to row 6 (release/v1.0 - tracked)
   - Verify footer again hides push option
5. Press `↓` to move to row 7 (experiment/ml - ad-hoc)
   - Verify footer shows push option again

**Expected Results**:
- ✓ Help text updates dynamically as cursor moves
- ✓ Push option hidden for tracked branches (main, release/v1.0)
- ✓ Push option visible for ad-hoc branches
- ✓ All other help options always visible
- ✓ Text is readable (muted gray color ~241)

---

### TC-004: Exit - Quit Command (q/esc)
**Purpose**: Verify graceful exit without selection

**Steps**:
1. Launch: `gbm2 wt testls --delay 2000`
2. Press `q` key
   - Application should exit immediately
   - No output should be printed
3. Relaunch: `gbm2 wt testls --delay 2000`
4. Press `esc` key
   - Application should exit immediately
   - No output should be printed

**Expected Results**:
- ✓ `q` key quits application
- ✓ `esc` key quits application
- ✓ No error messages shown
- ✓ No output written to stdout

---

### TC-005: Selection Output (space/enter)
**Purpose**: Verify selection outputs the correct worktree path

**Steps**:
1. Launch: `gbm2 wt testls --delay 2000`
2. Cursor on row 2 (feature/auth)
3. Press `space` (spacebar)
   - Application should quit
   - Stdout should show: `/tmp/feature-auth`
4. Relaunch: `gbm2 wt testls --delay 2000`
5. Navigate to row 5 (hotfix/crash)
6. Press `enter`
   - Application should quit
   - Stdout should show: `/tmp/hotfix-crash`

**Expected Results**:
- ✓ `space` outputs selected worktree path to stdout
- ✓ `enter` outputs selected worktree path to stdout
- ✓ Correct path for selected row is output
- ✓ Application exits cleanly after output

---

### TC-006: Pull Operation (l key)
**Purpose**: Verify pull operation displays appropriate message

**Steps**:
1. Launch: `gbm2 wt testls --delay 2000`
2. Cursor on any row
3. Press `l` (pull)
   - Should display message to stderr: `Would pull: <worktree-name>`
   - Message appears while table is still displayed
4. Wait 2-3 seconds
   - Verify table continues to function (no freeze)
5. Continue navigation and repeat pull on different rows

**Expected Results**:
- ✓ `l` key triggers pull operation
- ✓ Message shown: "Would pull: <name>"
- ✓ Message visible but doesn't block table interaction
- ✓ Can continue navigation and other operations
- ✓ Pull works on any worktree type (tracked or ad-hoc)

---

### TC-007: Push Operation - Ad-hoc Worktree (p key)
**Purpose**: Verify push succeeds for ad-hoc worktrees

**Steps**:
1. Launch: `gbm2 wt testls --delay 2000`
2. Navigate to row 2 (feature/auth - ad-hoc)
3. Press `p` (push)
   - Should display: `Would push: feature/auth`
4. Navigate to row 3 (bugfix/login - ad-hoc)
5. Press `p`
   - Should display: `Would push: bugfix/login`
6. Navigate to row 7 (experiment/ml - ad-hoc)
7. Press `p`
   - Should display: `Would push: experiment/ml`

**Expected Results**:
- ✓ Push succeeds on ad-hoc worktrees
- ✓ Message shown: "Would push: <name>"
- ✓ Message appears to stderr
- ✓ Works for all ad-hoc branches

---

### TC-008: Push Operation - Tracked Worktree (p key)
**Purpose**: Verify push fails for tracked worktrees with error message

**Steps**:
1. Launch: `gbm2 wt testls --delay 2000`
2. Cursor on row 1 (main - tracked)
3. Press `p` (push)
   - Should display: `Cannot push tracked worktree`
   - No "Would push" message
4. Navigate to row 6 (release/v1.0 - tracked)
5. Press `p`
   - Should display: `Cannot push tracked worktree`

**Expected Results**:
- ✓ Push fails on tracked worktrees
- ✓ Error message shown: "Cannot push tracked worktree"
- ✓ No push operation attempted
- ✓ Table remains functional after error
- ✓ Help text also hides push option for tracked

---

### TC-009: Delete Operation (d key)
**Purpose**: Verify delete operation displays appropriate message

**Steps**:
1. Launch: `gbm2 wt testls --delay 2000`
2. Cursor on row 3 (bugfix/login)
3. Press `d` (delete)
   - Should display: `Would delete: bugfix/login`
4. Navigate to row 7 (experiment/ml)
5. Press `d`
   - Should display: `Would delete: experiment/ml`
6. Repeat for various rows

**Expected Results**:
- ✓ `d` key triggers delete message
- ✓ Message shown: "Would delete: <name>"
- ✓ Message appears to stderr
- ✓ Works on any worktree (tracked or ad-hoc)
- ✓ Table remains functional after operation

---

### TC-010: Async Data Loading - Spinner Animation
**Purpose**: Verify spinner animates while loading, displays value when complete

**Steps**:
1. Launch: `gbm2 wt testls --delay 2000` (2 second delay)
2. Observe Git Status column immediately
   - Spinner should animate (⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏)
3. Wait for spinner to complete (approx 2 seconds)
4. Verify each row's Git Status column shows one of: `✓`, `↑ 3`, `↓ 2`, `↕ 1↑2`, or `?`
5. Status remains stable after loading

**Expected Results**:
- ✓ Spinner animates immediately on launch
- ✓ Spinner animation is smooth and continuous
- ✓ After delay, status value appears
- ✓ All 8 rows eventually show their status
- ✓ Status is consistent (doesn't change) after loading

---

### TC-011: Stress Test - Rapid Navigation
**Purpose**: Verify table handles rapid input without freezing or crashing

**Steps**:
1. Launch: `gbm2 wt testls --delay 1000`
2. Rapidly press `↑` and `↓` keys alternating 20+ times
3. Verify table updates smoothly
4. Verify spinner continues animating
5. Verify no errors or freezing occurs
6. Press `q` to quit

**Expected Results**:
- ✓ Table responds to rapid input
- ✓ No lag or freezing during navigation
- ✓ Cursor position is always accurate
- ✓ Spinner continues animating smoothly
- ✓ Application exits cleanly

---

### TC-012: Stress Test - Operations During Loading
**Purpose**: Verify operations work correctly while spinner is loading

**Steps**:
1. Launch: `gbm2 wt testls --delay 5000` (5 second delay for clear observation)
2. Immediately press `l` (pull) - while spinner still animating
   - Should show "Would pull: main"
3. Navigate up/down while waiting for spinner
   - Navigation should work smoothly
4. Press `d` (delete) - while spinner still animating
   - Should show "Would delete: <current>"
5. Wait for spinner to complete
6. Verify status values appear correctly

**Expected Results**:
- ✓ Operations work during async loading
- ✓ Navigation works during loading
- ✓ No interference between operations and spinner
- ✓ Spinner continues animating
- ✓ Final status displays correctly

---

### TC-013: Verify Mock Data Consistency
**Purpose**: Ensure mock worktrees and branch tracking are consistent

**Steps**:
1. Launch: `gbm2 wt testls --delay 500` (quick load)
2. Review all 8 worktrees and their types:
   - Row 1: main (Kind=tracked, footer hides push)
   - Row 2: feature/auth (Kind=ad hoc, footer shows push)
   - Row 3: bugfix/login (Kind=ad hoc, footer shows push)
   - Row 4: wip/dashboard (Kind=ad hoc, footer shows push)
   - Row 5: hotfix/crash (Kind=ad hoc, footer shows push)
   - Row 6: release/v1.0 (Kind=tracked, footer hides push)
   - Row 7: experiment/ml (Kind=ad hoc, footer shows push)
   - Row 8: archived/old (Kind=ad hoc, footer shows push)

**Expected Results**:
- ✓ All 8 worktrees display correctly
- ✓ Kind column shows correct type
- ✓ Footer push visibility matches kind type
- ✓ Branch names match worktree names

---

## Execution Checklist

### Pre-Test Checks
- [ ] Code compiles: `just build`
- [ ] Installs successfully: `just install`
- [ ] Help text displays: `gbm2 wt testls --help`
- [ ] Default delay is 1000ms
- [ ] Command accepts --delay flag 0-5000

### Manual Test Execution
- [ ] TC-001: Launch and display
- [ ] TC-002: Navigation (up/down)
- [ ] TC-003: Footer help text
- [ ] TC-004: Exit (q/esc)
- [ ] TC-005: Selection output
- [ ] TC-006: Pull operation
- [ ] TC-007: Push (ad-hoc)
- [ ] TC-008: Push (tracked - error)
- [ ] TC-009: Delete operation
- [ ] TC-010: Async spinner animation
- [ ] TC-011: Rapid navigation stress
- [ ] TC-012: Operations during loading
- [ ] TC-013: Mock data consistency

### Test Verification
- [ ] All 13 test cases passed
- [ ] No crashes or errors observed
- [ ] All help text displays correctly
- [ ] Navigation is smooth and responsive
- [ ] Async loading shows spinner correctly
- [ ] Operations handle errors appropriately
- [ ] Application exits cleanly in all scenarios

## Notes

### Known Limitations
- Mock service doesn't actually delete/push/pull (displays "Would..." messages)
- Git status values are deterministic based on path hash
- Worktrees are mock data, not real git repos
- Testing should be manual via terminal for full TUI experience

### Tips for Testing
- Use `--delay 2000` or `--delay 5000` for easier observation of spinner
- Stderr messages may appear above/below table - this is expected behavior
- Ctrl+C will force quit if application hangs (shouldn't happen)
- Test on a terminal with at least 80x24 character dimensions

---

## Test Report Template

```
Date: [DATE]
Tester: [NAME]
Build Version: [COMMIT HASH or just validate output]
Terminal: [DIMENSIONS, e.g., 120x40]

Test Results:
TC-001: [ PASS / FAIL ]
TC-002: [ PASS / FAIL ]
TC-003: [ PASS / FAIL ]
... (all 13 cases)

Summary:
- Total Tests: 13
- Passed: [N]
- Failed: [N]
- Blockers: [NONE or description]

Notes:
[Any observations or issues]
```

---

End of Testing Framework

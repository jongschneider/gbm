# Complete Testing Guide for gbm2 wt testls

**Quick Links**: 
- [Test Framework (13 test cases)](./TESTLS_FRAMEWORK.md)
- [Validation Report](./TEST_VALIDATION_REPORT.md)
- [Run automated validation](./validate_testls.sh)

---

## 🚀 Quick Start (2 minutes)

```bash
# Build and install
just install

# Launch the test command with 2-second delay for visibility
gbm2 wt testls --delay 2000

# In the TUI:
# - Press ↑/↓ to navigate
# - Press space to select
# - Press q to quit
```

---

## 📋 All Test Cases

### Quick Reference Matrix

| TC | Title | Category | Status |
|----|-------|----------|--------|
| TC-001 | Application Launch and Display | Display | Ready |
| TC-002 | Navigation - Up/Down Arrows | Navigation | Ready |
| TC-003 | Footer Help Text - Dynamic Content | UI/UX | Ready |
| TC-004 | Exit - Quit Command (q/esc) | User Input | Ready |
| TC-005 | Selection Output (space/enter) | User Input | Ready |
| TC-006 | Pull Operation (l key) | Operations | Ready |
| TC-007 | Push Operation - Ad-hoc Worktree (p key) | Operations | Ready |
| TC-008 | Push Operation - Tracked Worktree (p key) | Operations | Ready |
| TC-009 | Delete Operation (d key) | Operations | Ready |
| TC-010 | Async Data Loading - Spinner Animation | Async | Ready |
| TC-011 | Stress Test - Rapid Navigation | Performance | Ready |
| TC-012 | Stress Test - Operations During Loading | Performance | Ready |
| TC-013 | Verify Mock Data Consistency | Data | Ready |

---

## 🎯 Key Features to Validate

### 1. Table Display ✓
```
┌──────────────┬──────────────┬──────┬────────────┐
│ Name         │ Branch       │ Kind │ Git Status │
├──────────────┼──────────────┼──────┼────────────┤
│ main         │ main         │ trkd │ ⠋          │  ← spinner animates
│ feature/auth │ feature/auth │ ad   │ ↑ 3        │
│ ...          │ ...          │ ...  │ ...        │
└──────────────┴──────────────┴──────┴────────────┘
↑/↓: navigate • space/enter: select • l: pull • p: push • d: delete • q/esc: quit
```

### 2. Navigation
- ✓ Arrow keys move cursor up/down
- ✓ Smooth, responsive movement
- ✓ Wrapping at boundaries (implementation dependent)

### 3. Dynamic Help Text
```
Tracked worktree selected (no push option):
↑/↓: navigate • space/enter: select • l: pull • d: delete • q/esc: quit

Ad-hoc worktree selected (with push option):
↑/↓: navigate • space/enter: select • l: pull • p: push • d: delete • q/esc: quit
```

### 4. Operations
| Key | Tracked | Ad-hoc | Message |
|-----|---------|--------|---------|
| **l** (pull) | ✓ Works | ✓ Works | "Would pull: <name>" |
| **p** (push) | ✗ Error | ✓ Works | "Cannot push tracked..." / "Would push: <name>" |
| **d** (delete) | ✓ Works | ✓ Works | "Would delete: <name>" |

### 5. Async Loading
- Spinner animates: `⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏`
- After delay: status appears (`✓`, `↑ 3`, `↓ 2`, `↕`, `?`)
- All rows load independently

### 6. Exit & Selection
- **q/esc**: Quit without output
- **space/enter**: Output path, then quit
  - Example: Navigate to row 2, press space → outputs `/tmp/feature-auth`

---

## 🧪 Test Execution Options

### Option 1: Manual Testing (Recommended for First Run)
1. Launch: `gbm2 wt testls --delay 2000`
2. Follow [TESTLS_FRAMEWORK.md](./TESTLS_FRAMEWORK.md) test procedures
3. Record results in the Test Report Template (in framework document)

### Option 2: Automated Validation Script
```bash
./validate_testls.sh
```
- Runs automated build/install checks ✓
- Creates tmux session for guided manual testing
- Tracks pass/fail results
- Generates summary report

### Option 3: Interactive Tmux Session
```bash
tmux new-session -s testls -c /path/to/repo
gbm2 wt testls --delay 2000
# ... test manually ...
tmux kill-session -t testls
```

---

## 🔍 Detailed Test Instructions

### TC-001: Launch and Display (1 min)
```bash
$ gbm2 wt testls --delay 2000
```
**Verify**:
- [ ] Table renders with 4 columns (Name, Branch, Kind, Git Status)
- [ ] All 8 worktrees visible
- [ ] Git Status column shows animated spinner
- [ ] Cursor on row 1 (main)
- [ ] Help footer visible at bottom

### TC-002: Navigation (1 min)
```bash
$ gbm2 wt testls --delay 2000
# Press ↓ repeatedly to move down
# Press ↑ repeatedly to move up
# Observe cursor position in table
```
**Verify**:
- [ ] Cursor moves with arrow keys
- [ ] Movement is smooth and responsive
- [ ] Cursor position accurate
- [ ] Wrapping at boundaries (check top/bottom)

### TC-003: Help Text Dynamic (1 min)
```bash
$ gbm2 wt testls --delay 2000
# Move cursor to row 1 (main - tracked)
# Check help text for "p: push"
# Move to row 2 (feature/auth - ad-hoc)
# Check help text for "p: push"
```
**Verify**:
- [ ] Help text hides "p: push" on tracked rows
- [ ] Help text shows "p: push" on ad-hoc rows
- [ ] Other help options always visible
- [ ] Text updates as cursor moves

### TC-004: Exit (30 sec)
```bash
$ gbm2 wt testls --delay 2000
# Press q
```
**Verify**:
- [ ] App quits immediately
- [ ] No error shown
- [ ] No output to stdout

```bash
$ gbm2 wt testls --delay 2000
# Press esc
```
**Verify**:
- [ ] App quits immediately

### TC-005: Selection Output (1 min)
```bash
$ gbm2 wt testls --delay 2000
# Navigate to row 2 (feature/auth)
# Press space
```
**Verify**:
- [ ] App quits
- [ ] Stdout shows: `/tmp/feature-auth`

```bash
$ gbm2 wt testls --delay 2000
# Navigate to row 5 (hotfix/crash)
# Press enter
```
**Verify**:
- [ ] App quits
- [ ] Stdout shows: `/tmp/hotfix/crash`

### TC-006: Pull Operation (1 min)
```bash
$ gbm2 wt testls --delay 2000
# Move to any row
# Press l
```
**Verify**:
- [ ] Message appears: "Would pull: <name>"
- [ ] Message goes to stderr (visible in terminal)
- [ ] Table remains functional
- [ ] Can continue navigating

### TC-007: Push - Ad-hoc (1 min)
```bash
$ gbm2 wt testls --delay 2000
# Move to row 2 (feature/auth - ad-hoc)
# Press p
```
**Verify**:
- [ ] Message appears: "Would push: feature/auth"

```bash
# Move to row 7 (experiment/ml - ad-hoc)
# Press p
```
**Verify**:
- [ ] Message appears: "Would push: experiment/ml"

### TC-008: Push - Tracked (1 min)
```bash
$ gbm2 wt testls --delay 2000
# Move to row 1 (main - tracked)
# Press p
```
**Verify**:
- [ ] Message appears: "Cannot push tracked worktree"
- [ ] No "Would push" message

```bash
# Move to row 6 (release/v1.0 - tracked)
# Press p
```
**Verify**:
- [ ] Message appears: "Cannot push tracked worktree"
- [ ] Help text also hides "p: push" option

### TC-009: Delete Operation (1 min)
```bash
$ gbm2 wt testls --delay 2000
# Move to row 3 (bugfix/login)
# Press d
```
**Verify**:
- [ ] Message appears: "Would delete: bugfix/login"

### TC-010: Async Spinner Animation (2 min)
```bash
$ gbm2 wt testls --delay 2000
# Watch Git Status column on first row
```
**Verify**:
- [ ] Spinner animates immediately (⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏)
- [ ] Animation is smooth and continuous
- [ ] After ~2 seconds, spinner stops and shows value
- [ ] Value stays consistent: one of ✓, ↑ 3, ↓ 2, ↕ 1↑2, ?
- [ ] All 8 rows eventually show their status

### TC-011: Stress Test - Rapid Navigation (1 min)
```bash
$ gbm2 wt testls --delay 1000
# Rapidly press ↑/↓ alternating 20+ times
```
**Verify**:
- [ ] Table responds smoothly
- [ ] No lag or freezing
- [ ] Cursor position always accurate
- [ ] Spinner continues animating
- [ ] Press q to quit cleanly

### TC-012: Operations During Loading (2 min)
```bash
$ gbm2 wt testls --delay 5000  # 5-second delay
# Immediately press l (pull) while spinner animating
```
**Verify**:
- [ ] Pull message appears
- [ ] Spinner continues animating
- [ ] No interference

```bash
# Continue navigating while spinner loads
# Press d, then other operations
```
**Verify**:
- [ ] Navigation works during loading
- [ ] Operations work during loading
- [ ] Final status displays correctly when loading completes

### TC-013: Mock Data Consistency (2 min)
```bash
$ gbm2 wt testls --delay 500
# Review all 8 worktrees:
```
**Verify**:
- [ ] Row 1: main (Kind=tracked, no push in help)
- [ ] Row 2: feature/auth (Kind=ad hoc, push in help)
- [ ] Row 3: bugfix/login (Kind=ad hoc, push in help)
- [ ] Row 4: wip/dashboard (Kind=ad hoc, push in help)
- [ ] Row 5: hotfix/crash (Kind=ad hoc, push in help)
- [ ] Row 6: release/v1.0 (Kind=tracked, no push in help)
- [ ] Row 7: experiment/ml (Kind=ad hoc, push in help)
- [ ] Row 8: archived/old (Kind=ad hoc, push in help)

---

## 📊 Test Report Template

```markdown
# gbm2 wt testls Test Report

**Date**: [YYYY-MM-DD]
**Tester**: [Name]
**Terminal Dimensions**: [e.g., 120x40]
**Build Commit**: [git log --oneline -1]

## Test Results

| TC# | Title | Status | Notes |
|-----|-------|--------|-------|
| TC-001 | Launch and Display | PASS/FAIL | [observations] |
| TC-002 | Navigation | PASS/FAIL | [observations] |
| TC-003 | Help Text Dynamic | PASS/FAIL | [observations] |
| TC-004 | Exit (q/esc) | PASS/FAIL | [observations] |
| TC-005 | Selection Output | PASS/FAIL | [observations] |
| TC-006 | Pull Operation | PASS/FAIL | [observations] |
| TC-007 | Push (Ad-hoc) | PASS/FAIL | [observations] |
| TC-008 | Push (Tracked) | PASS/FAIL | [observations] |
| TC-009 | Delete Operation | PASS/FAIL | [observations] |
| TC-010 | Async Spinner | PASS/FAIL | [observations] |
| TC-011 | Rapid Navigation | PASS/FAIL | [observations] |
| TC-012 | Ops During Loading | PASS/FAIL | [observations] |
| TC-013 | Mock Data | PASS/FAIL | [observations] |

## Summary

- **Total Tests**: 13
- **Passed**: [N]
- **Failed**: [N]
- **Blockers**: [NONE or description]
- **Overall**: [PASS or FAIL]

## Notes

[Any additional observations, issues, or recommendations]

---

Validated against: [TESTLS_FRAMEWORK.md](./TESTLS_FRAMEWORK.md)
```

---

## 🐛 Troubleshooting

### "No git repository found" error
**Solution**: Run gbm2 from within the git repo directory
```bash
cd /path/to/repo  # Must be a git repo
gbm2 wt testls --delay 2000
```

### Terminal too small
**Error**: Table doesn't render properly
**Solution**: Make terminal at least 80x24, preferably 120x40

### Spinner not animating
**Solution**: Check if terminal supports the spinner characters
- Try: `printf '\u280b\u2819\u2839\n'`
- Should print: ⠋⠙⠹

### App freezes
**Solution**: Ctrl+C to force quit, then check error logs

### Help text doesn't show "p: push" toggle
**Issue**: Implementation verified, may need to rebuild
```bash
just build
sudo mv gbm /usr/local/bin/gbm2
```

---

## ✅ Validation Checklist

Before marking tests complete:

### Code Quality
- [ ] `just build` succeeds
- [ ] `just validate` passes all checks
- [ ] No compiler warnings
- [ ] All tests passing

### Installation
- [ ] `gbm2 wt testls --help` works
- [ ] `--delay` flag accepted
- [ ] Default delay is 1000ms

### Manual Testing
- [ ] All 13 test cases executed
- [ ] Test results documented
- [ ] No blocking issues found
- [ ] TUI is responsive

### Documentation
- [ ] TESTLS_FRAMEWORK.md reviewed
- [ ] TEST_VALIDATION_REPORT.md reviewed
- [ ] Test results recorded
- [ ] Issues documented (if any)

---

## 📚 Documentation

- **[TESTLS_FRAMEWORK.md](./TESTLS_FRAMEWORK.md)** - Complete test framework with 13 test cases
- **[TEST_VALIDATION_REPORT.md](./TEST_VALIDATION_REPORT.md)** - Validation status and implementation details
- **[TESTING_GUIDE.md](./TESTING_GUIDE.md)** - This guide
- **[validate_testls.sh](./validate_testls.sh)** - Automated validation script
- **[AGENTS.md](./AGENTS.md)** - Development guidelines
- **[CLAUDE.md](./CLAUDE.md)** - Repository overview

---

## 🎓 Implementation Details

### User Stories Implemented (10/10)
1. ✓ TableTheme in pkg/tui/theme.go
2. ✓ async.Cell[T] for per-cell loading
3. ✓ Table component in pkg/tui/table.go
4. ✓ AsyncRow support for spinners
5. ✓ Helper constructors (NewTable, SetAsyncCell)
6. ✓ gbm2 wt testls command
7. ✓ Navigation & footer validation
8. ✓ Pull/push operations validation
9. ✓ Delete confirmation validation
10. ✓ Test suite (16 unit tests)

### Code Structure
```
pkg/tui/
├── theme.go           # TableStyles addition
├── table.go           # Table component
├── table_test.go      # 16 unit tests
└── async/
    └── cell.go        # Per-cell async loading

cmd/service/
└── worktree_testls.go # testls command (mock service)
```

---

## 🚀 Ready to Test!

All systems ready. Choose your testing method:

1. **Quick Test** (2 min): Manual launch, basic operations
2. **Standard Test** (10 min): Follow TC-001 through TC-005
3. **Comprehensive Test** (30 min): All 13 test cases
4. **Automated** (optional): Run `./validate_testls.sh`

**Recommendation**: Start with quick test, then run comprehensive if time permits.

---

Last Updated: 2026-01-10
Framework Status: ✓ Complete
Implementation Status: ✓ Complete
Ready for Validation: ✓ YES

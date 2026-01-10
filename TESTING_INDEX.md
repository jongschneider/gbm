# Testing Framework Index

**Complete testing framework for `gbm2 wt testls` TUI Table component**

---

## 📚 Documentation Files

### 1. [README_TESTING.md](./README_TESTING.md) ⭐ START HERE (9.3K)
**Main entry point** - Overview, quick start, feature checklist
- Quick start guide (2 min)
- Feature summary and checklist
- Test category matrix
- Current status summary
- Troubleshooting tips

### 2. [TESTING_GUIDE.md](./TESTING_GUIDE.md) (13K)
**Step-by-step instructions** - All 13 test cases with detailed procedures
- Quick test option (2 min)
- Comprehensive test option (30 min)
- Detailed instructions for each test case
- Test report template
- Troubleshooting guide

### 3. [TESTLS_FRAMEWORK.md](./TESTLS_FRAMEWORK.md) (13K)
**Formal test specification** - Complete test suite definition
- Setup and prerequisites
- Test environment description
- Mock data specification (13 tables)
- All 13 test cases with acceptance criteria
- Execution checklist
- Test report template

### 4. [TEST_VALIDATION_REPORT.md](./TEST_VALIDATION_REPORT.md) (11K)
**Implementation status** - Build verification and validation results
- Build and installation verification ✓
- Implementation checklist (10/10 complete)
- Code quality results
- Feature validation matrix
- Known behaviors and testing notes
- Validation status table

---

## 🛠️ Automation Files

### [validate_testls.sh](./validate_testls.sh) (6.6K)
Automated validation script
- Runs automated build/install checks
- Creates tmux session for guided testing
- Tracks test results
- Generates summary report

---

## 🎯 Quick Navigation by Need

### "I want to..."

| Goal | Read | Time |
|------|------|------|
| Understand what we're testing | [README_TESTING.md](./README_TESTING.md) | 5 min |
| Run a quick test | [TESTING_GUIDE.md](./TESTING_GUIDE.md#-quick-start-2-minutes) | 2 min |
| Complete all tests | [TESTING_GUIDE.md](./TESTING_GUIDE.md#-detailed-test-instructions) | 30 min |
| Use automation | [validate_testls.sh](./validate_testls.sh) | varies |
| Check implementation status | [TEST_VALIDATION_REPORT.md](./TEST_VALIDATION_REPORT.md) | 5 min |
| See formal test spec | [TESTLS_FRAMEWORK.md](./TESTLS_FRAMEWORK.md) | 10 min |

---

## 📋 Test Case Overview

All 13 test cases organized by category:

### Display & Layout (2 tests)
- **TC-001**: Application Launch and Display
- **TC-013**: Verify Mock Data Consistency

### Navigation (2 tests)
- **TC-002**: Navigation - Up/Down Arrows
- **TC-011**: Stress Test - Rapid Navigation

### User Input (3 tests)
- **TC-003**: Footer Help Text - Dynamic Content
- **TC-004**: Exit - Quit Command (q/esc)
- **TC-005**: Selection Output (space/enter)

### Operations (4 tests)
- **TC-006**: Pull Operation (l key)
- **TC-007**: Push Operation - Ad-hoc Worktree (p key)
- **TC-008**: Push Operation - Tracked Worktree (p key)
- **TC-009**: Delete Operation (d key)

### Async Loading (2 tests)
- **TC-010**: Async Data Loading - Spinner Animation
- **TC-012**: Stress Test - Operations During Loading

---

## ✅ Implementation Status

| Component | Status | Details |
|-----------|--------|---------|
| Build | ✓ PASS | Compiles without warnings |
| Installation | ✓ PASS | gbm2 installed to /usr/local/bin |
| Unit Tests | ✓ PASS | 16/16 tests passing |
| Code Quality | ✓ PASS | Lint and vet checks pass |
| Testing Framework | ✓ COMPLETE | 13 test cases defined |
| Documentation | ✓ COMPLETE | 4 comprehensive guides |
| Manual Testing | ⧖ READY | Follow test guides |

---

## 🚀 Getting Started

### Option 1: Quick Manual Test (2 min)
```bash
gbm2 wt testls --delay 2000
# Press: ↓ (navigate), space (select), q (quit)
```

### Option 2: Comprehensive Manual Test (30 min)
Read [TESTING_GUIDE.md](./TESTING_GUIDE.md) and follow all test cases

### Option 3: Automated Validation
```bash
./validate_testls.sh
```

---

## 📊 Feature Checklist

Test coverage includes:
- ✓ Table display (4 columns, 8 rows, spinner animation)
- ✓ Navigation (up/down arrows with smooth movement)
- ✓ Dynamic help footer (shows/hides based on row type)
- ✓ Operations (pull, push, delete with appropriate messages)
- ✓ Selection (outputs path to stdout)
- ✓ Exit (graceful quit with q/esc)
- ✓ Async loading (spinner → status value)
- ✓ Stress testing (rapid navigation, operations during loading)

---

## 🔗 Related Files

### Implementation
- `cmd/service/worktree_testls.go` - testls command
- `pkg/tui/theme.go` - TableStyles
- `pkg/tui/table.go` - Table component
- `pkg/tui/async/cell.go` - Async Cell[T]
- `pkg/tui/table_test.go` - Unit tests (16 tests)

### Configuration
- `prd.json` - User stories (10 stories)
- `progress.txt` - Development log
- `AGENTS.md` - Development guidelines
- `justfile` - Build commands

---

## 📈 Test Metrics

- **Total Test Cases**: 13
- **Manual Test Time**: ~30 minutes (comprehensive)
- **Quick Test Time**: 2 minutes
- **Unit Tests**: 16 (in table_test.go)
- **Code Lines**: ~500 new (table.go, async/cell.go)
- **Mock Worktrees**: 8 (tracked + ad-hoc)

---

## ✨ Key Features

### Table Component
- Wraps bubbles/table with theme support
- Builder pattern for configuration
- Async cell support with spinners
- Direct bubbles/table integration

### Async Loading
- Per-cell spinner animation
- Smooth animation during async operations
- Independent cell loading
- Consistent status display

### Dynamic UI
- Help footer updates based on selected row
- Shows/hides "push" option based on worktree type
- Responsive to user navigation

### Operations
- Pull (l): Works on all worktrees
- Push (p): Works on ad-hoc, errors on tracked
- Delete (d): Works on all worktrees
- Selection (space/enter): Outputs path to stdout

---

## 🐛 Troubleshooting

### Terminal Too Small
Resize terminal to at least 80x24 (preferably 120x40)

### No Git Repository
Must run from within a git repository:
```bash
cd /path/to/repo
gbm2 wt testls --delay 2000
```

### Spinner Not Animating
Check terminal Unicode support:
```bash
printf '\u280b'  # Should print ⠋
```

### App Freezes
Press Ctrl+C to force quit, then rebuild:
```bash
just build && sudo mv gbm /usr/local/bin/gbm2
```

---

## 📝 Test Report Template

Use this structure to document your test results:

```markdown
# Test Report: gbm2 wt testls

**Date**: [Date]
**Tester**: [Name]
**Terminal**: [Dimensions]

## Results
| TC# | Title | Status | Notes |
|-----|-------|--------|-------|
| 001 | Launch & Display | PASS/FAIL | |
| ... | ... | ... | |
| 013 | Mock Data | PASS/FAIL | |

## Summary
- Passed: [N]/13
- Failed: [N]/13
- Blockers: [None or description]
```

---

## 🎓 Implementation Summary

**All 10 user stories complete:**
1. ✓ TableTheme - consistent table styling
2. ✓ async.Cell[T] - per-cell async loading
3. ✓ Table component - reusable wrapper
4. ✓ AsyncRow support - spinners per row
5. ✓ Helper constructors - clean API
6. ✓ testls command - mock git service
7. ✓ Navigation & footer - dynamic help
8. ✓ Pull/push operations - with error handling
9. ✓ Delete confirmation - mock deletion
10. ✓ Test suite - 16 unit tests

---

## ✅ Ready for Validation

All code quality checks pass:
- ✓ Build successful
- ✓ Tests passing (16/16)
- ✓ Lint checks pass
- ✓ Vet checks pass
- ✓ Installation successful

Choose your testing method and get started!

---

**Last Updated**: 2026-01-10  
**Status**: Ready for Manual Validation ✓  
**Implementation**: Complete (10/10 user stories) ✓  
**Code Quality**: All checks pass ✓

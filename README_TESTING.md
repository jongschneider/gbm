# gbm2 wt testls - Complete Testing Framework

## 📚 Documentation Overview

This directory contains a **comprehensive testing framework** for validating the `gbm2 wt testls` command, including:

### 🎯 Main Documents
1. **[TESTING_GUIDE.md](./TESTING_GUIDE.md)** ⭐ **START HERE**
   - Quick start (2 minutes)
   - All 13 test cases with detailed instructions
   - Test execution options
   - Troubleshooting guide
   - Test report template

2. **[TESTLS_FRAMEWORK.md](./TESTLS_FRAMEWORK.md)**
   - Complete formal test specification
   - 13 detailed test cases (TC-001 through TC-013)
   - Test checklist
   - Known limitations & tips

3. **[TEST_VALIDATION_REPORT.md](./TEST_VALIDATION_REPORT.md)**
   - Current validation status
   - Implementation checklist (10/10 user stories complete)
   - Code quality results
   - Feature validation matrix

### 🛠️ Automation
- **[validate_testls.sh](./validate_testls.sh)**
  - Automated build/install verification
  - Guided manual testing in tmux
  - Results tracking and summary

---

## 🚀 Quick Start (2 minutes)

```bash
# Build and install
just install

# Launch the test command
gbm2 wt testls --delay 2000

# In the TUI:
# ↑/↓ = navigate, space = select, l/p/d = operations, q = quit
```

---

## 🧪 Complete Test Suite

### Test Categories (13 total)

| Category | Tests | Time |
|----------|-------|------|
| Display & Layout | TC-001, TC-013 | 3 min |
| Navigation | TC-002, TC-011 | 3 min |
| User Input | TC-003, TC-004, TC-005 | 3 min |
| Operations | TC-006, TC-007, TC-008, TC-009 | 4 min |
| Async Loading | TC-010, TC-012 | 4 min |
| **Total** | **13 tests** | **~17 min** |

### Test Matrix

```
TC-001: Application Launch and Display          ✓ Ready
TC-002: Navigation - Up/Down Arrows             ✓ Ready
TC-003: Footer Help Text - Dynamic Content      ✓ Ready
TC-004: Exit - Quit Command (q/esc)             ✓ Ready
TC-005: Selection Output (space/enter)          ✓ Ready
TC-006: Pull Operation (l key)                  ✓ Ready
TC-007: Push Operation - Ad-hoc (p key)         ✓ Ready
TC-008: Push Operation - Tracked (p key)        ✓ Ready
TC-009: Delete Operation (d key)                ✓ Ready
TC-010: Async Data Loading - Spinner            ✓ Ready
TC-011: Stress Test - Rapid Navigation          ✓ Ready
TC-012: Stress Test - Operations During Load    ✓ Ready
TC-013: Verify Mock Data Consistency            ✓ Ready
```

---

## ✅ Current Status

### Build & Installation ✓
- ✓ Compiles without errors
- ✓ Installs successfully to /usr/local/bin
- ✓ All build checks pass

### Code Quality ✓
- ✓ All unit tests passing
- ✓ Lint checks pass
- ✓ Vet checks pass
- ✓ 16 TUI tests in table_test.go

### Features Implemented ✓
- ✓ Table display with 4 columns
- ✓ Async data loading with spinners
- ✓ Navigation (↑/↓)
- ✓ Dynamic help footer
- ✓ Operations (l, p, d)
- ✓ Selection output (space/enter)
- ✓ Clean exit (q/esc)

### Ready for Manual Testing ✓
All automated checks pass. Ready for comprehensive manual validation.

---

## 📋 Testing Instructions

### Option 1: Quick Manual Test (2 min)
```bash
cd /path/to/repo  # Must be a git repository
gbm2 wt testls --delay 2000

# Test basic features:
# - Press ↓ to navigate down
# - Check footer text changes when on different rows
# - Press space to select and output path
# - Press q to quit
```

### Option 2: Guided Automated Script (10 min)
```bash
./validate_testls.sh
```
- Runs automated checks
- Creates tmux session
- Guides through key test cases
- Tracks results

### Option 3: Comprehensive Manual Test (30 min)
Follow all 13 test cases in [TESTING_GUIDE.md](./TESTING_GUIDE.md)

---

## 🎯 Key Features to Validate

### 1. **Table Display** ✓
```
Name         │ Branch       │ Kind │ Git Status
─────────────┼──────────────┼──────┼────────────
main         │ main         │ trkd │ ⠋ (spinner)
feature/auth │ feature/auth │ ad   │ ↑ 3
```

### 2. **Navigation** ✓
- Arrow keys work smoothly
- Cursor updates correctly
- Wrapping at boundaries

### 3. **Dynamic Help Footer** ✓
```
Tracked row:
↑/↓: navigate • space/enter: select • l: pull • d: delete • q/esc: quit

Ad-hoc row:
↑/↓: navigate • space/enter: select • l: pull • p: push • d: delete • q/esc: quit
```

### 4. **Operations** ✓
| Key | Ad-hoc | Tracked | Output |
|-----|--------|---------|--------|
| **l** | ✓ | ✓ | "Would pull: X" |
| **p** | ✓ | ✗ | "Would push: X" / "Cannot push..." |
| **d** | ✓ | ✓ | "Would delete: X" |

### 5. **Async Loading** ✓
- Spinner animates while loading
- Status appears after delay
- Independent per-row loading

### 6. **Exit & Selection** ✓
- **q/esc**: Quit cleanly
- **space/enter**: Output path and quit

---

## 📊 Mock Data

8 worktrees with mix of tracked and ad-hoc:

| # | Name | Branch | Kind | Can Push? |
|---|------|--------|------|-----------|
| 1 | main | main | tracked | ✗ |
| 2 | feature/auth | feature/auth | ad hoc | ✓ |
| 3 | bugfix/login | bugfix/login | ad hoc | ✓ |
| 4 | wip/dashboard | wip/dashboard | ad hoc | ✓ |
| 5 | hotfix/crash | hotfix/crash | ad hoc | ✓ |
| 6 | release/v1.0 | release/v1.0 | tracked | ✗ |
| 7 | experiment/ml | experiment/ml | ad hoc | ✓ |
| 8 | archived/old | archived/old | ad hoc | ✓ |

---

## 🐛 Troubleshooting

### "No git repository found"
```bash
# Solution: Run from within the git repo
cd /path/to/repo
gbm2 wt testls --delay 2000
```

### Terminal too small
```bash
# Resize to at least 80x24, preferably 120x40
# Check: printf "$LINES x $COLUMNS\n"
```

### Spinner not showing
- Terminal may not support Unicode
- Try different terminal (iterm2, alacritty, etc.)
- Check: `printf '\u280b'` should print ⠋

### App freezes
- Press Ctrl+C to force quit
- Check for git repository
- Rebuild: `just build && sudo mv gbm /usr/local/bin/gbm2`

---

## 🔗 Related Files

### Implementation Files
- `cmd/service/worktree_testls.go` - testls command
- `pkg/tui/theme.go` - TableStyles
- `pkg/tui/table.go` - Table component
- `pkg/tui/async/cell.go` - Async Cell[T]
- `pkg/tui/table_test.go` - Unit tests (16 tests)

### Configuration
- `justfile` - Build and test commands
- `AGENTS.md` - Development guidelines
- `CLAUDE.md` - Repository overview

---

## ✨ Implementation Summary

### User Stories (10/10 Complete)
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

### Code Quality
- ✓ Compiles without warnings
- ✓ All lint checks pass
- ✓ All vet checks pass
- ✓ All unit tests pass (16 in table_test.go)
- ✓ Follows AGENTS.md guidelines

---

## 📝 Test Report Format

Use this template to document results:

```markdown
# Test Report: gbm2 wt testls

**Date**: [Date]
**Tester**: [Name]
**Terminal**: [Dimensions, e.g., 120x40]

## Results

| TC | Title | Status | Notes |
|----|-------|--------|-------|
| 01 | Launch & Display | PASS/FAIL | |
| 02 | Navigation | PASS/FAIL | |
| 03 | Help Footer | PASS/FAIL | |
| ... | ... | ... | |
| 13 | Mock Data | PASS/FAIL | |

## Summary
- Passed: [N]/13
- Failed: [N]/13
- Blockers: [None or description]

## Notes
[Any observations]
```

---

## 🎓 Learning Resources

### TUI Framework Components
- **bubbletea**: Tea framework for TUI applications
- **bubbles/table**: Reusable table component
- **bubbles/spinner**: Spinner animation
- **async.Eval[T]**: Lazy evaluation with caching
- **lipgloss**: Terminal styling library

### Code Patterns
- **Builder pattern**: Table configuration (WithColumns, WithRows, etc.)
- **Async cell loading**: Per-cell spinners during fetch
- **Dynamic UI**: Help text based on data
- **Error handling**: Tracked vs ad-hoc operation differences

---

## 🚀 Next Steps

1. **Read**: Start with [TESTING_GUIDE.md](./TESTING_GUIDE.md)
2. **Quick Test**: Run `gbm2 wt testls --delay 2000`
3. **Complete Test**: Follow all 13 test cases
4. **Document**: Fill in test report template
5. **Approve**: All tests should pass

---

## 📞 Support

### If Tests Fail
1. Check terminal size (80x24 minimum)
2. Ensure git repository is present
3. Rebuild: `just build && sudo mv gbm /usr/local/bin/gbm2`
4. Review error messages in test output
5. Check [TESTING_GUIDE.md](./TESTING_GUIDE.md) troubleshooting section

### If You Need Help
- Review [AGENTS.md](./AGENTS.md) for development guidelines
- Check [TESTLS_FRAMEWORK.md](./TESTLS_FRAMEWORK.md) for detailed specs
- See [TEST_VALIDATION_REPORT.md](./TEST_VALIDATION_REPORT.md) for implementation details

---

## ✅ Checklist Before Approving

- [ ] Read TESTING_GUIDE.md
- [ ] Run quick manual test (2 min)
- [ ] Run automated validation script
- [ ] Complete all 13 test cases
- [ ] Document results in test report
- [ ] No blocking issues found
- [ ] Ready to merge to main

---

**Status**: ✓ Ready for Manual Validation  
**Implementation**: ✓ Complete (10/10 user stories)  
**Code Quality**: ✓ All checks pass  
**Documentation**: ✓ Comprehensive testing framework  

---

Generated: 2026-01-10

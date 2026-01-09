## Metadata
name: CLI Test Suite Skill
description: Execute a comprehensive test of all GBM (Git Branch Manager) CLI functionality. Tracks all test results in a timestamped markdown report.


## Overview
Execute a comprehensive test of all GBM (Git Branch Manager) CLI functionality. Tracks all test results in a timestamped markdown report.

---

## How to Use

When asked to run the CLI test suite:

1. **Ask about VHS recording**: "Do you want to record this test session with VHS? (yes/no)"
2. **Set up test environment** with timestamped directories
3. **Build the GBM binary** from source
4. **Execute all test phases** in a tmux session
5. **Record results** in a timestamped markdown report
6. **Generate VHS recordings** (if user requested)

---

## Quick Start

```bash
# Set timestamp for this test run
export TIMESTAMP=$(date +%Y%m%d-%H%M%S)

# Create test results directory
mkdir -p /Users/jschneider/code/scratch/gbm/test-results/gbm-test-results-$TIMESTAMP

# Build GBM binary
cd /Users/jschneider/code/scratch/gbm
just build

# Create temporary test directory
mkdir -p /tmp/gbm-test-$TIMESTAMP

# Start tmux session
tmux new-session -s "gbm-test-$TIMESTAMP"
```

---

## Test Phases Overview

| Phase | Description | Tests |
|-------|-------------|-------|
| 1 | Build and Initialization | Build binary, help, init commands |
| 2 | Worktree Management | Add, list, switch, remove worktrees |
| 3 | Clone Tests | Clone remote repos with GBM structure |
| 4 | Configuration and Sync | Config files, sync worktrees |
| 5 | Shell Integration | Shell integration scripts |
| 6 | Edge Cases | Error handling, edge cases |
| 7 | Cleanup and Summary | Final verification |

---

## Phase 1: Build and Initialization Tests

### Test 1.1: Build GBM Binary
```bash
cd /Users/jschneider/code/scratch/gbm
just build
```
**Expected**: Binary built successfully at `./gbm`

### Test 1.2: Help Command
```bash
/Users/jschneider/code/scratch/gbm/gbm --help
```
**Expected**: Shows all available commands (init, clone, worktree, sync, shell-integration)

### Test 1.3: Init New Repository
```bash
cd /tmp/gbm-test-$TIMESTAMP
/Users/jschneider/code/scratch/gbm/gbm init test-repo --branch main
```
**Expected**:
- Creates `test-repo/.git` (bare repository)
- Creates `test-repo/worktrees/main/` (worktree)
- Creates `test-repo/.gbm/config.yaml`

### Test 1.4: Init with Dry Run
```bash
/Users/jschneider/code/scratch/gbm/gbm init test-repo-dry --branch main --dry-run
```
**Expected**: Shows commands without executing

### Test 1.5: Init in Current Directory
```bash
cd /tmp/gbm-test-$TIMESTAMP
mkdir current-dir-test && cd current-dir-test
/Users/jschneider/code/scratch/gbm/gbm init --branch develop
```
**Expected**: Initializes in current directory

---

## Phase 2: Worktree Management Tests

### Test 2.1: Add Worktree - New Branch (CLI Mode)
```bash
cd /tmp/gbm-test-$TIMESTAMP/test-repo
/Users/jschneider/code/scratch/gbm/gbm worktree add feature-1 feature-1 -b
```
**Expected**: Creates worktree at `worktrees/feature-1` with new branch `feature-1`

### Test 2.2: Add Worktree - New Branch with Base
```bash
/Users/jschneider/code/scratch/gbm/gbm worktree add feature-2 feature-2 -b --base main
```
**Expected**: Creates worktree from `main` branch

### Test 2.3: Add Worktree - Existing Branch
```bash
/Users/jschneider/code/scratch/gbm/gbm worktree add main-copy main
```
**Expected**: Creates worktree for existing `main` branch

### Test 2.4: Add Worktree - Branch Doesn't Exist (prompt test)
```bash
echo "n" | /Users/jschneider/code/scratch/gbm/gbm worktree add nonexistent nonexistent
```
**Expected**: Prompts to create branch, user declines

### Test 2.5: Add Worktree - Dry Run
```bash
/Users/jschneider/code/scratch/gbm/gbm worktree add feature-dry feature-dry -b --dry-run
```
**Expected**: Shows what would be created without executing

### Test 2.6: List Worktrees (TUI)
```bash
/Users/jschneider/code/scratch/gbm/gbm worktree list
```
**Expected**: Shows interactive TUI table with all worktrees
**Note**: This launches a TUI - send 'q' to quit

### Test 2.7: List Worktrees - Alias
```bash
/Users/jschneider/code/scratch/gbm/gbm wt ls
```
**Expected**: Same as `worktree list` (wt is alias)

### Test 2.8: Switch Worktree - Print Path
```bash
/Users/jschneider/code/scratch/gbm/gbm worktree switch feature-1 --print-path
```
**Expected**: Prints absolute path to feature-1 worktree

### Test 2.9: Switch Worktree - Without Shell Integration
```bash
/Users/jschneider/code/scratch/gbm/gbm worktree switch feature-2
```
**Expected**: Shows instructions for cd command and shell integration

### Test 2.10: Remove Worktree
```bash
echo "n" | /Users/jschneider/code/scratch/gbm/gbm worktree remove feature-1
```
**Expected**: Removes worktree, prompts about branch deletion (decline)

### Test 2.11: Remove Worktree - Force
```bash
# First, create uncommitted changes
cd /tmp/gbm-test-$TIMESTAMP/test-repo/worktrees/feature-2
echo "test" > uncommitted.txt
cd ../..
echo "y" | /Users/jschneider/code/scratch/gbm/gbm worktree remove feature-2 --force
```
**Expected**: Force removes worktree with uncommitted changes, deletes branch

### Test 2.12: Remove Current Worktree (using ".")
```bash
cd /tmp/gbm-test-$TIMESTAMP/test-repo/worktrees/main-copy
echo "y" | /Users/jschneider/code/scratch/gbm/gbm worktree remove .
```
**Expected**:
- Shows "Switching to repository root before removing current worktree..."
- Automatically changes to repo root before removal
- Removes current worktree successfully
- No "Unable to read current working directory" error

---

## Phase 3: Clone Tests

### Test 3.1: Clone Remote Repository
```bash
cd /tmp/gbm-test-$TIMESTAMP
/Users/jschneider/code/scratch/gbm/gbm clone https://github.com/bubbletea-examples/minimal.git
```
**Expected**:
- Clones as bare repo
- Creates worktree for default branch
- Creates .gbm/config.yaml

### Test 3.2: Clone with Custom Name
```bash
/Users/jschneider/code/scratch/gbm/gbm clone https://github.com/bubbletea-examples/minimal.git custom-name
```
**Expected**: Clones into `custom-name/` directory

### Test 3.3: Clone - Dry Run
```bash
/Users/jschneider/code/scratch/gbm/gbm clone https://github.com/bubbletea-examples/minimal.git --dry-run
```
**Expected**: Shows clone commands without executing

---

## Phase 4: Configuration and Sync Tests

### Test 4.1: Verify Config File Structure
```bash
cd /tmp/gbm-test-$TIMESTAMP/test-repo
cat .gbm/config.yaml
```
**Expected**: Shows valid YAML with worktrees configuration

### Test 4.2: Manual Config Edit - Add Worktree Definition
```bash
cat >> .gbm/config.yaml << EOF
  feature-3:
    branch: feature-3
EOF
```
**Expected**: Config updated successfully

### Test 4.3: Sync - Show Missing Worktrees
```bash
/Users/jschneider/code/scratch/gbm/gbm sync --dry-run
```
**Expected**: Shows feature-3 as missing (needs to be created)

### Test 4.4: Sync - Create Missing Worktrees
```bash
/Users/jschneider/code/scratch/gbm/gbm sync --force
```
**Expected**: Creates feature-3 worktree and branch

### Test 4.5: Sync - Detect Ad Hoc Worktrees
```bash
# Create an ad hoc worktree not in config
/Users/jschneider/code/scratch/gbm/gbm worktree add adhoc-test adhoc-test -b
# Run sync
/Users/jschneider/code/scratch/gbm/gbm sync --dry-run
```
**Expected**: Shows adhoc-test as orphaned (not in config)

### Test 4.6: Sync - Branch Mismatch Detection
```bash
# Manually edit config to change branch for existing worktree
# Then run sync
/Users/jschneider/code/scratch/gbm/gbm sync --dry-run
```
**Expected**: Detects branch mismatches

### Test 4.7: Sync - Worktree Rename Detection
```bash
# Create worktree with one name
/Users/jschneider/code/scratch/gbm/gbm worktree add old-name feature-rename -b
# Edit config to rename it, then run sync
/Users/jschneider/code/scratch/gbm/gbm sync --dry-run
```
**Expected**: Suggests rename from old-name to new name

---

## Phase 5: Shell Integration Tests

### Test 5.1: Generate Shell Integration
```bash
/Users/jschneider/code/scratch/gbm/gbm shell-integration
```
**Expected**: Outputs shell script for integration

### Test 5.2: Verify Shell Integration Format
```bash
/Users/jschneider/code/scratch/gbm/gbm shell-integration | head -n 5
```
**Expected**: Shows shell function definition

---

## Phase 6: Edge Cases and Error Handling

### Test 6.1: Add Worktree - Duplicate Name
```bash
cd /tmp/gbm-test-$TIMESTAMP/test-repo
/Users/jschneider/code/scratch/gbm/gbm worktree add main main
```
**Expected**: Error - worktree already exists

### Test 6.2: Remove Non-Existent Worktree
```bash
/Users/jschneider/code/scratch/gbm/gbm worktree remove nonexistent-worktree
```
**Expected**: Error - worktree not found

### Test 6.3: Switch to Non-Existent Worktree
```bash
/Users/jschneider/code/scratch/gbm/gbm worktree switch nonexistent
```
**Expected**: Error - worktree not found

### Test 6.4: Init in Existing Git Repository
```bash
cd /tmp/gbm-test-$TIMESTAMP/test-repo
/Users/jschneider/code/scratch/gbm/gbm init already-exists
```
**Expected**: Error or warning about existing git repo

### Test 6.5: Command Outside Git Repository
```bash
cd /tmp
/Users/jschneider/code/scratch/gbm/gbm worktree list
```
**Expected**: Error - not in a git repository

### Test 6.6: Switch Using "-" (Previous Worktree)
```bash
cd /tmp/gbm-test-$TIMESTAMP/test-repo
/Users/jschneider/code/scratch/gbm/gbm worktree switch feature-3 --print-path
/Users/jschneider/code/scratch/gbm/gbm worktree switch main --print-path
/Users/jschneider/code/scratch/gbm/gbm worktree switch - --print-path
```
**Expected**: Returns to feature-3

---

## Phase 7: Cleanup and Summary

### Test 7.1: Count All Worktrees
```bash
cd /tmp/gbm-test-$TIMESTAMP/test-repo
git worktree list | wc -l
```
**Expected**: Shows total worktree count

### Test 7.2: Verify Git Bare Repository Structure
```bash
ls -la .git/
```
**Expected**: Shows bare repository structure (refs, objects, etc.)

### Test 7.3: Final State Documentation
```bash
tree -L 3 -a
```
**Expected**: Shows complete repository structure

---

## Results File Format

Create a markdown file at `/Users/jschneider/code/scratch/gbm/test-results/gbm-test-results-$TIMESTAMP/gbm-test-results-$TIMESTAMP.md`:

```markdown
# GBM CLI Test Results
**Timestamp**: [ISO-8601 timestamp]
**Test Directory**: [path]
**GBM Binary**: [version/commit]
**System**: [OS/arch]

## Summary
- Total Tests: X
- Passed: Y
- Failed: Z
- Success Rate: N%

## Test Results

### Phase 1: Build and Initialization Tests

#### [PASS/FAIL] Test 1.1: Build GBM Binary
**Command**: `just build`
**Status**: SUCCESS/FAILURE
**Duration**: Xs
**Output**:
```
[command output]
```
**Notes**: [any observations]

[... continue for all tests ...]

## Failures Summary
[List all failed tests with details]

## Recommendations
[Any suggestions based on test results]
```

---

## VHS Recording Setup (Optional)

Only if user answered "yes" to VHS recording:

### Directory Structure
```
test-results/
└── gbm-test-results-$TIMESTAMP/
    ├── gbm-test-results-$TIMESTAMP.md
    └── vhs-recordings/
        ├── 01-init-repository.tape
        ├── 02-worktree-add.tape
        ├── init-repository.gif
        └── worktree-add.gif
```

### Create VHS Recordings Directory
```bash
mkdir -p /Users/jschneider/code/scratch/gbm/test-results/gbm-test-results-$TIMESTAMP/vhs-recordings
```

### Example Tape File
Create `01-init-repository.tape`:
```tape
# VHS Recording: Initialize Repository
Output init-repository.gif

Set Shell bash
Set FontSize 12
Set Width 1400
Set Height 600
Set PlaybackSpeed 1.0
Set TypingSpeed 10ms

# Setup
Type "# GBM Test: Initialize Repository" Enter
Sleep 500ms
Type "TIMESTAMP=$(date +%Y%m%d-%H%M%S)" Enter
Type "cd /tmp && mkdir gbm-vhs-init-$TIMESTAMP && cd gbm-vhs-init-$TIMESTAMP" Enter
Sleep 1s

# Test init command - ALWAYS USE ABSOLUTE PATH
Type "/Users/jschneider/code/scratch/gbm/gbm init demo-repo --branch main" Enter
Sleep 3s

Type "# Verify repository structure" Enter
Sleep 500ms
Type "ls -la demo-repo/" Enter
Sleep 2s

Type "# Success!" Enter
Sleep 500ms
```

### Run VHS Recordings
```bash
cd /Users/jschneider/code/scratch/gbm/test-results/gbm-test-results-$TIMESTAMP/vhs-recordings
vhs 01-init-repository.tape
vhs 02-worktree-add.tape
# ... for each tape file
```

---

## Success Criteria

- [ ] All core commands execute without errors
- [ ] Dry-run modes work correctly
- [ ] Error messages are clear and helpful
- [ ] Configuration files are created properly
- [ ] Worktrees are created in correct locations
- [ ] TUI interfaces launch successfully
- [ ] Edge cases are handled gracefully

---

## Critical Notes

1. **ALWAYS use absolute paths** for the GBM binary: `/Users/jschneider/code/scratch/gbm/gbm`
2. **NEVER use relative paths** like `../../gbm` or `./gbm` - they fail in VHS recordings
3. **Capture both stdout and stderr** for all commands
4. **Document any unexpected behavior**
5. **Test both long-form commands and aliases** (wt, ls, sw, etc.)
6. **Test 2.12**: GBM auto-switches to repo root when removing current worktree

---

## Post-Execution

```bash
# Review markdown results
less /Users/jschneider/code/scratch/gbm/test-results/gbm-test-results-$TIMESTAMP/gbm-test-results-$TIMESTAMP.md

# [VHS ONLY] View recordings
open /Users/jschneider/code/scratch/gbm/test-results/gbm-test-results-$TIMESTAMP/vhs-recordings/*.gif

# Archive test run
tar -czf gbm-test-results-$TIMESTAMP.tar.gz gbm-test-results-$TIMESTAMP/
```

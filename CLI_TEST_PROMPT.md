# GBM CLI Test Suite - Agent Instructions

## Objective
Execute a comprehensive test of all GBM (Git Branch Manager) CLI functionality in a new tmux session. Track all test results in a timestamped markdown report.

## Setup Requirements

1. **Create a new tmux session** named `gbm-test-$(date +%s)`
2. **Create a temporary test directory** at `/tmp/gbm-test-$(date +%s)`
3. **Build the GBM binary** from `/Users/jschneider/code/scratch/gbm`
4. **Create a timestamped results file** at `/tmp/gbm-test-results-$(date +%Y%m%d-%H%M%S).md`

## Test Execution Guidelines

- Run all commands in the tmux session for visibility
- Capture stdout and stderr for each command
- Mark each test as ✅ SUCCESS or ❌ FAILURE
- Include command output snippets in the results file
- Test both success and failure scenarios
- Use `--dry-run` flags where appropriate to verify command planning

## Test Suite

### Phase 1: Build and Initialization Tests

#### Test 1.1: Build GBM Binary
```bash
cd /Users/jschneider/code/scratch/gbm
just build
```
**Expected**: Binary built successfully at `./gbm`
**Log**: Build output, binary size, compilation time

#### Test 1.2: Help Command
```bash
./gbm --help
```
**Expected**: Shows all available commands (init, clone, worktree, sync, shell-integration)
**Log**: Full help output

#### Test 1.3: Init New Repository (with name)
```bash
cd /tmp/gbm-test-*
./gbm init test-repo --branch main
```
**Expected**:
- Creates `test-repo/.git` (bare repository)
- Creates `test-repo/worktrees/main/` (worktree)
- Creates `test-repo/.gbm/config.yaml`
**Log**: Directory structure, config file contents

#### Test 1.4: Init with Dry Run
```bash
cd /tmp/gbm-test-*
./gbm init test-repo-dry --branch main --dry-run
```
**Expected**: Shows commands without executing
**Log**: Dry-run output

#### Test 1.5: Init in Current Directory
```bash
cd /tmp/gbm-test-*
mkdir current-dir-test && cd current-dir-test
../../gbm init --branch develop
```
**Expected**: Initializes in current directory
**Log**: Resulting directory structure

### Phase 2: Worktree Management Tests

#### Test 2.1: Add Worktree - New Branch (CLI Mode)
```bash
cd /tmp/gbm-test-*/test-repo
../../gbm worktree add feature-1 feature-1 -b
```
**Expected**: Creates worktree at `worktrees/feature-1` with new branch `feature-1`
**Log**: Command output, verify worktree exists

#### Test 2.2: Add Worktree - New Branch with Base (CLI Mode)
```bash
cd /tmp/gbm-test-*/test-repo
../../gbm worktree add feature-2 feature-2 -b --base main
```
**Expected**: Creates worktree from `main` branch
**Log**: Command output, verify branch base

#### Test 2.3: Add Worktree - Existing Branch (CLI Mode)
```bash
cd /tmp/gbm-test-*/test-repo
../../gbm worktree add main-copy main
```
**Expected**: Creates worktree for existing `main` branch
**Log**: Command output

#### Test 2.4: Add Worktree - Branch Doesn't Exist (should prompt)
```bash
cd /tmp/gbm-test-*/test-repo
echo "n" | ../../gbm worktree add nonexistent nonexistent
```
**Expected**: Prompts to create branch, user declines
**Log**: Prompt output and cancellation message

#### Test 2.5: Add Worktree - Dry Run
```bash
cd /tmp/gbm-test-*/test-repo
../../gbm worktree add feature-dry feature-dry -b --dry-run
```
**Expected**: Shows what would be created without executing
**Log**: Dry-run output

#### Test 2.6: List Worktrees
```bash
cd /tmp/gbm-test-*/test-repo
../../gbm worktree list
```
**Expected**: Shows interactive TUI table with all worktrees
**Note**: This launches a TUI - you may need to send 'q' to quit
**Log**: Number of worktrees shown, any errors

#### Test 2.7: List Worktrees - Alias
```bash
cd /tmp/gbm-test-*/test-repo
../../gbm wt ls
```
**Expected**: Same as `worktree list` (wt is alias)
**Log**: Verify alias works

#### Test 2.8: Switch Worktree - Print Path
```bash
cd /tmp/gbm-test-*/test-repo
../../gbm worktree switch feature-1 --print-path
```
**Expected**: Prints absolute path to feature-1 worktree
**Log**: Path output

#### Test 2.9: Switch Worktree - Without Shell Integration
```bash
cd /tmp/gbm-test-*/test-repo
../../gbm worktree switch feature-2
```
**Expected**: Shows instructions for cd command and shell integration
**Log**: Instruction output

#### Test 2.10: Remove Worktree
```bash
cd /tmp/gbm-test-*/test-repo
echo "n" | ../../gbm worktree remove feature-1
```
**Expected**: Removes worktree, prompts about branch deletion (decline)
**Log**: Removal output, branch still exists

#### Test 2.11: Remove Worktree - Force
```bash
cd /tmp/gbm-test-*/test-repo
# First, create some uncommitted changes
cd worktrees/feature-2
echo "test" > uncommitted.txt
cd ../..
echo "y" | ../../gbm worktree remove feature-2 --force
```
**Expected**: Force removes worktree with uncommitted changes, deletes branch
**Log**: Force removal output

#### Test 2.12: Remove Current Worktree (using ".")
```bash
cd /tmp/gbm-test-*/test-repo/worktrees/main-copy
echo "y" | ../../../gbm worktree remove .
```
**Expected**:
- Shows message "Switching to repository root before removing current worktree..."
- Automatically changes to repo root before removal
- Removes current worktree successfully
- Prompts for branch deletion
- Deletes branch successfully (no "Unable to read current working directory" error)
**Log**: Complete removal output including automatic directory switch message

### Phase 3: Clone Tests

#### Test 3.1: Clone Remote Repository
```bash
cd /tmp/gbm-test-*
./gbm clone https://github.com/bubbletea-examples/minimal.git
```
**Note**: Use a small public repo for testing
**Expected**:
- Clones as bare repo
- Creates worktree for default branch
- Creates .gbm/config.yaml
**Log**: Clone output, directory structure

#### Test 3.2: Clone with Custom Name
```bash
cd /tmp/gbm-test-*
./gbm clone https://github.com/bubbletea-examples/minimal.git custom-name
```
**Expected**: Clones into `custom-name/` directory
**Log**: Clone output, verify directory name

#### Test 3.3: Clone - Dry Run
```bash
cd /tmp/gbm-test-*
./gbm clone https://github.com/bubbletea-examples/minimal.git --dry-run
```
**Expected**: Shows clone commands without executing
**Log**: Dry-run output

### Phase 4: Configuration and Sync Tests

#### Test 4.1: Verify Config File Structure
```bash
cd /tmp/gbm-test-*/test-repo
cat .gbm/config.yaml
```
**Expected**: Shows valid YAML with worktrees configuration
**Log**: Config file contents

#### Test 4.2: Manual Config Edit - Add Worktree Definition
```bash
cd /tmp/gbm-test-*/test-repo
# Edit .gbm/config.yaml to add a new worktree definition
cat >> .gbm/config.yaml << EOF
  feature-3:
    branch: feature-3
EOF
```
**Expected**: Config updated successfully
**Log**: Updated config contents

#### Test 4.3: Sync - Show Missing Worktrees
```bash
cd /tmp/gbm-test-*/test-repo
../../gbm sync --dry-run
```
**Expected**: Shows feature-3 as missing (needs to be created)
**Log**: Sync dry-run output

#### Test 4.4: Sync - Create Missing Worktrees
```bash
cd /tmp/gbm-test-*/test-repo
../../gbm sync --force
```
**Expected**: Creates feature-3 worktree and branch
**Log**: Sync output, verify worktree created

#### Test 4.5: Sync - Detect Ad Hoc Worktrees
```bash
cd /tmp/gbm-test-*/test-repo
# Create an ad hoc worktree not in config
../../gbm worktree add adhoc-test adhoc-test -b
# Run sync
../../gbm sync --dry-run
```
**Expected**: Shows adhoc-test as orphaned (not in config)
**Log**: Sync status showing orphaned worktree

#### Test 4.6: Sync - Branch Mismatch Detection
```bash
cd /tmp/gbm-test-*/test-repo
# Manually edit config to change branch for existing worktree
# Then run sync
../../gbm sync --dry-run
```
**Expected**: Detects branch mismatches
**Log**: Mismatch detection output

#### Test 4.7: Sync - Worktree Rename Detection
```bash
cd /tmp/gbm-test-*/test-repo
# Create worktree with one name
../../gbm worktree add old-name feature-rename -b
# Edit config to rename it
# Run sync to detect rename opportunity
../../gbm sync --dry-run
```
**Expected**: Suggests rename from old-name to new name
**Log**: Rename detection output

### Phase 5: Shell Integration Tests

#### Test 5.1: Generate Shell Integration
```bash
cd /tmp/gbm-test-*/test-repo
../../gbm shell-integration
```
**Expected**: Outputs shell script for integration
**Log**: Shell integration script

#### Test 5.2: Verify Shell Integration Format
```bash
cd /tmp/gbm-test-*/test-repo
../../gbm shell-integration | head -n 5
```
**Expected**: Shows shell function definition
**Log**: First 5 lines of integration script

### Phase 6: Edge Cases and Error Handling

#### Test 6.1: Add Worktree - Duplicate Name
```bash
cd /tmp/gbm-test-*/test-repo
../../gbm worktree add main main
```
**Expected**: Error - worktree already exists
**Log**: Error message

#### Test 6.2: Remove Non-Existent Worktree
```bash
cd /tmp/gbm-test-*/test-repo
../../gbm worktree remove nonexistent-worktree
```
**Expected**: Error - worktree not found
**Log**: Error message

#### Test 6.3: Switch to Non-Existent Worktree
```bash
cd /tmp/gbm-test-*/test-repo
../../gbm worktree switch nonexistent
```
**Expected**: Error - worktree not found
**Log**: Error message

#### Test 6.4: Init in Existing Git Repository
```bash
cd /tmp/gbm-test-*/test-repo
../../gbm init already-exists
```
**Expected**: Error or warning about existing git repo
**Log**: Error/warning message

#### Test 6.5: Command Outside Git Repository
```bash
cd /tmp
./gbm-test-*/gbm worktree list
```
**Expected**: Error - not in a git repository
**Log**: Error message

#### Test 6.6: Switch Using "-" (Previous Worktree)
```bash
cd /tmp/gbm-test-*/test-repo
../../gbm worktree switch feature-3 --print-path
../../gbm worktree switch main --print-path
../../gbm worktree switch - --print-path
```
**Expected**: Returns to feature-3
**Log**: All three switch outputs

### Phase 7: Cleanup and Summary

#### Test 7.1: Count All Worktrees
```bash
cd /tmp/gbm-test-*/test-repo
../../gbm worktree list --dry-run 2>&1 | grep -c "worktree" || git worktree list | wc -l
```
**Expected**: Shows total worktree count
**Log**: Total count

#### Test 7.2: Verify Git Bare Repository Structure
```bash
cd /tmp/gbm-test-*/test-repo
ls -la .git/
```
**Expected**: Shows bare repository structure (refs, objects, etc.)
**Log**: Directory listing

#### Test 7.3: Final State Documentation
```bash
cd /tmp/gbm-test-*/test-repo
tree -L 3 -a
```
**Expected**: Shows complete repository structure
**Log**: Full tree output

## Results File Format

Create a markdown file with the following structure:

```markdown
# GBM CLI Test Results
**Timestamp**: [ISO-8601 timestamp]
**Test Directory**: [path]
**GBM Binary**: [version/commit]
**System**: [OS/arch]

## Summary
- Total Tests: X
- Passed: ✅ Y
- Failed: ❌ Z
- Success Rate: N%

## Test Results

### Phase 1: Build and Initialization Tests

#### ✅/❌ Test 1.1: Build GBM Binary
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

## Execution Instructions

1. Start tmux session: `tmux new-session -s "gbm-test-$(date +%s)"`
2. Navigate to test directory
3. Execute each test sequentially
4. Record results in real-time
5. Generate final summary
6. Leave tmux session running for review: `<Ctrl-b> d` to detach

## Success Criteria

- ✅ All core commands execute without errors
- ✅ Dry-run modes work correctly
- ✅ Error messages are clear and helpful
- ✅ Configuration files are created properly
- ✅ Worktrees are created in correct locations
- ✅ TUI interfaces launch successfully
- ✅ Edge cases are handled gracefully

## Notes for the Agent

- Use absolute paths for the GBM binary to avoid PATH issues
- Capture both stdout and stderr for all commands
- Take screenshots of TUI interfaces if possible
- Document any unexpected behavior
- Include git command outputs where relevant for verification
- Test both long-form commands and aliases (wt, etc.)
- Test 2.12: GBM automatically switches to repo root when removing current worktree to prevent "Unable to read current working directory" errors

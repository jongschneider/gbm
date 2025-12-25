# GBM CLI Test Suite - Agent Instructions

## Objective
Execute a comprehensive test of all GBM (Git Branch Manager) CLI functionality in a new tmux session. Track all test results in a timestamped markdown report.

## Setup Requirements

1. **Prompt user for VHS recording**: Ask the user "Do you want to record this test session with VHS? (yes/no)"
   - If user answers **"yes"**: Proceed with VHS recording setup and execution (steps 7-8 below)
   - If user answers **"no"**: Skip all VHS-related setup and recording steps, but execute all tests normally
2. **Set timestamp variable** for this test run: `TIMESTAMP=$(date +%Y%m%d-%H%M%S)`
3. **Create timestamped test results directory** at `/Users/jschneider/code/scratch/gbm/test-results/gbm-test-results-$TIMESTAMP/`
4. **[VHS ONLY]** Create VHS recordings subdirectory at `/Users/jschneider/code/scratch/gbm/test-results/gbm-test-results-$TIMESTAMP/vhs-recordings/`
5. **Create a new tmux session** named `gbm-test-$TIMESTAMP`
6. **Create a temporary test directory** at `/tmp/gbm-test-$TIMESTAMP`
7. **Build the GBM binary** from `/Users/jschneider/code/scratch/gbm`
8. **Create timestamped results file** at `/Users/jschneider/code/scratch/gbm/test-results/gbm-test-results-$TIMESTAMP/gbm-test-results-$TIMESTAMP.md`
9. **[VHS ONLY]** Install VHS if not already installed: `brew install vhs` or `go install github.com/charmbracelet/vhs@latest`
10. **[VHS ONLY]** Create VHS tape files for recording individual operations

## VHS Recording Setup (Optional - Only if User Answered "Yes")

**NOTE**: This entire section is ONLY applicable if the user answered "yes" to VHS recording. If they answered "no", skip this section entirely.

VHS (Video Hosting Service) allows recording terminal sessions as GIF or video files. Create a tape file to automate the recording:

**CRITICAL VHS REQUIREMENT**:
- **ALWAYS use ABSOLUTE PATHS** for the GBM binary in VHS tape files: `/Users/jschneider/code/scratch/gbm/gbm`
- **NEVER use relative paths** like `../../gbm` or `./gbm` - they will fail when VHS executes the tape
- VHS tape files can be executed from any directory, so relative paths break
- All test commands in this document already use absolute paths - use them as-is in tape files

**VHS Directory Structure**:
- All VHS tape files go in: `/Users/jschneider/code/scratch/gbm/test-results/gbm-test-results-$TIMESTAMP/vhs-recordings/`
- VHS must be run FROM this directory: `cd /Users/jschneider/code/scratch/gbm/test-results/gbm-test-results-$TIMESTAMP/vhs-recordings && vhs *.tape`
- Output GIFs use relative paths (e.g., `Output init-repository.gif`) and will be created in the vhs-recordings directory

**Create Individual VHS Tape Files** in `/Users/jschneider/code/scratch/gbm/test-results/gbm-test-results-$TIMESTAMP/vhs-recordings/`:

Example tape file (`01-init-repository.tape`):
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

Create individual tape files for each major operation (init, worktree add, pull, push, etc.)

**Recording Options**:
- **Manual recording**: Use `vhs record` in the tmux session, then manually execute tests
- **Automated recording**: Generate a complete `.tape` file with all test commands and run `vhs < tape-file.tape`
- **Hybrid approach**: Record key test phases separately, then combine

## Test Execution Guidelines

- Run all commands in the tmux session for visibility
- **[VHS ONLY]** Record the session using VHS for visual playback and review (if user answered "yes")
- Capture stdout and stderr for each command
- Mark each test as ✅ SUCCESS or ❌ FAILURE
- Include command output snippets in the results file
- Test both success and failure scenarios
- Use `--dry-run` flags where appropriate to verify command planning
- **[VHS ONLY]** Save VHS recording to `/Users/jschneider/code/scratch/gbm/test-results/gbm-test-$TIMESTAMP.gif` (if user answered "yes")

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
/Users/jschneider/code/scratch/gbm/gbm init --branch develop
```
**Expected**: Initializes in current directory
**Log**: Resulting directory structure

### Phase 2: Worktree Management Tests

#### Test 2.1: Add Worktree - New Branch (CLI Mode)
```bash
cd /tmp/gbm-test-*/test-repo
/Users/jschneider/code/scratch/gbm/gbm worktree add feature-1 feature-1 -b
```
**Expected**: Creates worktree at `worktrees/feature-1` with new branch `feature-1`
**Log**: Command output, verify worktree exists

#### Test 2.2: Add Worktree - New Branch with Base (CLI Mode)
```bash
cd /tmp/gbm-test-*/test-repo
/Users/jschneider/code/scratch/gbm/gbm worktree add feature-2 feature-2 -b --base main
```
**Expected**: Creates worktree from `main` branch
**Log**: Command output, verify branch base

#### Test 2.3: Add Worktree - Existing Branch (CLI Mode)
```bash
cd /tmp/gbm-test-*/test-repo
/Users/jschneider/code/scratch/gbm/gbm worktree add main-copy main
```
**Expected**: Creates worktree for existing `main` branch
**Log**: Command output

#### Test 2.4: Add Worktree - Branch Doesn't Exist (should prompt)
```bash
cd /tmp/gbm-test-*/test-repo
echo "n" | /Users/jschneider/code/scratch/gbm/gbm worktree add nonexistent nonexistent
```
**Expected**: Prompts to create branch, user declines
**Log**: Prompt output and cancellation message

#### Test 2.5: Add Worktree - Dry Run
```bash
cd /tmp/gbm-test-*/test-repo
/Users/jschneider/code/scratch/gbm/gbm worktree add feature-dry feature-dry -b --dry-run
```
**Expected**: Shows what would be created without executing
**Log**: Dry-run output

#### Test 2.6: List Worktrees
```bash
cd /tmp/gbm-test-*/test-repo
/Users/jschneider/code/scratch/gbm/gbm worktree list
```
**Expected**: Shows interactive TUI table with all worktrees
**Note**: This launches a TUI - you may need to send 'q' to quit
**Log**: Number of worktrees shown, any errors

#### Test 2.7: List Worktrees - Alias
```bash
cd /tmp/gbm-test-*/test-repo
/Users/jschneider/code/scratch/gbm/gbm wt ls
```
**Expected**: Same as `worktree list` (wt is alias)
**Log**: Verify alias works

#### Test 2.8: Switch Worktree - Print Path
```bash
cd /tmp/gbm-test-*/test-repo
/Users/jschneider/code/scratch/gbm/gbm worktree switch feature-1 --print-path
```
**Expected**: Prints absolute path to feature-1 worktree
**Log**: Path output

#### Test 2.9: Switch Worktree - Without Shell Integration
```bash
cd /tmp/gbm-test-*/test-repo
/Users/jschneider/code/scratch/gbm/gbm worktree switch feature-2
```
**Expected**: Shows instructions for cd command and shell integration
**Log**: Instruction output

#### Test 2.10: Remove Worktree
```bash
cd /tmp/gbm-test-*/test-repo
echo "n" | /Users/jschneider/code/scratch/gbm/gbm worktree remove feature-1
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
echo "y" | /Users/jschneider/code/scratch/gbm/gbm worktree remove feature-2 --force
```
**Expected**: Force removes worktree with uncommitted changes, deletes branch
**Log**: Force removal output

#### Test 2.12: Remove Current Worktree (using ".")
```bash
cd /tmp/gbm-test-*/test-repo/worktrees/main-copy
echo "y" | /Users/jschneider/code/scratch/gbm/gbm worktree remove .
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
/Users/jschneider/code/scratch/gbm/gbm sync --dry-run
```
**Expected**: Shows feature-3 as missing (needs to be created)
**Log**: Sync dry-run output

#### Test 4.4: Sync - Create Missing Worktrees
```bash
cd /tmp/gbm-test-*/test-repo
/Users/jschneider/code/scratch/gbm/gbm sync --force
```
**Expected**: Creates feature-3 worktree and branch
**Log**: Sync output, verify worktree created

#### Test 4.5: Sync - Detect Ad Hoc Worktrees
```bash
cd /tmp/gbm-test-*/test-repo
# Create an ad hoc worktree not in config
/Users/jschneider/code/scratch/gbm/gbm worktree add adhoc-test adhoc-test -b
# Run sync
/Users/jschneider/code/scratch/gbm/gbm sync --dry-run
```
**Expected**: Shows adhoc-test as orphaned (not in config)
**Log**: Sync status showing orphaned worktree

#### Test 4.6: Sync - Branch Mismatch Detection
```bash
cd /tmp/gbm-test-*/test-repo
# Manually edit config to change branch for existing worktree
# Then run sync
/Users/jschneider/code/scratch/gbm/gbm sync --dry-run
```
**Expected**: Detects branch mismatches
**Log**: Mismatch detection output

#### Test 4.7: Sync - Worktree Rename Detection
```bash
cd /tmp/gbm-test-*/test-repo
# Create worktree with one name
/Users/jschneider/code/scratch/gbm/gbm worktree add old-name feature-rename -b
# Edit config to rename it
# Run sync to detect rename opportunity
/Users/jschneider/code/scratch/gbm/gbm sync --dry-run
```
**Expected**: Suggests rename from old-name to new name
**Log**: Rename detection output

### Phase 5: Shell Integration Tests

#### Test 5.1: Generate Shell Integration
```bash
cd /tmp/gbm-test-*/test-repo
/Users/jschneider/code/scratch/gbm/gbm shell-integration
```
**Expected**: Outputs shell script for integration
**Log**: Shell integration script

#### Test 5.2: Verify Shell Integration Format
```bash
cd /tmp/gbm-test-*/test-repo
/Users/jschneider/code/scratch/gbm/gbm shell-integration | head -n 5
```
**Expected**: Shows shell function definition
**Log**: First 5 lines of integration script

### Phase 6: Edge Cases and Error Handling

#### Test 6.1: Add Worktree - Duplicate Name
```bash
cd /tmp/gbm-test-*/test-repo
/Users/jschneider/code/scratch/gbm/gbm worktree add main main
```
**Expected**: Error - worktree already exists
**Log**: Error message

#### Test 6.2: Remove Non-Existent Worktree
```bash
cd /tmp/gbm-test-*/test-repo
/Users/jschneider/code/scratch/gbm/gbm worktree remove nonexistent-worktree
```
**Expected**: Error - worktree not found
**Log**: Error message

#### Test 6.3: Switch to Non-Existent Worktree
```bash
cd /tmp/gbm-test-*/test-repo
/Users/jschneider/code/scratch/gbm/gbm worktree switch nonexistent
```
**Expected**: Error - worktree not found
**Log**: Error message

#### Test 6.4: Init in Existing Git Repository
```bash
cd /tmp/gbm-test-*/test-repo
/Users/jschneider/code/scratch/gbm/gbm init already-exists
```
**Expected**: Error or warning about existing git repo
**Log**: Error/warning message

#### Test 6.5: Command Outside Git Repository
```bash
cd /tmp
/Users/jschneider/code/scratch/gbm/gbm worktree list
```
**Expected**: Error - not in a git repository
**Log**: Error message

#### Test 6.6: Switch Using "-" (Previous Worktree)
```bash
cd /tmp/gbm-test-*/test-repo
/Users/jschneider/code/scratch/gbm/gbm worktree switch feature-3 --print-path
/Users/jschneider/code/scratch/gbm/gbm worktree switch main --print-path
/Users/jschneider/code/scratch/gbm/gbm worktree switch - --print-path
```
**Expected**: Returns to feature-3
**Log**: All three switch outputs

### Phase 7: Cleanup and Summary

#### Test 7.1: Count All Worktrees
```bash
cd /tmp/gbm-test-*/test-repo
/Users/jschneider/code/scratch/gbm/gbm worktree list --dry-run 2>&1 | grep -c "worktree" || git worktree list | wc -l
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

### Setup Phase
```bash
# Set timestamp for this test run
export TIMESTAMP=$(date +%Y%m%d-%H%M%S)

# Create timestamped test results directory
mkdir -p /Users/jschneider/code/scratch/gbm/test-results/gbm-test-results-$TIMESTAMP

# [VHS ONLY] Create VHS recordings subdirectory
mkdir -p /Users/jschneider/code/scratch/gbm/test-results/gbm-test-results-$TIMESTAMP/vhs-recordings

# Build GBM binary
cd /Users/jschneider/code/scratch/gbm
just build

# Create temporary test directory
mkdir -p /tmp/gbm-test-$TIMESTAMP
```

### Recording Options (Only if User Answered "Yes" to VHS)

**NOTE**: The following recording options are ONLY for users who chose to use VHS. If the user answered "no", proceed directly to executing tests without any recording setup.

#### Option 1: Manual Recording with VHS
```bash
# Start tmux session
tmux new-session -s "gbm-test-$TIMESTAMP"

# Inside tmux, start VHS recording
vhs record --output /Users/jschneider/code/scratch/gbm/test-results/gbm-test-$TIMESTAMP.gif

# Execute tests manually
# Press Ctrl-D to stop recording when done
```

#### Option 2: Automated Recording with VHS Tape Files
```bash
# Create individual tape files for each operation in vhs-recordings/
# Then run from the vhs-recordings directory:
cd /Users/jschneider/code/scratch/gbm/test-results/gbm-test-results-$TIMESTAMP/vhs-recordings
vhs 01-init-repository.tape
vhs 02-worktree-add.tape
# ... etc for each operation
```

#### Option 3: Screen Recording (Alternative)
```bash
# Use asciinema or ttyrec as alternatives
asciinema rec /Users/jschneider/code/scratch/gbm/test-results/gbm-test-$TIMESTAMP.cast
```

### Execution Flow

**If User Answered "Yes" to VHS:**
1. Start tmux session: `tmux new-session -s "gbm-test-$TIMESTAMP"`
2. Navigate to test directory
3. Execute each test sequentially
4. Record results in `/Users/jschneider/code/scratch/gbm/test-results/gbm-test-results-$TIMESTAMP/gbm-test-results-$TIMESTAMP.md`
5. Create individual VHS tape files in `/Users/jschneider/code/scratch/gbm/test-results/gbm-test-results-$TIMESTAMP/vhs-recordings/`
6. Generate VHS recordings: `cd vhs-recordings && vhs *.tape`
7. Verify recordings saved: `ls -lh /Users/jschneider/code/scratch/gbm/test-results/gbm-test-results-$TIMESTAMP/vhs-recordings/*.gif`
8. Leave tmux session running for review: `<Ctrl-b> d` to detach

**If User Answered "No" to VHS:**
1. Start tmux session: `tmux new-session -s "gbm-test-$TIMESTAMP"`
2. Navigate to test directory
3. Execute each test sequentially
4. Record results in `/Users/jschneider/code/scratch/gbm/test-results/gbm-test-results-$TIMESTAMP/gbm-test-results-$TIMESTAMP.md`
5. Leave tmux session running for review: `<Ctrl-b> d` to detach

### Post-Execution
- Review markdown results: `less /Users/jschneider/code/scratch/gbm/test-results/gbm-test-results-$TIMESTAMP/gbm-test-results-$TIMESTAMP.md`
- **[VHS ONLY]** View individual VHS recordings: `open /Users/jschneider/code/scratch/gbm/test-results/gbm-test-results-$TIMESTAMP/vhs-recordings/*.gif`
- **[VHS ONLY]** Archive the entire test run directory: `tar -czf gbm-test-results-$TIMESTAMP.tar.gz gbm-test-results-$TIMESTAMP/`
- **[NO VHS]** Archive the test run directory: `tar -czf gbm-test-results-$TIMESTAMP.tar.gz gbm-test-results-$TIMESTAMP/`

### Directory Structure After Test Run

**With VHS:**
```
test-results/
└── gbm-test-results-20251225-093816/
    ├── gbm-test-results-20251225-093816.md        # Original test report
    ├── gbm-test-results-with-recordings.md         # Enhanced report with GIF references
    └── vhs-recordings/
        ├── README.md                               # VHS recordings documentation
        ├── 01-init-repository.tape                 # VHS tape files
        ├── 02-worktree-add.tape
        ├── ... (more tape files)
        ├── init-repository.gif                     # Generated recordings
        ├── worktree-add.gif
        └── ... (more GIF files)
```

**Without VHS:**
```
test-results/
└── gbm-test-results-20251225-093816/
    └── gbm-test-results-20251225-093816.md        # Test report
```

## Success Criteria

- ✅ All core commands execute without errors
- ✅ Dry-run modes work correctly
- ✅ Error messages are clear and helpful
- ✅ Configuration files are created properly
- ✅ Worktrees are created in correct locations
- ✅ TUI interfaces launch successfully
- ✅ Edge cases are handled gracefully

## Notes for the Agent

- **FIRST**: Prompt the user with "Do you want to record this test session with VHS? (yes/no)" and wait for their response
- If user answers "yes": Follow all VHS-related instructions throughout this document
- If user answers "no": Skip ALL VHS-related setup and recording steps, but execute all tests normally
- **Directory Structure**: All test artifacts go in `/Users/jschneider/code/scratch/gbm/test-results/gbm-test-results-$TIMESTAMP/`
- **VHS Structure**: VHS tape files and recordings go in `gbm-test-results-$TIMESTAMP/vhs-recordings/`
- **CRITICAL**: Use absolute paths for the GBM binary in all commands: `/Users/jschneider/code/scratch/gbm/gbm`
- **CRITICAL**: Never use relative paths like `../../gbm` or `./gbm` - they will fail in VHS recordings
- Capture both stdout and stderr for all commands
- Document any unexpected behavior
- Include git command outputs where relevant for verification
- Test both long-form commands and aliases (wt, etc.)
- Test 2.12: GBM automatically switches to repo root when removing current worktree to prevent "Unable to read current working directory" errors
- **[VHS ONLY]** Create individual tape files for each major operation (init, worktree add, pull, push, etc.)
- **[VHS ONLY]** VHS recordings provide visual proof of TUI interfaces and interactive commands
- **[VHS ONLY]** All test artifacts (markdown reports, VHS tapes, GIF recordings) use the same timestamp for easy correlation

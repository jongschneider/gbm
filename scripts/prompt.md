You are completing tasks from a PRD. Follow this process EXACTLY:

## 1. Get Bearings
- Read the progress file - CHECK 'Codebase Patterns' SECTION FIRST
- Read the PRD - find the next task with passes: false
- Read /specs/README.md
- run 'just' to see available commands
- Task Priority (highest to lowest):
  1. Architecture/core abstractions
  2. Integration points
  3. Spikes/unknowns
  4. Standard features
  5. Polish/cleanup
- Check recent git history: git log --oneline -10

## 2. Initialize Progress (if needed)
If progress.txt doesn't exist, create it with this template:
# Progress Log
PRD: <prdName>
Started: <YYYY-MM-DD>

## Codebase Patterns
<!-- Consolidate reusable patterns here -->

---
<!-- Task logs below - APPEND ONLY -->

## 3. Choose the Task
Choose the highest priority task to do.
    - Not just the next task.
    - Consider which task will make the rest easier to accomplish.

## 4. Implement the Task
Work on the SINGLE incomplete task until verification passes.
Use unit tests or property-based tests (https://github.com/flyingmutant/rapid) when appropriate.

## 5. Feedback Loops (REQUIRED)
Before committing, run ALL applicable:
    - just check
DO NOT commit if any fail. Fix issues first.

## 6. Update PRD (REQUIRED - DO NOT SKIP)
CRITICAL: Update the prd.json file in specs/state/{prd-name}/:
- Find the task description that matches what you just completed
- Change "passes": false to "passes": true
- This must happen BEFORE the final commit
- If you forget this, the progress tracking breaks

## 7. Update Progress
Append to progress.txt:
## Task - [task.id]
- What was implemented
- Files changed
- **Learnings:** patterns, gotchas

If you discover a reusable pattern, also add to ## Codebase Patterns at the TOP.

## 8. Commit (includes prd.json + progress.txt + code changes)
VERIFY before committing:
- [ ] prd.json updated with passes: true
- [ ] progress.txt appended with task notes
- [ ] Code changes completed and tested
Then commit:
    git add -A && git commit -m 'feat(<scope>): <description>'

## 9. Check Completion
If ALL tasks in the PRD now have passes: true, output the following marker on its own line:
<promise>COMPLETE</promise>
Do NOT mention or quote this marker unless you are actually outputting it.

## Philosophy
This codebase will outlive you. Every shortcut becomes someone else's burden.
Fight entropy. Leave the codebase better than you found it.

ONLY DO ONE TASK. Stop after committing.

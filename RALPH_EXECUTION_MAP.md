# Ralph Execution Map: Phase 2 Stories

## Story Dependency Graph

```
Story 1: Async Messages (Foundation)
  ├─ no dependencies
  └─ OUTPUT: FetchMsg, FetchCmd patterns

Story 2: Update Filterable
  ├─ DEPENDS ON: Story 1
  └─ OUTPUT: Spinner UI, non-blocking field

Story 4: Teatest Helpers (parallel with 2-3)
  ├─ no dependencies
  └─ OUTPUT: Test utilities + examples

Story 3: VHS Recordings (parallel with 4)
  ├─ DEPENDS ON: Story 2 (async visible in recordings)
  └─ OUTPUT: .tape scripts + .gif baselines

Story 5: Async Integration Tests
  ├─ DEPENDS ON: Stories 1, 2, 4
  └─ OUTPUT: Comprehensive teatest suite

Story 6: Navigator Root Model (can start anytime)
  ├─ no dependencies
  └─ OUTPUT: Multi-screen composition pattern

Story 7: Update Testadd
  ├─ DEPENDS ON: Story 6
  └─ OUTPUT: Cleaner stage management

Story 8: Custom Field Storage (can start anytime)
  ├─ no dependencies
  └─ OUTPUT: Generic field map

Story 9: Merge Workflow
  ├─ DEPENDS ON: Stories 2, 4, 8
  └─ OUTPUT: Reference workflow using all patterns

Story 10: Documentation
  ├─ DEPENDS ON: All stories (comes last)
  └─ OUTPUT: Architecture docs + comments
```

## Recommended Execution Order (Ralph Linear Loop)

Ralph executes stories **in this exact order**:

```
1. Story 1 (Async Messages)
   ↓ [3-5 hours]
2. Story 4 (Teatest Helpers) ← Can run in parallel with 2
   ↓ [1-2 hours]
3. Story 2 (Update Filterable)
   ↓ [2-3 hours]
4. Story 3 (VHS Recordings)
   ↓ [1-2 hours]
5. Story 5 (Async Integration Tests)
   ↓ [2-3 hours]
6. Story 6 (Navigator)
   ↓ [1-2 hours]
7. Story 7 (Update Testadd)
   ↓ [1-2 hours]
8. Story 8 (Custom Field Storage)
   ↓ [1-2 hours]
9. Story 9 (Merge Workflow)
   ↓ [2-3 hours]
10. Story 10 (Documentation)
   ↓ [1-2 hours]
[DONE] ← All 10 stories complete
```

**Why this order?**
- Foundation first (1): Async patterns needed by everything
- Teatest helpers (4): Used by all test stories
- Filterable (2): Demonstrates async pattern immediately
- VHS (3): Shows spinners from step 2
- Tests (5): Validates everything before architecture changes
- Architecture (6-8): Refactors on proven foundation
- Merge (9): Uses all three pattern types
- Docs (10): Written after implementation

---

## Ralph's Decision Loop (Per Story)

For each story, Ralph:

```
┌─ Read Story from PRD_PHASE2.md
│
├─ Parse:
│  ├─ Title
│  ├─ Description
│  ├─ Acceptance Criteria (list)
│  ├─ Files (create/update)
│  └─ Depends On (verify completed)
│
├─ Check Dependencies:
│  ├─ Are all required stories done?
│  ├─ Are those stories' tests passing?
│  └─ Is git in clean state?
│
├─ Make Code Changes:
│  ├─ Read relevant files
│  ├─ Review TUI_IMPROVEMENTS.md for patterns
│  ├─ Implement acceptance criteria
│  ├─ Follow AGENTS.md style guide
│  └─ Copy patterns from TESTING_VALIDATION_STRATEGY.md
│
├─ Run Tests:
│  ├─ go test ./... -v
│  ├─ go test ./... -race
│  ├─ go test ./... -cover (check > 80%)
│  └─ All must pass
│
├─ VHS Step (if applicable):
│  ├─ Create .tape script in spec/vhs/
│  ├─ vhs < script.tape → script.gif
│  └─ Verify .gif looks reasonable
│
├─ Commit:
│  ├─ git add .
│  ├─ git commit -m "story-N: [Title]
│  │   Acceptance Criteria:
│  │   - [ ] Criterion 1
│  │   ..."
│  └─ Verify: git log --oneline | head -1
│
├─ Report:
│  ├─ Story N complete ✓
│  ├─ X tests passed
│  ├─ Coverage: Y%
│  ├─ N VHS recordings
│  └─ Ready for story N+1
│
└─ Loop to next story OR exit if N=10
```

---

## File Changes by Story

**Story 1**:
- CREATE: `pkg/tui/async/messages.go`
- CREATE: `pkg/tui/async/messages_test.go`

**Story 2**:
- UPDATE: `pkg/tui/fields/filterable.go`
- UPDATE: `pkg/tui/fields/filterable_test.go`

**Story 3**:
- CREATE: `spec/vhs/*.tape` (4 files)
- CREATE: `spec/vhs/*.gif` (4 files)
- UPDATE: `justfile` (add vhs-record target)

**Story 4**:
- CREATE: `testutil/teatest_helpers.go`
- CREATE: `testutil/teatest_helpers_test.go`
- CREATE: `testutil/README.md`

**Story 5**:
- UPDATE: `pkg/tui/fields/filterable_test.go` (add teatest cases)
- UPDATE: `pkg/tui/fields/*_test.go` (if other fields needed)

**Story 6**:
- CREATE: `pkg/tui/navigator.go`
- CREATE: `pkg/tui/navigator_test.go`

**Story 7**:
- UPDATE: `cmd/service/worktree_testadd.go`
- UPDATE: `pkg/tui/workflows/workflows.go` (if needed)

**Story 8**:
- UPDATE: `pkg/tui/context.go`
- UPDATE: `pkg/tui/wizard.go`
- CREATE: `pkg/tui/context_test.go` (if not exists)

**Story 9**:
- CREATE: `pkg/tui/workflows/merge_custom.go`
- CREATE: `pkg/tui/workflows/merge_custom_test.go`
- CREATE: `spec/vhs/testadd_merge_custom_workflow.tape`
- UPDATE: `pkg/tui/workflows/workflows.go` (router)

**Story 10**:
- UPDATE: `pkg/tui/wizard.go` (add comments)
- UPDATE: `pkg/tui/field.go` (add comments)
- UPDATE: `pkg/tui/fields/filterable.go` (add comments)
- CREATE: `pkg/tui/ARCHITECTURE.md`

---

## Commit Format Ralph Should Use

```bash
git commit -m "story-N: [Title from PRD]

Implements acceptance criteria:
- FetchMsg type with Value and Err fields
- FetchCmd command factory
- Unit tests for async operations
- No blocking calls

Depends on: [previous story if any]
Files: [list of changed files]
Tests: X/Y passed, Z% coverage
VHS: [if applicable]"
```

---

## Git History After Ralph Completes

```bash
$ git log --oneline | head -15
story-10: Update Wizard with Architecture Documentation
story-9: Create Merge Workflow with Custom Fields
story-8: Add Custom Field Storage to WorkflowState
story-7: Update Testadd to Use Navigator
story-6: Create Navigator Root Model
story-5: Add Async Integration Tests for Filterable
story-4: Set Up Teatest Integration Test Framework
story-3: Add VHS Recording Scripts and Baseline GIFs
story-2: Update Filterable Field to Use Async Commands
story-1: Create Async Message Types and Commands
docs: Add Phase 2 summary and Ralph quick start guide
docs: Add Phase 2 testing strategy and PRD with 10 user stories
main: [previous commits]
```

---

## Success Criteria Ralph Must Meet

For each story:
- [ ] All acceptance criteria addressed
- [ ] All tests pass (`go test ./...`)
- [ ] Coverage > 80% for changed files
- [ ] No compiler warnings
- [ ] No lint issues (if linting enabled)
- [ ] VHS recordings created (if VHS story)
- [ ] Commit message clear and actionable

For Phase 2 completion:
- [ ] All 10 stories complete
- [ ] Full test suite passes
- [ ] All VHS recordings committed
- [ ] Git history clean (10 commits, 1 per story)
- [ ] Ready for manual review/testing

---

## Rollback Strategy (if something fails)

**If story N fails**:

```bash
# See what Ralph did
git log --oneline | head -1
git show HEAD

# Rollback to previous good state
git reset --hard HEAD~1

# Fix the issue (manually or update story)
# Then re-run Ralph on that story
ralph PRD_PHASE2.md --start-story N
```

**If multiple stories fail**:

```bash
# Rollback to baseline (before Ralph started)
git reset --hard <baseline-commit-hash>

# Review the failed story
# Update acceptance criteria if unclear
# Re-run Ralph from story 1
ralph PRD_PHASE2.md
```

---

## Monitoring During Execution

**What Ralph should output** (per story):

```
[Story 1/10] Create Async Message Types and Commands
├─ Reading: PRD_PHASE2.md story 1
├─ Checking dependencies: none ✓
├─ Files to create:
│  ├─ pkg/tui/async/messages.go
│  └─ pkg/tui/async/messages_test.go
├─ Implementing...
├─ Running: go test ./...
│  └─ ok: 45 tests passed in 1.2s
├─ Running: go test ./... -cover
│  └─ ok: coverage 82.3%
├─ Committing...
│  └─ [story-1 abc1234] Create Async Message Types and Commands
└─ ✅ Story 1 complete!

[Story 2/10] Update Filterable Field to Use Async Commands
├─ Reading: PRD_PHASE2.md story 2
├─ Checking dependencies: Story 1 ✓
├─ Files to update:
│  ├─ pkg/tui/fields/filterable.go
│  └─ pkg/tui/fields/filterable_test.go
├─ Implementing...
...
```

---

## Ralph Skill Integration

If using the Ralph skill:

```bash
# In Ralph skill, add Phase 2 PRD:
skill ralph \
  --prd PRD_PHASE2.md \
  --max-stories 10 \
  --commit-on-complete

# Ralph will:
# 1. Load PRD_PHASE2.md
# 2. Parse 10 stories
# 3. Execute each in order
# 4. Commit after each
# 5. Report progress
```

---

## Questions Ralph Might Need Clarification On

**If Ralph asks**: "What does 'acceptance criteria' mean?"
- Answer: The checklist under each story that must all be ✓ before story is done

**If Ralph asks**: "Should I commit after every test, or after whole story?"
- Answer: After whole story (when all acceptance criteria are met and tests pass)

**If Ralph asks**: "What if a test already existed for this code?"
- Answer: Update the test, don't replace it. Preserve existing test coverage.

**If Ralph asks**: "Should I fix linting issues?"
- Answer: Yes. Use `go fmt`, `golangci-lint` if available. Clean code.

---

## TL;DR

Ralph executes 10 stories in order, each with:
1. Read PRD criteria
2. Check dependencies
3. Code changes
4. Test validation
5. VHS recording (if applicable)
6. Git commit
7. Report status

Repeat until story 10 complete.

Total: 15-25 hours of autonomous work.

You review at the end. 👍

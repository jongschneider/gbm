# Quick Start: Running Ralph for Phase 2

## One-Minute Version

```bash
# 1. Read the plan (you are here)
# 2. Verify baseline commit exists
git log --oneline | head -3

# 3. Run Ralph with Phase 2 PRD
your-ralph-command --prd PRD_PHASE2.md

# 4. Come back in 2-3 hours to review
```

---

## The Plan (TL;DR)

**10 stories** to implement Phase 2 improvements:
1. Async message types (FetchMsg, FetchCmd)
2. Update Filterable field to use async
3. VHS recording scripts
4. Teatest test helpers
5. Async integration tests
6. Navigator root model
7. Update testadd to use Navigator
8. Custom field storage
9. Merge workflow with custom fields
10. Architecture documentation

**Each story**:
- Has acceptance criteria
- Includes test requirements (>80% coverage)
- Commits at end
- Takes 1-3 hours

**Total**: 15-25 hours of Ralph work (fully autonomous)

---

## Before Running Ralph

### Checklist

```bash
# 1. Branch is clean
git status
# Expected: nothing to commit, working tree clean

# 2. Latest baseline commit exists
git log --oneline | grep "docs: Add Phase 2"
# Expected: see the commit

# 3. PRD file exists
ls -l PRD_PHASE2.md
# Expected: file exists, readable

# 4. Can run tests
go test ./... -v
# Expected: tests pass (or at least run without crash)

# 5. Go version
go version
# Expected: go 1.25+

# 6. VHS installed
vhs --version
# Expected: vhs 0.10.0 (or similar)
```

All ✅? Ready for Ralph.

---

## Ralph Execution

### If Ralph already knows PRD format:

```bash
ralph PRD_PHASE2.md
```

### If Ralph needs a command:

```bash
ralph \
  --project "GBM Phase 2 TUI Improvements" \
  --prd PRD_PHASE2.md \
  --output-dir ~/results
```

### To run specific stories only:

```bash
# Just stories 1-2 (foundation)
ralph PRD_PHASE2.md --stories 1,2

# Or stories 3-5 (testing)
ralph PRD_PHASE2.md --stories 3,4,5
```

---

## What Ralph Will Do

For **each story** (in order):

1. **Read story** from PRD_PHASE2.md
2. **Understand requirements** from acceptance criteria
3. **Make code changes**:
   - Create/update files listed in "Files" section
   - Follow patterns from TUI_IMPROVEMENTS.md
4. **Run tests**:
   ```bash
   go test ./... -v
   go test ./... -race
   go test ./... -coverprofile=coverage.out
   ```
5. **Check coverage** (must be > 80% for changed files)
6. **If VHS story**:
   ```bash
   cd spec/vhs
   vhs < filename.tape
   # Creates filename.gif
   cd -
   ```
7. **Commit**:
   ```bash
   git add .
   git commit -m "story-N: [title]"
   ```
8. **Report**: Story complete, tests passing, coverage OK

---

## Monitoring Progress

### Live (if Ralph logs to stdout):

```bash
# Watch Ralph output
tail -f ralph.log

# Or in another terminal, watch tests
watch -n 5 "go test ./... -v"
```

### After completion:

```bash
# See all commits Ralph made
git log --oneline | head -20

# See what files changed
git diff main..HEAD --stat

# Review test coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# Check VHS recordings
ls -la spec/vhs/*.gif
```

---

## Troubleshooting

### Ralph stops on Story X

**Check the logs**:
```bash
git log --oneline | head -5
# See what commit was last made

git show HEAD
# See what Ralph did
```

**Common issues**:
1. **Test failure**: Ralph should report which test failed
   - Read the assertion failure
   - Review Ralph's code change
   - Hand-fix or rollback: `git reset --hard HEAD~1`

2. **VHS fails**: Usually means the TUI crashed
   - Run manually: `go run ./cmd/service testadd`
   - Check error output
   - Fix code bug
   - Re-run Ralph on that story

3. **Coverage drops**: Ralph made change but didn't add tests
   - Review the code
   - Add missing tests
   - Re-run Ralph on that story

### To pause Ralph

```bash
# Ralph will stop after current story completes
# Just interrupt: Ctrl+C

# To resume later
git log --oneline | head -1
# Note the story number from commit message

# Restart Ralph from that story
ralph PRD_PHASE2.md --start-story N+1
```

### To rollback a story

```bash
# Last commit was bad
git reset --hard HEAD~1

# Check you're at a good state
git log --oneline | head -1

# Restart Ralph on that story
ralph PRD_PHASE2.md --start-story N
```

---

## Success Criteria

When Ralph finishes, verify:

```bash
# 1. All 10 stories committed
git log --oneline | grep "story-" | wc -l
# Expected: 10

# 2. All tests pass
go test ./...
# Expected: ok

# 3. Coverage > 80%
go test ./... -cover
# Expected: coverage >= 80.0%

# 4. VHS recordings created
ls spec/vhs/*.gif | wc -l
# Expected: 4 (or more)

# 5. Can run TUI manually
go run ./cmd/service testadd
# Expected: TUI starts, can interact (ESC to quit)
```

All ✅? Phase 2 is complete! 🎉

---

## Next Steps After Ralph

1. **Review**: Read the 10 commits Ralph made
2. **Test manually**: Try the TUI with various workflows
3. **Code review**: Check async patterns against BUBBLETEA.md
4. **Merge**: If satisfied, merge to main

---

## Questions During Ralph Execution?

1. **What's Ralph doing?** Check the commit message
2. **Why did it fail?** Look at test output in git show
3. **Need to adjust?** Review the story criteria—was it clear enough?

---

## Need Help?

- **Testing strategy**: Read `spec/TESTING_VALIDATION_STRATEGY.md`
- **Story details**: Read `PRD_PHASE2.md`
- **Implementation ref**: Read `spec/TUI_IMPROVEMENTS.md`
- **Best practices**: Read `spec/BUBBLETEA.md`

---

## Estimated Timeline

- Stories 1-2: 3-5 hours
- Stories 3-5: 4-7 hours
- Stories 6-8: 4-6 hours
- Stories 9-10: 3-5 hours
- **Total**: 15-25 hours (mostly sleeping time ☺️)

Ralph should have Phase 2 done by tomorrow morning.

---

Ready? Run Ralph! 🚀

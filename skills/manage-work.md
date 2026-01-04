# Manage Work Skill

Provides a standardized workflow for implementing planned work on GBM using task plans and progress tracking files.

---

## How to Use

When asked to implement work on GBM:

1. **Read the current task plan and progress files** for the priority you're working on
2. **Choose a logical task or phase** to implement (based on dependencies and effort)
3. **Track progress** in the *-progress.md file using the provided checkboxes and logs
4. **Build and test frequently** using `just` commands
5. **When finished, ask for review** - don't mark complete or commit until approved
6. **Update the progress file** only after approval

---

## Choosing Available Work

```
1. Open cli-flags-implementation-plan.md (read goals & tasks)
2. Open cli-flags-implementation-progress.md (track progress)
3. Choose logical next task
4. Use just build / just test / just run
5. Update checkboxes as you complete tasks
6. When finished, ask for review
```

---

## Key Commands

These are the commands you'll use frequently while implementing work:

```bash
just build              # Build the binary
just run                # Build and run
just test               # Run all tests
just test-changed       # Run tests for changed files only
just validate           # Full validation (format, vet, lint, test)
just quick              # Quick validation (format, vet only)
just format             # Format changed files
just lint               # Lint changed files
just show-changed       # Show what files have changed
```

---

## Checklist Template

Use these sections to track progress in the *-progress.md file:

### Phase Tracking
```
**Estimated Effort:** 1 hour  
**Status:** 🔴 NOT STARTED / 🟡 IN PROGRESS / 🟢 COMPLETE

### Task 1.1: [Description]
- [ ] Item 1
- [ ] Item 2
- [ ] Tests passing
```

### Checkpoint Validation
After each phase, run:
```bash
just validate
```

All tests must pass before moving to next phase.

---

## Common Patterns

### File Creation
When creating a new file, follow these steps:
1. Create the file with proper package declaration
2. Add comprehensive comments
3. Write basic tests immediately
4. Run `just format` to ensure formatting
5. Update progress file

### Test Writing
```go
func TestFeatureName(t *testing.T) {
    // Arrange
    // Act
    // Assert - use require for critical, assert for validation
}
```

Use testify assertions (already imported in tests):
- `require.NoError(t, err)` - Fail-fast for critical checks
- `assert.Equal(t, expected, actual)` - Collect all failures

---

## Tips for Success

1. **Read the plan carefully** - Don't skip sections, understand the big picture
2. **Test frequently** - Run `just test` after each significant change
3. **Update progress as you go** - Don't wait until the end
4. **Follow the architecture** - See CLAUDE.md for patterns
5. **Keep commits clean** - Work in phases, each phase is a logical unit
6. **Ask questions** - If something is unclear, ask before implementing
7. **Use stdout/stderr properly** - Data to stdout, messages to stderr (GBM pattern)

---

## Troubleshooting

### Tests fail after changes
```bash
just quick              # Fast format/vet check
just format             # Auto-format the code
just validate           # Full validation
```

### Not sure what changed
```bash
just show-changed       # See changed files
git diff                # See actual changes
```

### Compilation errors
```bash
just compile            # Check if code compiles
go build ./cmd          # Direct build (more detailed errors)
```

### Specific test failing
```bash
go test -run TestName ./cmd/service -v
go test ./cmd/service -v -run TestName
```

---

## When You're Done

1. ✅ All tests passing
2. ✅ Manual testing complete
3. ✅ Progress file updated with details
4. ✅ Code formatted and linted
5. ❌ Do NOT commit yet
6. ❌ Do NOT mark complete in progress file yet
7. ✅ Ask for review: "The work is complete, please evaluate"
8. ⏳ Wait for approval
9. ✅ After approval, update progress file status
10. ✅ After approval, commit changes

---

## File Organization

```
gbm/worktrees/cli_improvements/
├── cli-flags-implementation-plan.md      # Current task plan
├── cli-flags-implementation-progress.md   # Current task progress
├── cli-improvement-analysis.md           # Analysis document
├── skills/
│   └── manage-work.md                    # This file
└── cmd/service/
    ├── root.go                           # Where to add flags
    ├── service.go                        # Main service
    └── ...
```
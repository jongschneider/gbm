# Phase 2 Documentation Structure

## The Problem

We created comprehensive documentation (PHASE2_SUMMARY.md, PRD_PHASE2.md, RALPH_EXECUTION_MAP.md, etc.) but it's verbose and hard to navigate.

## The Solution

### 🎯 NEW: [spec/IMPLEMENTATION_PLAN.md](./spec/IMPLEMENTATION_PLAN.md)
Single source of truth for Phase 2 work:
- **10 stories** with bullet points (not prose)
- **Execution order** with timeline
- **Citations to specifications** for deep dives
- **Acceptance criteria** from PRD
- **File locations** to change
- **Dependencies** between stories

**Read time**: 15 minutes  
**Use case**: "I want to know what to build and why"

---

### 📚 Original Documentation (Still Available)

| File | Use If | Read Time |
|------|--------|-----------|
| [QUICK_START_RALPH.md](./QUICK_START_RALPH.md) | You're running Ralph now | 5 min |
| [PRD_PHASE2.md](./PRD_PHASE2.md) | You need detailed acceptance criteria | 20 min |
| [PHASE2_SUMMARY.md](./PHASE2_SUMMARY.md) | You want executive overview | 5 min |
| [START_HERE.md](./START_HERE.md) | You're undecided on approach | 10 min |
| [RALPH_EXECUTION_MAP.md](./RALPH_EXECUTION_MAP.md) | You want dependency graph | 10 min |
| [PHASE2_INDEX.md](./PHASE2_INDEX.md) | You want comprehensive reference | 20 min |

---

## Navigation

### If you have 15 minutes
→ Read [spec/IMPLEMENTATION_PLAN.md](./spec/IMPLEMENTATION_PLAN.md)

### If you have 5 minutes
→ Read [PHASE2_QUICK_REF.md](./PHASE2_QUICK_REF.md)

### If you're implementing Story X
→ Look up in [spec/IMPLEMENTATION_PLAN.md](./spec/IMPLEMENTATION_PLAN.md) § Story X
→ Click "Reference" links for detailed spec
→ See "Acceptance Criteria" from PRD_PHASE2.md

### If Ralph fails on Story Y
→ Check [PRD_PHASE2.md](./PRD_PHASE2.md) § Story Y for exact requirements
→ Check [spec/TUI_IMPROVEMENTS.md](./spec/TUI_IMPROVEMENTS.md) § for pattern examples

### If you need to explain this to someone
→ Show [spec/IMPLEMENTATION_PLAN.md](./spec/IMPLEMENTATION_PLAN.md) § Quick Reference table (5 min read)

---

## File Relationships

```
IMPLEMENTATION_PLAN.md (condensed)
    ↓ (cites from)
    ├→ PRD_PHASE2.md (detailed user stories)
    ├→ TUI_IMPROVEMENTS.md (implementation patterns)
    ├→ TESTING_VALIDATION_STRATEGY.md (testing approach)
    ├→ BUBBLETEA.md (best practices)
    ├→ AGENTS.md (code style)
    └→ Code files (pkg/tui/*.go, cmd/service/*.go, etc.)
```

---

## Quick Links by Story

| Story | Implementation Plan | Detailed Spec | Code Location |
|-------|-------------------|---------------|---------------|
| 1: Async Messages | [§Story 1](./spec/IMPLEMENTATION_PLAN.md#story-1-async-messages-fetchmsgfetchcmd) | [PRD §1](./PRD_PHASE2.md#story-1-create-async-message-types-and-commands) | `pkg/tui/async/messages.go` |
| 2: Filterable Spinner | [§Story 2](./spec/IMPLEMENTATION_PLAN.md#story-2-update-filterable-field-with-async--spinner) | [PRD §2](./PRD_PHASE2.md#story-2-update-filterable-field-to-use-async-commands) | `pkg/tui/fields/filterable.go` |
| 3: VHS Recordings | [§Story 3](./spec/IMPLEMENTATION_PLAN.md#story-3-vhs-recording-scripts) | [PRD §3](./PRD_PHASE2.md#story-3-add-vhs-recording-scripts-and-baseline-gifs) | `spec/vhs/*.tape` |
| 4: Teatest Helpers | [§Story 4](./spec/IMPLEMENTATION_PLAN.md#story-4-teatest-integration-test-helpers) | [PRD §4](./PRD_PHASE2.md#story-4-set-up-teatest-integration-test-framework) | `testutil/teatest_helpers.go` |
| 5: Integration Tests | [§Story 5](./spec/IMPLEMENTATION_PLAN.md#story-5-async-integration-tests-for-filterable) | [PRD §5](./PRD_PHASE2.md#story-5-add-async-integration-tests-for-filterable) | `pkg/tui/fields/filterable_test.go` |
| 6: Navigator | [§Story 6](./spec/IMPLEMENTATION_PLAN.md#story-6-navigator-root-model) | [PRD §6](./PRD_PHASE2.md#story-6-create-navigator-root-model) | `pkg/tui/navigator.go` |
| 7: Update Testadd | [§Story 7](./spec/IMPLEMENTATION_PLAN.md#story-7-update-testadd-to-use-navigator) | [PRD §7](./PRD_PHASE2.md#story-7-update-testadd-to-use-navigator) | `cmd/service/worktree_testadd.go` |
| 8: Custom Fields | [§Story 8](./spec/IMPLEMENTATION_PLAN.md#story-8-custom-field-storage-in-workflowstate) | [PRD §8](./PRD_PHASE2.md#story-8-add-custom-field-storage-to-workflowstate) | `pkg/tui/context.go` |
| 9: Merge Workflow | [§Story 9](./spec/IMPLEMENTATION_PLAN.md#story-9-merge-workflow-with-custom-fields) | [PRD §9](./PRD_PHASE2.md#story-9-create-merge-workflow-with-custom-fields) | `pkg/tui/workflows/merge_custom.go` |
| 10: Docs | [§Story 10](./spec/IMPLEMENTATION_PLAN.md#story-10-architecture-documentation) | [PRD §10](./PRD_PHASE2.md#story-10-update-wizard-with-architecture-documentation) | `pkg/tui/ARCHITECTURE.md` |

---

## For Future Reference

- **Why we did this**: 1 hour of clear planning = 15-25 hours of focused execution
- **Why it's condensed**: Verbose docs are hard to act on. Bullet points with citations are easier.
- **Why keep both**: PRD_PHASE2.md has exhaustive acceptance criteria. IMPLEMENTATION_PLAN.md is the quick reference.

---

**You should read**: [spec/IMPLEMENTATION_PLAN.md](./spec/IMPLEMENTATION_PLAN.md) first (15 min)

Then either:
1. **Code it** (Ctrl+F for story number)
2. **Run Ralph** ([QUICK_START_RALPH.md](./QUICK_START_RALPH.md))
3. **Deep dive** (Follow citation links)

# Phase 2: Quick Reference

**TL;DR**: 10 stories in [spec/IMPLEMENTATION_PLAN.md](./spec/IMPLEMENTATION_PLAN.md). Start there.

---

## File Guide

| File | Purpose | Read If |
|------|---------|---------|
| **[spec/IMPLEMENTATION_PLAN.md](./spec/IMPLEMENTATION_PLAN.md)** | 10 stories w/ bullet points | You want action items (15 min) |
| [START_HERE.md](./START_HERE.md) | 3 reading paths | You want to choose your approach |
| [PHASE2_SUMMARY.md](./PHASE2_SUMMARY.md) | High-level strategy | You want executive summary (5 min) |
| [PRD_PHASE2.md](./PRD_PHASE2.md) | Detailed user stories (verbose) | Ralph needs this or deep dive |
| [QUICK_START_RALPH.md](./QUICK_START_RALPH.md) | How to run Ralph | You're ready to execute |
| [RALPH_EXECUTION_MAP.md](./RALPH_EXECUTION_MAP.md) | Story dependency graph | You want to understand sequencing |

---

## For Different Audiences

### I want to CODE this (5 min)
→ Read [spec/IMPLEMENTATION_PLAN.md](./spec/IMPLEMENTATION_PLAN.md) Stories 1-5

### I want to UNDERSTAND this (20 min)
→ Read [spec/IMPLEMENTATION_PLAN.md](./spec/IMPLEMENTATION_PLAN.md) + [spec/BUBBLETEA.md](./spec/BUBBLETEA.md) §1-2

### I want to RUN Ralph (10 min)
→ [QUICK_START_RALPH.md](./QUICK_START_RALPH.md) checklist → run

### I need ALL the details
→ [PHASE2_INDEX.md](./PHASE2_INDEX.md) (old comprehensive guide)

---

## The 10 Stories (Quick)

| # | What | Why |
|---|------|-----|
| 1 | FetchMsg/FetchCmd async messages | Foundation for non-blocking operations |
| 2 | Filterable with spinner | Demonstrates async + loading state |
| 3 | VHS recordings | Visual validation of workflows |
| 4 | Teatest helpers | Reduce test boilerplate |
| 5 | Async integration tests | Validate complete async flow |
| 6 | Navigator root model | Enable multi-screen composition |
| 7 | Update testadd to use Navigator | Simplify multi-screen logic |
| 8 | Custom field storage | Extensible fields without Wizard changes |
| 9 | Merge workflow (demo) | Reference impl using all patterns |
| 10 | Architecture docs | Explain patterns for future devs |

**See** [spec/IMPLEMENTATION_PLAN.md](./spec/IMPLEMENTATION_PLAN.md) for details on each.

---

## Key References

- **Problems**: [spec/REVIEW_SUMMARY.md](./spec/REVIEW_SUMMARY.md)
- **Solutions**: [spec/TUI_IMPROVEMENTS.md](./spec/TUI_IMPROVEMENTS.md)
- **Testing**: [spec/TESTING_VALIDATION_STRATEGY.md](./spec/TESTING_VALIDATION_STRATEGY.md)
- **Best Practices**: [spec/BUBBLETEA.md](./spec/BUBBLETEA.md)
- **Code Guidelines**: [AGENTS.md](./AGENTS.md)

---

## Success Checklist

- [ ] Read [spec/IMPLEMENTATION_PLAN.md](./spec/IMPLEMENTATION_PLAN.md)
- [ ] Understand 10 stories + why they matter
- [ ] Can cite which story fixes which pain point
- [ ] Know where to find detailed spec for any story
- [ ] Ready to run Ralph or code manually

---

**You're here because**: We did 1 hour of planning to create 15-25 hours of focused work.

**Next**: Pick a story from [spec/IMPLEMENTATION_PLAN.md](./spec/IMPLEMENTATION_PLAN.md) and start coding.

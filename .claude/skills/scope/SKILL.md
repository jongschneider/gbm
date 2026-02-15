---
name: scope
description: Plan and decompose work into Linear tickets using the project-manager agent. Creates issue hierarchies with dependencies and validation criteria. No code is written. Use when you need to plan a feature, bug fix, refactor, or any body of work before implementation.
---

# Project Scoping

Plan work and create Linear tickets. No code is written. The output is a set of well-structured
tickets ready for execution via `/execute`.

## Subagents Used

| Agent | Purpose |
|-------|---------|
| `project-manager` | Explores codebase, creates Linear issues with hierarchy and dependencies |
| `staff-engineer` | Only for technical investigation when the PM needs deeper analysis (no code changes) |

## Phase 1: Plan

1. Spawn a `project-manager` subagent with the user's task description:
   ```
   Task(subagent_type="project-manager", prompt="<user's task description>")
   ```

2. The PM will:
   - Initialize its Linear context (team, project, labels, statuses)
   - Explore the codebase to understand scope
   - Create Linear issues with hierarchy, dependencies, and validation criteria
   - Return a summary including:
     - Issue IDs and structure (parent → subtasks)
     - Which tickets are immediately ready (status = "Todo", no unresolved `blockedBy`)
     - Which tickets can be worked in parallel
     - Critical path
     - Any open technical questions or investigation requests

3. Review the PM's output. If the PM surfaced investigation requests that require deeper
   technical analysis, spawn a `staff-engineer` subagent in **exploration mode** (no code
   changes — just investigation and reporting):
   ```
   Task(subagent_type="staff-engineer", prompt="Investigate the following technical questions for planning purposes. Do NOT make any code changes. Read, explore, and report findings.\n\n<investigation requests from PM>")
   ```

4. If investigation was needed, re-spawn the PM with the findings to refine the plan:
   ```
   Task(subagent_type="project-manager", prompt="Refine the existing plan based on these technical findings:\n\n<staff-engineer findings>\n\nUpdate or create additional tickets as needed.")
   ```

## Phase 2: Review

Present the plan summary to the user:

- Total tickets created, with issue IDs
- Issue hierarchy (parent → subtasks)
- Which tickets are ready for immediate execution (no blockers)
- Which tickets can be worked in parallel
- Critical path
- Any assumptions or open questions

The user may request changes to the plan. If so, re-spawn the PM with the feedback. Scoping
is complete when the user is satisfied with the ticket structure.

**Scoping does NOT proceed to execution automatically.** Ask the user if they want to proceed
to execution via `/execute`, but do not start it unprompted.

## Rules

- **Linear is the source of truth.** All ticket state lives in Linear.
- **Dispatch via Task tool.** Use `subagent_type="project-manager"` for planning,
  `subagent_type="staff-engineer"` only for technical investigation.
- **Never implement directly.** The orchestrator dispatches and coordinates. It does not write
  code or run tests itself during scoping.
- **Staff-engineer exploration mode means NO code changes.** When spawning a staff-engineer
  for investigation during scoping, explicitly instruct it to only read, explore, and report.
  It must not edit files, write code, or move any tickets.
- **Discovered work flows through the PM.** If the staff-engineer's investigation surfaces
  additional work, feed it back to the PM to create tickets. Do not create issues directly.
